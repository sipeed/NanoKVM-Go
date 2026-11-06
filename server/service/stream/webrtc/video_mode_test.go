package webrtc

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"NanoKVM-Server/common"
	"NanoKVM-Server/service/stream"

	"github.com/gorilla/websocket"
)

func TestVideoModeNotificationPausesWithoutCaptureLoop(t *testing.T) {
	writer, reader := newStatusTestWebSocketPair(t)
	manager := NewWebRTCManager()
	client := NewClient(writer, nil, nil)

	manager.mutex.Lock()
	manager.clients[writer] = client
	manager.updateClientSnapshotLocked()
	manager.mutex.Unlock()

	broker := stream.NewVideoModeBroker()
	subscription := broker.Subscribe()
	watcherDone := make(chan struct{})
	go func() {
		manager.watchVideoMode(subscription)
		close(watcherDone)
	}()

	vision := common.GetKvmVision()
	previousMode := vision.StreamType()
	t.Cleanup(func() {
		vision.SetStreamType(previousMode)
		subscription.Close()
		select {
		case <-watcherDone:
		case <-time.After(time.Second):
			t.Error("video-mode watcher did not stop")
		}
	})

	vision.SetStreamType(common.STREAM_TYPE_H264_DIRECT)
	broker.Publish()

	if err := reader.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	var message Message
	if err := reader.ReadJSON(&message); err != nil {
		t.Fatalf("read video-mode status: %v", err)
	}
	if message.Event != stream.VideoStatusEvent || message.Data != stream.VideoStatusData(stream.VideoStatusInconsistentMode) {
		t.Fatalf("unexpected status message: %#v", message)
	}
}

func newStatusTestWebSocketPair(t *testing.T) (*websocket.Conn, *websocket.Conn) {
	t.Helper()

	upgrader := websocket.Upgrader{}
	serverConnection := make(chan *websocket.Conn, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connection, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		serverConnection <- connection
	}))
	t.Cleanup(server.Close)

	endpoint, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	endpoint.Scheme = "ws" + strings.TrimPrefix(endpoint.Scheme, "http")

	writer, _, err := websocket.DefaultDialer.Dial(endpoint.String(), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	t.Cleanup(func() { _ = writer.Close() })

	reader := <-serverConnection
	t.Cleanup(func() { _ = reader.Close() })
	return writer, reader
}
