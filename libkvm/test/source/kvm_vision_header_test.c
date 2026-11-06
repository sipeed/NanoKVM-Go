#include "libkvm/kvm_vision.h"

int main(void)
{
    kvmv_audio_playback_t* playback = 0;
    uint32_t flags = KVMV_AUDIO_FLAG_NONE;

    flags |= KVMV_AUDIO_FLAG_DISCONTINUITY;
    if (playback != 0 || flags != KVMV_AUDIO_FLAG_VALID_MASK) {
        return 1;
    }
    if (KVMV_AUDIO_MAX_OPUS_PACKET_BYTES != UINT32_C(1500)
        || KVMV_AUDIO_MAX_TIMEOUT_MS != UINT32_C(5000)) {
        return 1;
    }
    return KVMV_AUDIO_OK;
}
