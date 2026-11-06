//go:build linux && cgo

package common

/*
	#cgo CFLAGS: -I../include
	#cgo LDFLAGS: -L../dl_lib -lkvm
	#include "kvm_vision.h"
*/
import "C"

import (
	"runtime"
	"sync"
	"unsafe"
)

const (
	audioPlaybackABIFlagNone          = AudioPlaybackFlags(C.KVMV_AUDIO_FLAG_NONE)
	audioPlaybackABIFlagDiscontinuity = AudioPlaybackFlags(C.KVMV_AUDIO_FLAG_DISCONTINUITY)
	audioPlaybackABIFlagValidMask     = AudioPlaybackFlags(C.KVMV_AUDIO_FLAG_VALID_MASK)

	audioPlaybackABIMaxOpusPacketBytes = uint32(C.KVMV_AUDIO_MAX_OPUS_PACKET_BYTES)
	audioPlaybackABIMaxTimeoutMS       = uint32(C.KVMV_AUDIO_MAX_TIMEOUT_MS)

	audioPlaybackABIOK              = AudioPlaybackErrorCode(C.KVMV_AUDIO_OK)
	audioPlaybackABIInvalidArgument = AudioPlaybackErrorCode(C.KVMV_AUDIO_ERR_INVALID_ARGUMENT)
	audioPlaybackABIInvalidState    = AudioPlaybackErrorCode(C.KVMV_AUDIO_ERR_INVALID_STATE)
	audioPlaybackABICodec           = AudioPlaybackErrorCode(C.KVMV_AUDIO_ERR_CODEC)
	audioPlaybackABITimeout         = AudioPlaybackErrorCode(C.KVMV_AUDIO_ERR_TIMEOUT)
	audioPlaybackABIUnavailable     = AudioPlaybackErrorCode(C.KVMV_AUDIO_ERR_UNAVAILABLE)
	audioPlaybackABIBusy            = AudioPlaybackErrorCode(C.KVMV_AUDIO_ERR_BUSY)
	audioPlaybackABIPermission      = AudioPlaybackErrorCode(C.KVMV_AUDIO_ERR_PERMISSION)
	audioPlaybackABIDeviceIO        = AudioPlaybackErrorCode(C.KVMV_AUDIO_ERR_DEVICE_IO)
	audioPlaybackABINoMemory        = AudioPlaybackErrorCode(C.KVMV_AUDIO_ERR_NO_MEMORY)
	audioPlaybackABIInternal        = AudioPlaybackErrorCode(C.KVMV_AUDIO_ERR_INTERNAL)
)

type AudioPlayback struct {
	mu     sync.Mutex
	handle *C.kvmv_audio_playback_t
}

func OpenAudioPlayback() (*AudioPlayback, error) {
	playback := &AudioPlayback{}
	result := int(C.kvmv_audio_playback_open(&playback.handle))
	if result != int(AudioPlaybackOK) {
		if playback.handle != nil {
			C.kvmv_audio_playback_close(&playback.handle)
		}
		return nil, newAudioPlaybackError(result)
	}
	if playback.handle == nil {
		return nil, newAudioPlaybackError(int(AudioPlaybackErrInternal))
	}

	return playback, nil
}

func (p *AudioPlayback) WriteOpus(frame []byte, timeoutMS uint32, flags AudioPlaybackFlags) error {
	if p == nil {
		return newAudioPlaybackError(int(AudioPlaybackErrInvalidState))
	}
	if len(frame) == 0 || uint64(len(frame)) > uint64(AudioPlaybackMaxOpusPacketBytes) ||
		timeoutMS == 0 || timeoutMS > AudioPlaybackMaxTimeoutMS ||
		flags&^AudioPlaybackFlagValidMask != 0 {
		return newAudioPlaybackError(int(AudioPlaybackErrInvalidArgument))
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handle == nil {
		return newAudioPlaybackError(int(AudioPlaybackErrInvalidState))
	}

	data := (*C.uint8_t)(unsafe.Pointer(unsafe.SliceData(frame)))

	result := int(C.kvmv_audio_playback_write_opus(
		p.handle,
		data,
		C.uint32_t(len(frame)),
		C.uint32_t(timeoutMS),
		C.uint32_t(flags),
	))
	runtime.KeepAlive(frame)
	return newAudioPlaybackError(result)
}

func (p *AudioPlayback) Close() {
	if p == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	C.kvmv_audio_playback_close(&p.handle)
}
