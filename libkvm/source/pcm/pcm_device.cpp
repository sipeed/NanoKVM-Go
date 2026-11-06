#include <algorithm>
#include <cerrno>
#include <chrono>
#include <cstdint>
#include <utility>

#include "pcm_device.hpp"

namespace kvm::audio {
namespace {

std::string makePcmErrorMessage(const std::string& context, int error_code)
{ return context + ": " + snd_strerror(error_code); }

void checkPcmResult(int result, const std::string& context)
{
    if (result < 0) {
        throw PcmError(context, result);
    }
}

struct PcmHandleDeleter {
    void operator()(snd_pcm_t* handle) const noexcept
    {
        if (handle != nullptr) {
            snd_pcm_close(handle);
        }
    }
};

struct HwParamsDeleter {
    void operator()(snd_pcm_hw_params_t* parameters) const noexcept
    {
        if (parameters != nullptr) {
            snd_pcm_hw_params_free(parameters);
        }
    }
};

struct SwParamsDeleter {
    void operator()(snd_pcm_sw_params_t* parameters) const noexcept
    {
        if (parameters != nullptr) {
            snd_pcm_sw_params_free(parameters);
        }
    }
};

using PcmHandle = std::unique_ptr<snd_pcm_t, PcmHandleDeleter>;
using HwParams = std::unique_ptr<snd_pcm_hw_params_t, HwParamsDeleter>;
using SwParams = std::unique_ptr<snd_pcm_sw_params_t, SwParamsDeleter>;

}  // namespace

PcmError::PcmError(std::string context, int error_code)
    : std::runtime_error(makePcmErrorMessage(context, error_code))
    , error_code_(error_code)
{}

PcmOperationCancelled::PcmOperationCancelled()
    : std::runtime_error("PCM operation cancelled")
{}

PcmReadTimeout::PcmReadTimeout()
    : std::runtime_error("Timed out waiting for PCM capture data")
{}

PcmWriteTimeout::PcmWriteTimeout()
    : std::runtime_error("Timed out waiting for PCM playback space")
{}

std::chrono::steady_clock::time_point PcmReadBackend::now() const
{ return std::chrono::steady_clock::now(); }

std::chrono::steady_clock::time_point PcmWriteBackend::now() const
{ return std::chrono::steady_clock::now(); }

void readInterleavedFrames(void* buffer,
                           snd_pcm_uframes_t frames,
                           std::size_t frame_bytes,
                           const std::atomic<bool>& cancelled,
                           int timeout_ms,
                           PcmReadBackend& backend)
{
    if (buffer == nullptr || frames == 0 || frame_bytes == 0) {
        throw std::invalid_argument(
            "PCM read buffer, frame count, and frame size must be non-zero");
    }
    if (timeout_ms <= 0) {
        throw std::invalid_argument("PCM read timeout must be positive");
    }

    constexpr int MAX_WAIT_SLICE_MS = 50;
    const auto deadline = backend.now() + std::chrono::milliseconds(timeout_ms);
    auto* output = static_cast<std::uint8_t*>(buffer);
    snd_pcm_uframes_t frames_read = 0;

    while (frames_read < frames) {
        if (cancelled.load(std::memory_order_acquire)) {
            throw PcmOperationCancelled();
        }
        if (backend.now() >= deadline) {
            throw PcmReadTimeout();
        }

        const snd_pcm_sframes_t result = backend.read(
            output + frames_read * frame_bytes, frames - frames_read);
        if (result > 0) {
            const auto completed = static_cast<snd_pcm_uframes_t>(result);
            if (completed > frames - frames_read) {
                throw std::runtime_error(
                    "PCM backend returned more frames than requested");
            }
            frames_read += completed;
            continue;
        }

        if (result == 0 || result == -EAGAIN) {
            const auto remaining =
                std::chrono::duration_cast<std::chrono::milliseconds>(
                    deadline - backend.now())
                    .count();
            if (remaining <= 0) {
                throw PcmReadTimeout();
            }
            const int wait_time = static_cast<int>(
                std::min<std::int64_t>(remaining, MAX_WAIT_SLICE_MS));
            const int wait_result = backend.wait(wait_time);
            if (wait_result >= 0) {
                continue;
            }
            if (backend.recover(wait_result) == PcmRecoveryAction::Restarted) {
                frames_read = 0;
            }
            continue;
        }

        if (backend.recover(static_cast<int>(result))
            == PcmRecoveryAction::Restarted) {
            frames_read = 0;
        }
    }
}

void writeInterleavedFrames(const void* buffer,
                            snd_pcm_uframes_t frames,
                            std::size_t frame_bytes,
                            int timeout_ms,
                            PcmWriteBackend& backend)
{
    if (buffer == nullptr || frames == 0 || frame_bytes == 0) {
        throw std::invalid_argument(
            "PCM write buffer, frame count, and frame size must be non-zero");
    }
    if (timeout_ms <= 0) {
        throw std::invalid_argument("PCM write timeout must be positive");
    }

    constexpr int MAX_WAIT_SLICE_MS = 50;
    const auto deadline = backend.now() + std::chrono::milliseconds(timeout_ms);
    const auto* input = static_cast<const std::uint8_t*>(buffer);
    snd_pcm_uframes_t frames_written = 0;

    while (frames_written < frames) {
        if (backend.now() >= deadline) {
            throw PcmWriteTimeout();
        }

        const snd_pcm_sframes_t result = backend.write(
            input + frames_written * frame_bytes, frames - frames_written);
        if (result > 0) {
            const auto completed = static_cast<snd_pcm_uframes_t>(result);
            if (completed > frames - frames_written) {
                throw std::runtime_error(
                    "PCM backend wrote more frames than requested");
            }
            frames_written += completed;
            continue;
        }

        if (result == 0 || result == -EAGAIN) {
            const auto remaining =
                std::chrono::duration_cast<std::chrono::milliseconds>(
                    deadline - backend.now())
                    .count();
            if (remaining <= 0) {
                throw PcmWriteTimeout();
            }
            const int wait_time = static_cast<int>(
                std::min<std::int64_t>(remaining, MAX_WAIT_SLICE_MS));
            const int wait_result = backend.wait(wait_time);
            if (wait_result >= 0) {
                continue;
            }
            if (backend.recover(wait_result) == PcmRecoveryAction::Restarted) {
                // Recovery prepares an empty playback stream. Replay the
                // complete PCM block so a partial pre-XRUN prefix is not lost.
                frames_written = 0;
            }
            continue;
        }

        if (backend.recover(static_cast<int>(result))
            == PcmRecoveryAction::Restarted) {
            frames_written = 0;
        }
    }
}

class PcmDevice::Impl
    : public PcmReadBackend
    , public PcmWriteBackend {
  public:
    Impl(std::string device_name, snd_pcm_stream_t stream, int open_mode)
        : device_name_(std::move(device_name))
    {
        snd_pcm_t* raw_handle = nullptr;
        const int result =
            snd_pcm_open(&raw_handle, device_name_.c_str(), stream, open_mode);
        checkPcmResult(result, "Failed to open PCM device " + device_name_);
        handle_.reset(raw_handle);
    }

    void configure(const PcmParameters& requested)
    {
        if (requested.rate == 0 || requested.channels == 0
            || requested.period_frames == 0
            || (requested.buffer_frames != 0
                && requested.buffer_frames < requested.period_frames)) {
            throw std::invalid_argument(
                "PCM rate, channels, period, and buffer must be valid");
        }

        snd_pcm_hw_params_t* raw_parameters = nullptr;
        checkPcmResult(snd_pcm_hw_params_malloc(&raw_parameters),
                       "Failed to allocate PCM hardware parameters");
        HwParams hardware_parameters(raw_parameters);

        checkPcmResult(
            snd_pcm_hw_params_any(handle_.get(), hardware_parameters.get()),
            "Failed to initialize PCM hardware parameters");
        checkPcmResult(
            snd_pcm_hw_params_set_access(handle_.get(),
                                         hardware_parameters.get(),
                                         SND_PCM_ACCESS_RW_INTERLEAVED),
            "Failed to set PCM access mode");
        checkPcmResult(
            snd_pcm_hw_params_set_format(
                handle_.get(), hardware_parameters.get(), requested.format),
            "Failed to set PCM sample format");
        checkPcmResult(
            snd_pcm_hw_params_set_channels(
                handle_.get(), hardware_parameters.get(), requested.channels),
            "Failed to set PCM channel count");

        unsigned int rate = requested.rate;
        int direction = 0;
        checkPcmResult(
            snd_pcm_hw_params_set_rate_near(
                handle_.get(), hardware_parameters.get(), &rate, &direction),
            "Failed to set PCM sample rate");
        if (rate != requested.rate) {
            throw std::runtime_error(
                "PCM device " + device_name_ + " does not support "
                + std::to_string(requested.rate) + " Hz exactly");
        }

        snd_pcm_uframes_t period_frames = requested.period_frames;
        direction = 0;
        checkPcmResult(
            snd_pcm_hw_params_set_period_size_near(handle_.get(),
                                                   hardware_parameters.get(),
                                                   &period_frames,
                                                   &direction),
            "Failed to set PCM period size");

        snd_pcm_uframes_t buffer_frames = requested.buffer_frames;
        if (buffer_frames > 0) {
            checkPcmResult(
                snd_pcm_hw_params_set_buffer_size_near(
                    handle_.get(), hardware_parameters.get(), &buffer_frames),
                "Failed to set PCM buffer size");
        }
        checkPcmResult(
            snd_pcm_hw_params(handle_.get(), hardware_parameters.get()),
            "Failed to apply PCM hardware parameters");

        checkPcmResult(snd_pcm_hw_params_get_buffer_size(
                           hardware_parameters.get(), &buffer_frames),
                       "Failed to read PCM buffer size");

        parameters_ = requested;
        parameters_.period_frames = period_frames;
        parameters_.buffer_frames = buffer_frames;

        const int sample_bits = snd_pcm_format_physical_width(requested.format);
        if (sample_bits <= 0 || sample_bits % 8 != 0) {
            throw std::runtime_error("Unsupported PCM sample width");
        }
        frame_bytes_ =
            static_cast<std::size_t>(sample_bits / 8) * requested.channels;
    }

    void configureSoftware(snd_pcm_uframes_t start_threshold,
                           snd_pcm_uframes_t avail_min)
    {
        snd_pcm_sw_params_t* raw_parameters = nullptr;
        checkPcmResult(snd_pcm_sw_params_malloc(&raw_parameters),
                       "Failed to allocate PCM software parameters");
        SwParams software_parameters(raw_parameters);

        checkPcmResult(
            snd_pcm_sw_params_current(handle_.get(), software_parameters.get()),
            "Failed to initialize PCM software parameters");
        checkPcmResult(
            snd_pcm_sw_params_set_start_threshold(
                handle_.get(), software_parameters.get(), start_threshold),
            "Failed to set PCM start threshold");
        checkPcmResult(snd_pcm_sw_params_set_avail_min(
                           handle_.get(), software_parameters.get(), avail_min),
                       "Failed to set PCM minimum availability");
        checkPcmResult(
            snd_pcm_sw_params(handle_.get(), software_parameters.get()),
            "Failed to apply PCM software parameters");
    }

    void prepare()
    {
        checkPcmResult(snd_pcm_prepare(handle_.get()),
                       "Failed to prepare PCM device " + device_name_);
    }

    void drop()
    {
        checkPcmResult(snd_pcm_drop(handle_.get()),
                       "Failed to drop PCM device " + device_name_);
    }

    void dropNoThrow() noexcept { snd_pcm_drop(handle_.get()); }

    snd_pcm_sframes_t read(void* buffer, snd_pcm_uframes_t frames) override
    { return snd_pcm_readi(handle_.get(), buffer, frames); }

    snd_pcm_sframes_t write(const void* buffer,
                            snd_pcm_uframes_t frames) override
    { return snd_pcm_writei(handle_.get(), buffer, frames); }

    int wait(int timeout_ms) override
    { return snd_pcm_wait(handle_.get(), timeout_ms); }

    PcmRecoveryAction recover(int error) override
    {
        if (error == -EINTR) {
            return PcmRecoveryAction::Retry;
        }

        if (error == -ESTRPIPE) {
            // snd_pcm_recover() can wait forever while a suspended device
            // repeatedly returns EAGAIN. Capture may be stopped by
            // kvmv_deinit(), so try resume once and fall back to prepare.
            const int resume_result = snd_pcm_resume(handle_.get());
            if (resume_result == 0) {
                return PcmRecoveryAction::Retry;
            }
            prepare();
            return PcmRecoveryAction::Restarted;
        }

        if (error == -EPIPE) {
            checkPcmResult(snd_pcm_recover(handle_.get(), error, 1),
                           "Failed to recover PCM device " + device_name_);
            return PcmRecoveryAction::Restarted;
        }

        throw PcmError("Failed to access PCM device " + device_name_, error);
    }

    void readInterleaved(void* buffer,
                         snd_pcm_uframes_t frames,
                         const std::atomic<bool>& cancelled,
                         int timeout_ms)
    {
        if (frame_bytes_ == 0) {
            throw std::logic_error(
                "PCM device must be configured before reading");
        }
        readInterleavedFrames(
            buffer, frames, frame_bytes_, cancelled, timeout_ms, *this);
    }

    void writeInterleaved(const void* buffer,
                          snd_pcm_uframes_t frames,
                          int timeout_ms)
    {
        if (frame_bytes_ == 0) {
            throw std::logic_error(
                "PCM device must be configured before writing");
        }
        writeInterleavedFrames(buffer, frames, frame_bytes_, timeout_ms, *this);
    }

    const std::string& deviceName() const noexcept { return device_name_; }
    const PcmParameters& parameters() const noexcept { return parameters_; }
    std::size_t frameBytes() const noexcept { return frame_bytes_; }

  private:
    std::string device_name_;
    PcmHandle handle_;
    PcmParameters parameters_ {};
    std::size_t frame_bytes_ {0};
};

PcmDevice::PcmDevice(std::string device_name,
                     snd_pcm_stream_t stream,
                     int open_mode)
    : impl_(std::make_unique<Impl>(std::move(device_name), stream, open_mode))
{}

PcmDevice::~PcmDevice() = default;
PcmDevice::PcmDevice(PcmDevice&&) noexcept = default;
PcmDevice& PcmDevice::operator=(PcmDevice&&) noexcept = default;

void PcmDevice::configure(const PcmParameters& parameters)
{ impl_->configure(parameters); }

void PcmDevice::configureSoftware(snd_pcm_uframes_t start_threshold,
                                  snd_pcm_uframes_t avail_min)
{ impl_->configureSoftware(start_threshold, avail_min); }

void PcmDevice::prepare() { impl_->prepare(); }

void PcmDevice::drop() { impl_->drop(); }

void PcmDevice::dropNoThrow() noexcept { impl_->dropNoThrow(); }

void PcmDevice::readInterleaved(void* buffer,
                                snd_pcm_uframes_t frames,
                                const std::atomic<bool>& cancelled,
                                int timeout_ms)
{ impl_->readInterleaved(buffer, frames, cancelled, timeout_ms); }

void PcmDevice::writeInterleaved(const void* buffer,
                                 snd_pcm_uframes_t frames,
                                 int timeout_ms)
{ impl_->writeInterleaved(buffer, frames, timeout_ms); }

const std::string& PcmDevice::deviceName() const noexcept
{ return impl_->deviceName(); }

const PcmParameters& PcmDevice::parameters() const noexcept
{ return impl_->parameters(); }

std::size_t PcmDevice::frameBytes() const noexcept
{ return impl_->frameBytes(); }

}  // namespace kvm::audio
