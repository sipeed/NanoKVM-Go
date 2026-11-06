package stream

import (
	"sync"

	"NanoKVM-Server/common"
)

// VideoModeBroker wakes stream transports when the active mode changes. Each
// subscriber has a coalescing queue because consumers only need to re-read the
// latest atomic mode; publishing must never block the SetMode HTTP request.
type VideoModeBroker struct {
	mu          sync.Mutex
	subscribers map[uint64]chan struct{}
	nextID      uint64
}

type VideoModeSubscription struct {
	C       <-chan struct{}
	updates chan struct{}
	broker  *VideoModeBroker
	id      uint64
	once    sync.Once
}

func NewVideoModeBroker() *VideoModeBroker {
	return &VideoModeBroker{subscribers: make(map[uint64]chan struct{})}
}

func (b *VideoModeBroker) Subscribe() *VideoModeSubscription {
	b.mu.Lock()
	b.nextID++
	id := b.nextID
	updates := make(chan struct{}, 1)
	b.subscribers[id] = updates
	b.mu.Unlock()

	return &VideoModeSubscription{C: updates, updates: updates, broker: b, id: id}
}

func (s *VideoModeSubscription) Close() {
	if s == nil || s.broker == nil {
		return
	}

	s.once.Do(func() {
		s.broker.mu.Lock()
		delete(s.broker.subscribers, s.id)
		close(s.updates)
		s.broker.mu.Unlock()
	})
}

func (b *VideoModeBroker) Publish() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, subscriber := range b.subscribers {
		select {
		case subscriber <- struct{}{}:
		default:
		}
	}
}

var defaultVideoModeBroker = NewVideoModeBroker()

func DefaultVideoModeBroker() *VideoModeBroker { return defaultVideoModeBroker }

// SetActiveVideoMode is the single write path for the transport-level mode.
// Store the mode before publishing so subscribers always observe the new value.
func SetActiveVideoMode(mode uint8) {
	common.GetKvmVision().SetStreamType(mode)
	defaultVideoModeBroker.Publish()
}
