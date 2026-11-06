#include <cerrno>
#include <limits>
#include <stdexcept>
#include <utility>

#include "pcm_playback.hpp"

#include "pcm_device.hpp"

namespace kvm::audio {

class PcmPlayback::Impl {
  public:
    explicit Impl(std::string device_name)
    {
        device_ = std::make_unique<PcmDevice>(std::move(device_name),
                                              SND_PCM_STREAM_PLAYBACK,
                                              SND_PCM_NONBLOCK | SND_PCM_EINTR);
        device_->configure(PcmParameters {
            RATE,
            SND_PCM_FORMAT_S16_LE,
            CHANNELS,
            PERIOD_FRAMES,
            BUFFER_FRAMES,
        });

        const auto& negotiated = device_->parameters();
        device_->configureSoftware(negotiated.period_frames,
                                   negotiated.period_frames);
        device_->prepare();

        parameters_.rate = negotiated.rate;
        parameters_.channels = negotiated.channels;
        parameters_.period_frames =
            static_cast<std::uint32_t>(negotiated.period_frames);
        parameters_.buffer_frames =
            static_cast<std::uint32_t>(negotiated.buffer_frames);
        parameters_.frame_bytes =
            static_cast<std::uint32_t>(device_->frameBytes());
    }

    ~Impl() { close(); }

    void write(const std::int16_t* pcm,
               std::uint32_t frames,
               std::uint32_t timeout_ms)
    {
        if (pcm == nullptr || frames == 0 || timeout_ms == 0
            || timeout_ms
                > static_cast<std::uint32_t>(std::numeric_limits<int>::max())) {
            throw std::invalid_argument(
                "PCM playback data, frame count, and timeout must be valid");
        }
        if (device_ == nullptr) {
            throw PcmError("PCM playback is closed", -EBADFD);
        }

        try {
            device_->writeInterleaved(
                pcm, frames, static_cast<int>(timeout_ms));
        } catch (const PcmWriteTimeout&) {
            // A timed-out block may already be partially queued. Discard it so
            // the next successful frame cannot play behind stale PCM.
            resetNoThrow();
            throw;
        }
    }

    void reset()
    {
        if (device_ == nullptr) {
            throw PcmError("PCM playback is closed", -EBADFD);
        }
        device_->drop();
        device_->prepare();
    }

    void close() noexcept
    {
        if (device_ == nullptr) {
            return;
        }
        device_->dropNoThrow();
        device_.reset();
    }

    const PcmPlaybackParameters& parameters() const noexcept
    { return parameters_; }

  private:
    void resetNoThrow() noexcept
    {
        if (device_ == nullptr) {
            return;
        }
        device_->dropNoThrow();
        try {
            device_->prepare();
        } catch (...) {}
    }

    std::unique_ptr<PcmDevice> device_;
    PcmPlaybackParameters parameters_;
};

PcmPlayback::PcmPlayback(std::string device_name)
    : impl_(std::make_unique<Impl>(std::move(device_name)))
{}

PcmPlayback::~PcmPlayback() = default;

void PcmPlayback::write(const std::int16_t* pcm,
                        std::uint32_t frames,
                        std::uint32_t timeout_ms)
{ impl_->write(pcm, frames, timeout_ms); }

void PcmPlayback::reset() { impl_->reset(); }

void PcmPlayback::close() noexcept { impl_->close(); }

const PcmPlaybackParameters& PcmPlayback::parameters() const noexcept
{ return impl_->parameters(); }

}  // namespace kvm::audio
