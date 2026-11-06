package opus

import (
	"bytes"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pion/rtp"
)

func marshalMicrophonePacket(t *testing.T, sequence uint16, timestamp uint32, payload byte) []byte {
	return marshalMicrophonePayload(t, sequence, timestamp, []byte{payload})
}

func marshalMicrophonePayload(t *testing.T, sequence uint16, timestamp uint32, payload []byte) []byte {
	t.Helper()
	packet := &rtp.Packet{
		Header: rtp.Header{
			Version:        2,
			Marker:         true,
			PayloadType:    111,
			SequenceNumber: sequence,
			Timestamp:      timestamp,
			SSRC:           1,
		},
		Payload: payload,
	}
	raw, err := packet.Marshal()
	if err != nil {
		t.Fatalf("failed to marshal RTP packet: %v", err)
	}
	return raw
}

func TestUnmarshalOwnedPacketSurvivesReadBufferReuse(t *testing.T) {
	want := []byte{0xfc, 0x7d, 0x0d, 0x93, 0x80, 0x32, 0xdf, 0x18}
	first := marshalMicrophonePayload(t, 1, 0, want)
	second := marshalMicrophonePayload(t, 2, 960, []byte{0xfc, 0x53, 0xc1})
	buffer := make([]byte, max(len(first), len(second)))
	copy(buffer, first)
	packet, err := unmarshalOwnedPacket(buffer[:len(first)])
	if err != nil {
		t.Fatal(err)
	}
	copy(buffer, second) // simulate the next track.Read reusing the same storage
	if !bytes.Equal(packet.Payload, want) {
		t.Fatalf("payload corrupted by reused RTP read buffer: got %x want %x", packet.Payload, want)
	}
}

func TestReceiveMicrophonePacketsStopsCurrentSessionOnReadEnd(t *testing.T) {
	opened := make(chan struct{})
	var closeCalls atomic.Int32
	player := newAudioInputPlayer(func() OpusSink {
		return &fakeOpusSink{
			openFn: func() error {
				close(opened)
				return nil
			},
			closeFn: func() { closeCalls.Add(1) },
		}
	}, testAudioPlayerConfig(2))
	session := startTestSession(t, player)
	t.Cleanup(session.Stop)
	select {
	case <-opened:
	case <-time.After(time.Second):
		t.Fatal("session did not open")
	}

	receiveMicrophonePackets(func([]byte) (int, error) {
		return 0, io.EOF
	}, session)

	if session.IsActive() || player.IsActive() {
		t.Fatal("track read end did not stop the current session")
	}
	if closeCalls.Load() != 1 {
		t.Fatalf("unexpected Close count: %d", closeCalls.Load())
	}
}

func TestReceiveMicrophonePacketsStopsReadingSupersededSession(t *testing.T) {
	player := newAudioInputPlayer(func() OpusSink {
		return &fakeOpusSink{}
	}, testAudioPlayerConfig(2))
	oldSession := startTestSession(t, player)
	replacement := startTestSession(t, player)
	t.Cleanup(replacement.Stop)

	packets := [][]byte{
		marshalMicrophonePacket(t, 1, 0, 1),
		marshalMicrophonePacket(t, 2, 960, 2),
	}
	reads := 0
	readPastSample := false
	receiveMicrophonePackets(func(buffer []byte) (int, error) {
		if reads >= len(packets) {
			readPastSample = true
			return 0, io.EOF
		}
		packet := packets[reads]
		reads++
		return copy(buffer, packet), nil
	}, oldSession)

	if readPastSample {
		t.Fatal("superseded consumer continued reading after Enqueue became inactive")
	}
	if reads != 2 {
		t.Fatalf("unexpected packet read count: %d", reads)
	}
	if !replacement.IsActive() {
		t.Fatal("old consumer cleanup stopped the replacement session")
	}
}
