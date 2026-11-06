package inputrouter

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const (
	absoluteReportIDMouse  = 0x01
	absoluteReportIDStylus = 0x02
	touchscreenReportID    = 0x03
	touchscreenReportSize  = 62
)

type Output interface {
	WriteKeyboardReport([]byte) error
	WriteRelativeMouseReport([]byte) error
	WriteAbsoluteMouseReport([]byte) error
	WriteTouchScreenReport([]byte) error
}

type InputReleaser interface {
	ReleaseAllInputs() error
}

type EventGate func(Kind) error

type Encoder func(Event) ([]byte, string, error)

type SessionID uint64

type Neutralizer func([]byte) []byte

type Restorer func([]byte) []byte

type Recoverer func(previous, failed []byte) []byte

type RouteDefinition struct {
	Kind                   Kind
	Name                   string
	Encode                 Encoder
	Write                  func([]byte) error
	Neutralize             Neutralizer
	Restore                Restorer
	Recover                Recoverer
	ClearBeforeOwnerChange Neutralizer
	// KeepReleasedPosition keeps the released session's own coordinates when
	// another session still has state. Absolute pointers have no position-free
	// report form, so restoring the other session's stale position would undo a
	// one-shot move instead of only dropping its buttons.
	KeepReleasedPosition bool
	written              *atomic.Uint64
	failed               *atomic.Uint64
	retried              *atomic.Uint64
	dropped              *atomic.Uint64
}

type routeState struct {
	session SessionID
	report  []byte
	updated uint64
}

type Router struct {
	routeMutex             sync.RWMutex
	routes                 map[Kind]RouteDefinition
	dispatch               sync.Mutex
	routingEpoch           atomic.Uint64
	states                 map[Kind]map[SessionID]routeState
	owners                 map[Kind]SessionID
	sessions               map[SessionID]*sessionRuntime
	sources                map[Source]uint64
	stateSeq               atomic.Uint64
	releaser               InputReleaser
	traces                 *TraceHub
	now                    func() time.Time
	received               atomic.Uint64
	dispatched             atomic.Uint64
	rejected               atomic.Uint64
	decodeRejected         atomic.Uint64
	dispatchFailed         atomic.Uint64
	backpressureDropped    atomic.Uint64
	browserCaptured        atomic.Uint64
	browserCaptureRejected atomic.Uint64
	internalSequence       atomic.Uint32
	lastActive             atomic.Int64
	errorMutex             sync.RWMutex
	lastError              string
	eventGate              EventGate
}

func New(output Output, traces *TraceHub) *Router {
	return newRouter(output, traces, nil)
}

func newRouter(output Output, traces *TraceHub, eventGate EventGate) *Router {
	router := NewWithRoutes(defaultRoutes(output), traces)
	router.eventGate = eventGate
	if releaser, ok := output.(InputReleaser); ok {
		router.releaser = releaser
	}
	return router
}

func NewWithRoutes(routes []RouteDefinition, traces *TraceHub) *Router {
	if traces == nil {
		traces = NewTraceHub(256)
	}
	router := &Router{
		routes:   make(map[Kind]RouteDefinition, len(routes)),
		states:   make(map[Kind]map[SessionID]routeState, len(routes)),
		owners:   make(map[Kind]SessionID, len(routes)),
		sessions: make(map[SessionID]*sessionRuntime),
		sources:  make(map[Source]uint64),
		traces:   traces,
		now:      time.Now,
	}
	for _, route := range routes {
		if err := router.Register(route); err != nil {
			panic(err)
		}
	}
	return router
}

func (router *Router) Register(route RouteDefinition) error {
	if route.Name == "" || route.Encode == nil || route.Write == nil || route.Neutralize == nil || route.Restore == nil || route.Recover == nil {
		return errors.New("input route requires a name, encoder, writer, neutralizer, restorer, and recoverer")
	}
	router.routeMutex.Lock()
	defer router.routeMutex.Unlock()
	if _, exists := router.routes[route.Kind]; exists {
		return fmt.Errorf("input route already registered for kind %d", route.Kind)
	}
	route.written = &atomic.Uint64{}
	route.failed = &atomic.Uint64{}
	route.retried = &atomic.Uint64{}
	route.dropped = &atomic.Uint64{}
	router.routes[route.Kind] = route
	return nil
}

func (router *Router) Route(data []byte) error {
	return router.RouteSession(0, data)
}

func (router *Router) RouteVirtualKeyboardSession(session uint64, report []byte) error {
	eventEpoch := router.routingEpoch.Load()
	sequence := router.internalSequence.Add(1)
	capturedAtMS := uint32(router.now().UnixMilli())
	frame, kind, err := adaptLegacyKeyboard(report, sequence, capturedAtMS)
	if len(frame) > 1 {
		frame[1] = byte(SourceVirtualKeyboard)
	}
	return router.routeAdaptedSession(
		SessionID(session),
		SourceVirtualKeyboard,
		kind,
		report,
		frame,
		sequence,
		capturedAtMS,
		err,
		eventEpoch,
		"internal-adapted",
		"virtual keyboard HID report converted to input-router v1",
	)
}

func (router *Router) RouteVirtualPointerSession(session uint64, report []byte) error {
	eventEpoch := router.routingEpoch.Load()
	sequence := router.internalSequence.Add(1)
	capturedAtMS := uint32(router.now().UnixMilli())
	frame, kind, err := adaptLegacyPointer(report, sequence, capturedAtMS)
	if len(frame) > 1 {
		frame[1] = byte(SourceVirtualPointer)
	}
	return router.routeAdaptedSession(
		SessionID(session),
		SourceVirtualPointer,
		kind,
		report,
		frame,
		sequence,
		capturedAtMS,
		err,
		eventEpoch,
		"internal-adapted",
		"virtual pointer HID report converted to input-router v1",
	)
}

func (router *Router) RouteVirtualTouchSession(session uint64, report []byte) error {
	eventEpoch := router.routingEpoch.Load()
	sequence := router.internalSequence.Add(1)
	capturedAtMS := uint32(router.now().UnixMilli())
	frame, kind, err := adaptLegacyTouchscreen(report, sequence, capturedAtMS)
	if len(frame) > 1 {
		frame[1] = byte(SourceVirtualTouch)
	}
	return router.routeAdaptedSession(
		SessionID(session),
		SourceVirtualTouch,
		kind,
		report,
		frame,
		sequence,
		capturedAtMS,
		err,
		eventEpoch,
		"internal-adapted",
		"virtual touchscreen HID report converted to input-router v1",
	)
}

func (router *Router) ReleaseInputSession(session uint64, reason string) error {
	return router.ReleaseSession(SessionID(session), reason)
}

func (router *Router) ResetInputSessions(reason string) error {
	router.dispatch.Lock()
	defer router.dispatch.Unlock()
	return router.releaseAndClearInputSessionsLocked("reset", reason)
}

func (router *Router) RouteSession(session SessionID, data []byte) error {
	return router.routeSessionAtEpoch(session, data, router.routingEpoch.Load())
}

func (router *Router) routeSessionAtEpoch(session SessionID, data []byte, eventEpoch uint64) error {
	routerStarted := time.Now()
	router.received.Add(1)
	now := router.now().UnixMilli()
	router.lastActive.Store(now)

	event, err := Decode(data)
	if err != nil {
		router.decodeRejected.Add(1)
		router.reject(err)
		router.traces.Publish(TraceRecord{
			Timestamp: now,
			Stage:     "decode",
			Capture:   hex.EncodeToString(data),
			Error:     err.Error(),
		})
		return err
	}

	record := traceEvent(event, now)
	record.SessionID = uint64(session)
	route, ok := router.route(event.Kind)
	if ok {
		record.Route = route.Name
	}
	router.dispatch.Lock()
	defer router.dispatch.Unlock()
	if event.Kind == KindPointerRelease {
		if gateErr, stage, meaning := inputEpochGate(eventEpoch, router.routingEpoch.Load()); gateErr != nil {
			err = gateErr
			record.Stage = stage
			record.Meaning = meaning
		} else if router.eventGate != nil {
			err = router.eventGate(event.Kind)
			if err != nil {
				record.Stage = "mode-gate"
				record.Meaning = "input kind rejected by active pointer mode"
			}
		}
		if err == nil {
			err = router.routePointerReleaseLocked(session, event, &record, now)
		} else {
			router.dispatchFailed.Add(1)
			router.reject(err)
			record.Error = err.Error()
		}
		record.RouterDurationUS = time.Since(routerStarted).Microseconds()
		router.traces.Publish(record)
		return err
	}
	if !ok {
		err = fmt.Errorf("no route registered for input event kind: %d", event.Kind)
	} else {
		if gateErr, stage, meaning := inputEpochGate(eventEpoch, router.routingEpoch.Load()); gateErr != nil {
			err = gateErr
			record.Stage = stage
			record.Meaning = meaning
		} else if router.eventGate != nil {
			err = router.eventGate(event.Kind)
			if err != nil {
				record.Stage = "mode-gate"
				record.Meaning = "input kind rejected by active pointer mode"
			}
		}
		var report []byte
		var meaning string
		if err == nil {
			report, meaning, err = route.Encode(event)
			record.Report = hex.EncodeToString(report)
			record.Meaning = meaning
		}
		if err == nil {
			owner, hasOwner := router.owners[event.Kind]
			if route.ClearBeforeOwnerChange != nil && hasOwner && owner != session {
				err = router.clearBeforeOwnerChange(
					route,
					session,
					event.Kind,
					"owner-switch-clear",
					fmt.Sprintf("clear owner session %d before session %d", owner, session),
				)
				if err == nil {
					delete(router.owners, event.Kind)
				}
			}
		}
		if err == nil {
			var result routeWriteResult
			writeStarted := time.Now()
			result, err = writeRouteReport(route, router.sessionReport(event.Kind, session), report)
			record.HIDWriteUS = time.Since(writeStarted).Microseconds()
			if result.report != nil {
				report = result.report
				record.Report = hex.EncodeToString(report)
			}
			if result.dropped {
				router.backpressureDropped.Add(1)
				record.Stage = "backpressure-drop"
				record.Error = result.cause.Error()
				record.RouterDurationUS = time.Since(routerStarted).Microseconds()
				router.traces.Publish(record)
				return nil
			}
			if result.retried {
				record.Stage = "dispatched-retry"
			}
		}
		if err == nil {
			route.written.Add(1)
			states := router.states[event.Kind]
			if states == nil {
				states = make(map[SessionID]routeState)
				router.states[event.Kind] = states
			}
			states[session] = routeState{
				session: session,
				report:  append([]byte(nil), report...),
				updated: router.stateSeq.Add(1),
			}
			router.owners[event.Kind] = session
			router.recordSessionEvent(session, event, route.Name, now)
		}
	}

	if err != nil {
		router.dispatchFailed.Add(1)
		if ok {
			route.failed.Add(1)
		}
		router.reject(err)
		if record.Stage == "" {
			record.Stage = "error"
		}
		record.Error = err.Error()
	} else {
		router.dispatched.Add(1)
		if record.Stage == "" {
			record.Stage = "dispatched"
		}
	}
	record.RouterDurationUS = time.Since(routerStarted).Microseconds()
	router.traces.Publish(record)
	return err
}

func (router *Router) ReleaseSession(session SessionID, reason string) error {
	router.dispatch.Lock()
	defer router.dispatch.Unlock()

	var releaseErrors []error
	delete(router.sessions, session)
	for kind := range router.states {
		if err := router.releaseRouteLocked(kind, session, "release", reason); err != nil {
			releaseErrors = append(releaseErrors, err)
		}
	}
	return errors.Join(releaseErrors...)
}

func (router *Router) releaseRouteLocked(kind Kind, session SessionID, stage string, reason string) error {
	states := router.states[kind]
	if states == nil {
		return nil
	}
	released, exists := states[session]
	if !exists {
		return nil
	}
	delete(states, session)
	owner, hasOwner := router.owners[kind]
	if !hasOwner || owner != session {
		return nil
	}
	delete(router.owners, kind)
	route, ok := router.route(kind)
	if !ok {
		return nil
	}
	var report []byte
	var nextSession SessionID
	var latest routeState
	for candidateSession, candidate := range states {
		if candidate.updated > latest.updated {
			latest = candidate
			nextSession = candidateSession
		}
	}
	if latest.updated == 0 || route.KeepReleasedPosition {
		report = route.Neutralize(released.report)
	} else {
		report = route.Restore(latest.report)
		stage = "restore"
	}
	var err error
	if latest.updated != 0 && route.ClearBeforeOwnerChange != nil {
		err = router.clearBeforeOwnerChange(
			route,
			session,
			kind,
			"restore-clear",
			fmt.Sprintf("clear session %d before restoring session %d", session, nextSession),
		)
	}
	if err == nil {
		err = writeReleaseReport(route, report)
	}
	record := TraceRecord{
		SessionID: uint64(session),
		Timestamp: router.now().UnixMilli(),
		Stage:     stage,
		Kind:      kindName(kind),
		Route:     route.Name,
		Report:    hex.EncodeToString(report),
		Meaning:   reason,
	}
	if err != nil {
		router.dispatchFailed.Add(1)
		route.failed.Add(1)
		router.reject(err)
		record.Error = err.Error()
		router.traces.Publish(record)
		return fmt.Errorf("release %s: %w", route.Name, err)
	}
	if latest.updated != 0 {
		router.owners[kind] = nextSession
	}
	router.traces.Publish(record)
	return nil
}

func (router *Router) routePointerReleaseLocked(session SessionID, event Event, record *TraceRecord, now int64) error {
	var releaseErrors []error
	for _, candidate := range []struct {
		mask uint8
		kind Kind
	}{
		{mask: ReleaseRouteRelativePointer, kind: KindRelativePointer},
		{mask: ReleaseRouteAbsolutePointer, kind: KindAbsolutePointer},
		{mask: ReleaseRouteStylus, kind: KindStylus},
		{mask: ReleaseRouteTouchscreen, kind: KindTouchscreen},
	} {
		if event.PointerRelease.Routes&candidate.mask == 0 {
			continue
		}
		routeName := ""
		if route, ok := router.route(candidate.kind); ok {
			routeName = route.Name
		}
		if router.eventGate != nil {
			if gateErr := router.eventGate(candidate.kind); gateErr != nil {
				router.dropReleasedRouteLocked(candidate.kind, session)
				router.dropSessionRouteRecord(session, routeName)
				router.traces.Publish(TraceRecord{
					SessionID: uint64(session),
					Timestamp: now,
					Stage:     "mode-gate",
					Kind:      kindName(candidate.kind),
					Route:     routeName,
					Meaning:   "input kind rejected by active pointer mode",
					Error:     gateErr.Error(),
				})
				continue
			}
		}
		if states := router.states[candidate.kind]; states != nil {
			if state, ok := states[session]; ok && (candidate.kind == KindAbsolutePointer || candidate.kind == KindStylus) && len(state.report) == 7 && state.report[1] == 0 {
				router.dropReleasedRouteLocked(candidate.kind, session)
				router.dropSessionRouteRecord(session, routeName)
				continue
			}
		}
		releaseErr := router.releaseRouteLocked(candidate.kind, session, "release", "client relinquished pointer routes")
		router.dropSessionRouteRecord(session, routeName)
		if releaseErr != nil {
			releaseErrors = append(releaseErrors, releaseErr)
		}
	}
	if record.Stage == "" {
		record.Stage = "client-release"
	}
	record.Meaning = "client relinquished pointer routes"
	if len(releaseErrors) > 0 {
		err := errors.Join(releaseErrors...)
		record.Error = err.Error()
		return err
	}
	router.dispatched.Add(1)
	return nil
}

func (router *Router) dropReleasedRouteLocked(kind Kind, session SessionID) {
	if states := router.states[kind]; states != nil {
		delete(states, session)
	}
	if owner, hasOwner := router.owners[kind]; hasOwner && owner == session {
		delete(router.owners, kind)
	}
}

func (router *Router) dropSessionRouteRecord(session SessionID, routeName string) {
	if routeName == "" {
		return
	}
	runtime := router.sessions[session]
	if runtime == nil {
		return
	}
	delete(runtime.routes, routeName)
}

func (router *Router) clearBeforeOwnerChange(
	route RouteDefinition,
	session SessionID,
	kind Kind,
	stage string,
	meaning string,
) error {
	report := route.ClearBeforeOwnerChange(nil)
	err := writeReleaseReport(route, report)
	record := TraceRecord{
		SessionID: uint64(session),
		Timestamp: router.now().UnixMilli(),
		Stage:     stage,
		Kind:      kindName(kind),
		Route:     route.Name,
		Report:    hex.EncodeToString(report),
		Meaning:   meaning,
	}
	if err != nil {
		record.Error = err.Error()
	}
	router.traces.Publish(record)
	return err
}

const (
	criticalWriteAttempts   = 3
	criticalWriteRetryDelay = 5 * time.Millisecond
)

func writeReleaseReport(route RouteDefinition, report []byte) error {
	// An empty report means there is no observed state to release (for
	// example an absolute pointer that never reported a position); skipping
	// the write is success, not a dispatch failure.
	if len(report) == 0 {
		return nil
	}
	var err error
	for attempt := 0; attempt < criticalWriteAttempts; attempt++ {
		err = route.Write(report)
		if err == nil {
			return nil
		}
		if !errors.Is(err, os.ErrDeadlineExceeded) && !errors.Is(err, os.ErrClosed) {
			return err
		}
		if attempt+1 < criticalWriteAttempts {
			route.retried.Add(1)
			time.Sleep(criticalWriteRetryDelay)
		}
	}
	return err
}

type routeWriteResult struct {
	report  []byte
	cause   error
	retried bool
	dropped bool
}

func writeRouteReport(route RouteDefinition, previous, report []byte) (routeWriteResult, error) {
	result := routeWriteResult{report: report}
	err := route.Write(report)
	if err == nil {
		return result, nil
	}
	if !transientWriteError(err) {
		return result, err
	}

	recovery := route.Recover(previous, report)
	if recovery == nil {
		route.dropped.Add(1)
		result.cause = err
		result.dropped = true
		return result, nil
	}
	result.report = recovery
	for attempt := 1; attempt < criticalWriteAttempts; attempt++ {
		route.retried.Add(1)
		result.retried = true
		time.Sleep(criticalWriteRetryDelay)
		err = route.Write(recovery)
		if err == nil {
			return result, nil
		}
		if !transientWriteError(err) {
			return result, err
		}
	}
	return result, err
}

func transientWriteError(err error) bool {
	return errors.Is(err, os.ErrDeadlineExceeded) || errors.Is(err, os.ErrClosed)
}

func (router *Router) Release(reason string) error {
	router.dispatch.Lock()
	defer router.dispatch.Unlock()
	if router.releaser == nil {
		return nil
	}
	err := router.releaser.ReleaseAllInputs()
	if err == nil {
		clear(router.states)
		clear(router.owners)
		clear(router.sessions)
	}
	record := TraceRecord{
		Timestamp: router.now().UnixMilli(),
		Stage:     "release",
		Meaning:   reason,
	}
	if err != nil {
		router.dispatchFailed.Add(1)
		router.reject(err)
		record.Error = err.Error()
	}
	router.traces.Publish(record)
	return err
}

func (router *Router) Traces() *TraceHub {
	return router.traces
}

func (router *Router) route(kind Kind) (RouteDefinition, bool) {
	router.routeMutex.RLock()
	defer router.routeMutex.RUnlock()
	route, ok := router.routes[kind]
	return route, ok
}

func (router *Router) sessionReport(kind Kind, session SessionID) []byte {
	states := router.states[kind]
	if states == nil {
		return nil
	}
	return states[session].report
}

func (router *Router) reject(err error) {
	router.rejected.Add(1)
	router.errorMutex.Lock()
	router.lastError = err.Error()
	router.errorMutex.Unlock()
}

func defaultRoutes(output Output) []RouteDefinition {
	return []RouteDefinition{
		{Kind: KindKeyboardState, Name: "hidg0/keyboard", Encode: encodeKeyboard, Write: output.WriteKeyboardReport, Neutralize: neutralKeyboard, Restore: cloneReport, Recover: recoverStateReport},
		{Kind: KindRelativePointer, Name: "hidg1/relative-pointer", Encode: encodeRelativePointer, Write: output.WriteRelativeMouseReport, Neutralize: neutralRelativePointer, Restore: restoreRelativePointer, Recover: recoverRelativePointer},
		{Kind: KindAbsolutePointer, Name: "hidg2/absolute-pointer", Encode: encodeAbsolutePointer, Write: output.WriteAbsoluteMouseReport, Neutralize: neutralAbsoluteMouse, Restore: restoreAbsolutePointer, Recover: recoverAbsolutePointer, KeepReleasedPosition: true},
		{Kind: KindStylus, Name: "hidg2/stylus", Encode: encodeStylus, Write: output.WriteAbsoluteMouseReport, Neutralize: neutralStylus, Restore: cloneReport, Recover: recoverAbsolutePointer, KeepReleasedPosition: true},
		{Kind: KindTouchscreen, Name: "hidg2/touchscreen", Encode: encodeTouchscreen, Write: output.WriteTouchScreenReport, Neutralize: neutralTouchscreen, Restore: cloneReport, Recover: recoverTouchscreen, ClearBeforeOwnerChange: neutralTouchscreen},
	}
}

func cloneReport(last []byte) []byte {
	return append([]byte(nil), last...)
}

func recoverStateReport(_, failed []byte) []byte {
	return cloneReport(failed)
}

func recoverRelativePointer(previous, failed []byte) []byte {
	if len(failed) != 4 {
		return nil
	}
	if len(previous) == 4 && previous[0] == failed[0] {
		return nil
	}
	return []byte{failed[0], 0, 0, 0}
}

func recoverAbsolutePointer(previous, failed []byte) []byte {
	if len(failed) != 7 {
		return nil
	}
	if len(previous) == 7 && previous[0] == failed[0] && previous[1] == failed[1] {
		return nil
	}
	return cloneReport(failed)
}

func recoverTouchscreen(previous, failed []byte) []byte {
	if len(failed) != touchscreenReportSize {
		return nil
	}
	if len(previous) != touchscreenReportSize {
		previous = neutralTouchscreen(nil)
	}
	if sameTouchTopology(previous, failed) {
		return nil
	}
	return cloneReport(failed)
}

func sameTouchTopology(left, right []byte) bool {
	if len(left) != touchscreenReportSize || len(right) != touchscreenReportSize {
		return false
	}
	if left[touchscreenReportSize-1] != right[touchscreenReportSize-1] {
		return false
	}
	for offset := 1; offset < touchscreenReportSize-1; offset += 6 {
		if left[offset] != right[offset] || left[offset+1] != right[offset+1] {
			return false
		}
	}
	return true
}

func neutralKeyboard([]byte) []byte {
	return make([]byte, 8)
}

func neutralRelativePointer([]byte) []byte {
	return make([]byte, 4)
}

func restoreRelativePointer(last []byte) []byte {
	report := make([]byte, 4)
	if len(last) == len(report) {
		report[0] = last[0]
	}
	return report
}

func neutralAbsoluteMouse(last []byte) []byte {
	return neutralAbsolutePointer(absoluteReportIDMouse, last)
}

func neutralStylus(last []byte) []byte {
	return neutralAbsolutePointer(absoluteReportIDStylus, last)
}

// neutralAbsolutePointer reuses the last observed absolute position with
// buttons and wheel cleared. It returns nil when no usable position report
// exists.
func neutralAbsolutePointer(reportID byte, last []byte) []byte {
	if len(last) != 7 {
		return nil
	}
	return []byte{reportID, 0, last[2], last[3], last[4], last[5], 0}
}

func restoreAbsolutePointer(last []byte) []byte {
	if len(last) != 7 {
		return nil
	}
	return []byte{last[0], last[1], last[2], last[3], last[4], last[5], 0}
}

func neutralTouchscreen([]byte) []byte {
	report := make([]byte, touchscreenReportSize)
	report[0] = touchscreenReportID
	for index := 0; index < MaxContacts; index++ {
		report[1+index*6+1] = byte(index + 1)
	}
	report[len(report)-1] = MaxContacts
	return report
}

func encodeKeyboard(event Event) ([]byte, string, error) {
	report := append([]byte(nil), event.Keyboard.Report[:]...)
	return report, fmt.Sprintf("keyboard modifiers=0x%02x", report[0]), nil
}

func encodeRelativePointer(event Event) ([]byte, string, error) {
	pointer := event.Relative
	report := []byte{
		pointer.Buttons,
		byte(clampInt16(pointer.DeltaX, -127, 127)),
		byte(clampInt16(pointer.DeltaY, -127, 127)),
		byte(clampInt16(pointer.Wheel, -127, 127)),
	}
	meaning := fmt.Sprintf(
		"relative pointer buttons=0x%02x dx=%d dy=%d wheel=%d",
		pointer.Buttons,
		pointer.DeltaX,
		pointer.DeltaY,
		pointer.Wheel,
	)
	return report, meaning, nil
}

func encodeAbsolutePointer(event Event) ([]byte, string, error) {
	pointer := event.Absolute
	report := absoluteReport(
		absoluteReportIDMouse,
		pointer.Buttons,
		pointer.X,
		pointer.Y,
		byte(clampInt16(pointer.Wheel, -127, 127)),
	)
	meaning := fmt.Sprintf(
		"absolute pointer buttons=0x%02x x=%.4f y=%.4f wheel=%d",
		pointer.Buttons,
		normalizeAxis(pointer.X),
		normalizeAxis(pointer.Y),
		pointer.Wheel,
	)
	return report, meaning, nil
}

func encodeStylus(event Event) ([]byte, string, error) {
	stylus := event.Stylus
	report := absoluteReport(
		absoluteReportIDStylus,
		stylus.State,
		stylus.X,
		stylus.Y,
		stylus.Pressure,
	)
	meaning := fmt.Sprintf(
		"stylus state=0x%02x x=%.4f y=%.4f pressure=%d",
		stylus.State,
		normalizeAxis(stylus.X),
		normalizeAxis(stylus.Y),
		stylus.Pressure,
	)
	return report, meaning, nil
}

func encodeTouchscreen(event Event) ([]byte, string, error) {
	report := make([]byte, touchscreenReportSize)
	report[0] = touchscreenReportID
	for index, contact := range event.Touchscreen.Contacts {
		offset := 1 + index*6
		report[offset] = contact.State & 0x03
		report[offset+1] = contact.ID
		binary.LittleEndian.PutUint16(report[offset+2:offset+4], toHIDAxis(contact.X))
		binary.LittleEndian.PutUint16(report[offset+4:offset+6], toHIDAxis(contact.Y))
	}
	report[len(report)-1] = byte(len(event.Touchscreen.Contacts))
	return report, fmt.Sprintf("touchscreen contacts=%d", len(event.Touchscreen.Contacts)), nil
}

func absoluteReport(reportID, state uint8, x, y uint16, tail uint8) []byte {
	report := make([]byte, 7)
	report[0] = reportID
	report[1] = state
	binary.LittleEndian.PutUint16(report[2:4], toHIDAxis(x))
	binary.LittleEndian.PutUint16(report[4:6], toHIDAxis(y))
	report[6] = tail
	return report
}

func toHIDAxis(value uint16) uint16 {
	return uint16(math.Round(float64(value) * 0x7fff / float64(^uint16(0))))
}

func clampInt16(value, min, max int16) int16 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
