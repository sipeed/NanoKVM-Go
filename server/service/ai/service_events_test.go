package ai

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type testSSEvent struct {
	name string
	data string
}

func TestServiceStateEventsSSE(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withAIStateFiles(t)
	writeScreenTimelapseServiceState(t, false)

	service := &Service{}
	router := gin.New()
	router.GET("/api/ai/services/events", service.StreamServiceStateEvents)

	server := httptest.NewServer(router)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/ai/services/events")
	if err != nil {
		t.Fatalf("open service state events: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if contentType := resp.Header.Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("expected text/event-stream content type, got %q", contentType)
	}
	if cacheControl := resp.Header.Get("Cache-Control"); cacheControl != "no-cache" {
		t.Fatalf("expected no-cache header, got %q", cacheControl)
	}

	reader := bufio.NewReader(resp.Body)
	snapshotEvent := readTestSSEvent(t, reader)
	if snapshotEvent.name != serviceStateEventSnapshot {
		t.Fatalf("expected snapshot event, got %#v", snapshotEvent)
	}

	var snapshot serviceStateSnapshotEvent
	if err := json.Unmarshal([]byte(snapshotEvent.data), &snapshot); err != nil {
		t.Fatalf("decode snapshot event: %v", err)
	}
	if snapshot.Type != serviceStateEventSnapshot || snapshot.Services[aiServiceScreenTimelapse].Enabled {
		t.Fatalf("expected disabled screen timelapse snapshot, got %#v", snapshot)
	}

	if _, err := DefaultServiceStateManager().Set(aiServiceScreenTimelapse, true, "test"); err != nil {
		t.Fatalf("set screen timelapse service state: %v", err)
	}

	changedEvent := readTestSSEvent(t, reader)
	if changedEvent.name != serviceStateEventChanged {
		t.Fatalf("expected service_changed event, got %#v", changedEvent)
	}

	var changed ServiceStateEvent
	if err := json.Unmarshal([]byte(changedEvent.data), &changed); err != nil {
		t.Fatalf("decode changed event: %v", err)
	}
	if changed.Service != aiServiceScreenTimelapse || !changed.Enabled || changed.Source != "test" {
		t.Fatalf("expected enabled screen timelapse change event, got %#v", changed)
	}

	manager := DefaultServiceStateManager()
	waitForServiceStateSubscribers(t, manager, 1)

	if err := resp.Body.Close(); err != nil {
		t.Fatalf("close events response body: %v", err)
	}
	waitForServiceStateSubscribers(t, manager, 0)
}

func readTestSSEvent(t *testing.T, reader *bufio.Reader) testSSEvent {
	t.Helper()

	resultCh := make(chan struct {
		event testSSEvent
		err   error
	}, 1)
	go func() {
		event, err := readTestSSEEventBlocking(reader)
		resultCh <- struct {
			event testSSEvent
			err   error
		}{event: event, err: err}
	}()

	select {
	case result := <-resultCh:
		if result.err != nil {
			t.Fatalf("read SSE event: %v", result.err)
		}
		return result.event
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for SSE event")
		return testSSEvent{}
	}
}

func readTestSSEEventBlocking(reader *bufio.Reader) (testSSEvent, error) {
	var event testSSEvent
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return testSSEvent{}, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if event.name != "" || event.data != "" {
				return event, nil
			}
			continue
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		switch key {
		case "event":
			event.name = value
		case "data":
			if event.data != "" {
				event.data += "\n"
			}
			event.data += value
		}
	}
}
