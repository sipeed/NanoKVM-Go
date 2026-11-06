package ws

import (
	"encoding/json"
	"strconv"
	"sync/atomic"
	"time"

	"NanoKVM-Server/service/hid"
	"NanoKVM-Server/service/inputrouter"
	"NanoKVM-Server/service/vm/jiggler"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

const (
	Heartbeat = iota
	KeyboardEvent
	MouseEvent
	InputRouterEvent
	InputCaptureDebugEvent
	maxInputWebSocketMessageSize  = 64 * 1024
	inputCaptureDebugPollInterval = 250 * time.Millisecond
)

var nextInputSession atomic.Uint64

func NewClient(ws *websocket.Conn) *Client {
	client := &Client{
		ws:            ws,
		inputSession:  inputrouter.SessionID(nextInputSession.Add(1)),
		debugStop:     make(chan struct{}),
		lastHeartbeat: time.Time{},
	}

	hid.GetHid().Open()

	return client
}

func (c *Client) Start() {
	defer c.Close()
	go c.watchInputCaptureDebug()
	_ = c.Read()
}

func (c *Client) Read() error {
	var zeroTime time.Time
	c.ws.SetReadLimit(maxInputWebSocketMessageSize)
	_ = c.ws.SetReadDeadline(zeroTime)

	for {
		messageType, data, err := c.ws.ReadMessage()
		if err != nil {
			return err
		}

		log.Debugf("received message %d: %v", messageType, data)
		if len(data) == 0 {
			log.Debug("ignore empty websocket message")
			continue
		}
		switch data[0] {
		case Heartbeat:
			c.UpdateHeartbeat()
		case KeyboardEvent:
			c.RouteLegacyKeyboard(data[1:])
		case MouseEvent:
			c.RouteLegacyPointer(data[1:])
		case InputRouterEvent:
			if err := inputrouter.GetRouter().RouteSession(c.inputSession, data[1:]); err != nil {
				log.Debugf("input router rejected frame: %s", err)
			} else {
				jiggler.GetJiggler().Update()
			}
		case InputCaptureDebugEvent:
			if !c.captureDebugEnabled.Load() {
				continue
			}
			if err := inputrouter.GetRouter().RecordBrowserCapture(c.inputSession, data[1:]); err != nil {
				log.Debugf("input router rejected browser capture: %s", err)
			}
		}
	}
}

func (c *Client) syncInputCaptureDebug() {
	enabled := inputrouter.GetRouter().Traces().StreamSubscribers() > 0
	if enabled == c.captureDebugEnabled.Load() {
		return
	}
	if err := c.Write("input-debug-capture", strconv.FormatBool(enabled)); err != nil {
		return
	}
	c.captureDebugEnabled.Store(enabled)
}

func (c *Client) watchInputCaptureDebug() {
	c.syncInputCaptureDebug()
	ticker := time.NewTicker(inputCaptureDebugPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.syncInputCaptureDebug()
		case <-c.debugStop:
			return
		}
	}
}

func (c *Client) Write(event string, data string) error {
	message := &Message{
		Type: event,
		Data: data,
	}

	messageByte, err := json.Marshal(message)
	if err != nil {
		log.Errorf("failed to marshal message: %s", err)
		return err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	_ = c.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return c.ws.WriteMessage(websocket.TextMessage, messageByte)
}

func (c *Client) UpdateHeartbeat() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.lastHeartbeat = time.Now()
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		if c.debugStop != nil {
			close(c.debugStop)
		}
		_ = c.ws.Close()
		if err := inputrouter.GetRouter().ReleaseSession(c.inputSession, "websocket disconnected"); err != nil {
			log.Debugf("release input router state: %s", err)
		}

		log.Debug("websocket disconnected")
	})
}

func (c *Client) CloseWithMessage(message []byte) {
	c.mutex.Lock()
	_ = c.ws.SetWriteDeadline(time.Now().Add(closeMessageWait))
	_ = c.ws.WriteMessage(websocket.CloseMessage, message)
	c.mutex.Unlock()

	_ = c.ws.Close()
}

func (c *Client) RouteLegacyKeyboard(report []byte) {
	c.legacySequence++
	err := inputrouter.GetRouter().RouteLegacyKeyboardSession(
		c.inputSession,
		report,
		c.legacySequence,
		uint32(time.Now().UnixMilli()),
	)
	c.finishLegacyRoute(err)
}

func (c *Client) RouteLegacyPointer(report []byte) {
	c.legacySequence++
	err := inputrouter.GetRouter().RouteLegacyPointerSession(
		c.inputSession,
		report,
		c.legacySequence,
		uint32(time.Now().UnixMilli()),
	)
	c.finishLegacyRoute(err)
}

func (c *Client) finishLegacyRoute(err error) {
	if err != nil {
		log.Debugf("input router rejected legacy frame: %s", err)
		return
	}
	jiggler.GetJiggler().Update()
}
