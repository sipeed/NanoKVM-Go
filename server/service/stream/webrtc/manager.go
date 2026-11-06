package webrtc

import (
	"runtime"
	"sync/atomic"
	"time"

	"NanoKVM-Server/common"
	"NanoKVM-Server/config"
	"NanoKVM-Server/service/stream"
	"NanoKVM-Server/service/stream/opus"

	"github.com/pion/webrtc/v4"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4/pkg/media"
	log "github.com/sirupsen/logrus"
)

func NewWebRTCManager() *WebRTCManager {
	m := &WebRTCManager{
		clients:      make(map[*websocket.Conn]*Client),
		videoSending: 0,
		audioSending: 0,
		videoStatus:  0,
	}
	m.updateClientSnapshotLocked()

	return m
}

func (m *WebRTCManager) watchVideoMode(subscription *stream.VideoModeSubscription) {
	for range subscription.C {
		if m.GetClientCount() == 0 {
			continue
		}
		if common.GetKvmVision().StreamType() != common.STREAM_TYPE_H264_WEBRTC {
			m.updateStatus(stream.VideoStatusInconsistentMode)
		}
	}
}

func (m *WebRTCManager) AddClient(ws *websocket.Conn, client *Client) {
	m.mutex.Lock()
	m.clients[ws] = client
	total := m.updateClientSnapshotLocked()
	if stream.IsKnownVideoStatus(m.videoStatus) {
		if err := client.WriteMessage(stream.VideoStatusEvent, stream.VideoStatusData(m.videoStatus)); err != nil {
			log.Debugf("failed to replay h264 video status to %s: %v", ws.RemoteAddr(), err)
		}
	}
	m.mutex.Unlock()

	stream.SetActiveVideoMode(common.STREAM_TYPE_H264_WEBRTC)
	m.RequestKeyframe()

	log.Debugf("added client %s, total clients: %d", ws.RemoteAddr(), total)
}

func (m *WebRTCManager) RemoveClient(ws *websocket.Conn) {
	m.mutex.Lock()
	delete(m.clients, ws)
	remaining := m.updateClientSnapshotLocked()
	m.mutex.Unlock()

	log.Debugf("removed client %s, total clients: %d", ws.RemoteAddr(), remaining)
}

func (m *WebRTCManager) GetClientCount() int {
	return len(m.getClients())
}

func (m *WebRTCManager) updateClientSnapshotLocked() int {
	clients := make([]*Client, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}
	m.clientSnapshot.Store(&clients)

	return len(clients)
}

func (m *WebRTCManager) getClients() []*Client {
	clients := m.clientSnapshot.Load()
	if clients == nil {
		return nil
	}

	return *clients
}

func (m *WebRTCManager) StartVideoStream() {
	if atomic.CompareAndSwapInt32(&m.videoSending, 0, 1) {
		go m.sendVideoStream()
		log.Debugf("start sending h264 stream")
	}
}

// RequestKeyframe asks the reader to force the encoder to emit an IDR as soon
// as possible. Called from the RTCP path on PLI/FIR and internally after
// encoder-side frame drops. It is a non-blocking, idempotent signal
// (coalesced until consumed).
func (m *WebRTCManager) RequestKeyframe() {
	atomic.StoreInt32(&m.keyframeReq, 1)
}

func (m *WebRTCManager) StartAudioStream() {
	if atomic.CompareAndSwapInt32(&m.audioSending, 0, 1) {
		go m.sendAudioStream()
	}
}

func (m *WebRTCManager) StartMicStream(track *webrtc.TrackRemote) {
	session, err := opus.GetAudioInputPlayer().StartSession()
	if err != nil {
		log.Errorf("failed to start audio player: %s", err)
		return
	}
	go opus.ReceiveMicrophoneTrack(track, session)
}

// sendVideoStream is the capture (reader) goroutine. It reads the encoder
// socket and hands frames to the writer goroutine through an unbuffered
// channel, so slow writers backpressure kvm_vin without accumulating old
// frames inside Go.
func (m *WebRTCManager) sendVideoStream() {
	defer atomic.StoreInt32(&m.videoSending, 0)

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := stream.SetCPUAffinity(0); err != nil {
		return
	}

	screen := common.GetScreen()
	vision := common.GetKvmVision()
	defer func() {
		if vision.StreamType() == common.STREAM_TYPE_H264_WEBRTC {
			vision.CloseVideoStream()
		}
	}()
	frameCh := make(chan videoFrame)
	go m.writeVideoStream(frameCh)
	defer close(frameCh)

	duration := time.Second / time.Duration(120)
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	// waitingKeyframe gates startup and encoder-side drops. If the P-frame
	// chain is broken, freeze until the next keyframe rather than feeding
	// corrupt P-frames to the browser decoder.
	waitingKeyframe := true
	consumeKeyframeRequest := func() {
		if atomic.SwapInt32(&m.keyframeReq, 0) != 1 {
			return
		}

		// ForceIDR is non-blocking: it only signals the C library's IDR
		// worker thread, which performs the request off this goroutine. The
		// library coalesces and rate-limits bursts, so calling it on every
		// pending request is cheap.
		vision.ForceIDR()
	}
	currentStreamType := vision.StreamType
	readFrame := func() ([]byte, int) {
		consumeKeyframeRequest()
		return vision.ReadH264(screen.Width, screen.Height, screen.BitRate)
	}

	for range ticker.C {
		if m.GetClientCount() == 0 {
			log.Debugf("stop sending h264 stream")
			return
		}

		if currentStreamType() != common.STREAM_TYPE_H264_WEBRTC {
			m.updateStatus(stream.VideoStatusInconsistentMode)
			continue
		}

		// Drain currently available socket frames this tick rather than one
		// frame per tick, so the reader keeps pace with the encoder.
		for {
			// ReadH264 waits for a bounded interval in libkvm. Re-check the
			// client count and stream type here so disconnects or a mode switch
			// stop the reader promptly instead of keeping this drain loop alive.
			if m.GetClientCount() == 0 {
				break
			}

			data, result, modeMatches := stream.ReadVideoFrameForMode(
				common.STREAM_TYPE_H264_WEBRTC,
				currentStreamType,
				readFrame,
			)
			m.updateStatus(result)
			if !modeMatches {
				break
			}

			if result == common.IMG_BUFFER_FULL {
				// Encoder dropped frames on its side: the chain is broken.
				waitingKeyframe = true
				m.RequestKeyframe()
				break
			}
			if result < 0 || len(data) == 0 {
				break // IMG_NOT_EXIST / VENC error: no frame available.
			}

			isKeyframe := common.IsH264KeyFrame(result)

			if waitingKeyframe {
				if !isKeyframe {
					continue // still mid-chain, keep dropping
				}
				waitingKeyframe = false
			}

			frame := videoFrame{
				data:       data,
				capturedAt: time.Now(),
			}

			frameCh <- frame
		}
	}
}

// writeVideoStream is the send (writer) goroutine. It consumes captured frames
// and fans them out to all clients. RTP timestamps are derived from capture
// time so playout cadence follows the encoder, not the network.
func (m *WebRTCManager) writeVideoStream(frameCh <-chan videoFrame) {
	var lastCaptured time.Time

	for frame := range frameCh {
		var elapsed time.Duration
		if !lastCaptured.IsZero() {
			elapsed = frame.capturedAt.Sub(lastCaptured)
		}
		lastCaptured = frame.capturedAt

		sample := media.Sample{
			Data:     frame.data,
			Duration: elapsed,
		}

		for _, client := range m.getClients() {
			client.track.writeVideo(sample)
		}

		stream.GetFrameRateCounter().Update()
	}
}

func (m *WebRTCManager) sendAudioStream() {
	defer atomic.StoreInt32(&m.audioSending, 0)
	subscription := stream.DefaultAudioBroker().Subscribe()
	defer subscription.Close()

	for {
		if m.GetClientCount() == 0 {
			return
		}
		data := <-subscription.C

		sample := media.Sample{
			Data:     data,
			Duration: 20 * time.Millisecond,
		}

		for _, client := range m.getClients() {
			client.track.writeAudio(sample)
		}
	}
}

func (m *WebRTCManager) updateStatus(videoStatus int) {
	newStatus := stream.NormalizeVideoStatus(videoStatus, config.GetHDMICaptureEnabled())

	m.statusMutex.Lock()
	defer m.statusMutex.Unlock()

	m.mutex.Lock()
	if m.videoStatus == newStatus {
		m.mutex.Unlock()
		return
	}

	m.videoStatus = newStatus
	clients := make([]*Client, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}
	m.mutex.Unlock()

	data := stream.VideoStatusData(newStatus)
	for _, client := range clients {
		if err := client.WriteMessage(stream.VideoStatusEvent, data); err != nil {
			log.Debugf("failed to send h264 video status: %v", err)
		}
	}
}
