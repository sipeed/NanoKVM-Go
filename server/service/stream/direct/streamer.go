package direct

import (
	"bytes"
	"encoding/binary"
	"sync"
	"sync/atomic"
	"time"

	"NanoKVM-Server/common"
	"NanoKVM-Server/config"
	"NanoKVM-Server/service/stream"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

type Streamer struct {
	mutex          sync.Mutex
	clients        map[*websocket.Conn]*stream.DirectClient
	clientSnapshot atomic.Pointer[[]*stream.DirectClient]
	running        int32
	videoStatus    int
}

func newStreamer() *Streamer {
	s := &Streamer{
		clients: make(map[*websocket.Conn]*stream.DirectClient),
	}
	s.updateClientSnapshotLocked()

	return s
}

func (s *Streamer) addClient(ws *websocket.Conn) {
	client := stream.NewDirectClient(ws)
	replayFailed := false

	s.mutex.Lock()
	s.clients[ws] = client
	s.updateClientSnapshotLocked()
	if stream.IsKnownVideoStatus(s.videoStatus) {
		if err := client.WriteVideoStatus(s.videoStatus); err != nil {
			log.Debugf("failed to replay h264 video status to %s: %v", ws.RemoteAddr(), err)
			replayFailed = true
		}
	}
	s.mutex.Unlock()
	if replayFailed {
		s.removeClient(ws)
		return
	}

	stream.SetActiveVideoMode(common.STREAM_TYPE_H264_DIRECT)
	common.GetKvmVision().ForceIDR()

	if atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		go s.run()
		log.Debug("h264 stream started")
	}
}

func (s *Streamer) removeClient(ws *websocket.Conn) {
	s.mutex.Lock()
	delete(s.clients, ws)
	remaining := s.updateClientSnapshotLocked()
	s.mutex.Unlock()

	log.Debugf("h264 websocket disconnected, remaining clients: %d", remaining)
}

func (s *Streamer) updateClientSnapshotLocked() int {
	clients := make([]*stream.DirectClient, 0, len(s.clients))
	for _, client := range s.clients {
		clients = append(clients, client)
	}
	s.clientSnapshot.Store(&clients)

	return len(clients)
}

func (s *Streamer) getClients() []*stream.DirectClient {
	clients := s.clientSnapshot.Load()
	if clients == nil {
		return nil
	}

	return *clients
}

func (s *Streamer) run() {
	defer atomic.StoreInt32(&s.running, 0)

	duration := time.Second / time.Duration(120)
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	screen := common.GetScreen()
	vision := common.GetKvmVision()
	defer func() {
		if vision.StreamType() == common.STREAM_TYPE_H264_DIRECT {
			vision.CloseVideoStream()
		}
	}()
	startTime := time.Now()
	waitingKeyframe := true

	for range ticker.C {
		clients := s.getClients()
		if len(clients) == 0 {
			log.Debug("h264 stream stopped due to no clients")
			return
		}

		if vision.StreamType() != common.STREAM_TYPE_H264_DIRECT {
			s.updateStatus(-4)
			continue
		}

		data, result := vision.ReadH264(screen.Width, screen.Height, screen.BitRate)
		s.updateStatus(result)
		if result == common.IMG_BUFFER_FULL {
			waitingKeyframe = true
			vision.ForceIDR()
			continue
		}
		if result < 0 || len(data) == 0 {
			continue
		}

		isKeyFrame := byte(0)
		if common.IsH264KeyFrame(result) {
			isKeyFrame = byte(1)
		}
		if waitingKeyframe {
			if isKeyFrame == 0 {
				continue
			}
			waitingKeyframe = false
		}

		timestamp := time.Since(startTime).Microseconds()

		if err := s.send(clients, isKeyFrame, timestamp, data); err != nil {
			continue
		}

		stream.GetFrameRateCounter().Update()
	}
}

func (s *Streamer) updateStatus(videoStatus int) {
	newStatus := stream.NormalizeVideoStatus(videoStatus, config.GetHDMICaptureEnabled())

	failedClients := make([]*websocket.Conn, 0)
	s.mutex.Lock()
	if s.videoStatus != newStatus {
		s.videoStatus = newStatus
		for ws, client := range s.clients {
			if err := client.WriteVideoStatus(newStatus); err != nil {
				log.Debugf("failed to send h264 video status to %s: %v", ws.RemoteAddr(), err)
				failedClients = append(failedClients, ws)
			}
		}
	}
	s.mutex.Unlock()

	for _, ws := range failedClients {
		s.removeClient(ws)
	}
}

func (s *Streamer) send(clients []*stream.DirectClient, isKeyFrame byte, timestamp int64, data []byte) error {
	buf := stream.BufferPool.Get().(*bytes.Buffer)
	defer stream.BufferPool.Put(buf)

	buf.Reset()

	if err := buf.WriteByte(isKeyFrame); err != nil {
		log.Errorf("failed to write keyframe flag: %s", err)
		return err
	}

	tsBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(tsBytes, uint64(timestamp))
	if _, err := buf.Write(tsBytes); err != nil {
		log.Errorf("failed to write timestamp: %s", err)
		return err
	}

	if _, err := buf.Write(data); err != nil {
		log.Errorf("failed to write h264 data: %s", err)
		return err
	}

	for _, client := range clients {
		if err := client.WriteMessage(websocket.BinaryMessage, buf.Bytes()); err != nil {
			log.Errorf("failed to write message to client %s: %s.", client.RemoteAddr(), err)

			s.removeClient(client.Connection())
		}
	}

	return nil
}
