package webrtc

import (
	"strings"
	"testing"

	"NanoKVM-Server/service/stream/playoutdelay"

	pion "github.com/pion/webrtc/v4"
)

func TestH265PeerConnectionAdvertisesOnlyHardwareCodecAndRTX(t *testing.T) {
	mediaEngine, err := createMediaEngine()
	if err != nil {
		t.Fatalf("create media engine: %v", err)
	}

	peerConnection, err := createPeerConnection(nil, mediaEngine, true)
	if err != nil {
		t.Fatalf("create peer connection: %v", err)
	}
	t.Cleanup(func() {
		if err := peerConnection.Close(); err != nil {
			t.Errorf("close peer connection: %v", err)
		}
	})

	track, err := pion.NewTrackLocalStaticRTP(
		pion.RTPCodecCapability{MimeType: pion.MimeTypeH265},
		"video",
		"pion-video",
	)
	if err != nil {
		t.Fatalf("create H.265 track: %v", err)
	}
	if _, err := peerConnection.AddTrack(track); err != nil {
		t.Fatalf("add H.265 track: %v", err)
	}

	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		t.Fatalf("create offer: %v", err)
	}

	sdp := offer.SDP
	for _, line := range []string{
		"a=rtpmap:96 H265/90000",
		"a=rtcp-fb:96 nack",
		"a=rtcp-fb:96 nack pli",
		"a=rtcp-fb:96 ccm fir",
		"a=rtpmap:97 rtx/90000",
		"a=fmtp:97 apt=96",
	} {
		if !strings.Contains(sdp, line+"\r\n") {
			t.Errorf("offer is missing %q:\n%s", line, sdp)
		}
	}

	if got := strings.Count(sdp, " H265/90000\r\n"); got != 1 {
		t.Errorf("H.265 codec count = %d, want 1:\n%s", got, sdp)
	}
	if got := strings.Count(sdp, " rtx/90000\r\n"); got != 1 {
		t.Errorf("RTX codec count = %d, want 1:\n%s", got, sdp)
	}

	for _, codec := range []string{"VP8/90000", "VP9/90000", "AV1/90000", "H264/90000"} {
		if strings.Contains(sdp, codec) {
			t.Errorf("offer unexpectedly advertises %s:\n%s", codec, sdp)
		}
	}
	if strings.Contains(sdp, "goog-remb") {
		t.Errorf("offer unexpectedly advertises goog-remb:\n%s", sdp)
	}
	if !strings.Contains(sdp, " "+playoutdelay.URI+"\r\n") {
		t.Errorf("offer is missing playout-delay extension:\n%s", sdp)
	}
}

func TestH265MinimalCodecNegotiatesPionDefaultOffer(t *testing.T) {
	browser, err := pion.NewPeerConnection(pion.Configuration{})
	if err != nil {
		t.Fatalf("create browser peer connection: %v", err)
	}
	t.Cleanup(func() {
		if err := browser.Close(); err != nil {
			t.Errorf("close browser peer connection: %v", err)
		}
	})

	if _, err := browser.AddTransceiverFromKind(
		pion.RTPCodecTypeVideo,
		pion.RTPTransceiverInit{Direction: pion.RTPTransceiverDirectionRecvonly},
	); err != nil {
		t.Fatalf("add browser video transceiver: %v", err)
	}
	browserOffer, err := browser.CreateOffer(nil)
	if err != nil {
		t.Fatalf("create browser offer: %v", err)
	}

	mediaEngine, err := createMediaEngine()
	if err != nil {
		t.Fatalf("create media engine: %v", err)
	}
	server, err := createPeerConnection(nil, mediaEngine, true)
	if err != nil {
		t.Fatalf("create server peer connection: %v", err)
	}
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Errorf("close server peer connection: %v", err)
		}
	})

	track, err := pion.NewTrackLocalStaticRTP(
		pion.RTPCodecCapability{MimeType: pion.MimeTypeH265},
		"video",
		"pion-video",
	)
	if err != nil {
		t.Fatalf("create H.265 track: %v", err)
	}
	if _, err := server.AddTrack(track); err != nil {
		t.Fatalf("add H.265 track: %v", err)
	}
	if err := server.SetRemoteDescription(browserOffer); err != nil {
		t.Fatalf("set browser offer: %v", err)
	}

	answer, err := server.CreateAnswer(nil)
	if err != nil {
		t.Fatalf("create server answer: %v", err)
	}

	sdp := answer.SDP
	h265PayloadType := payloadTypeForCodec(t, sdp, "H265")
	rtxPayloadType := payloadTypeForCodec(t, sdp, "rtx")
	for _, line := range []string{
		"a=rtcp-fb:" + h265PayloadType + " nack",
		"a=rtcp-fb:" + h265PayloadType + " nack pli",
		"a=rtcp-fb:" + h265PayloadType + " ccm fir",
		"a=fmtp:" + rtxPayloadType + " apt=" + h265PayloadType,
	} {
		if !strings.Contains(sdp, line+"\r\n") {
			t.Errorf("answer is missing %q:\n%s", line, sdp)
		}
	}

	if got := strings.Count(sdp, " H265/90000\r\n"); got != 1 {
		t.Errorf("negotiated H.265 codec count = %d, want 1:\n%s", got, sdp)
	}
	if got := strings.Count(sdp, " rtx/90000\r\n"); got != 1 {
		t.Errorf("negotiated RTX codec count = %d, want 1:\n%s", got, sdp)
	}
}

func payloadTypeForCodec(t *testing.T, sessionDescription, codec string) string {
	t.Helper()

	suffix := " " + codec + "/90000"
	for _, line := range strings.Split(sessionDescription, "\r\n") {
		if strings.HasPrefix(line, "a=rtpmap:") && strings.HasSuffix(line, suffix) {
			return strings.TrimSuffix(strings.TrimPrefix(line, "a=rtpmap:"), suffix)
		}
	}

	t.Fatalf("SDP is missing %s codec:\n%s", codec, sessionDescription)
	return ""
}
