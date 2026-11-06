package inputrouter

import (
	"errors"
	"testing"

	"NanoKVM-Server/service/hid"
)

func TestPointerModeEventGateMapping(t *testing.T) {
	tests := []struct {
		mode        hid.PointerMode
		allowedKind Kind
	}{
		{mode: hid.PointerModeAbsolute, allowedKind: KindAbsolutePointer},
		{mode: hid.PointerModeDigitizerStylus, allowedKind: KindStylus},
		{mode: hid.PointerModeTouchScreen, allowedKind: KindTouchscreen},
	}
	kinds := []Kind{
		KindKeyboardState,
		KindRelativePointer,
		KindAbsolutePointer,
		KindStylus,
		KindTouchscreen,
	}

	for _, test := range tests {
		gate := pointerModeEventGate(func() hid.PointerModeStatus {
			return hid.PointerModeStatus{Mode: test.mode}
		})
		for _, kind := range kinds {
			err := gate(kind)
			isHID2 := kind == KindAbsolutePointer || kind == KindStylus || kind == KindTouchscreen
			if !isHID2 || kind == test.allowedKind {
				if err != nil {
					t.Errorf("mode %q rejected kind %s: %v", test.mode, kindName(kind), err)
				}
				continue
			}
			if !errors.Is(err, hid.ErrPointerModeMismatch) {
				t.Errorf("mode %q kind %s error = %v, want pointer mode mismatch", test.mode, kindName(kind), err)
			}
		}
	}
}

func TestRouterModeGateAllowsOnlyMappedHID2Route(t *testing.T) {
	routes := []struct {
		kind  Kind
		frame []byte
	}{
		{
			kind: KindAbsolutePointer,
			frame: append(
				makeHeader(SourceBrowserPointer, KindAbsolutePointer),
				absolutePayload(0, 0x8000, 0x8000, 0)...,
			),
		},
		{
			kind: KindStylus,
			frame: append(
				makeHeader(SourceBrowserPointer, KindStylus),
				stylusPayload(0, 0x8000, 0x8000, 0)...,
			),
		},
		{
			kind:  KindTouchscreen,
			frame: append(makeHeader(SourceBrowserTouch, KindTouchscreen), 0),
		},
	}
	modes := []struct {
		mode        hid.PointerMode
		allowedKind Kind
	}{
		{mode: hid.PointerModeAbsolute, allowedKind: KindAbsolutePointer},
		{mode: hid.PointerModeDigitizerStylus, allowedKind: KindStylus},
		{mode: hid.PointerModeTouchScreen, allowedKind: KindTouchscreen},
	}

	for _, mode := range modes {
		for _, route := range routes {
			t.Run(string(mode.mode)+"/"+kindName(route.kind), func(t *testing.T) {
				output := &fakeOutput{}
				router := newRouter(output, NewTraceHub(4), pointerModeEventGate(func() hid.PointerModeStatus {
					return hid.PointerModeStatus{Mode: mode.mode}
				}))

				err := router.RouteSession(7, route.frame)
				if route.kind == mode.allowedKind {
					if err != nil {
						t.Fatalf("allowed route error: %v", err)
					}
					if len(output.writes) != 1 {
						t.Fatalf("allowed route writes = %v, want one", output.writes)
					}
					return
				}

				if !errors.Is(err, hid.ErrPointerModeMismatch) {
					t.Fatalf("inactive route error = %v, want pointer mode mismatch", err)
				}
				if len(output.writes) != 0 {
					t.Fatalf("inactive route writes = %v", output.writes)
				}
			})
		}
	}
}

func TestRouterModeGateRejectsBeforeWriteOrState(t *testing.T) {
	output := &fakeOutput{}
	traces := NewTraceHub(8)
	router := newRouter(output, traces, pointerModeEventGate(func() hid.PointerModeStatus {
		return hid.PointerModeStatus{Mode: hid.PointerModeAbsolute}
	}))
	frame := append(
		makeHeader(SourceBrowserPointer, KindStylus),
		stylusPayload(0x03, 0x8000, 0x8000, 0xff)...,
	)

	err := router.RouteSession(41, frame)
	if !errors.Is(err, hid.ErrPointerModeMismatch) {
		t.Fatalf("RouteSession error = %v, want pointer mode mismatch", err)
	}
	if len(output.writes) != 0 {
		t.Fatalf("mode-mismatched route wrote reports: %v", output.writes)
	}
	if err := router.ReleaseSession(41, "rejected session"); err != nil {
		t.Fatalf("ReleaseSession after rejection: %v", err)
	}
	if len(output.writes) != 0 {
		t.Fatalf("rejected state was restored or released later: %v", output.writes)
	}

	status := router.Status()
	if status.Received != 1 || status.Dispatched != 0 || status.Rejected != 1 || status.DispatchFailed != 1 {
		t.Fatalf("runtime status after mode rejection = %+v", status)
	}
	if status.ActiveSessions != 0 || len(status.Sessions) != 0 || len(status.Sources) != 0 {
		t.Fatalf("mode rejection retained session state: %+v", status)
	}
	for _, route := range status.Routes {
		if route.Route == "hidg2/stylus" && (route.Owner != 0 || route.Dispatched != 0 || route.Failed != 1) {
			t.Fatalf("stylus route after mode rejection = %+v", route)
		}
	}
	records := traces.Snapshot(1)
	if len(records) != 1 || records[0].Stage != "mode-gate" || records[0].Error == "" {
		t.Fatalf("mode rejection trace = %+v", records)
	}
}

func TestPointerModeGateLeavesKeyboardAndRelativePointerUnaffected(t *testing.T) {
	output := &fakeOutput{}
	router := newRouter(output, NewTraceHub(8), pointerModeEventGate(func() hid.PointerModeStatus {
		return hid.PointerModeStatus{Mode: hid.PointerModeTouchScreen}
	}))

	keyboard := append(makeHeader(SourceBrowserKeyboard, KindKeyboardState), make([]byte, 8)...)
	if err := router.RouteSession(1, keyboard); err != nil {
		t.Fatalf("route keyboard: %v", err)
	}
	relative := append(makeHeader(SourceBrowserPointer, KindRelativePointer), make([]byte, 7)...)
	if err := router.RouteSession(1, relative); err != nil {
		t.Fatalf("route relative pointer: %v", err)
	}
	if len(output.writes) != 2 || output.writes[0] == nil || output.writes[1] == nil {
		t.Fatalf("unaffected route writes = %v", output.writes)
	}
}

func TestPointerModeGateCoversAdaptedAndWebHIDInputs(t *testing.T) {
	touch := make([]byte, touchscreenReportSize)
	touch[0] = touchscreenReportID

	tests := []struct {
		name  string
		mode  hid.PointerMode
		route func(*Router) error
	}{
		{
			name: "AI absolute producer",
			mode: hid.PointerModeTouchScreen,
			route: func(router *Router) error {
				return router.RouteVirtualPointerSession(hid.InternalInputSessionAI, []byte{absoluteReportIDMouse, 0, 0, 0, 0, 0, 0})
			},
		},
		{
			name: "absolute jiggler producer",
			mode: hid.PointerModeDigitizerStylus,
			route: func(router *Router) error {
				return router.RouteVirtualPointerSession(hid.InternalInputSessionJiggler, []byte{absoluteReportIDMouse, 0, 0, 0, 0, 0, 0})
			},
		},
		{
			name: "virtual touchscreen producer",
			mode: hid.PointerModeAbsolute,
			route: func(router *Router) error {
				return router.RouteVirtualTouchSession(1002, touch)
			},
		},
		{
			name: "legacy stylus",
			mode: hid.PointerModeAbsolute,
			route: func(router *Router) error {
				return router.RouteLegacyPointerSession(
					1003,
					[]byte{absoluteReportIDStylus, 0, 0, 0, 0, 0, 0},
					1,
					1,
				)
			},
		},
		{
			name: "webhid touchscreen",
			mode: hid.PointerModeAbsolute,
			route: func(router *Router) error {
				return router.RouteSession(
					1004,
					append(makeHeader(SourceWebHID, KindTouchscreen), []byte{0}...),
				)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := &fakeOutput{}
			router := newRouter(output, NewTraceHub(8), pointerModeEventGate(func() hid.PointerModeStatus {
				return hid.PointerModeStatus{Mode: test.mode}
			}))

			err := test.route(router)
			if !errors.Is(err, hid.ErrPointerModeMismatch) {
				t.Fatalf("route error = %v, want pointer mode mismatch", err)
			}
			if len(output.writes) != 0 {
				t.Fatalf("mode-mismatched adapted input wrote reports: %v", output.writes)
			}
			status := router.Status()
			if status.ActiveSessions != 0 || len(status.Sessions) != 0 {
				t.Fatalf("mode-mismatched adapted input retained state: %+v", status)
			}
		})
	}
}
