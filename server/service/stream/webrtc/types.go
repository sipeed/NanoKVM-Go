package webrtc

import (
	"sync"
	"sync/atomic"
	"time"

	"NanoKVM-Server/service/media"
	"github.com/gorilla/websocket"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

type WebRTCManager struct {
	clients        map[*websocket.Conn]*Client
	clientSnapshot atomic.Pointer[[]*Client]
	videoSending   int32
	audioSending   int32
	videoStatus    int
	// keyframeReq is set (1) when a fresh keyframe is needed, either because a
	// downstream decoder asked for one via RTCP PLI/FIR, or because the encoder
	// dropped frames and the P-frame chain was broken. The reader goroutine
	// consumes it and forces the encoder to emit an IDR.
	keyframeReq int32
	mutex       sync.Mutex
	statusMutex sync.Mutex
}

// videoFrame is a captured, already-copied H.264 access-unit fragment moving
// from the capture (reader) goroutine to the send (writer) goroutine. capturedAt
// records when it left the encoder reader so the writer can derive RTP
// timestamps from the capture cadence rather than from send timing.
type videoFrame struct {
	data       []byte
	capturedAt time.Time
}

type Client struct {
	ws           *websocket.Conn
	video        *webrtc.PeerConnection
	audio        *webrtc.PeerConnection
	track        *Track
	capabilities media.Capabilities
	mutex        sync.Mutex
}

type SignalingHandler struct {
	client *Client
}

type Track struct {
	videoPacketizer rtp.Packetizer
	video           *webrtc.TrackLocalStaticRTP
	audio           *webrtc.TrackLocalStaticSample
}

type Message struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}
