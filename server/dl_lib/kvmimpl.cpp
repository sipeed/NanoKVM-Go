
#include <stdexcept>
#include <stdio.h>
#include <stdlib.h>

#include "kvm_vision.h"

#ifdef __cplusplus
extern "C" {
#endif

// #ifdef KVMIMPL_THROW
// #pragma message("Throw Impl")
// #else
// #pragma message("Empty Impl")
// #endif

static int __throw_error() {
#ifdef KVMIMPL_THROW
  throw std::runtime_error(
      "[Error] This shared library is exclusively for linking with GO "
      "applications. "
      "Please compile the implementation from support/libkvm directory.");
  return -1;
#else
  printf("[Error] This shared library is exclusively for linking with GO "
         "applications. "
         "Please compile the implementation from support/libkvm directory.");
  exit(-1);
  return -1;
#endif
}

void kvmv_init(uint8_t _debug_info_en) {
#ifdef KVMIMPL_ALLOW_INIT
  // Host-side test double (CI only): initialization succeeds so Go tests can
  // exercise code paths that merely call GetKvmVision(). Every capture and
  // encode entry point below still refuses loudly, so a stub that reaches a
  // real device cannot silently serve video.
  (void)_debug_info_en;
#else
  __throw_error();
#endif
}

void kvmv_free_all_data() {
  __throw_error();
}

void kvmv_deinit() {
  __throw_error();
}

int kvmv_read_img(uint16_t _width,
                  uint16_t _height,
                  uint8_t _type,
                  uint16_t _qlty,
                  uint8_t** _pp_kvm_data,
                  uint32_t* _p_kvmv_data_size) {
  return __throw_error();
}

int kvmv_get_sps_frame(uint8_t** _pp_kvm_data, uint32_t* _p_kvmv_data_size) {
  return __throw_error();
}

int kvmv_get_pps_frame(uint8_t** _pp_kvm_data, uint32_t* _p_kvmv_data_size) {
  return __throw_error();
}

int kvmv_free_data(uint8_t** _pp_kvm_data) {
  return __throw_error();
}

int kvmv_set_fps(uint8_t _fps) {
  return __throw_error();
}

int kvmv_get_fps(void) {
  return __throw_error();
}

int kvmv_set_gop(uint16_t _gop) {
  return __throw_error();
}

int kvmv_set_bitrate(uint16_t _bitrate) {
  return __throw_error();
}

int kvmv_hdmi_control(uint8_t _en) {
  return __throw_error();
}

int kvmv_read_audio(uint8_t** _pp_kvm_data, uint32_t* _p_kvmv_data_size) {
  return __throw_error();
}

void kvmv_reset_audio_capture(void) {
  __throw_error();
}

int kvmv_set_rate_control(uint8_t mode) {
  return __throw_error();
}

void kvmv_close_video_stream(void) {
  __throw_error();
}

int kvmv_force_idr(void) {
  return __throw_error();
}

int kvmv_audio_playback_open(kvmv_audio_playback_t** out) {
  return __throw_error();
}

int kvmv_audio_playback_write_opus(kvmv_audio_playback_t* playback,
                                   const uint8_t* opus,
                                   uint32_t size,
                                   uint32_t timeout_ms,
                                   uint32_t flags) {
  return __throw_error();
}

void kvmv_audio_playback_close(kvmv_audio_playback_t** playback) {
  __throw_error();
}

#ifdef __cplusplus
}
#endif
