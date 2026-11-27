package ai

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"NanoKVM-Server/service/hid"
	"NanoKVM-Server/service/inputrouter"
)

type remoteRouteOutput struct {
	writes [][]byte
}

func (output *remoteRouteOutput) WriteKeyboardReport(report []byte) error {
	output.writes = append(output.writes, bytes.Clone(report))
	return nil
}

func (output *remoteRouteOutput) WriteRelativeMouseReport(report []byte) error {
	output.writes = append(output.writes, bytes.Clone(report))
	return nil
}

func (output *remoteRouteOutput) WriteAbsoluteMouseReport(report []byte) error {
	output.writes = append(output.writes, bytes.Clone(report))
	return nil
}

func (output *remoteRouteOutput) WriteTouchScreenReport(report []byte) error {
	output.writes = append(output.writes, bytes.Clone(report))
	return nil
}

func TestNewRemoteUsesInputRouterForRelativePointer(t *testing.T) {
	output := &remoteRouteOutput{}
	router := inputrouter.New(output, inputrouter.NewTraceHub(8))
	hid.SetInputRouterSink(router)
	t.Cleanup(func() { hid.SetInputRouterSink(nil) })

	if err := NewRemote().MoveRelative(12, -7); err != nil {
		t.Fatalf("MoveRelative: %v", err)
	}
	if len(output.writes) < 2 {
		t.Fatalf("routed writes = %d, want motion and release", len(output.writes))
	}
	if !bytes.Equal(output.writes[0], []byte{0, 12, 0xf9, 0}) {
		t.Fatalf("routed motion report = %v", output.writes[0])
	}
	if !bytes.Equal(output.writes[len(output.writes)-1], []byte{0, 0, 0, 0}) {
		t.Fatalf("routed release report = %v", output.writes[len(output.writes)-1])
	}
	records := router.Traces().Snapshot(2)
	if len(records) != 2 || records[0].Source != "virtual-pointer" || records[1].Stage != "release" {
		t.Fatalf("remote route traces = %+v", records)
	}
}

func TestMCPTypeTextUsesSavedLayoutWithoutPartialCommands(t *testing.T) {
	output := &remoteRouteOutput{}
	remote := newRemoteWithHID(output)
	remote.keyboardLayoutPath = filepath.Join(t.TempDir(), "keyboard-layout")
	typeText := typeTextHandler(remote, newToolRateLimiter(0))
	if err := os.WriteFile(remote.keyboardLayoutPath, []byte("fr-CA"), 0600); err != nil {
		t.Fatal(err)
	}
	_, result, err := typeText(context.Background(), nil, TypeTextParams{Text: "#|é", DelayMS: 1})
	if err != nil || result.Typed != 3 || result.SkippedCount != 0 {
		t.Fatalf("type_text result: %+v, %v", result, err)
	}
	want := [][]byte{{0, 0, 53, 0, 0, 0, 0, 0}, keyUpReport, {2, 0, 53, 0, 0, 0, 0, 0}, keyUpReport, {0, 0, 56, 0, 0, 0, 0, 0}, keyUpReport}
	if !reflect.DeepEqual(output.writes, want) {
		t.Fatalf("MCP reports: %v", output.writes)
	}
	output.writes = nil
	_, _, err = typeText(context.Background(), nil, TypeTextParams{Text: "#🙂", DelayMS: 1})
	if err == nil || !strings.Contains(err.Error(), "U+1F642") || len(output.writes) != 0 {
		t.Fatalf("unsupported command: writes=%v, error=%v", output.writes, err)
	}
}
