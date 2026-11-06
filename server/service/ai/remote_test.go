package ai

import (
	"bytes"
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
