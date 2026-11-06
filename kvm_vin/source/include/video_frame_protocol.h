#pragma once

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

#define KVM_VIN_VIDEO_SOCK_PATH "/run/kvm/vin_video.sock"

#define KVM_VIDEO_FRAME_MAGIC 0x4656564BU /* "KVVF" in little-endian memory */
#define KVM_VIDEO_FRAME_VERSION 1U
#define KVM_VIDEO_FRAME_HEADER_LEN 64U
#define KVM_VIDEO_FRAME_FLAG_KEY 0x1U

typedef struct __attribute__((packed)) {
    uint32_t magic;
    uint16_t version;
    uint16_t header_len;
    uint32_t payload_len;
    uint32_t width;
    uint32_t height;
    uint32_t format;
    uint32_t flags;
    uint64_t pts_us;
    uint64_t seq;
    uint32_t fps;
    uint32_t reserved0;
    uint64_t reserved1;
    uint32_t reserved2;
} kvm_video_frame_header_t;

#ifdef __cplusplus
static_assert(sizeof(kvm_video_frame_header_t) == KVM_VIDEO_FRAME_HEADER_LEN,
              "kvm video frame header must be exactly 64 bytes");
#endif

#ifdef __cplusplus
}
#endif
