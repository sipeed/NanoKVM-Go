//go:build linux && cgo

package common

/*
	#cgo CFLAGS: -I../include
	#cgo LDFLAGS: -L../dl_lib -lkvm
	#include "kvm_vision.h"
*/
import "C"
import (
	"strings"
	"sync"
	"unsafe"

	"NanoKVM-Server/config"

	log "github.com/sirupsen/logrus"
)

const (
	RATE_CONTROL_CBR uint8 = iota
	RATE_CONTROL_VBR
)

const (
	IMG_H264_TYPE_SPS uint8 = 1
	IMG_H264_TYPE_PPS uint8 = 2
	IMG_H264_TYPE_IF  uint8 = 3
	IMG_H264_TYPE_PF  uint8 = 4
	IMG_H265_TYPE_SPS uint8 = 5
	IMG_H265_TYPE_PPS uint8 = 6
	IMG_H265_TYPE_IF  uint8 = 7
	IMG_H265_TYPE_PF  uint8 = 8
)

// kvmv_read_img result codes (negative = error/no-data).
const (
	IMG_BUFFER_FULL = -3
	IMG_VENC_ERROR  = -2
	IMG_NOT_EXIST   = -1
)

const (
	STREAM_TYPE_H264_WEBRTC = iota
	STREAM_TYPE_H264_DIRECT
	STREAM_TYPE_H265_WEBRTC
	STREAM_TYPE_H265_DIRECT
)

var (
	kvmVision     *KvmVision
	kvmVisionOnce sync.Once
)

func GetKvmVision() *KvmVision {
	kvmVisionOnce.Do(func() {
		kvmVision = newKvmVision()

		conf := config.GetInstance()
		logLevel := strings.ToLower(conf.Logger.Level)

		logEnable := C.uint8_t(0)
		if logLevel == "debug" {
			logEnable = C.uint8_t(1)
		}

		C.kvmv_init(logEnable)
		log.Debugf("kvm vision initialized")
	})

	return kvmVision
}

func (k *KvmVision) SetRateControl(mode uint8) int {
	if mode != RATE_CONTROL_CBR && mode != RATE_CONTROL_VBR {
		log.Debugf("invalid rate control mode: %d", mode)
		return -1
	}

	result := int(C.kvmv_set_rate_control(
		C.uint8_t(mode),
	))
	if result < 0 {
		log.Debugf("failed to set rate control mode: %d", result)
		return result
	}

	return result
}

func (k *KvmVision) ReadH264(width uint16, height uint16, bitRate uint16) (data []byte, result int) {
	k.frameMu.Lock()
	defer k.frameMu.Unlock()

	var (
		kvmData  *C.uint8_t
		dataSize C.uint32_t
	)

	result = int(C.kvmv_read_img(
		C.uint16_t(width),
		C.uint16_t(height),
		C.uint8_t(IMG_H264_TYPE_SPS),
		C.uint16_t(bitRate),
		&kvmData,
		&dataSize,
	))
	if result < 0 {
		log.Debugf("failed to read H.264: %d", result)
		return
	}
	if !IsH264FrameType(result) {
		log.Debugf("skip non-H.264 frame while reading H.264: %d", result)
		result = IMG_NOT_EXIST
		return
	}
	if kvmData == nil || dataSize == 0 {
		return
	}

	data = C.GoBytes(unsafe.Pointer(kvmData), C.int(dataSize))
	return
}

func (k *KvmVision) ReadH265(width uint16, height uint16, bitRate uint16) (data []byte, result int) {
	k.frameMu.Lock()
	defer k.frameMu.Unlock()

	var (
		kvmData  *C.uint8_t
		dataSize C.uint32_t
	)

	result = int(C.kvmv_read_img(
		C.uint16_t(width),
		C.uint16_t(height),
		C.uint8_t(IMG_H265_TYPE_SPS),
		C.uint16_t(bitRate),
		&kvmData,
		&dataSize,
	))
	if result < 0 {
		log.Debugf("failed to read H.265: %d", result)
		return
	}
	if !IsH265FrameType(result) {
		log.Debugf("skip non-H.265 frame while reading H.265: %d", result)
		result = IMG_NOT_EXIST
		return
	}
	if kvmData == nil || dataSize == 0 {
		return
	}

	data = C.GoBytes(unsafe.Pointer(kvmData), C.int(dataSize))
	return
}

func (k *KvmVision) ReadAudio() (data []byte, result int) {
	var (
		kvmData  *C.uint8_t
		dataSize C.uint32_t
	)

	result = int(C.kvmv_read_audio(&kvmData, &dataSize))
	if result < 0 {
		if result == -1 {
			log.Debugf("failed to read audio: %d", result)
		} else {
			log.Debugf("failed to read audio: %d", result)
		}
		return
	}
	if kvmData == nil || dataSize == 0 {
		return
	}

	data = C.GoBytes(unsafe.Pointer(kvmData), C.int(dataSize))
	return
}

func (k *KvmVision) ResetAudioCapture() {
	C.kvmv_reset_audio_capture()
}

func (k *KvmVision) SetFps(fps uint8) int {
	_fps := C.uint8_t(fps)
	result := C.kvmv_set_fps(_fps)
	return int(result)
}

func (k *KvmVision) GetFps() int {
	return int(C.kvmv_get_fps())
}

func (k *KvmVision) SetHDMI(enable bool) int {
	hdmiEnable := C.uint8_t(0)
	if enable {
		hdmiEnable = C.uint8_t(1)
	}

	result := int(C.kvmv_hdmi_control(hdmiEnable))
	if result < 0 {
		log.Errorf("failed to set hdmi to %t", enable)
		return result
	}

	return result
}

func (k *KvmVision) SetGop(gop uint16) int {
	_gop := C.uint16_t(gop)
	result := C.kvmv_set_gop(_gop)
	return int(result)
}

func (k *KvmVision) SetBitRate(bitRate uint16) int {
	_bitRate := C.uint16_t(bitRate)
	result := C.kvmv_set_bitrate(_bitRate)
	return int(result)
}

func (k *KvmVision) ForceIDR() int {
	return int(C.kvmv_force_idr())
}

func (k *KvmVision) CloseVideoStream() {
	k.frameMu.Lock()
	defer k.frameMu.Unlock()

	C.kvmv_close_video_stream()
}

func (k *KvmVision) Close() {
	k.frameMu.Lock()
	defer k.frameMu.Unlock()

	C.kvmv_deinit()
	log.Debugf("stop kvm vision...")
}
