//go:build !linux || !cgo

package common

import "testing"

func TestOpenAudioPlaybackUnsupportedPlatform(t *testing.T) {
	playback, err := OpenAudioPlayback()
	if playback != nil {
		t.Fatal("expected no playback on unsupported platform")
	}

	code, ok := AudioPlaybackErrorCodeOf(err)
	if !ok || code != AudioPlaybackErrUnavailable {
		t.Fatalf("unexpected open error: code=%d classified=%t err=%v", code, ok, err)
	}
}

func TestUnsupportedAudioPlaybackStillValidatesArguments(t *testing.T) {
	playback := &AudioPlayback{}

	code, ok := AudioPlaybackErrorCodeOf(
		playback.WriteOpus(nil, 20, AudioPlaybackFlagNone),
	)
	if !ok || code != AudioPlaybackErrInvalidArgument {
		t.Fatalf("unexpected invalid input error: code=%d classified=%t", code, ok)
	}

	code, ok = AudioPlaybackErrorCodeOf(
		playback.WriteOpus([]byte{0x01}, 20, AudioPlaybackFlagNone),
	)
	if !ok || code != AudioPlaybackErrInvalidState {
		t.Fatalf("unexpected unavailable state error: code=%d classified=%t", code, ok)
	}
}
