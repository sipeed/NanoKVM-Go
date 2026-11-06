// Package media owns short-lived capabilities for the WebRTC media plane.
package media

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

const DefaultTTL = 10 * time.Minute

type Capabilities struct {
	SpeakerReceive bool `json:"speakerReceive"`
	MicSend        bool `json:"micSend"`
	VideoReceive   bool `json:"videoReceive"`
}

type Session struct {
	ID           string       `json:"id"`
	Token        string       `json:"token,omitempty"`
	Owner        string       `json:"owner"`
	Capabilities Capabilities `json:"capabilities"`
	ExpiresAt    time.Time    `json:"expiresAt"`
}

type Manager struct {
	mu       sync.Mutex
	sessions map[string]Session
	now      func() time.Time
}

func NewManager() *Manager {
	return &Manager{sessions: make(map[string]Session), now: time.Now}
}

func (m *Manager) Create(owner string, capabilities Capabilities, ttl time.Duration) (Session, error) {
	if !capabilities.SpeakerReceive && !capabilities.MicSend && !capabilities.VideoReceive {
		return Session{}, errors.New("at least one media capability is required")
	}
	if ttl <= 0 || ttl > DefaultTTL {
		ttl = DefaultTTL
	}
	token, err := randomToken()
	if err != nil {
		return Session{}, err
	}
	now := m.now()
	session := Session{ID: token[:16], Token: token, Owner: owner, Capabilities: capabilities, ExpiresAt: now.Add(ttl)}
	m.mu.Lock()
	m.removeExpiredLocked(now)
	m.sessions[token] = session
	m.mu.Unlock()
	return session, nil
}

func (m *Manager) Get(token string) (Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	m.removeExpiredLocked(now)
	session, ok := m.sessions[token]
	if ok {
		session.Token = ""
	}
	return session, ok
}

func (m *Manager) Close(token string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[token]; !ok {
		return false
	}
	delete(m.sessions, token)
	return true
}

// CloseAll revokes every outstanding media bearer capability. Active media
// connections are closed separately through the auth-session registry.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	m.sessions = make(map[string]Session)
	m.mu.Unlock()
}

func (m *Manager) removeExpiredLocked(now time.Time) {
	for token, session := range m.sessions {
		if !session.ExpiresAt.After(now) {
			delete(m.sessions, token)
		}
	}
}

func randomToken() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

var defaultManager = NewManager()

func DefaultManager() *Manager { return defaultManager }
