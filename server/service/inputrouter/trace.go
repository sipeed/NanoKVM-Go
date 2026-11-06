package inputrouter

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const DebugSocketPath = "/run/nanokvm-input-debug.sock"

type TracePoint struct {
	ID       uint8   `json:"id,omitempty"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Touching bool    `json:"touching"`
	InRange  bool    `json:"in_range"`
	Pressure uint8   `json:"pressure,omitempty"`
}

type TraceSemantic struct {
	Modifiers uint8   `json:"modifiers,omitempty"`
	Keys      []uint8 `json:"keys,omitempty"`
	Buttons   uint16  `json:"buttons,omitempty"`
	DeltaX    int16   `json:"dx,omitempty"`
	DeltaY    int16   `json:"dy,omitempty"`
	Wheel     int16   `json:"wheel,omitempty"`
	State     uint8   `json:"state,omitempty"`
}

type TraceRecord struct {
	Sequence         uint64               `json:"sequence"`
	SessionID        uint64               `json:"session_id,omitempty"`
	SourceSeq        uint32               `json:"source_sequence,omitempty"`
	Timestamp        int64                `json:"t"`
	CapturedAtMS     uint32               `json:"captured_at_ms,omitempty"`
	CaptureSequence  uint32               `json:"capture_sequence,omitempty"`
	ClientDelayUS    uint32               `json:"client_delay_us,omitempty"`
	RouterDurationUS int64                `json:"router_duration_us,omitempty"`
	HIDWriteUS       int64                `json:"hid_write_us,omitempty"`
	Stage            string               `json:"stage"`
	Source           string               `json:"source,omitempty"`
	Kind             string               `json:"kind,omitempty"`
	Mode             string               `json:"mode,omitempty"`
	Capture          string               `json:"capture,omitempty"`
	Route            string               `json:"route,omitempty"`
	Report           string               `json:"report,omitempty"`
	Meaning          string               `json:"meaning,omitempty"`
	Semantic         *TraceSemantic       `json:"semantic,omitempty"`
	Points           []TracePoint         `json:"points,omitempty"`
	BrowserEvent     *BrowserInputCapture `json:"browser_event,omitempty"`
	Error            string               `json:"error,omitempty"`
}

type TraceHub struct {
	mutex         sync.RWMutex
	records       []TraceRecord
	capacity      int
	sequence      uint64
	subscribers   map[uint64]chan TraceRecord
	subscriberSeq uint64
	streamDropped atomic.Uint64
	socketPath    string
	debugQueue    chan []byte
	dropped       atomic.Uint64
	delivered     atomic.Uint64
	unavailable   atomic.Uint64
	connections   atomic.Uint64
	clientMutex   sync.Mutex
	client        *net.UnixConn
	listenerMutex sync.Mutex
	listener      *net.UnixListener
	stop          chan struct{}
	closeOnce     sync.Once
	workers       sync.WaitGroup
}

func NewTraceHub(capacity int) *TraceHub {
	if capacity < 0 {
		capacity = 0
	}
	return &TraceHub{capacity: capacity, subscribers: make(map[uint64]chan TraceRecord)}
}

func NewManagedTraceHub(capacity int, socketPath string) *TraceHub {
	hub := NewTraceHub(capacity)
	hub.socketPath = socketPath
	hub.debugQueue = make(chan []byte, 128)
	hub.stop = make(chan struct{})
	hub.workers.Add(2)
	go hub.serveDebugSocket()
	go hub.forwardDebugRecords()
	return hub
}

func NewDefaultManagedTraceHub(capacity int) *TraceHub {
	return NewTraceHub(capacity)
}

func (hub *TraceHub) Publish(record TraceRecord) {
	hub.mutex.Lock()
	hub.sequence++
	record.Sequence = hub.sequence
	if record.Timestamp == 0 {
		record.Timestamp = time.Now().UnixMilli()
	}
	if hub.capacity > 0 && len(hub.records) == hub.capacity {
		copy(hub.records, hub.records[1:])
		hub.records[len(hub.records)-1] = record
	} else if hub.capacity > 0 {
		hub.records = append(hub.records, record)
	}
	for _, subscriber := range hub.subscribers {
		select {
		case subscriber <- record:
		default:
			hub.streamDropped.Add(1)
		}
	}
	hub.mutex.Unlock()

	if hub.debugQueue == nil {
		return
	}
	data, err := json.Marshal(record)
	if err != nil {
		return
	}
	select {
	case hub.debugQueue <- data:
	default:
		hub.dropped.Add(1)
	}
}

func (hub *TraceHub) Subscribe(buffer int) (<-chan TraceRecord, func()) {
	if buffer < 1 {
		buffer = 1
	}
	hub.mutex.Lock()
	hub.subscriberSeq++
	id := hub.subscriberSeq
	records := make(chan TraceRecord, buffer)
	hub.subscribers[id] = records
	hub.mutex.Unlock()

	var once sync.Once
	return records, func() {
		once.Do(func() {
			hub.mutex.Lock()
			delete(hub.subscribers, id)
			close(records)
			hub.mutex.Unlock()
		})
	}
}

func (hub *TraceHub) StreamSubscribers() int {
	hub.mutex.RLock()
	defer hub.mutex.RUnlock()
	return len(hub.subscribers)
}

func (hub *TraceHub) DroppedStreamRecords() uint64 {
	return hub.streamDropped.Load()
}

func (hub *TraceHub) Snapshot(limit int) []TraceRecord {
	hub.mutex.RLock()
	defer hub.mutex.RUnlock()
	if limit <= 0 || limit > len(hub.records) {
		limit = len(hub.records)
	}
	start := len(hub.records) - limit
	result := make([]TraceRecord, limit)
	copy(result, hub.records[start:])
	return result
}

func (hub *TraceHub) DroppedDebugRecords() uint64 {
	return hub.dropped.Load()
}

func (hub *TraceHub) DeliveredDebugRecords() uint64 {
	return hub.delivered.Load()
}

func (hub *TraceHub) UnavailableDebugRecords() uint64 {
	return hub.unavailable.Load()
}

func (hub *TraceHub) DebugConnections() uint64 {
	return hub.connections.Load()
}

func (hub *TraceHub) DebugConnected() bool {
	hub.clientMutex.Lock()
	defer hub.clientMutex.Unlock()
	return hub.client != nil
}

func (hub *TraceHub) Close() {
	hub.closeOnce.Do(func() {
		if hub.stop == nil {
			return
		}
		close(hub.stop)
		hub.listenerMutex.Lock()
		if hub.listener != nil {
			_ = hub.listener.Close()
		}
		hub.listenerMutex.Unlock()
		hub.clientMutex.Lock()
		if hub.client != nil {
			_ = hub.client.Close()
			hub.client = nil
		}
		hub.clientMutex.Unlock()
		hub.workers.Wait()
		_ = os.Remove(hub.socketPath)
	})
}

func (hub *TraceHub) forwardDebugRecords() {
	defer hub.workers.Done()
	for {
		select {
		case data := <-hub.debugQueue:
			hub.sendDebugRecord(data)
		case <-hub.stop:
			return
		}
	}
}

func (hub *TraceHub) sendDebugRecord(data []byte) {
	hub.clientMutex.Lock()
	defer hub.clientMutex.Unlock()
	if hub.client == nil {
		hub.unavailable.Add(1)
		return
	}
	if err := hub.client.SetWriteDeadline(time.Now().Add(25 * time.Millisecond)); err != nil {
		hub.dropDebugClient()
		hub.unavailable.Add(1)
		return
	}
	data = append(data, '\n')
	if _, err := hub.client.Write(data); err != nil {
		hub.dropDebugClient()
		hub.unavailable.Add(1)
		return
	}
	hub.delivered.Add(1)
}

func (hub *TraceHub) serveDebugSocket() {
	defer hub.workers.Done()
	_ = os.Remove(hub.socketPath)
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: hub.socketPath, Net: "unix"})
	if err != nil {
		return
	}
	select {
	case <-hub.stop:
		_ = listener.Close()
		return
	default:
	}
	if err := os.Chmod(hub.socketPath, 0o660); err != nil {
		_ = listener.Close()
		return
	}
	hub.listenerMutex.Lock()
	hub.listener = listener
	hub.listenerMutex.Unlock()

	for {
		connection, err := listener.AcceptUnix()
		if err != nil {
			select {
			case <-hub.stop:
				return
			default:
			}
			if errors.Is(err, net.ErrClosed) {
				return
			}
			continue
		}
		hub.clientMutex.Lock()
		if hub.client != nil {
			_ = hub.client.Close()
		}
		hub.client = connection
		hub.connections.Add(1)
		hub.clientMutex.Unlock()
	}
}

func (hub *TraceHub) dropDebugClient() {
	if hub.client == nil {
		return
	}
	_ = hub.client.Close()
	hub.client = nil
}

func traceEvent(event Event, timestamp int64) TraceRecord {
	record := TraceRecord{
		SourceSeq:       event.Sequence,
		Timestamp:       timestamp,
		CapturedAtMS:    event.CapturedAtMS,
		CaptureSequence: event.CaptureSeq,
		ClientDelayUS:   event.ClientDelayUS,
		Source:          sourceName(event.Source),
		Kind:            kindName(event.Kind),
		Mode:            modeName(event.Kind),
		Capture:         hex.EncodeToString(event.Raw),
	}
	switch event.Kind {
	case KindKeyboardState:
		keys := make([]uint8, 0, 6)
		for _, key := range event.Keyboard.Report[2:] {
			if key != 0 {
				keys = append(keys, key)
			}
		}
		record.Semantic = &TraceSemantic{
			Modifiers: event.Keyboard.Report[0],
			Keys:      keys,
		}
	case KindRelativePointer:
		record.Semantic = &TraceSemantic{
			Buttons: uint16(event.Relative.Buttons),
			DeltaX:  event.Relative.DeltaX,
			DeltaY:  event.Relative.DeltaY,
			Wheel:   event.Relative.Wheel,
		}
	case KindAbsolutePointer:
		record.Semantic = &TraceSemantic{
			Buttons: uint16(event.Absolute.Buttons),
			Wheel:   event.Absolute.Wheel,
		}
		record.Points = []TracePoint{{X: normalizeAxis(event.Absolute.X), Y: normalizeAxis(event.Absolute.Y)}}
	case KindStylus:
		record.Semantic = &TraceSemantic{State: event.Stylus.State}
		record.Points = []TracePoint{{
			X:        normalizeAxis(event.Stylus.X),
			Y:        normalizeAxis(event.Stylus.Y),
			Touching: event.Stylus.State&0x01 != 0,
			InRange:  event.Stylus.State&0x02 != 0,
			Pressure: event.Stylus.Pressure,
		}}
	case KindTouchscreen:
		record.Points = make([]TracePoint, 0, len(event.Touchscreen.Contacts))
		for _, contact := range event.Touchscreen.Contacts {
			record.Points = append(record.Points, TracePoint{
				ID:       contact.ID,
				X:        contact.NormX,
				Y:        contact.NormY,
				Touching: contact.State&0x01 != 0,
				InRange:  contact.State&0x02 != 0,
				Pressure: contact.Pressure,
			})
		}
	}
	return record
}

func sourceName(source Source) string {
	switch source {
	case SourceBrowserKeyboard:
		return "browser-keyboard"
	case SourceBrowserPointer:
		return "browser-pointer"
	case SourceBrowserTouch:
		return "browser-touchscreen"
	case SourceVirtualKeyboard:
		return "virtual-keyboard"
	case SourceVirtualPointer:
		return "virtual-pointer"
	case SourceVirtualTouch:
		return "virtual-touchscreen"
	case SourceWebHID:
		return "webhid"
	case SourceLegacyKeyboard:
		return "legacy-websocket-keyboard"
	case SourceLegacyPointer:
		return "legacy-websocket-pointer"
	default:
		return "unknown"
	}
}

func kindName(kind Kind) string {
	switch kind {
	case KindKeyboardState:
		return "keyboard-state"
	case KindRelativePointer:
		return "relative-pointer"
	case KindAbsolutePointer:
		return "absolute-pointer"
	case KindStylus:
		return "stylus"
	case KindTouchscreen:
		return "touchscreen"
	case KindPointerRelease:
		return "pointer-release"
	case KindRawHID:
		return "raw-hid"
	default:
		return "unknown"
	}
}

func modeName(kind Kind) string {
	switch kind {
	case KindRelativePointer:
		return "relative"
	case KindAbsolutePointer:
		return "absolute"
	case KindStylus:
		return "digitizer-stylus"
	case KindTouchscreen:
		return "touch-screen"
	default:
		return kindName(kind)
	}
}
