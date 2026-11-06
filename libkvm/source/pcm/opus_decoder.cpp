#include <array>
#include <limits>
#include <string>
#include <type_traits>

#include "opus_decoder.hpp"

#include <opus/opus.h>

namespace kvm::audio {
namespace {

std::string makeErrorMessage(const char* context, int opus_code)
{
    const char* detail = opus_strerror(opus_code);
    return std::string(context) + ": "
        + (detail == nullptr ? "unknown Opus error" : detail);
}

struct DecoderDeleter {
    void operator()(::OpusDecoder* decoder) const noexcept
    {
        if (decoder != nullptr) {
            opus_decoder_destroy(decoder);
        }
    }
};

using DecoderPtr = std::unique_ptr<::OpusDecoder, DecoderDeleter>;

DecoderPtr createDecoder()
{
    int opus_code = OPUS_OK;
    ::OpusDecoder* decoder = opus_decoder_create(
        OpusDecoder::SAMPLE_RATE, OpusDecoder::CHANNELS, &opus_code);
    if (decoder == nullptr || opus_code != OPUS_OK) {
        if (opus_code == OPUS_OK) {
            opus_code = OPUS_INTERNAL_ERROR;
        }
        throw OpusDecodeError(OpusDecodeErrorStage::Initialization,
                              opus_code,
                              "Failed to create Opus decoder");
    }

    return DecoderPtr(decoder);
}

}  // namespace

OpusDecodeError::OpusDecodeError(OpusDecodeErrorStage stage,
                                 int opus_code,
                                 const char* context)
    : std::runtime_error(makeErrorMessage(context, opus_code))
    , stage_(stage)
    , opus_code_(opus_code)
{}

class OpusDecoder::Impl {
  public:
    Impl()
        : decoder_(createDecoder())
    {
        static_assert(std::is_same_v<opus_int16, std::int16_t>,
                      "libopus opus_int16 must match int16_t");
    }

    DecodedPcm decode(const std::uint8_t* encoded, std::size_t size)
    {
        if (encoded == nullptr || size == 0
            || size > static_cast<std::size_t>(
                   std::numeric_limits<opus_int32>::max())) {
            throw OpusDecodeError(OpusDecodeErrorStage::InvalidInput,
                                  OPUS_BAD_ARG,
                                  "Invalid complete Opus packet");
        }

        const int decoded_frames = opus_decode(decoder_.get(),
                                               encoded,
                                               static_cast<opus_int32>(size),
                                               pcm_.data(),
                                               MAX_FRAMES_PER_CHANNEL,
                                               0);
        if (decoded_frames < 0) {
            throw OpusDecodeError(OpusDecodeErrorStage::Decode,
                                  decoded_frames,
                                  "Failed to decode Opus packet");
        }

        return DecodedPcm {
            pcm_.data(),
            static_cast<std::uint32_t>(decoded_frames),
        };
    }

    void reset()
    {
        const int opus_code =
            opus_decoder_ctl(decoder_.get(), OPUS_RESET_STATE);
        if (opus_code != OPUS_OK) {
            throw OpusDecodeError(OpusDecodeErrorStage::Reset,
                                  opus_code,
                                  "Failed to reset Opus decoder");
        }
    }

  private:
    DecoderPtr decoder_;
    std::array<std::int16_t, MAX_FRAMES_PER_CHANNEL * CHANNELS> pcm_ {};
};

OpusDecoder::OpusDecoder()
    : impl_(std::make_unique<Impl>())
{}

OpusDecoder::~OpusDecoder() = default;

DecodedPcm OpusDecoder::decode(const std::uint8_t* encoded, std::size_t size)
{ return impl_->decode(encoded, size); }

void OpusDecoder::reset() { impl_->reset(); }

}  // namespace kvm::audio
