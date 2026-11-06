package stream

import (
	"net"
	"sync"

	"github.com/gorilla/websocket"
)

// DirectClient serializes all writes to a Direct stream connection. Direct
// video frames and status events share the same WebSocket, and gorilla/websocket
// permits only one concurrent writer per connection.
type DirectClient struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
}

func NewDirectClient(conn *websocket.Conn) *DirectClient {
	return &DirectClient{conn: conn}
}

func (c *DirectClient) WriteMessage(messageType int, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	return c.conn.WriteMessage(messageType, data)
}

func (c *DirectClient) WriteVideoStatus(status int) error {
	payload, err := MarshalVideoStatus(status)
	if err != nil {
		return err
	}

	return c.WriteMessage(websocket.TextMessage, payload)
}

func (c *DirectClient) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *DirectClient) Connection() *websocket.Conn {
	return c.conn
}
