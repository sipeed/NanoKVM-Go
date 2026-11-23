package stream

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/jpeg"
	"net"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func TestWebcamDemandAndRelease(t *testing.T) {
	// macOS limits local socket paths to 104 bytes.
	dir, err := os.MkdirTemp("/tmp", "uvc-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	usb, err := net.ListenPacket("unixgram", filepath.Join(dir, "live.sock"))
	if err != nil {
		t.Fatal(err)
	}
	defer usb.Close()
	if err := usb.(*net.UnixConn).SetReadBuffer(2 * cameraFrameLimit); err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.GET("/camera", func(c *gin.Context) { connectWebcam(c, dir+"/") })
	srv := httptest.NewServer(router)
	defer srv.Close()
	open := func() *websocket.Conn {
		conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/camera", nil)
		if err != nil {
			t.Fatal(err)
		}
		return conn
	}
	conn := open()
	defer conn.Close()
	read := func(want string) {
		t.Helper()
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		_, data, err := conn.ReadMessage()
		if err != nil || string(data) != want {
			t.Fatalf("want %s, got %q (%v)", want, data, err)
		}
	}
	read("idle")
	other := open()
	defer other.Close()
	other.SetReadDeadline(time.Now().Add(time.Second))
	if _, _, err := other.ReadMessage(); !websocket.IsCloseError(err, 1008) {
		t.Fatalf("second publisher: %v", err)
	}
	flag := filepath.Join(dir, "streaming")
	var jpg bytes.Buffer
	receive := func(want []byte) {
		t.Helper()
		buf := make([]byte, cameraFrameLimit+1)
		usb.SetReadDeadline(time.Now().Add(time.Second))
		n, _, err := usb.ReadFrom(buf)
		if err != nil || !bytes.Equal(buf[:n], want) {
			t.Fatalf("unexpected USB frame (%d bytes, %v)", n, err)
		}
	}
	for i, name := range []string{"720p25", "720p30"} {
		if i > 0 {
			if err := os.Remove(flag); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.Symlink(name, flag); err != nil {
			t.Fatal(err)
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte("poll")); err != nil {
			t.Fatal(err)
		}
		read("720p25")
		jpg.Reset()
		if err := jpeg.Encode(&jpg, image.NewGray(image.Rect(0, 0, cameraFrameWidth, cameraFrameHeight)), nil); err != nil {
			t.Fatal(err)
		}
		// Two JPEG comment segments bring the payload to the advertised ceiling.
		// A single segment is limited to a 16-bit length.
		payload := append([]byte{}, jpg.Bytes()[:2]...)
		padding := cameraFrameLimit - jpg.Len()
		for _, length := range []int{padding / 2, padding - padding/2} {
			segment := make([]byte, length)
			segment[0], segment[1] = 0xff, 0xfe
			binary.BigEndian.PutUint16(segment[2:], uint16(length-2))
			payload = append(payload, segment...)
		}
		payload = append(payload, jpg.Bytes()[2:]...)
		if err := conn.WriteMessage(websocket.BinaryMessage, payload); err != nil {
			t.Fatal(err)
		}
		receive(append([]byte{cameraFrameIndex}, payload...))
		read("720p25")
	}
	conn.Close()
	receive([]byte{0})
	if validWebcamJPEG(jpg.Bytes()[:jpg.Len()-2]) || validWebcamJPEG(make([]byte, cameraFrameLimit+1)) {
		t.Fatal("accepted invalid frame")
	}
	jpg.Reset()
	jpeg.Encode(&jpg, image.NewGray(image.Rect(0, 0, 480, 640)), nil)
	if validWebcamJPEG(jpg.Bytes()) {
		t.Fatal("accepted incorrect dimensions")
	}
}

func TestUSBDisconnectClosesIdleWebcamPublisher(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "uvc-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "live.sock")
	usb, err := net.ListenPacket("unixgram", path)
	if err != nil {
		t.Fatal(err)
	}
	defer usb.Close()
	router := gin.New()
	router.GET("/camera", func(c *gin.Context) { connectWebcam(c, dir+"/") })
	srv := httptest.NewServer(router)
	defer srv.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/camera", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, msg, err := conn.ReadMessage(); err != nil || string(msg) != "idle" {
		t.Fatalf("first grant %s, %v", msg, err)
	}
	// The feeder changes the socket timestamp on a physical USB disconnect.
	changed := time.Now().Add(time.Minute)
	if err := os.Chtimes(path, changed, changed); err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, []byte("poll")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := conn.ReadMessage(); !websocket.IsCloseError(err, websocket.CloseGoingAway) {
		t.Fatalf("publisher survived disconnect: %v", err)
	}
}

func TestWebcamStartupCalibration(t *testing.T) {
	for _, hostOpens := range []bool{false, true} {
		name := "expires"
		if hostOpens {
			name = "host takes over"
		}
		t.Run(name, func(t *testing.T) {
			dir, err := os.MkdirTemp("/tmp", "uvc-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(dir)
			usb, err := net.ListenPacket("unixgram", filepath.Join(dir, "live.sock"))
			if err != nil {
				t.Fatal(err)
			}
			defer usb.Close()
			if err := usb.(*net.UnixConn).SetReadBuffer(2 * cameraFrameLimit); err != nil {
				t.Fatal(err)
			}
			router := gin.New()
			router.GET("/camera", func(c *gin.Context) { connectWebcam(c, dir+"/") })
			srv := httptest.NewServer(router)
			defer srv.Close()
			conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/camera?calibrate=1&maxFps=30", nil)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			conn.SetReadDeadline(time.Now().Add(20 * time.Second))
			read := func() string {
				t.Helper()
				_, msg, err := conn.ReadMessage()
				if err != nil {
					t.Fatal(err)
				}
				return string(msg)
			}
			send := func(kind int, data []byte) {
				t.Helper()
				if err := conn.WriteMessage(kind, data); err != nil {
					t.Fatal(err)
				}
			}
			var jpg bytes.Buffer
			if err := jpeg.Encode(&jpg, image.NewGray(image.Rect(0, 0, 1280, 720)), nil); err != nil {
				t.Fatal(err)
			}
			start := time.Now()
			if cmd := read(); cmd != "calibrate" {
				t.Fatalf("first grant %s", cmd)
			}
			flag := filepath.Join(dir, "streaming")
			if hostOpens {
				if err := os.Symlink("720p30", flag); err != nil {
					t.Fatal(err)
				}
			}
			send(websocket.BinaryMessage, jpg.Bytes())
			cmd := read()
			if hostOpens {
				if cmd != "720p30" {
					t.Fatalf("host waiting: %s", cmd)
				}
				if err := os.Remove(flag); err != nil {
					t.Fatal(err)
				}
				send(websocket.BinaryMessage, jpg.Bytes())
				cmd = read()
				buf := make([]byte, cameraFrameLimit+1)
				usb.SetReadDeadline(time.Now().Add(time.Second))
				n, _, err := usb.ReadFrom(buf)
				if err != nil || !bytes.Equal(buf[:n], append([]byte{1}, jpg.Bytes()...)) {
					t.Fatalf("live frame: %v", err)
				}
			} else {
				for cmd == "calibrate" {
					send(websocket.BinaryMessage, jpg.Bytes())
					cmd = read()
				}
				if elapsed := time.Since(start); elapsed < 10500*time.Millisecond || elapsed > 12500*time.Millisecond {
					t.Fatalf("benchmark duration %v", elapsed)
				}
			}
			if cmd != "idle" {
				t.Fatalf("final grant %s", cmd)
			}
			usb.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			_, _, err = usb.ReadFrom(make([]byte, cameraFrameLimit+1))
			if timeout, ok := err.(net.Error); !ok || !timeout.Timeout() {
				t.Fatalf("unexpected USB traffic: %v", err)
			}
			conn.Close()
			usb.SetReadDeadline(time.Now().Add(time.Second))
			buf := make([]byte, 8)
			n, _, err := usb.ReadFrom(buf)
			if err != nil || !bytes.Equal(buf[:n], []byte{0}) {
				t.Fatalf("publisher release %v", err)
			}
		})
	}
}
