package inputrouter

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	ProtocolVersion        = 1
	HeaderSize             = 12
	MaxContacts            = 10
	FlagCaptureContext     = 1 << 0
	CaptureContextSize     = 8
	supportedProtocolFlags = FlagCaptureContext
)

type Source uint8

const (
	SourceBrowserKeyboard Source = 1
	SourceBrowserPointer  Source = 2
	SourceBrowserTouch    Source = 3
	SourceVirtualKeyboard Source = 16
	SourceVirtualPointer  Source = 17
	SourceVirtualTouch    Source = 19
	SourceWebHID          Source = 32
	SourceLegacyKeyboard  Source = 240
	SourceLegacyPointer   Source = 241
)

type Kind uint8

const (
	KindKeyboardState   Kind = 1
	KindRelativePointer Kind = 2
	KindAbsolutePointer Kind = 3
	KindStylus          Kind = 4
	KindTouchscreen     Kind = 5
	KindPointerRelease  Kind = 6
	KindRawHID          Kind = 127
)

const (
	ReleaseRouteRelativePointer uint8 = 1 << 0
	ReleaseRouteAbsolutePointer uint8 = 1 << 1
	ReleaseRouteStylus          uint8 = 1 << 2
	ReleaseRouteTouchscreen     uint8 = 1 << 3
	ReleaseRouteAllPointers     uint8 = ReleaseRouteRelativePointer |
		ReleaseRouteAbsolutePointer |
		ReleaseRouteStylus |
		ReleaseRouteTouchscreen
)

type Event struct {
	Version        uint8
	Source         Source
	Kind           Kind
	Flags          uint8
	Sequence       uint32
	CapturedAtMS   uint32
	CaptureSeq     uint32
	ClientDelayUS  uint32
	Keyboard       *KeyboardState
	Relative       *RelativePointer
	Absolute       *AbsolutePointer
	Stylus         *Stylus
	Touchscreen    *Touchscreen
	PointerRelease *PointerRelease
	Raw            []byte
}

type KeyboardState struct {
	Report [8]byte
}

type RelativePointer struct {
	Buttons uint8
	DeltaX  int16
	DeltaY  int16
	Wheel   int16
}

type AbsolutePointer struct {
	Buttons uint8
	X       uint16
	Y       uint16
	Wheel   int16
}

type Stylus struct {
	State    uint8
	X        uint16
	Y        uint16
	Pressure uint8
}

type Contact struct {
	ID       uint8   `json:"id"`
	State    uint8   `json:"state"`
	X        uint16  `json:"-"`
	Y        uint16  `json:"-"`
	Pressure uint8   `json:"pressure"`
	NormX    float64 `json:"x"`
	NormY    float64 `json:"y"`
}

type Touchscreen struct {
	Contacts []Contact
}

type PointerRelease struct {
	Routes uint8
}

func Decode(data []byte) (Event, error) {
	if len(data) < HeaderSize {
		return Event{}, fmt.Errorf("input router frame too short: %d", len(data))
	}

	event := Event{
		Version:      data[0],
		Source:       Source(data[1]),
		Kind:         Kind(data[2]),
		Flags:        data[3],
		Sequence:     binary.LittleEndian.Uint32(data[4:8]),
		CapturedAtMS: binary.LittleEndian.Uint32(data[8:12]),
		Raw:          append([]byte(nil), data...),
	}
	if event.Version != ProtocolVersion {
		return Event{}, fmt.Errorf("unsupported input router protocol version: %d", event.Version)
	}
	if event.Flags&^uint8(supportedProtocolFlags) != 0 {
		return Event{}, fmt.Errorf("unsupported input router flags: 0x%02x", event.Flags)
	}
	if !event.Source.valid() {
		return Event{}, fmt.Errorf("unsupported input source: %d", event.Source)
	}
	payload := data[HeaderSize:]
	if event.Flags&FlagCaptureContext != 0 {
		if len(payload) < CaptureContextSize {
			return Event{}, fmt.Errorf("input capture context is truncated: %d", len(payload))
		}
		contextOffset := len(payload) - CaptureContextSize
		event.CaptureSeq = binary.LittleEndian.Uint32(payload[contextOffset : contextOffset+4])
		event.ClientDelayUS = binary.LittleEndian.Uint32(payload[contextOffset+4:])
		if event.CaptureSeq == 0 {
			return Event{}, errors.New("input capture sequence must be non-zero")
		}
		payload = payload[:contextOffset]
	}
	switch event.Kind {
	case KindKeyboardState:
		if len(payload) != 8 {
			return Event{}, payloadSizeError(event.Kind, len(payload), 8)
		}
		state := &KeyboardState{}
		copy(state.Report[:], payload)
		event.Keyboard = state
	case KindRelativePointer:
		if len(payload) != 7 {
			return Event{}, payloadSizeError(event.Kind, len(payload), 7)
		}
		event.Relative = &RelativePointer{
			Buttons: payload[0],
			DeltaX:  int16(binary.LittleEndian.Uint16(payload[1:3])),
			DeltaY:  int16(binary.LittleEndian.Uint16(payload[3:5])),
			Wheel:   int16(binary.LittleEndian.Uint16(payload[5:7])),
		}
	case KindAbsolutePointer:
		if len(payload) != 7 {
			return Event{}, payloadSizeError(event.Kind, len(payload), 7)
		}
		event.Absolute = &AbsolutePointer{
			Buttons: payload[0],
			X:       binary.LittleEndian.Uint16(payload[1:3]),
			Y:       binary.LittleEndian.Uint16(payload[3:5]),
			Wheel:   int16(binary.LittleEndian.Uint16(payload[5:7])),
		}
	case KindStylus:
		if len(payload) != 6 {
			return Event{}, payloadSizeError(event.Kind, len(payload), 6)
		}
		event.Stylus = &Stylus{
			State:    payload[0],
			X:        binary.LittleEndian.Uint16(payload[1:3]),
			Y:        binary.LittleEndian.Uint16(payload[3:5]),
			Pressure: payload[5],
		}
	case KindTouchscreen:
		touchscreen, err := decodeTouchscreen(payload)
		if err != nil {
			return Event{}, err
		}
		event.Touchscreen = touchscreen
	case KindPointerRelease:
		if len(payload) != 1 {
			return Event{}, payloadSizeError(event.Kind, len(payload), 1)
		}
		mask := payload[0]
		if mask == 0 {
			return Event{}, errors.New("pointer release route mask must be non-zero")
		}
		if mask&^ReleaseRouteAllPointers != 0 {
			return Event{}, fmt.Errorf("unsupported pointer release route mask: 0x%02x", mask)
		}
		event.PointerRelease = &PointerRelease{Routes: mask}
	case KindRawHID:
		return Event{}, fmt.Errorf("raw HID routing requires a registered device profile")
	default:
		return Event{}, fmt.Errorf("unsupported input event kind: %d", event.Kind)
	}
	if err := validateEvent(event); err != nil {
		return Event{}, err
	}

	return event, nil
}

func decodeTouchscreen(payload []byte) (*Touchscreen, error) {
	if len(payload) < 1 {
		return nil, errors.New("touchscreen payload is empty")
	}
	count := int(payload[0])
	if count > MaxContacts {
		return nil, fmt.Errorf("touchscreen contact count %d exceeds %d", count, MaxContacts)
	}
	const contactSize = 8
	if len(payload) != 1+count*contactSize {
		return nil, payloadSizeError(KindTouchscreen, len(payload), 1+count*contactSize)
	}

	contacts := make([]Contact, 0, count)
	seen := make(map[uint8]struct{}, count)
	for index := 0; index < count; index++ {
		offset := 1 + index*contactSize
		id := payload[offset]
		if id == 0 || id > MaxContacts {
			return nil, fmt.Errorf("touchscreen contact id %d is outside 1..%d", id, MaxContacts)
		}
		if _, exists := seen[id]; exists {
			return nil, fmt.Errorf("duplicate touchscreen contact id: %d", id)
		}
		if payload[offset+1]&^uint8(0x03) != 0 {
			return nil, fmt.Errorf("unsupported touchscreen contact state: 0x%02x", payload[offset+1])
		}
		if payload[offset+7] != 0 {
			return nil, fmt.Errorf("touchscreen contact reserved byte must be zero: 0x%02x", payload[offset+7])
		}
		seen[id] = struct{}{}
		x := binary.LittleEndian.Uint16(payload[offset+2 : offset+4])
		y := binary.LittleEndian.Uint16(payload[offset+4 : offset+6])
		contacts = append(contacts, Contact{
			ID:       id,
			State:    payload[offset+1],
			X:        x,
			Y:        y,
			Pressure: payload[offset+6],
			NormX:    normalizeAxis(x),
			NormY:    normalizeAxis(y),
		})
	}
	return &Touchscreen{Contacts: contacts}, nil
}

func validateEvent(event Event) error {
	switch event.Kind {
	case KindKeyboardState:
		if event.Keyboard.Report[1] != 0 {
			return fmt.Errorf("keyboard reserved byte must be zero: 0x%02x", event.Keyboard.Report[1])
		}
		for _, usage := range event.Keyboard.Report[2:] {
			if usage > 0xe7 {
				return fmt.Errorf("keyboard usage exceeds descriptor range: 0x%02x", usage)
			}
		}
	case KindRelativePointer:
		if event.Relative.Buttons&^uint8(0x1f) != 0 {
			return fmt.Errorf("unsupported relative pointer buttons: 0x%02x", event.Relative.Buttons)
		}
	case KindAbsolutePointer:
		if event.Absolute.Buttons&^uint8(0x1f) != 0 {
			return fmt.Errorf("unsupported absolute pointer buttons: 0x%02x", event.Absolute.Buttons)
		}
	case KindStylus:
		if event.Stylus.State&^uint8(0x03) != 0 {
			return fmt.Errorf("unsupported stylus state: 0x%02x", event.Stylus.State)
		}
	}
	return nil
}

func (source Source) valid() bool {
	switch source {
	case SourceBrowserKeyboard,
		SourceBrowserPointer,
		SourceBrowserTouch,
		SourceVirtualKeyboard,
		SourceVirtualPointer,
		SourceVirtualTouch,
		SourceWebHID,
		SourceLegacyKeyboard,
		SourceLegacyPointer:
		return true
	default:
		return false
	}
}

func payloadSizeError(kind Kind, got, want int) error {
	return fmt.Errorf("invalid payload size for input event kind %d: got %d, want %d", kind, got, want)
}

func normalizeAxis(value uint16) float64 {
	return float64(value) / float64(^uint16(0))
}
