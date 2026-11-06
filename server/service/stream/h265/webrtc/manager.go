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
		tracks:       make(map[*websocket.Conn]*Track),
		videoSending: 0,
		audioSending: 0,
		videoStatus:  0,
	}
	m.updateTrackSnapshotLocked()

	return m
}

func (m *WebRTCManager) watchVideoMode(subscription *stream.VideoModeSubscription) {
	for range subscription.C {
		if m.GetTrackCount() == 0 {
			continue
		}
		if common.GetKvmVision().StreamType() != common.STREAM_TYPE_H265_WEBRTC {
			m.updateStatus(stream.VideoStatusInconsistentMode)
		}
	}
}

func (m *WebRTCManager) AddTrack(ws *websocket.Conn, track *Track) {
	m.mutex.Lock()
	m.tracks[ws] = track
	total := m.updateTrackSnapshotLocked()
	if stream.IsKnownVideoStatus(m.videoStatus) {
		if err := track.client.WriteMessage(stream.VideoStatusEvent, stream.VideoStatusData(m.videoStatus)); err != nil {
			log.Debugf("failed to replay h265 video status to %s: %v", ws.RemoteAddr(), err)
		}
	}
	m.mutex.Unlock()

	stream.SetActiveVideoMode(common.STREAM_TYPE_H265_WEBRTC)
	m.RequestKeyframe()

	log.Debugf("added track %s, total tracks: %d", ws.RemoteAddr(), total)
}

func (m *WebRTCManager) RemoveTrack(ws *websocket.Conn) {
	m.mutex.Lock()
	delete(m.tracks, ws)
	remaining := m.updateTrackSnapshotLocked()
	m.mutex.Unlock()

	log.Debugf("removed track %s, total tracks: %d", ws.RemoteAddr(), remaining)
}

func (m *WebRTCManager) GetTrackCount() int {
	return len(m.getTracks())
}

func (m *WebRTCManager) updateTrackSnapshotLocked() int {
	tracks := make([]*Track, 0, len(m.tracks))
	for _, track := range m.tracks {
		tracks = append(tracks, track)
	}
	m.trackSnapshot.Store(&tracks)

	return len(tracks)
}

func (m *WebRTCManager) getTracks() []*Track {
	tracks := m.trackSnapshot.Load()
	if tracks == nil {
		return nil
	}

	return *tracks
}

func (m *WebRTCManager) StartVideoStream() {
	if atomic.CompareAndSwapInt32(&m.videoSending, 0, 1) {
		go m.sendVideoStream()
		log.Debugf("start sending h265 stream")
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
		if vision.StreamType() == common.STREAM_TYPE_H265_WEBRTC {
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
		return vision.ReadH265(screen.Width, screen.Height, screen.BitRate)
	}

	for range ticker.C {
		if m.GetTrackCount() == 0 {
			log.Debugf("stop sending h265 stream")
			return
		}

		if currentStreamType() != common.STREAM_TYPE_H265_WEBRTC {
			m.updateStatus(stream.VideoStatusInconsistentMode)
			continue
		}

		// Drain currently available socket frames this tick rather than one
		// frame per tick, so the reader keeps pace with the encoder.
		for {
			// ReadH265 waits for a bounded interval in libkvm. Re-check the
			// client count and stream type here so disconnects or a mode switch
			// stop the reader promptly instead of keeping this drain loop alive.
			if m.GetTrackCount() == 0 {
				break
			}

			data, result, modeMatches := stream.ReadVideoFrameForMode(
				common.STREAM_TYPE_H265_WEBRTC,
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

			isKeyframe := common.IsH265KeyFrame(result)

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
// and fans them out to all tracks. RTP timestamps are derived from capture
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

		for _, track := range m.getTracks() {
			track.writeVideo(sample)
		}

		stream.GetFrameRateCounter().Update()
	}
}

func (m *WebRTCManager) sendAudioStream() {
	defer atomic.StoreInt32(&m.audioSending, 0)
	subscription := stream.DefaultAudioBroker().Subscribe()
	defer subscription.Close()

	for {
		if m.GetTrackCount() == 0 {
			return
		}
		data := <-subscription.C

		sample := media.Sample{
			Data:     data,
			Duration: 20 * time.Millisecond,
		}

		for _, track := range m.getTracks() {
			track.writeAudio(sample)
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
	tracks := make([]*Track, 0, len(m.tracks))
	for _, track := range m.tracks {
		tracks = append(tracks, track)
	}
	m.mutex.Unlock()

	data := stream.VideoStatusData(newStatus)
	for _, track := range tracks {
		if err := track.client.WriteMessage(stream.VideoStatusEvent, data); err != nil {
			log.Debugf("failed to send h265 video status: %v", err)
		}
	}
}
