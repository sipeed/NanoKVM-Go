#pragma once

#include <atomic>
#include <chrono>
#include <cstddef>
#include <memory>
#include <stdexcept>
#include <string>

#include <alsa/asoundlib.h>

namespace kvm::audio {

struct PcmParameters {
    unsigned int rate;
    snd_pcm_format_t format;
    unsigned int channels;
    snd_pcm_uframes_t period_frames;
    snd_pcm_uframes_t buffer_frames {0};
};

class PcmError : public std::runtime_error {
  public:
    PcmError(std::string context, int error_code);

    int errorCode() const noexcept { return error_code_; }

  private:
    int error_code_;
};

class PcmOperationCancelled : public std::runtime_error {
  public:
    PcmOperationCancelled();
};

class PcmReadTimeout : public std::runtime_error {
  public:
    PcmReadTimeout();
};

class PcmWriteTimeout : public std::runtime_error {
  public:
    PcmWriteTimeout();
};

enum class PcmRecoveryAction {
    Retry,
    Restarted,
};

class PcmReadBackend {
  public:
    virtual ~PcmReadBackend() = default;

    virtual snd_pcm_sframes_t read(void* buffer, snd_pcm_uframes_t frames) = 0;
    virtual int wait(int timeout_ms) = 0;
    virtual PcmRecoveryAction recover(int error) = 0;
    virtual std::chrono::steady_clock::time_point now() const;
};

class PcmWriteBackend {
  public:
    virtual ~PcmWriteBackend() = default;

    virtual snd_pcm_sframes_t write(const void* buffer,
                                    snd_pcm_uframes_t frames) = 0;
    virtual int wait(int timeout_ms) = 0;
    virtual PcmRecoveryAction recover(int error) = 0;
    virtual std::chrono::steady_clock::time_point now() const;
};

void readInterleavedFrames(void* buffer,
                           snd_pcm_uframes_t frames,
                           std::size_t frame_bytes,
                           const std::atomic<bool>& cancelled,
                           int timeout_ms,
                           PcmReadBackend& backend);

void writeInterleavedFrames(const void* buffer,
                            snd_pcm_uframes_t frames,
                            std::size_t frame_bytes,
                            int timeout_ms,
                            PcmWriteBackend& backend);

class PcmDevice {
  public:
    PcmDevice(std::string device_name,
              snd_pcm_stream_t stream,
              int open_mode = SND_PCM_NONBLOCK);
    ~PcmDevice();

    PcmDevice(const PcmDevice&) = delete;
    PcmDevice& operator=(const PcmDevice&) = delete;
    PcmDevice(PcmDevice&&) noexcept;
    PcmDevice& operator=(PcmDevice&&) noexcept;

    void configure(const PcmParameters& parameters);
    void configureSoftware(snd_pcm_uframes_t start_threshold,
                           snd_pcm_uframes_t avail_min);
    void prepare();
    void drop();
    void dropNoThrow() noexcept;

    void readInterleaved(void* buffer,
                         snd_pcm_uframes_t frames,
                         const std::atomic<bool>& cancelled,
                         int timeout_ms = 200);
    void writeInterleaved(const void* buffer,
                          snd_pcm_uframes_t frames,
                          int timeout_ms);

    const std::string& deviceName() const noexcept;
    const PcmParameters& parameters() const noexcept;
    std::size_t frameBytes() const noexcept;

  private:
    class Impl;
    std::unique_ptr<Impl> impl_;
};

}  // namespace kvm::audio
