package stream

import (
	"sync"
	"time"

	"NanoKVM-Server/common"
)

// AudioBroker is the sole reader of the selected ALSA capture path. Consumers get
// a small latest-wins queue so a slow WebRTC peer never delays capture or any
// other speaker subscriber.
type AudioBroker struct {
	mu          sync.Mutex
	subscribers map[uint64]chan []byte
	nextID      uint64
	running     bool
}

type AudioSubscription struct {
	C      <-chan []byte
	broker *AudioBroker
	id     uint64
	once   sync.Once
}

func NewAudioBroker() *AudioBroker {
	return &AudioBroker{subscribers: make(map[uint64]chan []byte)}
}

func (b *AudioBroker) Subscribe() *AudioSubscription {
	b.mu.Lock()
	b.nextID++
	id := b.nextID
	ch := make(chan []byte, 6)
	b.subscribers[id] = ch
	if !b.running {
		b.running = true
		go b.run()
	}
	b.mu.Unlock()
	return &AudioSubscription{C: ch, broker: b, id: id}
}

func (s *AudioSubscription) Close() {
	if s == nil || s.broker == nil {
		return
	}
	s.once.Do(func() {
		s.broker.mu.Lock()
		delete(s.broker.subscribers, s.id)
		s.broker.mu.Unlock()
	})
}

func (b *AudioBroker) run() {
	for {
		b.mu.Lock()
		if len(b.subscribers) == 0 {
			b.running = false
			b.mu.Unlock()
			return
		}
		b.mu.Unlock()

		frame, result := common.GetKvmVision().ReadAudio()
		if result < 0 || len(frame) == 0 {
			time.Sleep(5 * time.Millisecond)
			continue
		}

		b.mu.Lock()
		for _, subscriber := range b.subscribers {
			select {
			case subscriber <- frame:
			default:
				select {
				case <-subscriber:
				default:
				}
				select {
				case subscriber <- frame:
				default:
				}
			}
		}
		b.mu.Unlock()
	}
}

var defaultAudioBroker = NewAudioBroker()

func DefaultAudioBroker() *AudioBroker { return defaultAudioBroker }
