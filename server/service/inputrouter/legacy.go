package inputrouter

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

func (router *Router) RouteLegacyKeyboardSession(
	session SessionID,
	report []byte,
	sequence uint32,
	capturedAtMS uint32,
) error {
	eventEpoch := router.routingEpoch.Load()
	frame, kind, err := adaptLegacyKeyboard(report, sequence, capturedAtMS)
	return router.routeLegacySession(session, SourceLegacyKeyboard, kind, report, frame, sequence, capturedAtMS, err, eventEpoch)
}

func (router *Router) RouteLegacyPointerSession(
	session SessionID,
	report []byte,
	sequence uint32,
	capturedAtMS uint32,
) error {
	eventEpoch := router.routingEpoch.Load()
	frame, kind, err := adaptLegacyPointer(report, sequence, capturedAtMS)
	return router.routeLegacySession(session, SourceLegacyPointer, kind, report, frame, sequence, capturedAtMS, err, eventEpoch)
}

func (router *Router) routeLegacySession(
	session SessionID,
	source Source,
	kind Kind,
	report []byte,
	frame []byte,
	sequence uint32,
	capturedAtMS uint32,
	adaptErr error,
	eventEpoch uint64,
) error {
	return router.routeAdaptedSession(
		session,
		source,
		kind,
		report,
		frame,
		sequence,
		capturedAtMS,
		adaptErr,
		eventEpoch,
		"legacy-adapted",
		"legacy WebSocket HID report converted to input-router v1",
	)
}

func (router *Router) routeAdaptedSession(
	session SessionID,
	source Source,
	kind Kind,
	report []byte,
	frame []byte,
	sequence uint32,
	capturedAtMS uint32,
	adaptErr error,
	eventEpoch uint64,
	stage string,
	meaning string,
) error {
	now := router.now().UnixMilli()
	record := TraceRecord{
		SessionID:    uint64(session),
		SourceSeq:    sequence,
		Timestamp:    now,
		CapturedAtMS: capturedAtMS,
		Stage:        stage,
		Source:       sourceName(source),
		Kind:         kindName(kind),
		Mode:         modeName(kind),
		Capture:      hex.EncodeToString(report),
		Meaning:      meaning,
	}
	if adaptErr != nil {
		router.received.Add(1)
		router.lastActive.Store(now)
		router.decodeRejected.Add(1)
		router.reject(adaptErr)
		record.Stage = stage + "-error"
		if stage == "legacy-adapted" {
			record.Stage = "legacy-adapt-error"
		}
		record.Error = adaptErr.Error()
		router.traces.Publish(record)
		return adaptErr
	}

	router.traces.Publish(record)
	return router.routeSessionAtEpoch(session, frame, eventEpoch)
}

func adaptLegacyKeyboard(report []byte, sequence, capturedAtMS uint32) ([]byte, Kind, error) {
	if len(report) != 8 {
		return nil, KindKeyboardState, fmt.Errorf("invalid legacy keyboard report length: %d", len(report))
	}
	return semanticFrame(SourceLegacyKeyboard, KindKeyboardState, report, sequence, capturedAtMS), KindKeyboardState, nil
}

func adaptLegacyPointer(report []byte, sequence, capturedAtMS uint32) ([]byte, Kind, error) {
	switch len(report) {
	case 4:
		payload := make([]byte, 7)
		payload[0] = report[0]
		putLegacyInt16(payload[1:3], int16(int8(report[1])))
		putLegacyInt16(payload[3:5], int16(int8(report[2])))
		putLegacyInt16(payload[5:7], int16(int8(report[3])))
		return semanticFrame(SourceLegacyPointer, KindRelativePointer, payload, sequence, capturedAtMS), KindRelativePointer, nil
	case 7:
		return adaptLegacyAbsolutePointer(report, sequence, capturedAtMS)
	case touchscreenReportSize:
		return adaptLegacyTouchscreen(report, sequence, capturedAtMS)
	default:
		return nil, 0, fmt.Errorf("invalid legacy pointer report length: %d", len(report))
	}
}

func adaptLegacyAbsolutePointer(report []byte, sequence, capturedAtMS uint32) ([]byte, Kind, error) {
	x := expandLegacyAxis(binary.LittleEndian.Uint16(report[2:4]))
	y := expandLegacyAxis(binary.LittleEndian.Uint16(report[4:6]))
	switch report[0] {
	case absoluteReportIDMouse:
		payload := make([]byte, 7)
		payload[0] = report[1]
		binary.LittleEndian.PutUint16(payload[1:3], x)
		binary.LittleEndian.PutUint16(payload[3:5], y)
		putLegacyInt16(payload[5:7], int16(int8(report[6])))
		return semanticFrame(SourceLegacyPointer, KindAbsolutePointer, payload, sequence, capturedAtMS), KindAbsolutePointer, nil
	case absoluteReportIDStylus:
		payload := make([]byte, 6)
		payload[0] = report[1]
		binary.LittleEndian.PutUint16(payload[1:3], x)
		binary.LittleEndian.PutUint16(payload[3:5], y)
		payload[5] = report[6]
		return semanticFrame(SourceLegacyPointer, KindStylus, payload, sequence, capturedAtMS), KindStylus, nil
	default:
		return nil, 0, fmt.Errorf("invalid legacy absolute pointer report id: %d", report[0])
	}
}

func adaptLegacyTouchscreen(report []byte, sequence, capturedAtMS uint32) ([]byte, Kind, error) {
	if len(report) != touchscreenReportSize {
		return nil, KindTouchscreen, fmt.Errorf("invalid legacy touchscreen report length: %d", len(report))
	}
	if report[0] != touchscreenReportID {
		return nil, KindTouchscreen, fmt.Errorf("invalid legacy touchscreen report id: %d", report[0])
	}
	count := int(report[len(report)-1])
	if count > MaxContacts {
		return nil, KindTouchscreen, fmt.Errorf("legacy touchscreen contact count %d exceeds %d", count, MaxContacts)
	}

	payload := make([]byte, 1+count*8)
	payload[0] = byte(count)
	for index := 0; index < count; index++ {
		reportOffset := 1 + index*6
		payloadOffset := 1 + index*8
		payload[payloadOffset] = report[reportOffset+1]
		payload[payloadOffset+1] = report[reportOffset]
		binary.LittleEndian.PutUint16(
			payload[payloadOffset+2:payloadOffset+4],
			expandLegacyAxis(binary.LittleEndian.Uint16(report[reportOffset+2:reportOffset+4])),
		)
		binary.LittleEndian.PutUint16(
			payload[payloadOffset+4:payloadOffset+6],
			expandLegacyAxis(binary.LittleEndian.Uint16(report[reportOffset+4:reportOffset+6])),
		)
	}
	return semanticFrame(SourceLegacyPointer, KindTouchscreen, payload, sequence, capturedAtMS), KindTouchscreen, nil
}

func semanticFrame(source Source, kind Kind, payload []byte, sequence, capturedAtMS uint32) []byte {
	frame := make([]byte, HeaderSize+len(payload))
	frame[0] = ProtocolVersion
	frame[1] = byte(source)
	frame[2] = byte(kind)
	binary.LittleEndian.PutUint32(frame[4:8], sequence)
	binary.LittleEndian.PutUint32(frame[8:12], capturedAtMS)
	copy(frame[HeaderSize:], payload)
	return frame
}

func expandLegacyAxis(value uint16) uint16 {
	if value >= 0x7fff {
		return 0xffff
	}
	return uint16((uint32(value)*0xffff + 0x3fff) / 0x7fff)
}

func putLegacyInt16(target []byte, value int16) {
	binary.LittleEndian.PutUint16(target, uint16(value))
}
