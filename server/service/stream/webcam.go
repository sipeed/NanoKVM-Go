package stream

import (
	"bytes"
	"errors"
	"image/jpeg"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"NanoKVM-Server/middleware"
	"NanoKVM-Server/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	cameraRuntime     = "/run/nanokvm-webcam/"
	cameraFrameLimit  = 128 * 1024
	cameraFrameWidth  = 1024
	cameraFrameHeight = 600
	cameraFrameIndex  = 1
)

var cameraPublisher = make(chan struct{}, 1)
var cameraUpgrade = websocket.Upgrader{CheckOrigin: utils.CheckWebSocketOrigin}

func validWebcamJPEG(data []byte) bool {
	if len(data) < 4 || len(data) > cameraFrameLimit || !bytes.HasSuffix(data, []byte{0xff, 0xd9}) {
		return false
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
	return err == nil && cfg.Width == cameraFrameWidth && cfg.Height == cameraFrameHeight
}

// ConnectWebcam forwards compressed frames directly to the local UVC output.
// Each reply grants one upload. An idle host only needs periodic demand checks.
func ConnectWebcam(c *gin.Context) {
	connectWebcam(c, cameraRuntime)
}

func connectWebcam(c *gin.Context, runtime string) {
	var expires time.Time
	if token := c.GetString("jwtToken"); token != "" {
		claims, err := middleware.ParseJWT(token)
		if err != nil || claims.ExpiresAt == nil {
			c.Status(http.StatusUnauthorized)
			return
		}
		expires = claims.ExpiresAt.Time
	}
	conn, err := cameraUpgrade.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	closeSocket := func(code int, reason string) {
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(time.Second))
	}
	select {
	case cameraPublisher <- struct{}{}:
		defer func() { <-cameraPublisher }()
	default:
		closeSocket(1008, "Camera already shared")
		return
	}
	if codec := c.Query("codec"); codec != "" && codec != "jpeg" {
		closeSocket(1008, "Only JPEG is supported")
		return
	}
	socketFile, err := os.Stat(runtime + "live.sock")
	if err != nil {
		closeSocket(1011, "Enable Webcam in Settings → Device")
		return
	}
	usb, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: runtime + "live.sock", Net: "unixgram"})
	if err != nil {
		closeSocket(1011, "USB camera unavailable")
		return
	}
	defer func() {
		_ = usb.SetWriteDeadline(time.Now().Add(100 * time.Millisecond))
		_, _ = usb.Write([]byte{0}) // Immediately replace the last camera image.
		_ = usb.Close()
	}()
	if err := usb.SetWriteBuffer(2 * cameraFrameLimit); err != nil {
		return
	}
	conn.SetReadLimit(cameraFrameLimit)
	packet := make([]byte, cameraFrameLimit+2)
	frame := packet[1:]
	supports30 := c.Query("maxFps") == "30"
	var nextFrame, calibrateUntil time.Time
	if c.Query("calibrate") == "1" {
		calibrateUntil = time.Now().Add(11 * time.Second)
	}
	for {
		current, err := os.Stat(runtime + "live.sock")
		if err != nil || !os.SameFile(socketFile, current) || !socketFile.ModTime().Equal(current.ModTime()) {
			closeSocket(1001, "USB webcam disconnected")
			return
		}
		deadline := time.Now().Add(8 * time.Second)
		if !expires.IsZero() && expires.Before(deadline) {
			deadline = expires
		}
		if !time.Now().Before(deadline) {
			return
		}
		selected, err := os.Readlink(runtime + "streaming")
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return
		}
		command := "idle"
		fps := 25
		if supports30 && selected != "600p25" {
			fps = 30
		}
		switch selected {
		case "600p25", "600p30":
			command = "600p25" // Older dashboards only recognize the 25 fps grant.
			if fps == 30 {
				command = "600p30"
			}
			calibrateUntil = time.Time{}
		default:
			if time.Now().Before(calibrateUntil) {
				command = "calibrate"
			}
		}
		if command != "idle" {
			time.Sleep(time.Until(nextFrame))
			period := time.Second / time.Duration(fps)
			now := time.Now()
			// Preserve the phase across small scheduler delays. After a long
			// stall, resume at the normal rate without issuing catch-up grants.
			nextFrame = nextFrame.Add(period)
			if nextFrame.Before(now.Add(period / 2)) {
				nextFrame = now.Add(period)
			}
		}
		_ = conn.SetWriteDeadline(deadline)
		if conn.WriteMessage(websocket.TextMessage, []byte(command)) != nil {
			return
		}
		_ = conn.SetReadDeadline(deadline)
		kind, reader, err := conn.NextReader()
		if err != nil {
			return
		}
		n, err := io.ReadFull(reader, frame)
		if err != io.ErrUnexpectedEOF || n > cameraFrameLimit {
			closeSocket(1008, "Invalid webcam message size")
			return
		}
		data := frame[:n]
		if !time.Now().Before(deadline) {
			return
		}
		if kind == websocket.TextMessage && string(data) == "poll" {
			// Bound status polling even if a client ignores the suggested delay.
			if command == "idle" {
				time.Sleep(100 * time.Millisecond)
			}
			continue
		}
		if kind != websocket.BinaryMessage || command == "idle" || !validWebcamJPEG(data) {
			closeSocket(1008, "Invalid webcam frame")
			return
		}
		if command == "calibrate" {
			continue
		}
		_ = usb.SetWriteDeadline(time.Now().Add(100 * time.Millisecond))
		packet[0] = cameraFrameIndex
		if _, err := usb.Write(packet[:n+1]); err != nil {
			return
		}
	}
}
