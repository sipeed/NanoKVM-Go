package common

import (
	"errors"
	"testing"
)

func TestAudioPlaybackErrorCodeOf(t *testing.T) {
	err := newAudioPlaybackError(int(AudioPlaybackErrCodec))
	if err == nil {
		t.Fatal("expected codec error")
	}

	code, ok := AudioPlaybackErrorCodeOf(err)
	if !ok {
		t.Fatal("expected AudioPlaybackError")
	}
	if code != AudioPlaybackErrCodec {
		t.Fatalf("unexpected error code: got %d want %d", code, AudioPlaybackErrCodec)
	}

	var typedErr *AudioPlaybackError
	if !errors.As(err, &typedErr) {
		t.Fatal("expected errors.As to preserve AudioPlaybackError")
	}
}

func TestAudioPlaybackSuccessHasNoError(t *testing.T) {
	if err := newAudioPlaybackError(int(AudioPlaybackOK)); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestAudioPlaybackConstantsMatchABI(t *testing.T) {
	if AudioPlaybackFlagValidMask != AudioPlaybackFlagDiscontinuity {
		t.Fatalf("unexpected flag mask: %d", AudioPlaybackFlagValidMask)
	}
	if AudioPlaybackMaxOpusPacketBytes != 1500 {
		t.Fatalf("unexpected max Opus packet size: %d", AudioPlaybackMaxOpusPacketBytes)
	}
	if AudioPlaybackMaxTimeoutMS != 5000 {
		t.Fatalf("unexpected max timeout: %d", AudioPlaybackMaxTimeoutMS)
	}
}

func TestNilAudioPlaybackIsClosed(t *testing.T) {
	var playback *AudioPlayback
	err := playback.WriteOpus([]byte{0x01}, 20, AudioPlaybackFlagNone)
	code, ok := AudioPlaybackErrorCodeOf(err)
	if !ok || code != AudioPlaybackErrInvalidState {
		t.Fatalf("unexpected nil playback error: code=%d classified=%t err=%v", code, ok, err)
	}

	playback.Close()
}
