#include <atomic>
#include <condition_variable>
#include <cstdlib>
#include <filesystem>
#include <limits>
#include <mutex>
#include <stdexcept>
#include <utility>
#include <vector>

#include "pcm_capture.hpp"

#include "opus_encoder.hpp"
#include "pcm_device.hpp"

namespace kvm::audio {
namespace {

AudioCaptureReadResult makeStatus(AudioCaptureReadStatus status)
{
    AudioCaptureReadResult result;
    result.status = status;
    return result;
}

std::string configuredCaptureDevice()
{
    const char* device = std::getenv(AudioCapturer::DEVICE_ENV);
    if (device != nullptr && *device != '\0') {
        return device;
    }
    if (std::filesystem::exists(AudioCapturer::VIRTUAL_SPEAKER_FLAG)) {
        return AudioCapturer::VIRTUAL_SPEAKER_DEVICE;
    }
    return AudioCapturer::DEFAULT_DEVICE;
}

}  // namespace

class AudioCapturer::Impl {
  public:
    explicit Impl(std::string device_name)
        : device_(std::move(device_name), SND_PCM_STREAM_CAPTURE)
        , pcm_buffer_(PERIOD_FRAMES * CHANNELS)
    {
        device_.configure(PcmParameters {
            RATE,
            SND_PCM_FORMAT_S16_LE,
            CHANNELS,
            PERIOD_FRAMES,
        });
        device_.prepare();
    }

    std::uint32_t read(std::uint8_t* output, std::size_t capacity)
    {
        if (output == nullptr || capacity == 0
            || capacity > static_cast<std::size_t>(
                   std::numeric_limits<std::int32_t>::max())) {
            throw std::invalid_argument(
                "Opus output buffer and capacity must be valid");
        }

        device_.readInterleaved(
            pcm_buffer_.data(), PERIOD_FRAMES, stop_requested_);

        return encoder_.encode(
            pcm_buffer_.data(), PERIOD_FRAMES, output, capacity);
    }

    void requestStop() noexcept
    { stop_requested_.store(true, std::memory_order_release); }

  private:
    PcmDevice device_;
    OpusEncoder encoder_;
    std::vector<std::int16_t> pcm_buffer_;
    std::atomic<bool> stop_requested_ {false};
};

AudioCapturer::AudioCapturer()
    : AudioCapturer(configuredCaptureDevice())
{}

AudioCapturer::AudioCapturer(std::string device_name)
    : impl_(std::make_unique<Impl>(std::move(device_name)))
{}

AudioCapturer::~AudioCapturer() = default;

std::uint32_t AudioCapturer::read(std::uint8_t* output, std::size_t capacity)
{ return impl_->read(output, capacity); }

void AudioCapturer::requestStop() noexcept { impl_->requestStop(); }

class AudioCaptureState::Impl {
  public:
    Impl(Factory factory, std::chrono::milliseconds retry_delay)
        : factory_(std::move(factory))
        , retry_delay_(retry_delay)
    {
        if (!factory_) {
            factory_ = []() { return std::make_unique<AudioCapturer>(); };
        }
        if (retry_delay_ < std::chrono::milliseconds::zero()) {
            throw std::invalid_argument(
                "Audio capture retry delay cannot be negative");
        }
    }

    ~Impl() { stop(); }

    void start()
    {
        std::lock_guard<std::mutex> lock(mutex_);
        running_ = true;
        next_retry_ = std::chrono::steady_clock::time_point::min();
    }

    void stop() noexcept
    {
        std::unique_lock<std::mutex> lock(mutex_);
        running_ = false;
        if (capture_ != nullptr) {
            capture_->requestStop();
        }
        condition_.notify_all();
        condition_.wait(lock, [this]() { return !read_active_; });
        capture_.reset();
        next_retry_ = std::chrono::steady_clock::time_point::min();
    }

    AudioCaptureReadResult read(std::uint8_t* output, std::size_t capacity)
    {
        if (output == nullptr || capacity == 0) {
            AudioCaptureReadResult result;
            result.status = AudioCaptureReadStatus::Failed;
            result.error = "Audio output buffer and capacity must be valid";
            return result;
        }

        std::unique_lock<std::mutex> lock(mutex_);
        condition_.wait(lock, [this]() { return !read_active_ || !running_; });

        if (!running_) {
            return makeStatus(AudioCaptureReadStatus::Stopped);
        }

        const auto now = std::chrono::steady_clock::now();
        if (capture_ == nullptr && now < next_retry_) {
            return makeStatus(AudioCaptureReadStatus::RetryLater);
        }

        read_active_ = true;
        const bool needs_initialization = capture_ == nullptr;
        lock.unlock();

        std::unique_ptr<AudioCaptureSource> new_capture;
        try {
            if (needs_initialization) {
                new_capture = factory_();
                if (new_capture == nullptr) {
                    throw std::runtime_error(
                        "Audio capture factory returned a null device");
                }
            }

            lock.lock();
            if (!running_) {
                if (new_capture != nullptr) {
                    new_capture->requestStop();
                }
                finishReadLocked();
                return makeStatus(AudioCaptureReadStatus::Stopped);
            }
            if (capture_ == nullptr) {
                capture_ = std::move(new_capture);
            }
            AudioCaptureSource* capture = capture_.get();
            lock.unlock();

            const std::uint32_t size = capture->read(output, capacity);
            if (size > capacity) {
                throw std::runtime_error(
                    "Audio capture returned more bytes than requested");
            }

            lock.lock();
            if (!running_) {
                finishReadLocked();
                return makeStatus(AudioCaptureReadStatus::Stopped);
            }
            AudioCaptureReadResult result;
            result.status = AudioCaptureReadStatus::Success;
            result.size = size;
            result.device_initialized = needs_initialization;
            finishReadLocked();
            return result;
        } catch (const PcmReadTimeout&) {
            lock.lock();
            const bool stopped = !running_;
            AudioCaptureReadResult result;
            result.status = stopped ? AudioCaptureReadStatus::Stopped
                                    : AudioCaptureReadStatus::NoData;
            result.device_initialized = !stopped && needs_initialization;
            finishReadLocked();
            return result;
        } catch (const PcmOperationCancelled&) {
            lock.lock();
            const bool stopped = !running_;
            resetCaptureAfterFailureLocked();
            finishReadLocked();
            return makeStatus(stopped ? AudioCaptureReadStatus::Stopped
                                      : AudioCaptureReadStatus::Failed);
        } catch (const std::exception& error) {
            lock.lock();
            const bool stopped = !running_;
            resetCaptureAfterFailureLocked();

            AudioCaptureReadResult result;
            result.status = stopped ? AudioCaptureReadStatus::Stopped
                                    : AudioCaptureReadStatus::Failed;
            if (!stopped) {
                result.error = error.what();
            }
            finishReadLocked();
            return result;
        } catch (...) {
            lock.lock();
            const bool stopped = !running_;
            resetCaptureAfterFailureLocked();

            AudioCaptureReadResult result;
            result.status = stopped ? AudioCaptureReadStatus::Stopped
                                    : AudioCaptureReadStatus::Failed;
            if (!stopped) {
                result.error = "Unknown audio capture error";
            }
            finishReadLocked();
            return result;
        }
    }

  private:
    void resetCaptureAfterFailureLocked()
    {
        capture_.reset();
        next_retry_ = std::chrono::steady_clock::now() + retry_delay_;
    }

    void finishReadLocked()
    {
        read_active_ = false;
        condition_.notify_all();
    }

    Factory factory_;
    std::chrono::milliseconds retry_delay_;
    std::mutex mutex_;
    std::condition_variable condition_;
    std::unique_ptr<AudioCaptureSource> capture_;
    std::chrono::steady_clock::time_point next_retry_ {
        std::chrono::steady_clock::time_point::min()};
    bool running_ {true};
    bool read_active_ {false};
};

AudioCaptureState::AudioCaptureState(Factory factory,
                                     std::chrono::milliseconds retry_delay)
    : impl_(std::make_unique<Impl>(std::move(factory), retry_delay))
{}

AudioCaptureState::~AudioCaptureState() = default;

void AudioCaptureState::start() { impl_->start(); }

void AudioCaptureState::stop() noexcept { impl_->stop(); }

AudioCaptureReadResult AudioCaptureState::read(std::uint8_t* output,
                                               std::size_t capacity)
{ return impl_->read(output, capacity); }

}  // namespace kvm::audio
