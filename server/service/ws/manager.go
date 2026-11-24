package ws

import (
	"sync"
	"time"

	"NanoKVM-Server/service/vm/jiggler"

	"github.com/gorilla/websocket"
)

const closeMessageWait = 2 * time.Second

var (
	globalManager *Manager
	managerOnce   sync.Once
)

func GetManager() *Manager {
	managerOnce.Do(func() {
		globalManager = &Manager{
			clients: make(map[*websocket.Conn]*Client),
			mutex:   sync.RWMutex{},
		}
	})
	return globalManager
}

func (m *Manager) AddClient(ws *websocket.Conn, client *Client) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.clients[ws] = client
	if len(m.clients) == 1 {
		jiggler.GetJiggler().SetConnected(true)
	}
}

func (m *Manager) RemoveClient(ws *websocket.Conn) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.clients, ws)
	if len(m.clients) == 0 {
		jiggler.GetJiggler().SetConnected(false)
	}
}

func (m *Manager) GetClients() []*Client {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	clients := make([]*Client, 0, len(m.clients))
	for _, c := range m.clients {
		clients = append(clients, c)
	}

	return clients
}

func (m *Manager) CloseAll(code int, text string) {
	clients := m.GetClients()
	message := websocket.FormatCloseMessage(code, text)
	for _, client := range clients {
		client.CloseWithMessage(message)
	}
}
