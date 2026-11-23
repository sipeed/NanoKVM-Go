package ai

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"testing"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/hid"
	"NanoKVM-Server/service/inputrouter"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const dragName = "drag_mouse"

func TestMCPDragSchemaAndValidation(t *testing.T) {
	installDragCalibration(t)
	output := &dragRouteOutput{}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	server, _ := newDragTestServer(t, output)
	session := connectInMemoryMCP(t, ctx, server)
	list, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	var dragTool *mcp.Tool
	for _, tool := range list.Tools {
		if tool.Name == dragName {
			dragTool = tool
			break
		}
	}
	if dragTool == nil {
		t.Fatalf("MCP tools do not include %s", dragName)
	}

	data, err := json.Marshal(dragTool.InputSchema)
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Properties map[string]struct {
			Type json.RawMessage
		}
		Required []string
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	slices.Sort(schema.Required)
	if !slices.Equal(schema.Required, []string{"endX", "endY", "startX", "startY"}) {
		t.Errorf("required fields = %v", schema.Required)
	}
	durationType := schema.Properties["durationMs"].Type
	var nullableDurationTypes []string
	_ = json.Unmarshal(durationType, &nullableDurationTypes)
	if len(schema.Properties) != 5 || (string(durationType) != `"integer"` && !slices.Contains(nullableDurationTypes, "integer")) {
		t.Errorf("expected four coordinates and an optional integer durationMs, got %s", data)
	}
	for _, field := range []string{"startX", "startY", "endX", "endY"} {
		if string(schema.Properties[field].Type) != `"number"` {
			t.Errorf("%s should be a number, got %s", field, data)
		}
		t.Run("missing "+field, func(t *testing.T) {
			args := dragArguments()
			delete(args, field)
			assertMCPDragRejected(t, ctx, session, args)
		})
	}
	for _, test := range []struct {
		name  string
		field string
		value any
	}{
		{"negative start", "startX", -0.1},
		{"pixel endpoint", "endY", 300},
		{"null coordinate", "startX", nil},
		{"short duration", "durationMs", 99},
		{"long duration", "durationMs", 1001},
		{"fractional duration", "durationMs", 100.5},
	} {
		t.Run(test.name, func(t *testing.T) {
			args := dragArguments()
			args[test.field] = test.value
			assertMCPDragRejected(t, ctx, session, args)
		})
	}
	if len(output.writes) != 0 {
		t.Fatalf("invalid requests touched HID: %x", output.writes)
	}
}

func TestMCPDragFollowsCalibrationAndReleases(t *testing.T) {
	portal, profile := installDragCalibration(t)
	output := &dragRouteOutput{beforeWrite: func(report []byte) error {
		if bytes.Equal(report, []byte{1, 0, 0, 0}) {
			*portal = proto.PortalConfig{Mode: "none"}
			*profile = proto.InputProfileConfig{SelectedProfileID: defaultInputProfileID}
		}
		return nil
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	server, _ := newDragTestServer(t, output)
	session := connectInMemoryMCP(t, ctx, server)
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: dragName, Arguments: dragArguments()})
	if err != nil || result == nil || result.IsError {
		t.Fatalf("MCP drag = %+v, %v", result, err)
	}
	data, err := json.Marshal(result.StructuredContent)
	if err != nil || string(data) != `{"ok":true}` {
		t.Errorf("MCP drag result = %s, %v", data, err)
	}

	// Crop maps the diagonal to (0.125, 0.125)..(0.875, 0.875) in
	// source space. The profile then maps x to clamp(2*x-0.5)
	// and y to 0.5*y+0.25, even after the active settings change.
	start := []byte{1, 0, 0x00, 0x00, 0x00, 0x28, 0}
	end := []byte{1, 0, 0xff, 0x7f, 0xff, 0x57, 0}
	held := assertDragReports(t, output.writes, start, end)
	foundClippedInterior := false
	for _, report := range held {
		y := binary.LittleEndian.Uint16(report[4:6])
		if y > 0x2800 && y < 0x57ff && binary.LittleEndian.Uint16(report[2:4]) == 0 {
			foundClippedInterior = true
		}
	}
	if !foundClippedInterior {
		t.Errorf("drag never applied calibration to an intermediate content point: %x", held)
	}

	result, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name: "move_mouse", Arguments: map[string]any{"mode": "absolute", "x": 0.25, "y": 0.25},
	})
	if err != nil || result == nil || result.IsError {
		t.Fatalf("move_mouse after drag = %+v, %v", result, err)
	}
	if got := output.writes[len(output.writes)-1]; !bytes.Equal(got, []byte{1, 0, 0, 0x20, 0, 0x20, 0}) {
		t.Errorf("move after release = %x", got)
	}
}

func TestMCPDragWithPortalOffOverHTTP(t *testing.T) {
	portal, _ := installDragCalibration(t)
	portal.Mode = "none"
	output := &dragRouteOutput{}
	server, _ := newDragTestServer(t, output)
	httpServer := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil))
	t.Cleanup(httpServer.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: httpServer.URL, HTTPClient: httpServer.Client(), MaxRetries: -1, DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	args := dragArguments()
	args["durationMs"] = 100
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: dragName, Arguments: args})
	if err != nil || result == nil || result.IsError {
		t.Fatalf("MCP drag over HTTP = %+v, %v", result, err)
	}
	start := []byte{1, 0, 0, 0, 0, 0x20, 0}
	end := []byte{1, 0, 0xff, 0x7f, 0xff, 0x5f, 0}
	assertDragReports(t, output.writes, start, end)
}

func TestMCPDragReleasesAfterHIDFailure(t *testing.T) {
	installDragCalibration(t)
	pressed := false
	output := &dragRouteOutput{beforeWrite: func(report []byte) error {
		if bytes.Equal(report, []byte{1, 0, 0, 0}) {
			pressed = true
		}
		if pressed && len(report) == 7 && binary.LittleEndian.Uint16(report[4:6]) > 0x2800 {
			return os.ErrDeadlineExceeded
		}
		return nil
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	server, _ := newDragTestServer(t, output)
	session := connectInMemoryMCP(t, ctx, server)
	assertMCPDragRejected(t, ctx, session, dragArguments())
	start := []byte{1, 0, 0, 0, 0, 0x28, 0}
	if held := assertDragReports(t, output.writes, start, nil); len(held) != 0 {
		t.Errorf("unexpected position changes after failed HID write: %x", held)
	}
}

func TestMCPDragReleasesAfterCancellation(t *testing.T) {
	installDragCalibration(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	callCtx, stopDrag := context.WithCancel(ctx)
	defer stopDrag()

	output := &dragRouteOutput{beforeWrite: func(report []byte) error {
		if bytes.Equal(report, []byte{1, 0, 0, 0}) {
			stopDrag()
		}
		return nil
	}}
	server, _ := newDragTestServer(t, output)
	done := make(chan struct{})
	server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method == "tools/call" {
				defer close(done)
			}
			return next(ctx, method, req)
		}
	})
	session := connectInMemoryMCP(t, ctx, server)
	args := dragArguments()
	args["durationMs"] = 1000
	_, err := session.CallTool(callCtx, &mcp.CallToolParams{Name: dragName, Arguments: args})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled MCP drag = %v", err)
	}
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatal("canceled drag handler did not return")
	}

	start := []byte{1, 0, 0, 0, 0, 0x28, 0}
	held := assertDragReports(t, output.writes, start, nil)
	for _, report := range held {
		if binary.LittleEndian.Uint16(report[4:6]) == 0x57ff {
			t.Errorf("canceled drag reached its requested endpoint: %x", held)
		}
	}
}

func TestMCPDragRequiresReadyPointerMode(t *testing.T) {
	for _, test := range []struct {
		name       string
		mode       hid.PointerMode
		switching  bool
		switchHeld bool
	}{
		{name: "touchscreen", mode: hid.PointerModeTouchScreen},
		{name: "switching", mode: hid.PointerModeAbsolute, switching: true},
		{name: "switching after press", mode: hid.PointerModeAbsolute, switchHeld: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			installDragCalibration(t)
			status := hid.PointerModeStatus{Mode: test.mode, Switching: test.switching}
			output := &dragRouteOutput{beforeWrite: func(report []byte) error {
				if test.switchHeld && bytes.Equal(report, []byte{1, 0, 0, 0}) {
					status.Switching = true
				}
				return nil
			}}
			server, remote := newDragTestServer(t, output)
			remote.pointerModeStatus = func() hid.PointerModeStatus { return status }
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			session := connectInMemoryMCP(t, ctx, server)
			assertMCPDragRejected(t, ctx, session, dragArguments())
			if test.switchHeld {
				start := []byte{1, 0, 0, 0, 0, 0x28, 0}
				if held := assertDragReports(t, output.writes, start, nil); len(held) != 0 {
					t.Errorf("drag continued after mode began switching: %x", held)
				}
			} else if len(output.writes) != 0 {
				t.Errorf("drag touched HID with unavailable pointer mode: %x", output.writes)
			}
		})
	}
}

func assertMCPDragRejected(t *testing.T, ctx context.Context, session *mcp.ClientSession, args any) {
	t.Helper()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: dragName, Arguments: args})
	if err != nil {
		t.Fatalf("MCP drag did not return a tool error for %v: %v", args, err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("MCP drag accepted %v: %+v", args, result)
	}
}

func assertDragReports(t *testing.T, reports [][]byte, start, end []byte) [][]byte {
	t.Helper()
	if len(reports) < 3 || !bytes.Equal(reports[0], start) || !bytes.Equal(reports[1], []byte{1, 0, 0, 0}) {
		t.Fatalf("drag should position before pressing the relative left button: %x", reports)
	}
	var held [][]byte
	lastPosition := start
	released := false
	for _, report := range reports[2:] {
		switch {
		case bytes.Equal(report, []byte{0, 0, 0, 0}):
			if !released && end != nil && !bytes.Equal(lastPosition, end) {
				t.Errorf("left button released at %x, want %x", lastPosition, end)
			}
			released = true
		case len(report) == 7 && report[0] == 1 && report[1] == 0 && report[6] == 0:
			if released && !bytes.Equal(report, lastPosition) {
				t.Errorf("cleanup moved the pointer from %x to %x", lastPosition, report)
			}
			if !released {
				held = append(held, report)
				lastPosition = report
			}
		default:
			t.Errorf("unexpected drag HID report %x", report)
		}
	}
	if !released {
		t.Fatalf("drag never sent a left button-up report: %x", reports)
	}
	return held
}

func dragArguments() map[string]any {
	return map[string]any{"startX": 0, "startY": 0, "endX": 1, "endY": 1}
}

func newDragTestServer(t *testing.T, output *dragRouteOutput) (*mcp.Server, *Remote) {
	t.Helper()
	hid.SetInputRouterSink(inputrouter.New(output, nil))
	t.Cleanup(func() { hid.SetInputRouterSink(nil) })
	remote := NewRemote()
	remote.pointerModeStatus = func() hid.PointerModeStatus {
		return hid.PointerModeStatus{Mode: hid.PointerModeAbsolute}
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	registerTools(server, remote, nil)
	return server, remote
}

func installDragCalibration(t *testing.T) (*proto.PortalConfig, *proto.InputProfileConfig) {
	t.Helper()
	oldPortal, oldProfile := loadPortalConfig, loadInputProfileConfig
	width, height, cached := portalCaptureSize.get()
	portalCaptureSize.set(800, 800)
	t.Cleanup(func() {
		loadPortalConfig, loadInputProfileConfig = oldPortal, oldProfile
		portalCaptureSize.invalidate()
		if cached {
			portalCaptureSize.set(width, height)
		}
	})
	portal := &proto.PortalConfig{
		Mode:             "manual",
		SelectedRegionID: "test-crop",
		ManualRegions: []proto.PortalManualRegion{{
			ID: "test-crop", SourceWidth: 800, SourceHeight: 800,
			ContentRect: proto.PortalContentRect{Left: 100, Top: 100, Width: 600, Height: 600},
		}},
	}
	profile := &proto.InputProfileConfig{
		SelectedProfileID: "test-profile",
		Profiles: []proto.InputProfile{{
			ID: "test-profile",
			Calibrations: map[string]proto.InputProfileCalibration{
				"absolute": {Points: map[string]proto.InputProfileCalibrationPoint{
					"top-left":     {Source: proto.InputProfilePoint{X: 0.25, Y: 0}, Target: proto.InputProfilePoint{X: 0, Y: 0.25}},
					"top-right":    {Source: proto.InputProfilePoint{X: 0.75, Y: 0}, Target: proto.InputProfilePoint{X: 1, Y: 0.25}},
					"bottom-left":  {Source: proto.InputProfilePoint{X: 0.25, Y: 1}, Target: proto.InputProfilePoint{X: 0, Y: 0.75}},
					"bottom-right": {Source: proto.InputProfilePoint{X: 0.75, Y: 1}, Target: proto.InputProfilePoint{X: 1, Y: 0.75}},
				}},
			},
		}},
	}
	loadPortalConfig = func() (*proto.PortalConfig, bool, error) { return portal, true, nil }
	loadInputProfileConfig = func() (*proto.InputProfileConfig, bool, error) { return profile, true, nil }
	return portal, profile
}

type dragRouteOutput struct {
	remoteRouteOutput
	beforeWrite func([]byte) error
}

func (o *dragRouteOutput) WriteRelativeMouseReport(report []byte) error {
	if o.beforeWrite != nil {
		if err := o.beforeWrite(report); err != nil {
			return err
		}
	}
	return o.remoteRouteOutput.WriteRelativeMouseReport(report)
}

func (o *dragRouteOutput) WriteAbsoluteMouseReport(report []byte) error {
	if o.beforeWrite != nil {
		if err := o.beforeWrite(report); err != nil {
			return err
		}
	}
	return o.remoteRouteOutput.WriteAbsoluteMouseReport(report)
}
