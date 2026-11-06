package webrtc

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

type WebRTCManager struct {
	tracks        map[*websocket.Conn]*Track
	trackSnapshot atomic.Pointer[[]*Track]
	videoSending  int32
	audioSending  int32
	videoStatus   int
	// keyframeReq is set (1) when a fresh keyframe is needed, either because a
	// downstream decoder asked for one via RTCP PLI/FIR, or because the encoder
	// dropped frames and the P-frame chain was broken. The reader goroutine
	// consumes it and forces the encoder to emit an IDR.
	keyframeReq int32
	mutex       sync.Mutex
	statusMutex sync.Mutex
}

// videoFrame is a captured, already-copied H.265 access-unit fragment moving
// from the capture (reader) goroutine to the send (writer) goroutine. capturedAt
// records when it left the encoder reader so the writer can derive RTP
// timestamps from the capture cadence rather than from send timing.
type videoFrame struct {
	data       []byte
	capturedAt time.Time
}

type Client struct {
	ws    *websocket.Conn
	video *webrtc.PeerConnection
	audio *webrtc.PeerConnection
	mutex sync.Mutex
}

type SignalingHandler struct {
	client *Client
}

type Track struct {
	client          *Client
	videoPacketizer rtp.Packetizer
	video           *webrtc.TrackLocalStaticRTP
	audio           *webrtc.TrackLocalStaticSample
}

type Message struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}
