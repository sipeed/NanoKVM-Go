package common

import (
	"errors"
	"fmt"
)

type AudioPlaybackErrorCode int

const (
	AudioPlaybackOK                 AudioPlaybackErrorCode = 0
	AudioPlaybackErrInvalidArgument AudioPlaybackErrorCode = -1001
	AudioPlaybackErrInvalidState    AudioPlaybackErrorCode = -1002
	AudioPlaybackErrCodec           AudioPlaybackErrorCode = -1003
	AudioPlaybackErrTimeout         AudioPlaybackErrorCode = -1004
	AudioPlaybackErrUnavailable     AudioPlaybackErrorCode = -1005
	AudioPlaybackErrBusy            AudioPlaybackErrorCode = -1006
	AudioPlaybackErrPermission      AudioPlaybackErrorCode = -1007
	AudioPlaybackErrDeviceIO        AudioPlaybackErrorCode = -1008
	AudioPlaybackErrNoMemory        AudioPlaybackErrorCode = -1009
	AudioPlaybackErrInternal        AudioPlaybackErrorCode = -1099
)

type AudioPlaybackFlags uint32

const (
	AudioPlaybackFlagNone          AudioPlaybackFlags = 0
	AudioPlaybackFlagDiscontinuity AudioPlaybackFlags = 1
	AudioPlaybackFlagValidMask     AudioPlaybackFlags = AudioPlaybackFlagDiscontinuity

	AudioPlaybackMaxOpusPacketBytes uint32 = 1500
	AudioPlaybackMaxTimeoutMS       uint32 = 5000
)

type AudioPlaybackError struct {
	Code AudioPlaybackErrorCode
}

func (e *AudioPlaybackError) Error() string {
	if e == nil {
		return "audio playback error"
	}

	return fmt.Sprintf("audio playback: %s (%d)", e.Code, e.Code)
}

func (code AudioPlaybackErrorCode) String() string {
	switch code {
	case AudioPlaybackOK:
		return "ok"
	case AudioPlaybackErrInvalidArgument:
		return "invalid argument"
	case AudioPlaybackErrInvalidState:
		return "invalid state"
	case AudioPlaybackErrCodec:
		return "codec error"
	case AudioPlaybackErrTimeout:
		return "timeout"
	case AudioPlaybackErrUnavailable:
		return "device unavailable"
	case AudioPlaybackErrBusy:
		return "device busy"
	case AudioPlaybackErrPermission:
		return "permission denied"
	case AudioPlaybackErrDeviceIO:
		return "device I/O error"
	case AudioPlaybackErrNoMemory:
		return "out of memory"
	case AudioPlaybackErrInternal:
		return "internal error"
	default:
		return "unknown error"
	}
}

func AudioPlaybackErrorCodeOf(err error) (AudioPlaybackErrorCode, bool) {
	var playbackErr *AudioPlaybackError
	if !errors.As(err, &playbackErr) {
		return AudioPlaybackOK, false
	}

	return playbackErr.Code, true
}

func newAudioPlaybackError(code int) error {
	if code == int(AudioPlaybackOK) {
		return nil
	}

	return &AudioPlaybackError{Code: AudioPlaybackErrorCode(code)}
}
