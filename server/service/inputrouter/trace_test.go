package inputrouter

import (
	"bufio"
	"bytes"
	"net"
	"path/filepath"
	"testing"
	"time"
)

func TestTraceEventIncludesStructuredSemanticState(t *testing.T) {
	keyboard := Event{Kind: KindKeyboardState, Keyboard: &KeyboardState{Report: [8]byte{2, 0, 4, 5}}}
	keyboardRecord := traceEvent(keyboard, 1)
	if keyboardRecord.Semantic == nil || keyboardRecord.Semantic.Modifiers != 2 || !bytes.Equal(keyboardRecord.Semantic.Keys, []byte{4, 5}) {
		t.Fatalf("keyboard semantic trace = %+v", keyboardRecord.Semantic)
	}
}

func TestTraceHubStreamsWithoutRetainingRecords(t *testing.T) {
	hub := NewTraceHub(0)
	records, unsubscribe := hub.Subscribe(2)
	hub.Publish(TraceRecord{Stage: "dispatched"})

	select {
	case record := <-records:
		if record.Sequence != 1 || record.Stage != "dispatched" {
			t.Fatalf("streamed record = %+v", record)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for streamed trace")
	}
	if snapshot := hub.Snapshot(10); len(snapshot) != 0 {
		t.Fatalf("zero-capacity snapshot = %+v", snapshot)
	}
	if hub.StreamSubscribers() != 1 {
		t.Fatalf("stream subscribers = %d, want 1", hub.StreamSubscribers())
	}
	unsubscribe()
	if hub.StreamSubscribers() != 0 {
		t.Fatalf("stream subscribers after unsubscribe = %d, want 0", hub.StreamSubscribers())
	}
	hub.Publish(TraceRecord{Stage: "discarded"})
}

func TestTraceHubAcceptsReplacementDebugClient(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "input-debug.sock")
	hub := NewManagedTraceHub(8, socketPath)
	t.Cleanup(hub.Close)

	first := connectDebugClient(t, socketPath)
	waitForDebugConnections(t, hub, 1)
	hub.Publish(TraceRecord{Stage: "first"})
	readDebugRecord(t, first, "first")
	_ = first.Close()

	second := connectDebugClient(t, socketPath)
	waitForDebugConnections(t, hub, 2)
	hub.Publish(TraceRecord{Stage: "second"})
	readDebugRecord(t, second, "second")
	_ = second.Close()

	if hub.DeliveredDebugRecords() != 2 || hub.DroppedDebugRecords() != 0 {
		t.Fatalf(
			"debug counters delivered=%d dropped=%d unavailable=%d",
			hub.DeliveredDebugRecords(),
			hub.DroppedDebugRecords(),
			hub.UnavailableDebugRecords(),
		)
	}
}

func waitForDebugConnections(t *testing.T, hub *TraceHub, want uint64) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for hub.DebugConnections() < want {
		if time.Now().After(deadline) {
			t.Fatalf("debug connections = %d, want %d", hub.DebugConnections(), want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func connectDebugClient(t *testing.T, path string) *net.UnixConn {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		connection, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: path, Net: "unix"})
		if err == nil {
			return connection
		}
		if time.Now().After(deadline) {
			t.Fatalf("connect debug socket: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func readDebugRecord(t *testing.T, connection *net.UnixConn, want string) {
	t.Helper()
	if err := connection.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set debug socket deadline: %v", err)
	}
	data, err := bufio.NewReader(connection).ReadBytes('\n')
	if err != nil {
		t.Fatalf("read debug record: %v", err)
	}
	needle := []byte(`"stage":"` + want + `"`)
	if !bytes.Contains(data, needle) {
		t.Fatalf("debug record = %q, want stage %q", data, want)
	}
}
