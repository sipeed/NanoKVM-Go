package hid

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type pasteEvent struct {
	report []byte
	delay  time.Duration
}

type pasteEventRecorder struct {
	mu     sync.Mutex
	events []pasteEvent
}

func (r *pasteEventRecorder) recordWrite(report []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, pasteEvent{report: bytes.Clone(report)})
}

func (r *pasteEventRecorder) recordSleep(delay time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, pasteEvent{delay: delay})
}

func (r *pasteEventRecorder) snapshot() []pasteEvent {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]pasteEvent(nil), r.events...)
}

type pasteEventFile struct {
	fakeHIDFile
	recorder *pasteEventRecorder
}

func (f *pasteEventFile) Write(data []byte) (int, error) {
	f.recorder.recordWrite(data)
	return f.fakeHIDFile.Write(data)
}

func TestPasteTimingContract(t *testing.T) {
	if pasteKeyDownHold != 10*time.Millisecond {
		t.Fatalf("pasteKeyDownHold = %v, want 10ms", pasteKeyDownHold)
	}
	if pastePostReleaseDelay != 25*time.Millisecond {
		t.Fatalf("pastePostReleaseDelay = %v, want 25ms", pastePostReleaseDelay)
	}
}

func TestPasteWritesMappedReportsWithConfiguredPacing(t *testing.T) {
	events := &pasteEventRecorder{}
	keyboard := &pasteEventFile{recorder: events}
	service := &Service{
		hid: &Hid{g0: hidEndpoint{file: keyboard}},
		pasteSleep: func(delay time.Duration) {
			events.recordSleep(delay)
		},
	}

	form := url.Values{"content": {"aA!"}}
	request := httptest.NewRequest(http.MethodPost, "/api/hid/paste", bytes.NewBufferString(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request

	service.Paste(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Paste HTTP status = %d, want %d", recorder.Code, http.StatusOK)
	}

	wantReports := [][]byte{
		{0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00},
		{0x02, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00},
		{0x02, 0x00, 0x1e, 0x00, 0x00, 0x00, 0x00, 0x00},
	}
	assertPasteEvents(t, events.snapshot(), pasteEventsForReports(wantReports))
}

func TestPasteReleasesShiftBeforeFollowingLowercaseReports(t *testing.T) {
	events := &pasteEventRecorder{}
	keyboard := &pasteEventFile{recorder: events}
	service := &Service{
		hid: &Hid{g0: hidEndpoint{file: keyboard}},
		pasteSleep: func(delay time.Duration) {
			events.recordSleep(delay)
		},
	}

	form := url.Values{"content": {"International"}}
	request := httptest.NewRequest(http.MethodPost, "/api/hid/paste", bytes.NewBufferString(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request

	service.Paste(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Paste HTTP status = %d, want %d", recorder.Code, http.StatusOK)
	}

	wantReports := [][]byte{
		{0x02, 0x00, 0x0c, 0x00, 0x00, 0x00, 0x00, 0x00}, // I
		{0x00, 0x00, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00}, // n
		{0x00, 0x00, 0x17, 0x00, 0x00, 0x00, 0x00, 0x00}, // t
		{0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00}, // e
		{0x00, 0x00, 0x15, 0x00, 0x00, 0x00, 0x00, 0x00}, // r
		{0x00, 0x00, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00}, // n
		{0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00}, // a
		{0x00, 0x00, 0x17, 0x00, 0x00, 0x00, 0x00, 0x00}, // t
		{0x00, 0x00, 0x0c, 0x00, 0x00, 0x00, 0x00, 0x00}, // i
		{0x00, 0x00, 0x12, 0x00, 0x00, 0x00, 0x00, 0x00}, // o
		{0x00, 0x00, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00}, // n
		{0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00}, // a
		{0x00, 0x00, 0x0f, 0x00, 0x00, 0x00, 0x00, 0x00}, // l
	}
	assertPasteEvents(t, events.snapshot(), pasteEventsForReports(wantReports))
}

func TestPasteWritesShiftedNumberRowReportsWithConfiguredPacing(t *testing.T) {
	events := &pasteEventRecorder{}
	keyboard := &pasteEventFile{recorder: events}
	service := &Service{
		hid: &Hid{g0: hidEndpoint{file: keyboard}},
		pasteSleep: func(delay time.Duration) {
			events.recordSleep(delay)
		},
	}

	form := url.Values{"content": {"%^&*()"}}
	request := httptest.NewRequest(http.MethodPost, "/api/hid/paste", bytes.NewBufferString(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request

	service.Paste(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Paste HTTP status = %d, want %d", recorder.Code, http.StatusOK)
	}

	wantReports := [][]byte{
		{0x02, 0x00, 0x22, 0x00, 0x00, 0x00, 0x00, 0x00},
		{0x02, 0x00, 0x23, 0x00, 0x00, 0x00, 0x00, 0x00},
		{0x02, 0x00, 0x24, 0x00, 0x00, 0x00, 0x00, 0x00},
		{0x02, 0x00, 0x25, 0x00, 0x00, 0x00, 0x00, 0x00},
		{0x02, 0x00, 0x26, 0x00, 0x00, 0x00, 0x00, 0x00},
		{0x02, 0x00, 0x27, 0x00, 0x00, 0x00, 0x00, 0x00},
	}
	assertPasteEvents(t, events.snapshot(), pasteEventsForReports(wantReports))
}

func TestPasteSkipsUnmappedRunesWithoutPacing(t *testing.T) {
	events := &pasteEventRecorder{}
	keyboard := &pasteEventFile{recorder: events}
	service := &Service{
		hid: &Hid{g0: hidEndpoint{file: keyboard}},
		pasteSleep: func(delay time.Duration) {
			events.recordSleep(delay)
		},
	}

	form := url.Values{"content": {"a€b"}}
	request := httptest.NewRequest(http.MethodPost, "/api/hid/paste", bytes.NewBufferString(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request

	service.Paste(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Paste HTTP status = %d, want %d", recorder.Code, http.StatusOK)
	}

	assertPasteEvents(t, events.snapshot(), pasteEventsForReports([][]byte{
		{0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00},
		{0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00},
	}))
}

func TestPasteCharacterSequenceBlocksConcurrentKeyboardReports(t *testing.T) {
	events := &pasteEventRecorder{}
	keyboard := &pasteEventFile{recorder: events}
	h := &Hid{g0: hidEndpoint{file: keyboard}}
	keyDownHeld := make(chan struct{})
	postReleaseHeld := make(chan struct{})
	releaseKeyDownHold := make(chan struct{})
	releasePostReleaseHold := make(chan struct{})
	var sleepCalls int
	service := &Service{
		hid: h,
		pasteSleep: func(delay time.Duration) {
			events.recordSleep(delay)
			sleepCalls++
			switch sleepCalls {
			case 1:
				if delay != pasteKeyDownHold {
					t.Errorf("key-down hold = %v, want %v", delay, pasteKeyDownHold)
				}
				close(keyDownHeld)
				<-releaseKeyDownHold
			case 2:
				if delay != pastePostReleaseDelay {
					t.Errorf("post-release delay = %v, want %v", delay, pastePostReleaseDelay)
				}
				close(postReleaseHeld)
				<-releasePostReleaseHold
			default:
				t.Errorf("pasteSleep called %d times, want 2", sleepCalls)
			}
		},
	}

	pasteDone := make(chan struct{})
	go func() {
		defer close(pasteDone)
		form := url.Values{"content": {"A"}}
		request := httptest.NewRequest(http.MethodPost, "/api/hid/paste", bytes.NewBufferString(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		context.Request = request
		service.Paste(context)
		if recorder.Code != http.StatusOK {
			t.Errorf("Paste HTTP status = %d, want %d", recorder.Code, http.StatusOK)
		}
	}()

	<-keyDownHeld

	directStarted := make(chan struct{})
	directDone := make(chan error, 1)
	directReport := []byte{0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00}
	go func() {
		close(directStarted)
		directDone <- h.WriteKeyboardReport(directReport)
	}()
	<-directStarted

	assertKeyboardMutexHeld(t, &h.kbMutex)
	close(releaseKeyDownHold)
	<-postReleaseHeld
	assertKeyboardMutexHeld(t, &h.kbMutex)

	wantSequence := []pasteEvent{
		{report: []byte{0x02, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{delay: pasteKeyDownHold},
		{report: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{delay: pastePostReleaseDelay},
	}
	assertPasteEvents(t, events.snapshot(), wantSequence)
	close(releasePostReleaseHold)

	if err := <-directDone; err != nil {
		t.Fatalf("concurrent WriteKeyboardReport error = %v", err)
	}
	<-pasteDone

	assertPasteEvents(t, events.snapshot(), append(wantSequence, pasteEvent{report: directReport}))
}

func assertKeyboardMutexHeld(t *testing.T, mu *sync.Mutex) {
	t.Helper()
	if mu.TryLock() {
		mu.Unlock()
		t.Fatal("keyboard mutex was not held during paste character sequence")
	}
}

func pasteEventsForReports(reports [][]byte) []pasteEvent {
	keyUp := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	events := make([]pasteEvent, 0, len(reports)*4)
	for _, report := range reports {
		events = append(events,
			pasteEvent{report: report},
			pasteEvent{delay: pasteKeyDownHold},
			pasteEvent{report: keyUp},
			pasteEvent{delay: pastePostReleaseDelay},
		)
	}
	return events
}

func assertPasteEvents(t *testing.T, got, want []pasteEvent) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("paste events = %d, want %d", len(got), len(want))
	}

	for index, wantEvent := range want {
		gotEvent := got[index]
		if wantEvent.report != nil {
			if !bytes.Equal(gotEvent.report, wantEvent.report) {
				t.Fatalf("paste event[%d] report = %v, want %v", index, gotEvent.report, wantEvent.report)
			}
			continue
		}
		if gotEvent.delay != wantEvent.delay {
			t.Fatalf("paste event[%d] delay = %v, want %v", index, gotEvent.delay, wantEvent.delay)
		}
	}
}
