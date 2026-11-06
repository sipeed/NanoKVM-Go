package ai

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPControlToolSet(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	registerTools(server, nil, nil)
	assertMCPToolSet(t, server, []string{
		"type_text", "press_keys", "move_mouse", "click_mouse", "scroll_mouse", "screenshot",
		"media_session_help", "media_session_create", "media_session_status", "media_session_close",
	})
}

func TestMediaSessionHelpContractIsComplete(t *testing.T) {
	help := newMediaSessionHelp()
	data, err := json.Marshal(help)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, required := range []string{"nanokvm-webrtc-media/v1", "speakerReceive", "micSend", "audio/opus", "48000", "960", "20", "s16le", "offer", "answer", "candidate", "WebSocket", "wss", "bounded", "Barge", "media_session_close"} {
		if !strings.Contains(strings.ToLower(text), strings.ToLower(required)) {
			t.Errorf("help contract is missing %q", required)
		}
	}
	if help.AudioContract.SamplesPerChannelPerFrame != help.AudioContract.RTPClockRate*help.AudioContract.FrameDurationMS/1000 {
		t.Fatal("audio frame size does not match RTP clock and duration")
	}
}

func TestMediaSessionHelpDoesNotSuggestMCPAudioTransport(t *testing.T) {
	help := newMediaSessionHelp()
	if len(help.ControlPlane.NeverCarries) == 0 || help.DataPlane.Transport == "" {
		t.Fatal("control/data-plane boundary must be explicit")
	}
	if help.Lifecycle.DefaultTTLSeconds != 600 || help.Lifecycle.MaximumTTLSeconds != 600 {
		t.Fatal("documented TTL must match media.DefaultTTL")
	}
}

func TestCreateMetadataMarksTokenSensitive(t *testing.T) {
	usage := newTokenUsage()
	if !usage.Opaque || !usage.Sensitive || usage.Log {
		t.Fatal("unsafe token metadata")
	}
	endpoint := newSignalingEndpoint("/api/media/audio/webrtc?media_session=secret")
	if endpoint.Transport != "websocket" || !endpoint.SameHostAsMCP || endpoint.HTTPSScheme != "wss" {
		t.Fatal("incomplete signaling metadata")
	}
}
