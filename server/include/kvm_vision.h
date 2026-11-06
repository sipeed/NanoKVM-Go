#ifndef KVM_VISION_H_
#define KVM_VISION_H_

#include <stdint.h>
#ifdef __cplusplus
extern "C" {
#endif

#define IMG_BUFFER_FULL   -3
#define IMG_VENC_ERROR    -2
#define IMG_NOT_EXIST     -1
#define IMG_MJPEG_TYPE    0
#define IMG_H264_TYPE_SPS 1
#define IMG_H264_TYPE_PPS 2
#define IMG_H264_TYPE_IF  3
#define IMG_H264_TYPE_PF  4
#define IMG_H265_TYPE_SPS 5
#define IMG_H265_TYPE_PPS 6
#define IMG_H265_TYPE_IF  7
#define IMG_H265_TYPE_PF  8

#define VENC_CBR 0
#define VENC_VBR 1

typedef struct kvmv_audio_playback kvmv_audio_playback_t;

#define KVMV_AUDIO_FLAG_NONE          UINT32_C(0)
#define KVMV_AUDIO_FLAG_DISCONTINUITY UINT32_C(1)
#define KVMV_AUDIO_FLAG_VALID_MASK    KVMV_AUDIO_FLAG_DISCONTINUITY

#define KVMV_AUDIO_MAX_OPUS_PACKET_BYTES UINT32_C(1500)
#define KVMV_AUDIO_MAX_TIMEOUT_MS        UINT32_C(5000)

#define KVMV_AUDIO_OK                   0
#define KVMV_AUDIO_ERR_INVALID_ARGUMENT (-1001)
#define KVMV_AUDIO_ERR_INVALID_STATE    (-1002)
#define KVMV_AUDIO_ERR_CODEC            (-1003)
#define KVMV_AUDIO_ERR_TIMEOUT          (-1004)
#define KVMV_AUDIO_ERR_UNAVAILABLE      (-1005)
#define KVMV_AUDIO_ERR_BUSY             (-1006)
#define KVMV_AUDIO_ERR_PERMISSION       (-1007)
#define KVMV_AUDIO_ERR_DEVICE_IO        (-1008)
#define KVMV_AUDIO_ERR_NO_MEMORY        (-1009)
#define KVMV_AUDIO_ERR_INTERNAL         (-1099)

void kvmv_init(uint8_t _debug_info_en);
void kvmv_free_all_data(void);
void kvmv_deinit(void);

/**********************************************************************************
 * @name    kvmv_read_img
 * @author  Sipeed BuGu
 * @date    2024/10/25
 * @version R1.0
 * @brief   Acquire the encoded image with auto init
 * @param	_width				@input: 	Output image width
 * @param	_height				@input: 	Output image height
 * @param	_type				@input: 	Encode type
 * @param	_qlty				@input: 	MJPEG: (50-100) | H264:  (500-10000)
 * @param	_pp_kvm_data		@output: 	Encode data
 * @param	_p_kvmv_data_size	@output: 	Encode data size
 * @return
        -3: img buffer full
        -2: VENC Error
        -1: No images were acquired
         0: Acquire MJPEG encoded images
         1: Acquire H264 encoded images(SPS)
         2: Acquire H264 encoded images(PPS)
         3: Acquire H264 encoded images(I)
         4: Acquire H264 encoded images(P)
 **********************************************************************************/
int kvmv_read_img(uint16_t _width,
                  uint16_t _height,
                  uint8_t _type,
                  uint16_t _qlty,
                  uint8_t** _pp_kvm_data,
                  uint32_t* _p_kvmv_data_size);
int kvmv_get_sps_frame(uint8_t** _pp_kvm_data, uint32_t* _p_kvmv_data_size);
int kvmv_get_pps_frame(uint8_t** _pp_kvm_data, uint32_t* _p_kvmv_data_size);
int kvmv_free_data(uint8_t** _pp_kvm_data);
int kvmv_set_fps(uint8_t _fps);
int kvmv_get_fps(void);
int kvmv_set_gop(uint16_t _gop);
int kvmv_set_bitrate(uint16_t _bitrate);
int kvmv_hdmi_control(uint8_t _en);
int kvmv_read_audio(uint8_t** _pp_kvm_data, uint32_t* _p_kvmv_data_size);
void kvmv_reset_audio_capture(void);

/*
 * The caller owns each playback handle and must eventually close it. Calls on
 * one handle must be serialized; close must not race with write_opus. The Opus
 * input is borrowed only for the duration of write_opus. kvmv_deinit() releases
 * the underlying device, while a later close still releases the opaque handle.
 */
int kvmv_audio_playback_open(kvmv_audio_playback_t** out);
int kvmv_audio_playback_write_opus(kvmv_audio_playback_t* playback,
                                   const uint8_t* opus,
                                   uint32_t size,
                                   uint32_t timeout_ms,
                                   uint32_t flags);
void kvmv_audio_playback_close(kvmv_audio_playback_t** playback);
int kvmv_set_rate_control(uint8_t mode);
void kvmv_close_video_stream(void);
/**********************************************************************************
 * @name    kvmv_force_idr
 * @brief   Asynchronously request the encoder to emit an IDR (key) frame as
 * soon as possible. Non-blocking: only signals a dedicated worker thread, which
 * performs the actual (blocking) request off the caller's thread. Bursts are
 * coalesced and rate-limited internally.
 * @return  0 on success (request enqueued)
 **********************************************************************************/
int kvmv_force_idr(void);
#ifdef __cplusplus
}
#endif

#endif  // KVM_VISION_H_
