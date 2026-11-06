package ws

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"NanoKVM-Server/service/inputrouter"
)

type Manager struct {
	clients map[*websocket.Conn]*Client
	mutex   sync.RWMutex
}

type Client struct {
	ws                  *websocket.Conn
	inputSession        inputrouter.SessionID
	legacySequence      uint32
	captureDebugEnabled atomic.Bool
	debugStop           chan struct{}
	lastHeartbeat       time.Time
	mutex               sync.Mutex
	closeOnce           sync.Once
}

type Message struct {
	Type string `json:"type"`
	Data string `json:"data"`
}
