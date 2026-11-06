//go:build linux && cgo

package common

import "testing"

func TestAudioPlaybackGoConstantsMatchCABI(t *testing.T) {
	flagPairs := []struct {
		name    string
		goValue AudioPlaybackFlags
		cValue  AudioPlaybackFlags
	}{
		{"none", AudioPlaybackFlagNone, audioPlaybackABIFlagNone},
		{"discontinuity", AudioPlaybackFlagDiscontinuity, audioPlaybackABIFlagDiscontinuity},
		{"valid mask", AudioPlaybackFlagValidMask, audioPlaybackABIFlagValidMask},
	}
	for _, pair := range flagPairs {
		if pair.goValue != pair.cValue {
			t.Fatalf("%s flag drifted: Go=%d C=%d", pair.name, pair.goValue, pair.cValue)
		}
	}

	if AudioPlaybackMaxOpusPacketBytes != audioPlaybackABIMaxOpusPacketBytes ||
		AudioPlaybackMaxTimeoutMS != audioPlaybackABIMaxTimeoutMS {
		t.Fatalf("audio playback limit constants drifted from C ABI")
	}

	errorPairs := []struct {
		name    string
		goValue AudioPlaybackErrorCode
		cValue  AudioPlaybackErrorCode
	}{
		{"ok", AudioPlaybackOK, audioPlaybackABIOK},
		{"invalid argument", AudioPlaybackErrInvalidArgument, audioPlaybackABIInvalidArgument},
		{"invalid state", AudioPlaybackErrInvalidState, audioPlaybackABIInvalidState},
		{"codec", AudioPlaybackErrCodec, audioPlaybackABICodec},
		{"timeout", AudioPlaybackErrTimeout, audioPlaybackABITimeout},
		{"unavailable", AudioPlaybackErrUnavailable, audioPlaybackABIUnavailable},
		{"busy", AudioPlaybackErrBusy, audioPlaybackABIBusy},
		{"permission", AudioPlaybackErrPermission, audioPlaybackABIPermission},
		{"device I/O", AudioPlaybackErrDeviceIO, audioPlaybackABIDeviceIO},
		{"no memory", AudioPlaybackErrNoMemory, audioPlaybackABINoMemory},
		{"internal", AudioPlaybackErrInternal, audioPlaybackABIInternal},
	}
	for _, pair := range errorPairs {
		if pair.goValue != pair.cValue {
			t.Fatalf("%s error drifted: Go=%d C=%d", pair.name, pair.goValue, pair.cValue)
		}
	}
}

func requireAudioPlaybackCode(t *testing.T, err error, expected AudioPlaybackErrorCode) {
	t.Helper()
	code, ok := AudioPlaybackErrorCodeOf(err)
	if !ok || code != expected {
		t.Fatalf("unexpected playback error: code=%d classified=%t err=%v", code, ok, err)
	}
}

func TestAudioPlaybackWriteValidatesBeforeCallingC(t *testing.T) {
	playback := &AudioPlayback{}
	validFrame := []byte{0x01}

	requireAudioPlaybackCode(t,
		playback.WriteOpus(nil, 20, AudioPlaybackFlagNone),
		AudioPlaybackErrInvalidArgument)
	requireAudioPlaybackCode(t,
		playback.WriteOpus(make([]byte, AudioPlaybackMaxOpusPacketBytes+1), 20, AudioPlaybackFlagNone),
		AudioPlaybackErrInvalidArgument)
	requireAudioPlaybackCode(t,
		playback.WriteOpus(validFrame, 0, AudioPlaybackFlagNone),
		AudioPlaybackErrInvalidArgument)
	requireAudioPlaybackCode(t,
		playback.WriteOpus(validFrame, AudioPlaybackMaxTimeoutMS+1, AudioPlaybackFlagNone),
		AudioPlaybackErrInvalidArgument)
	requireAudioPlaybackCode(t,
		playback.WriteOpus(validFrame, 20, AudioPlaybackFlagValidMask<<1),
		AudioPlaybackErrInvalidArgument)

	requireAudioPlaybackCode(t,
		playback.WriteOpus(validFrame, 20, AudioPlaybackFlagNone),
		AudioPlaybackErrInvalidState)
}
