package ai

import (
	"bytes"
	"context"
	"errors"
	"os"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"NanoKVM-Server/service/hid"
	"NanoKVM-Server/service/inputrouter"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPScrollReports(t *testing.T) {
	output := &scrollRouteOutput{}
	server, _ := newScrollTestServer(t, output)
	session := connectInMemoryMCP(t, t.Context(), server)

	for _, tt := range []struct {
		name  string
		args  map[string]any
		wheel []int8
	}{
		{"one larger delta", map[string]any{"deltaY": 4}, []int8{4, 0}},
		{"legacy split delta", map[string]any{"deltaY": -128, "repeat": 1}, []int8{-127, -1, 0}},
		{"zero delta", map[string]any{"deltaY": 0}, nil},
		{"two down", map[string]any{"deltaY": 1, "repeat": 2}, []int8{1, 0, 1, 0}},
		{"eight up", map[string]any{"deltaY": -1, "repeat": 8}, []int8{-1, 0, -1, 0, -1, 0, -1, 0, -1, 0, -1, 0, -1, 0, -1, 0}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			output.writes = nil
			result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "scroll_mouse", Arguments: tt.args})
			if err != nil || result == nil || result.IsError {
				t.Fatalf("scroll_mouse = %+v, %v", result, err)
			}
			var wheel []int8
			for _, b := range output.writes {
				if len(b) != 4 || !bytes.Equal(b[:3], []byte{0, 0, 0}) {
					t.Fatalf("scroll changed mouse position or buttons: %x", b)
				}
				wheel = append(wheel, int8(b[3]))
			}
			if !slices.Equal(wheel, tt.wheel) {
				t.Errorf("wheel reports = %v, want %v", wheel, tt.wheel)
			}
		})
	}
}

func TestMCPScrollRejectsInvalidRequestsBeforeWriting(t *testing.T) {
	output := &scrollRouteOutput{}
	server, _ := newScrollTestServer(t, output)
	session := connectInMemoryMCP(t, t.Context(), server)
	for _, args := range []map[string]any{
		{"repeat": 4},
		{"deltaY": 1, "repeat": 0},
		{"deltaY": 1, "repeat": 9},
		{"deltaY": 1, "repeat": 1.5},
		{"deltaY": 0, "repeat": 4},
		{"deltaY": -2, "repeat": 4},
		{"deltaY": 127*maxRelativeMouseReports + 1},
	} {
		result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "scroll_mouse", Arguments: args})
		if err != nil || result == nil || !result.IsError {
			t.Fatalf("expected tool error for %+v, got %+v, %v", args, result, err)
		}
	}
	if len(output.writes) != 0 {
		t.Fatalf("invalid requests wrote HID reports: %x", output.writes)
	}
}

func TestMCPScrollStopsOnWriteFailure(t *testing.T) {
	output := &scrollRouteOutput{}
	pulses := 0
	output.beforeWrite = func(b []byte) error {
		if len(b) == 4 && b[3] != 0 {
			pulses++
			if pulses == 2 {
				return os.ErrDeadlineExceeded
			}
		}
		return nil
	}
	server, _ := newScrollTestServer(t, output)
	session := connectInMemoryMCP(t, t.Context(), server)
	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "scroll_mouse", Arguments: map[string]int{"deltaY": 1, "repeat": 4}})
	if err != nil || result == nil || !result.IsError {
		t.Fatalf("expected HID failure, got %+v, %v", result, err)
	}
	if pulses != 2 {
		t.Fatalf("after an uncertain wheel write: %d pulse attempts, want 2", pulses)
	}
	if len(output.writes) != 2 || !bytes.Equal(output.writes[0], []byte{0, 0, 0, 1}) || !bytes.Equal(output.writes[1], []byte{0, 0, 0, 0}) {
		t.Fatalf("unexpected successful reports: %x", output.writes)
	}
}

func TestMCPScrollStopsOnReleaseFailure(t *testing.T) {
	output := &scrollRouteOutput{beforeWrite: func(b []byte) error {
		if bytes.Equal(b, []byte{0, 0, 0, 0}) {
			return errors.New("release failed")
		}
		return nil
	}}
	server, _ := newScrollTestServer(t, output)
	session := connectInMemoryMCP(t, t.Context(), server)
	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "scroll_mouse", Arguments: map[string]int{"deltaY": -1, "repeat": 4}})
	if err != nil || result == nil || !result.IsError {
		t.Fatalf("expected release failure, got %+v, %v", result, err)
	}
	if len(output.writes) != 1 || !bytes.Equal(output.writes[0], []byte{0, 0, 0, 0xff}) {
		t.Fatalf("scroll continued after failure to release first tick: %x", output.writes)
	}
}

func TestMCPScrollStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	output := &scrollRouteOutput{beforeWrite: func(b []byte) error {
		if len(b) == 4 && b[3] != 0 {
			cancel()
		}
		return nil
	}}
	_, remote := newScrollTestServer(t, output)
	count := 4
	_, _, err := scrollMouseHandler(remote, nil)(ctx, nil, ScrollMouseParams{DeltaY: 1, Repeat: &count})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if len(output.writes) != 2 || !bytes.Equal(output.writes[1], []byte{0, 0, 0, 0}) {
		t.Fatalf("expected first wheel tick and release, got %x", output.writes)
	}
}

func TestMCPScrollReservesEachTick(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, remote := newScrollTestServer(t, &scrollRouteOutput{})
		handler := scrollMouseHandler(remote, newToolRateLimiter(time.Second))
		count := 4
		start := time.Now()
		if _, _, err := handler(t.Context(), nil, ScrollMouseParams{DeltaY: 1, Repeat: &count}); err != nil {
			t.Fatal(err)
		}
		if elapsed := time.Since(start); elapsed != 3*scrollTickInterval {
			t.Fatalf("four ticks took %v, want three intervals", elapsed)
		}
		time.Sleep(2 * time.Second)
		if _, _, err := handler(t.Context(), nil, ScrollMouseParams{DeltaY: 1}); err == nil {
			t.Fatal("second scroll admitted before the four reserved rate-limit intervals elapsed")
		}
		time.Sleep(2 * time.Second)
		if _, _, err := handler(t.Context(), nil, ScrollMouseParams{DeltaY: 1}); err != nil {
			t.Fatalf("second scroll after cooldown: %v", err)
		}
	})
}

func newScrollTestServer(t *testing.T, output *scrollRouteOutput) (*mcp.Server, *Remote) {
	t.Helper()
	hid.SetInputRouterSink(inputrouter.New(output, nil))
	t.Cleanup(func() { hid.SetInputRouterSink(nil) })
	remote := NewRemote()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	registerTools(server, remote, nil)
	return server, remote
}

type scrollRouteOutput struct {
	remoteRouteOutput
	beforeWrite func([]byte) error
}

func (o *scrollRouteOutput) WriteRelativeMouseReport(report []byte) error {
	if o.beforeWrite != nil {
		if err := o.beforeWrite(report); err != nil {
			return err
		}
	}
	return o.remoteRouteOutput.WriteRelativeMouseReport(report)
}
