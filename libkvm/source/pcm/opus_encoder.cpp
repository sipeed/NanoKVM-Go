#include <limits>
#include <string>
#include <type_traits>

#include "opus_encoder.hpp"

#include <opus/opus.h>

namespace kvm::audio {
namespace {

std::string makeErrorMessage(const char* context, int opus_code)
{
    const char* detail = opus_strerror(opus_code);
    return std::string(context) + ": "
        + (detail == nullptr ? "unknown Opus error" : detail);
}

struct EncoderDeleter {
    void operator()(::OpusEncoder* encoder) const noexcept
    {
        if (encoder != nullptr) {
            opus_encoder_destroy(encoder);
        }
    }
};

using EncoderPtr = std::unique_ptr<::OpusEncoder, EncoderDeleter>;

EncoderPtr createEncoder()
{
    int opus_code = OPUS_OK;
    ::OpusEncoder* encoder = opus_encoder_create(OpusEncoder::SAMPLE_RATE,
                                                 OpusEncoder::CHANNELS,
                                                 OPUS_APPLICATION_AUDIO,
                                                 &opus_code);
    if (encoder == nullptr || opus_code != OPUS_OK) {
        if (encoder != nullptr) {
            opus_encoder_destroy(encoder);
        }
        if (opus_code == OPUS_OK) {
            opus_code = OPUS_INTERNAL_ERROR;
        }
        throw OpusEncodeError(OpusEncodeErrorStage::Initialization,
                              opus_code,
                              "Failed to create Opus encoder");
    }

    return EncoderPtr(encoder);
}

void checkConfiguration(int opus_code, const char* context)
{
    if (opus_code != OPUS_OK) {
        throw OpusEncodeError(
            OpusEncodeErrorStage::Configuration, opus_code, context);
    }
}

}  // namespace

OpusEncodeError::OpusEncodeError(OpusEncodeErrorStage stage,
                                 int opus_code,
                                 const char* context)
    : std::runtime_error(makeErrorMessage(context, opus_code))
    , stage_(stage)
    , opus_code_(opus_code)
{}

class OpusEncoder::Impl {
  public:
    Impl()
        : encoder_(createEncoder())
    {
        static_assert(std::is_same_v<opus_int16, std::int16_t>,
                      "libopus opus_int16 must match int16_t");

        checkConfiguration(
            opus_encoder_ctl(encoder_.get(), OPUS_SET_BITRATE(BITRATE)),
            "Failed to set Opus bitrate");
        checkConfiguration(
            opus_encoder_ctl(encoder_.get(), OPUS_SET_COMPLEXITY(COMPLEXITY)),
            "Failed to set Opus complexity");
        checkConfiguration(
            opus_encoder_ctl(encoder_.get(),
                             OPUS_SET_MAX_BANDWIDTH(OPUS_BANDWIDTH_FULLBAND)),
            "Failed to set Opus bandwidth");
        checkConfiguration(opus_encoder_ctl(encoder_.get(),
                                            OPUS_SET_SIGNAL(OPUS_SIGNAL_MUSIC)),
                           "Failed to set Opus signal type");
    }

    std::uint32_t encode(const std::int16_t* pcm,
                         std::uint32_t frames,
                         std::uint8_t* output,
                         std::size_t capacity)
    {
        if (pcm == nullptr || frames == 0
            || frames
                > static_cast<std::uint32_t>(std::numeric_limits<int>::max())
            || output == nullptr || capacity == 0
            || capacity > static_cast<std::size_t>(
                   std::numeric_limits<opus_int32>::max())) {
            throw OpusEncodeError(OpusEncodeErrorStage::InvalidInput,
                                  OPUS_BAD_ARG,
                                  "Invalid PCM or Opus output buffer");
        }

        const int encoded_bytes =
            opus_encode(encoder_.get(),
                        pcm,
                        static_cast<int>(frames),
                        output,
                        static_cast<opus_int32>(capacity));
        if (encoded_bytes < 0) {
            throw OpusEncodeError(OpusEncodeErrorStage::Encode,
                                  encoded_bytes,
                                  "Failed to encode Opus frame");
        }

        return static_cast<std::uint32_t>(encoded_bytes);
    }

  private:
    EncoderPtr encoder_;
};

OpusEncoder::OpusEncoder()
    : impl_(std::make_unique<Impl>())
{}

OpusEncoder::~OpusEncoder() = default;

std::uint32_t OpusEncoder::encode(const std::int16_t* pcm,
                                  std::uint32_t frames,
                                  std::uint8_t* output,
                                  std::size_t capacity)
{ return impl_->encode(pcm, frames, output, capacity); }

}  // namespace kvm::audio
