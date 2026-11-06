package stream

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestDirectClientSerializesStatusAndFrameWrites(t *testing.T) {
	upgrader := websocket.Upgrader{}
	serverConnCh := make(chan *websocket.Conn, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("Upgrade() error = %v", err)
			return
		}
		serverConnCh <- conn
	}))
	defer server.Close()

	wsURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	wsURL.Scheme = "ws" + strings.TrimPrefix(wsURL.Scheme, "http")

	writerConn, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer writerConn.Close()

	readerConn := <-serverConnCh
	defer readerConn.Close()

	client := NewDirectClient(writerConn)
	const writesPerType = 16

	var writers sync.WaitGroup
	for i := 0; i < writesPerType; i++ {
		writers.Add(2)
		go func() {
			defer writers.Done()
			if err := client.WriteVideoStatus(-1); err != nil {
				t.Errorf("WriteVideoStatus() error = %v", err)
			}
		}()
		go func(i int) {
			defer writers.Done()
			if err := client.WriteMessage(websocket.BinaryMessage, []byte{byte(i)}); err != nil {
				t.Errorf("WriteMessage() error = %v", err)
			}
		}(i)
	}

	readerConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	textMessages := 0
	binaryMessages := 0
	for textMessages+binaryMessages < writesPerType*2 {
		messageType, payload, err := readerConn.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage() error = %v", err)
		}

		switch messageType {
		case websocket.TextMessage:
			var message VideoStatusMessage
			if err := json.Unmarshal(payload, &message); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}
			if message.Event != VideoStatusEvent || message.Data != "-1" {
				t.Fatalf("status message = %#v, want video-status/-1", message)
			}
			textMessages++
		case websocket.BinaryMessage:
			if len(payload) != 1 {
				t.Fatalf("binary payload length = %d, want 1", len(payload))
			}
			binaryMessages++
		default:
			t.Fatalf("message type = %d, want text or binary", messageType)
		}
	}

	writers.Wait()
	if textMessages != writesPerType || binaryMessages != writesPerType {
		t.Fatalf("received text=%d binary=%d, want %d each", textMessages, binaryMessages, writesPerType)
	}
}
