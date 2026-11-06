package inputrouter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
)

const (
	maxBrowserCaptureBytes = 16 * 1024
	maxBrowserTouchPoints  = 32
)

type BrowserInputCapture struct {
	API          string          `json:"api"`
	Sequence     uint32          `json:"sequence"`
	CapturedAtMS uint32          `json:"capturedAtMS"`
	Type         string          `json:"type"`
	Content      json.RawMessage `json:"content"`
}

type BrowserPointerEventContent struct {
	PointerID           int32   `json:"pointerId"`
	PointerType         string  `json:"pointerType"`
	IsPrimary           bool    `json:"isPrimary"`
	ClientX             float64 `json:"clientX"`
	ClientY             float64 `json:"clientY"`
	PageX               float64 `json:"pageX"`
	PageY               float64 `json:"pageY"`
	ScreenX             float64 `json:"screenX"`
	ScreenY             float64 `json:"screenY"`
	OffsetX             float64 `json:"offsetX"`
	OffsetY             float64 `json:"offsetY"`
	MovementX           float64 `json:"movementX"`
	MovementY           float64 `json:"movementY"`
	Button              int16   `json:"button"`
	Buttons             uint16  `json:"buttons"`
	Pressure            float64 `json:"pressure"`
	TangentialPressure  float64 `json:"tangentialPressure"`
	Width               float64 `json:"width"`
	Height              float64 `json:"height"`
	TiltX               int16   `json:"tiltX"`
	TiltY               int16   `json:"tiltY"`
	Twist               uint16  `json:"twist"`
	AltitudeAngle       float64 `json:"altitudeAngle"`
	AzimuthAngle        float64 `json:"azimuthAngle"`
	CtrlKey             bool    `json:"ctrlKey"`
	ShiftKey            bool    `json:"shiftKey"`
	AltKey              bool    `json:"altKey"`
	MetaKey             bool    `json:"metaKey"`
	TimeStamp           float64 `json:"timeStamp"`
	CoalescedEventCount uint16  `json:"coalescedEventCount"`
	Cancelable          bool    `json:"cancelable"`
	Composed            bool    `json:"composed"`
	TargetID            string  `json:"targetId"`
}

type BrowserPointerCapture struct {
	API          string                     `json:"api"`
	Sequence     uint32                     `json:"sequence"`
	CapturedAtMS uint32                     `json:"capturedAtMS"`
	Type         string                     `json:"type"`
	Content      BrowserPointerEventContent `json:"content"`
}

type BrowserTouchPoint struct {
	Identifier    int32   `json:"identifier"`
	ClientX       float64 `json:"clientX"`
	ClientY       float64 `json:"clientY"`
	PageX         float64 `json:"pageX"`
	PageY         float64 `json:"pageY"`
	ScreenX       float64 `json:"screenX"`
	ScreenY       float64 `json:"screenY"`
	RadiusX       float64 `json:"radiusX"`
	RadiusY       float64 `json:"radiusY"`
	RotationAngle float64 `json:"rotationAngle"`
	Force         float64 `json:"force"`
}

type BrowserTouchEventContent struct {
	Touches        []BrowserTouchPoint `json:"touches"`
	TargetTouches  []BrowserTouchPoint `json:"targetTouches"`
	ChangedTouches []BrowserTouchPoint `json:"changedTouches"`
	CtrlKey        bool                `json:"ctrlKey"`
	ShiftKey       bool                `json:"shiftKey"`
	AltKey         bool                `json:"altKey"`
	MetaKey        bool                `json:"metaKey"`
	TimeStamp      float64             `json:"timeStamp"`
	Cancelable     bool                `json:"cancelable"`
	Composed       bool                `json:"composed"`
	TargetID       string              `json:"targetId"`
}

type BrowserTouchCapture struct {
	API          string                   `json:"api"`
	Sequence     uint32                   `json:"sequence"`
	CapturedAtMS uint32                   `json:"capturedAtMS"`
	Type         string                   `json:"type"`
	Content      BrowserTouchEventContent `json:"content"`
}

func decodeBrowserInputCapture(data []byte) (BrowserInputCapture, error) {
	if len(data) == 0 {
		return BrowserInputCapture{}, errors.New("browser input capture is empty")
	}
	if len(data) > maxBrowserCaptureBytes {
		return BrowserInputCapture{}, fmt.Errorf(
			"browser input capture exceeds %d bytes: %d",
			maxBrowserCaptureBytes,
			len(data),
		)
	}

	var capture BrowserInputCapture
	if err := decodeStrictJSON(data, &capture); err != nil {
		return BrowserInputCapture{}, fmt.Errorf("decode browser input capture: %w", err)
	}
	if capture.Sequence == 0 {
		return BrowserInputCapture{}, errors.New("browser capture sequence must be non-zero")
	}
	if len(capture.Content) == 0 || bytes.Equal(capture.Content, []byte("null")) {
		return BrowserInputCapture{}, errors.New("browser input capture content is empty")
	}

	switch capture.API {
	case "PointerEvent":
		var content BrowserPointerEventContent
		if err := decodeStrictJSON(capture.Content, &content); err != nil {
			return BrowserInputCapture{}, fmt.Errorf("decode PointerEvent content: %w", err)
		}
		if err := validateBrowserPointerCapture(capture.Type, content); err != nil {
			return BrowserInputCapture{}, err
		}
	case "TouchEvent":
		var content BrowserTouchEventContent
		if err := decodeStrictJSON(capture.Content, &content); err != nil {
			return BrowserInputCapture{}, fmt.Errorf("decode TouchEvent content: %w", err)
		}
		if err := validateBrowserTouchCapture(capture.Type, content); err != nil {
			return BrowserInputCapture{}, err
		}
	default:
		return BrowserInputCapture{}, fmt.Errorf("unsupported browser capture API: %q", capture.API)
	}

	return capture, nil
}

func decodeStrictJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("JSON has trailing data")
	}
	return nil
}

func validateBrowserPointerCapture(eventType string, content BrowserPointerEventContent) error {
	switch eventType {
	case "pointerdown", "pointermove", "pointerup", "pointercancel", "pointerleave":
	default:
		return fmt.Errorf("unsupported PointerEvent type: %q", eventType)
	}
	if len(content.PointerType) > 32 {
		return errors.New("PointerEvent pointerType is too long")
	}
	if len(content.TargetID) > 128 {
		return errors.New("PointerEvent targetId is too long")
	}
	values := []float64{
		content.ClientX,
		content.ClientY,
		content.PageX,
		content.PageY,
		content.ScreenX,
		content.ScreenY,
		content.OffsetX,
		content.OffsetY,
		content.MovementX,
		content.MovementY,
		content.Pressure,
		content.TangentialPressure,
		content.Width,
		content.Height,
		content.AltitudeAngle,
		content.AzimuthAngle,
		content.TimeStamp,
	}
	if !allFinite(values) {
		return errors.New("PointerEvent contains a non-finite number")
	}
	if content.Pressure < 0 || content.Pressure > 1 {
		return fmt.Errorf("PointerEvent pressure is outside 0..1: %f", content.Pressure)
	}
	if content.Width < 0 || content.Height < 0 {
		return errors.New("PointerEvent contact geometry must be non-negative")
	}
	return nil
}

func validateBrowserTouchCapture(eventType string, content BrowserTouchEventContent) error {
	switch eventType {
	case "touchstart", "touchmove", "touchend", "touchcancel":
	default:
		return fmt.Errorf("unsupported TouchEvent type: %q", eventType)
	}
	if len(content.TargetID) > 128 {
		return errors.New("TouchEvent targetId is too long")
	}
	if !allFinite([]float64{content.TimeStamp}) {
		return errors.New("TouchEvent contains a non-finite timestamp")
	}
	for name, points := range map[string][]BrowserTouchPoint{
		"touches":        content.Touches,
		"targetTouches":  content.TargetTouches,
		"changedTouches": content.ChangedTouches,
	} {
		if len(points) > maxBrowserTouchPoints {
			return fmt.Errorf("TouchEvent %s count %d exceeds %d", name, len(points), maxBrowserTouchPoints)
		}
		seen := make(map[int32]struct{}, len(points))
		for _, point := range points {
			if _, exists := seen[point.Identifier]; exists {
				return fmt.Errorf("TouchEvent %s has duplicate identifier %d", name, point.Identifier)
			}
			seen[point.Identifier] = struct{}{}
			if err := validateBrowserTouchPoint(point); err != nil {
				return fmt.Errorf("TouchEvent %s: %w", name, err)
			}
		}
	}
	return nil
}

func validateBrowserTouchPoint(point BrowserTouchPoint) error {
	if !allFinite([]float64{
		point.ClientX,
		point.ClientY,
		point.PageX,
		point.PageY,
		point.ScreenX,
		point.ScreenY,
		point.RadiusX,
		point.RadiusY,
		point.RotationAngle,
		point.Force,
	}) {
		return errors.New("touch point contains a non-finite number")
	}
	if point.RadiusX < 0 || point.RadiusY < 0 {
		return errors.New("touch point radius must be non-negative")
	}
	if point.Force < 0 || point.Force > 1 {
		return fmt.Errorf("touch point force is outside 0..1: %f", point.Force)
	}
	return nil
}

func allFinite(values []float64) bool {
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return true
}

func (router *Router) RecordBrowserCapture(session SessionID, data []byte) error {
	capture, err := decodeBrowserInputCapture(data)
	if err != nil {
		router.browserCaptureRejected.Add(1)
		return err
	}

	source := "browser-pointer-event"
	kind := "pointer-event"
	if capture.API == "TouchEvent" {
		source = "browser-touch-event"
		kind = "touch-event"
	}
	router.browserCaptured.Add(1)
	router.traces.Publish(TraceRecord{
		SessionID:       uint64(session),
		Timestamp:       router.now().UnixMilli(),
		CapturedAtMS:    capture.CapturedAtMS,
		Stage:           "browser-capture",
		Source:          source,
		Kind:            kind,
		CaptureSequence: capture.Sequence,
		BrowserEvent:    &capture,
		Meaning:         capture.API + " " + capture.Type,
	})
	return nil
}
