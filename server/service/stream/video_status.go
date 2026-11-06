package stream

import (
	"encoding/json"
	"strconv"
)

const (
	VideoStatusEvent            = "video-status"
	VideoStatusUnknown          = 0
	VideoStatusNormal           = 1
	VideoStatusInconsistentMode = -4
	VideoStatusCaptureDisabled  = -5
)

type VideoStatusMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

// NormalizeVideoStatus maps every successful capture result to the wire-level
// normal status while preserving the negative KvmVision error codes. A
// disabled capture setting takes precedence over the underlying capture result.
func NormalizeVideoStatus(result int, captureEnabled bool) int {
	if !captureEnabled {
		return VideoStatusCaptureDisabled
	}

	if result >= 0 {
		return VideoStatusNormal
	}

	return result
}

func IsKnownVideoStatus(status int) bool {
	return status != VideoStatusUnknown
}

func VideoStatusData(status int) string {
	return strconv.Itoa(status)
}

func MarshalVideoStatus(status int) ([]byte, error) {
	return json.Marshal(VideoStatusMessage{
		Event: VideoStatusEvent,
		Data:  VideoStatusData(status),
	})
}

// ReadVideoFrameForMode prevents an old transport from continuing to drain
// frames after another client changes the active video mode. The second check
// discards a frame if the mode changes while the read is blocked.
func ReadVideoFrameForMode(
	expected uint8,
	current func() uint8,
	readFrame func() ([]byte, int),
) ([]byte, int, bool) {
	if current() != expected {
		return nil, VideoStatusInconsistentMode, false
	}

	data, result := readFrame()
	if current() != expected {
		return nil, VideoStatusInconsistentMode, false
	}

	return data, result, true
}
