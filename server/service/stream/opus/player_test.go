package opus

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"NanoKVM-Server/common"
)

type fakeOpusSink struct {
	openFn  func() error
	writeFn func([]byte, time.Duration, bool) error
	closeFn func()
}

func (s *fakeOpusSink) Open() error {
	if s.openFn == nil {
		return nil
	}
	return s.openFn()
}

func (s *fakeOpusSink) WriteOpus(frame []byte, timeout time.Duration, discontinuity bool) error {
	if s.writeFn == nil {
		return nil
	}
	return s.writeFn(frame, timeout, discontinuity)
}

func (s *fakeOpusSink) Close() {
	if s.closeFn != nil {
		s.closeFn()
	}
}

type recordedWrite struct {
	frame         []byte
	timeout       time.Duration
	discontinuity bool
}

func testAudioPlayerConfig(capacity int) audioPlayerConfig {
	config := defaultAudioPlayerConfig()
	config.queueCapacity = capacity
	config.errorLogPeriod = time.Hour
	return config
}

func receiveWrite(t *testing.T, writes <-chan recordedWrite) recordedWrite {
	t.Helper()
	select {
	case write := <-writes:
		return write
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Opus write")
		return recordedWrite{}
	}
}

func receiveEvent(t *testing.T, events <-chan string) string {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for player event")
		return ""
	}
}

func closeOnce(channel chan struct{}) func() {
	var once sync.Once
	return func() {
		once.Do(func() { close(channel) })
	}
}

func startTestSession(t *testing.T, player *AudioInputPlayer) *AudioInputSession {
	t.Helper()
	session, err := player.StartSession()
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}
	return session
}

func TestOpusFrameQueueDropsOldestAndMarksNextPop(t *testing.T) {
	queue := newOpusFrameQueue(2)
	if queue.push([]byte{1}) {
		t.Fatal("first frame should not be dropped")
	}
	if queue.push([]byte{2}) {
		t.Fatal("second frame should not be dropped")
	}
	if !queue.push([]byte{3}) {
		t.Fatal("third frame should drop the oldest frame")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first, ok := queue.pop(ctx)
	if !ok {
		t.Fatal("expected first queued frame")
	}
	second, ok := queue.pop(ctx)
	if !ok {
		t.Fatal("expected second queued frame")
	}

	if !reflect.DeepEqual(first.data, []byte{2}) || !first.discontinuity {
		t.Fatalf("unexpected first pop: data=%v discontinuity=%t", first.data, first.discontinuity)
	}
	if !reflect.DeepEqual(second.data, []byte{3}) || second.discontinuity {
		t.Fatalf("unexpected second pop: data=%v discontinuity=%t", second.data, second.discontinuity)
	}
}

func TestAudioInputPlayerCopiesFramesAndOwnsSinkLifecycle(t *testing.T) {
	openStarted := make(chan struct{})
	allowOpen := make(chan struct{})
	releaseOpen := closeOnce(allowOpen)
	writes := make(chan recordedWrite, 1)
	var closeCalls atomic.Int32

	sink := &fakeOpusSink{
		openFn: func() error {
			close(openStarted)
			<-allowOpen
			return nil
		},
		writeFn: func(frame []byte, timeout time.Duration, discontinuity bool) error {
			writes <- recordedWrite{
				frame:         append([]byte(nil), frame...),
				timeout:       timeout,
				discontinuity: discontinuity,
			}
			return nil
		},
		closeFn: func() {
			closeCalls.Add(1)
		},
	}
	player := newAudioInputPlayer(func() OpusSink { return sink }, testAudioPlayerConfig(2))
	var session *AudioInputSession
	t.Cleanup(func() {
		releaseOpen()
		session.Stop()
	})

	session = startTestSession(t, player)
	select {
	case <-openStarted:
	case <-time.After(time.Second):
		t.Fatal("sink Open was not called")
	}

	frame := []byte{1, 2, 3}
	if err := session.Enqueue(frame); err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}
	frame[0] = 9
	releaseOpen()

	write := receiveWrite(t, writes)
	if !reflect.DeepEqual(write.frame, []byte{1, 2, 3}) {
		t.Fatalf("queued frame was not copied: %v", write.frame)
	}
	if write.timeout != defaultWriteTimeout || write.discontinuity {
		t.Fatalf("unexpected write metadata: timeout=%s discontinuity=%t", write.timeout, write.discontinuity)
	}

	session.Stop()
	if player.IsActive() {
		t.Fatal("player remained active after Stop")
	}
	if closeCalls.Load() != 1 {
		t.Fatalf("unexpected Close count: %d", closeCalls.Load())
	}
}

func TestAudioInputPlayerPauseClosesSinkAndResumesSessionLazily(t *testing.T) {
	events := make(chan string, 16)
	var factoryCalls atomic.Int32

	player := newAudioInputPlayer(func() OpusSink {
		id := factoryCalls.Add(1)
		events <- fmt.Sprintf("factory:%d", id)
		var opened atomic.Bool
		return &fakeOpusSink{
			openFn: func() error {
				opened.Store(true)
				events <- fmt.Sprintf("open:%d", id)
				return nil
			},
			writeFn: func(frame []byte, timeout time.Duration, discontinuity bool) error {
				events <- fmt.Sprintf("write:%d:%d", id, frame[0])
				return nil
			},
			closeFn: func() {
				if opened.Swap(false) {
					events <- fmt.Sprintf("close:%d", id)
				}
			},
		}
	}, testAudioPlayerConfig(2))

	session := startTestSession(t, player)
	t.Cleanup(func() {
		player.ResumePlayback()
		session.Stop()
	})
	if event := receiveEvent(t, events); event != "factory:1" {
		t.Fatalf("unexpected factory event: %s", event)
	}
	if event := receiveEvent(t, events); event != "open:1" {
		t.Fatalf("unexpected open event: %s", event)
	}
	if err := session.Enqueue([]byte{1}); err != nil {
		t.Fatalf("initial Enqueue failed: %v", err)
	}
	if event := receiveEvent(t, events); event != "write:1:1" {
		t.Fatalf("unexpected initial write: %s", event)
	}

	player.PausePlayback()
	if event := receiveEvent(t, events); event != "close:1" {
		t.Fatalf("PausePlayback returned before closing the sink: %s", event)
	}
	if !player.IsActive() || !session.IsActive() {
		t.Fatal("PausePlayback invalidated the TrackRemote session")
	}
	if err := session.Enqueue([]byte{2}); err != nil {
		t.Fatalf("paused Enqueue failed: %v", err)
	}
	select {
	case event := <-events:
		t.Fatalf("paused playback produced an event: %s", event)
	default:
	}

	player.ResumePlayback()
	if event := receiveEvent(t, events); event != "factory:2" {
		t.Fatalf("unexpected resumed factory event: %s", event)
	}
	select {
	case event := <-events:
		t.Fatalf("ResumePlayback opened an idle sink: %s", event)
	default:
	}
	if err := session.Enqueue([]byte{3}); err != nil {
		t.Fatalf("resumed Enqueue failed: %v", err)
	}
	if event := receiveEvent(t, events); event != "open:2" {
		t.Fatalf("unexpected resumed open event: %s", event)
	}
	if event := receiveEvent(t, events); event != "write:2:3" {
		t.Fatalf("paused frame leaked into resumed playback: %s", event)
	}

	session.Stop()
	if event := receiveEvent(t, events); event != "close:2" {
		t.Fatalf("unexpected final close event: %s", event)
	}
}

func TestAudioInputPlayerSessionStartedWhilePausedWaitsForResumeFrame(t *testing.T) {
	events := make(chan string, 4)
	player := newAudioInputPlayer(func() OpusSink {
		events <- "factory"
		return &fakeOpusSink{
			openFn: func() error {
				events <- "open"
				return nil
			},
			writeFn: func([]byte, time.Duration, bool) error {
				events <- "write"
				return nil
			},
		}
	}, testAudioPlayerConfig(2))

	player.PausePlayback()
	session := startTestSession(t, player)
	t.Cleanup(func() {
		player.ResumePlayback()
		session.Stop()
	})
	if !session.IsActive() {
		t.Fatal("session started during a pause is not active")
	}
	if err := session.Enqueue([]byte{1}); err != nil {
		t.Fatalf("paused Enqueue failed: %v", err)
	}
	select {
	case event := <-events:
		t.Fatalf("paused StartSession created a sink: %s", event)
	default:
	}

	player.ResumePlayback()
	if event := receiveEvent(t, events); event != "factory" {
		t.Fatalf("unexpected resumed factory event: %s", event)
	}
	select {
	case event := <-events:
		t.Fatalf("resumed idle session opened a sink: %s", event)
	default:
	}
	if err := session.Enqueue([]byte{2}); err != nil {
		t.Fatalf("resumed Enqueue failed: %v", err)
	}
	if event := receiveEvent(t, events); event != "open" {
		t.Fatalf("unexpected resumed open event: %s", event)
	}
	if event := receiveEvent(t, events); event != "write" {
		t.Fatalf("unexpected resumed write event: %s", event)
	}
}

func TestAudioInputPlayerSessionStopWhilePausedClearsOwnership(t *testing.T) {
	opened := make(chan struct{}, 1)
	player := newAudioInputPlayer(func() OpusSink {
		return &fakeOpusSink{
			openFn: func() error {
				opened <- struct{}{}
				return nil
			},
		}
	}, testAudioPlayerConfig(2))

	player.PausePlayback()
	first := startTestSession(t, player)
	if !first.IsActive() {
		t.Fatal("paused session did not acquire ownership")
	}
	first.Stop()
	if player.IsActive() || first.IsActive() {
		t.Fatal("stopping a paused session left its ownership active")
	}
	if err := first.Enqueue([]byte{1}); !errors.Is(err, ErrAudioSessionInactive) {
		t.Fatalf("stopped paused session Enqueue error = %v", err)
	}

	player.ResumePlayback()
	second := startTestSession(t, player)
	t.Cleanup(second.Stop)
	select {
	case <-opened:
	case <-time.After(time.Second):
		t.Fatal("new session did not open playback after resume")
	}
	if !second.IsActive() {
		t.Fatal("new session did not acquire ownership after resume")
	}
	second.Stop()
}

func TestAudioInputPlayerSessionTakeoverIsTokenScopedAndRestartable(t *testing.T) {
	opened := make(chan int32, 3)
	var factoryCalls atomic.Int32
	var closeCalls atomic.Int32

	player := newAudioInputPlayer(func() OpusSink {
		id := factoryCalls.Add(1)
		return &fakeOpusSink{
			openFn: func() error {
				opened <- id
				return nil
			},
			closeFn: func() { closeCalls.Add(1) },
		}
	}, testAudioPlayerConfig(2))

	first := startTestSession(t, player)
	t.Cleanup(first.Stop)
	select {
	case id := <-opened:
		if id != 1 {
			t.Fatalf("unexpected first sink id: %d", id)
		}
	case <-time.After(time.Second):
		t.Fatal("first session did not open")
	}

	second := startTestSession(t, player)
	t.Cleanup(second.Stop)
	select {
	case id := <-opened:
		if id != 2 {
			t.Fatalf("unexpected replacement sink id: %d", id)
		}
	case <-time.After(time.Second):
		t.Fatal("replacement session did not open")
	}
	if first.token == 0 || second.token == 0 || first.token == second.token {
		t.Fatalf("invalid session tokens: first=%d second=%d", first.token, second.token)
	}
	if first.IsActive() || !second.IsActive() {
		t.Fatalf("unexpected ownership state: first=%t second=%t", first.IsActive(), second.IsActive())
	}
	if err := first.Enqueue([]byte{1}); !errors.Is(err, ErrAudioSessionInactive) {
		t.Fatalf("stale session Enqueue error=%v", err)
	}
	first.Stop()
	first.Stop()
	if !second.IsActive() || closeCalls.Load() != 1 {
		t.Fatalf("stale Stop affected replacement: active=%t closes=%d", second.IsActive(), closeCalls.Load())
	}
	second.Stop()
	second.Stop()
	if player.IsActive() {
		t.Fatal("player remained active after current session Stop")
	}

	third := startTestSession(t, player)
	t.Cleanup(third.Stop)
	select {
	case id := <-opened:
		if id != 3 {
			t.Fatalf("unexpected restarted sink id: %d", id)
		}
	case <-time.After(time.Second):
		t.Fatal("restarted session did not open")
	}
	third.Stop()
	if factoryCalls.Load() != 3 || closeCalls.Load() != 3 {
		t.Fatalf("unexpected lifecycle counts: factories=%d closes=%d", factoryCalls.Load(), closeCalls.Load())
	}
}

func TestAudioInputPlayerTakeoverClosesOldSinkAndDropsOldQueue(t *testing.T) {
	events := make(chan string, 16)
	firstWriteStarted := make(chan struct{})
	releaseFirstWrite := make(chan struct{})
	releaseWrite := closeOnce(releaseFirstWrite)
	var factoryCalls atomic.Int32
	var openSinks atomic.Int32
	var maxOpenSinks atomic.Int32

	player := newAudioInputPlayer(func() OpusSink {
		id := factoryCalls.Add(1)
		var writes atomic.Int32
		return &fakeOpusSink{
			openFn: func() error {
				opened := openSinks.Add(1)
				for {
					maximum := maxOpenSinks.Load()
					if opened <= maximum || maxOpenSinks.CompareAndSwap(maximum, opened) {
						break
					}
				}
				events <- fmt.Sprintf("open:%d", id)
				return nil
			},
			writeFn: func(frame []byte, timeout time.Duration, discontinuity bool) error {
				events <- fmt.Sprintf("write:%d:%d", id, frame[0])
				if id == 1 && writes.Add(1) == 1 {
					close(firstWriteStarted)
					<-releaseFirstWrite
				}
				return nil
			},
			closeFn: func() {
				events <- fmt.Sprintf("close:%d", id)
				openSinks.Add(-1)
			},
		}
	}, testAudioPlayerConfig(4))

	first := startTestSession(t, player)
	t.Cleanup(func() {
		releaseWrite()
		first.Stop()
	})
	if event := receiveEvent(t, events); event != "open:1" {
		t.Fatalf("unexpected first event: %s", event)
	}
	if err := first.Enqueue([]byte{1}); err != nil {
		t.Fatalf("first Enqueue failed: %v", err)
	}
	select {
	case <-firstWriteStarted:
	case <-time.After(time.Second):
		t.Fatal("first write did not start")
	}
	if err := first.Enqueue([]byte{2}); err != nil {
		t.Fatalf("queued Enqueue 2 failed: %v", err)
	}
	if err := first.Enqueue([]byte{3}); err != nil {
		t.Fatalf("queued Enqueue 3 failed: %v", err)
	}

	type sessionResult struct {
		session *AudioInputSession
		err     error
	}
	replacementResult := make(chan sessionResult, 1)
	go func() {
		session, err := player.StartSession()
		replacementResult <- sessionResult{session: session, err: err}
	}()

	deadline := time.Now().Add(time.Second)
	for {
		err := first.Enqueue([]byte{9})
		if errors.Is(err, ErrAudioSessionInactive) {
			break
		}
		if err != nil {
			t.Fatalf("unexpected old-session Enqueue error: %v", err)
		}
		if time.Now().After(deadline) {
			t.Fatal("replacement did not invalidate the old token")
		}
		runtime.Gosched()
	}
	select {
	case result := <-replacementResult:
		result.session.Stop()
		t.Fatal("replacement returned before the old bounded write completed")
	default:
	}

	releaseWrite()
	if event := receiveEvent(t, events); event != "write:1:1" {
		t.Fatalf("unexpected in-flight write event: %s", event)
	}
	if event := receiveEvent(t, events); event != "close:1" {
		t.Fatalf("old sink was not closed before replacement: %s", event)
	}
	result := <-replacementResult
	if result.err != nil {
		t.Fatalf("replacement StartSession failed: %v", result.err)
	}
	second := result.session
	t.Cleanup(second.Stop)
	if event := receiveEvent(t, events); event != "open:2" {
		t.Fatalf("replacement sink opened out of order: %s", event)
	}

	first.Stop()
	first.Stop()
	if !second.IsActive() {
		t.Fatal("old Stop terminated the replacement session")
	}
	if err := second.Enqueue([]byte{4}); err != nil {
		t.Fatalf("replacement Enqueue failed: %v", err)
	}
	if event := receiveEvent(t, events); event != "write:2:4" {
		t.Fatalf("old queued frame leaked into replacement: %s", event)
	}
	second.Stop()
	if event := receiveEvent(t, events); event != "close:2" {
		t.Fatalf("unexpected replacement close event: %s", event)
	}
	if maxOpenSinks.Load() != 1 || openSinks.Load() != 0 {
		t.Fatalf("overlapping sinks detected: maximum=%d remaining=%d", maxOpenSinks.Load(), openSinks.Load())
	}
}

func TestAudioInputPlayerConcurrentOldStopAndReplacementStart(t *testing.T) {
	writes := make(chan byte, 1)
	openedEvents := make(chan struct{}, 2)
	var openSinks atomic.Int32
	var maxOpenSinks atomic.Int32

	player := newAudioInputPlayer(func() OpusSink {
		var opened atomic.Bool
		return &fakeOpusSink{
			openFn: func() error {
				opened.Store(true)
				opened := openSinks.Add(1)
				for {
					maximum := maxOpenSinks.Load()
					if opened <= maximum || maxOpenSinks.CompareAndSwap(maximum, opened) {
						break
					}
				}
				openedEvents <- struct{}{}
				return nil
			},
			writeFn: func(frame []byte, timeout time.Duration, discontinuity bool) error {
				writes <- frame[0]
				return nil
			},
			closeFn: func() {
				if opened.Swap(false) {
					openSinks.Add(-1)
				}
			},
		}
	}, testAudioPlayerConfig(2))

	first := startTestSession(t, player)
	t.Cleanup(first.Stop)
	select {
	case <-openedEvents:
	case <-time.After(time.Second):
		t.Fatal("first session did not open")
	}
	start := make(chan struct{})
	oldStopped := make(chan struct{})
	type sessionResult struct {
		session *AudioInputSession
		err     error
	}
	replacementResult := make(chan sessionResult, 1)
	go func() {
		<-start
		first.Stop()
		close(oldStopped)
	}()
	go func() {
		<-start
		session, err := player.StartSession()
		replacementResult <- sessionResult{session: session, err: err}
	}()
	close(start)

	result := <-replacementResult
	if result.err != nil {
		t.Fatalf("replacement StartSession failed: %v", result.err)
	}
	second := result.session
	t.Cleanup(second.Stop)
	select {
	case <-openedEvents:
	case <-time.After(time.Second):
		t.Fatal("replacement session did not open")
	}
	select {
	case <-oldStopped:
	case <-time.After(time.Second):
		t.Fatal("old session Stop did not finish")
	}
	first.Stop()
	if !second.IsActive() {
		t.Fatal("concurrent old Stop terminated the replacement")
	}
	if err := second.Enqueue([]byte{7}); err != nil {
		t.Fatalf("replacement Enqueue failed: %v", err)
	}
	select {
	case value := <-writes:
		if value != 7 {
			t.Fatalf("unexpected replacement frame: %d", value)
		}
	case <-time.After(time.Second):
		t.Fatal("replacement frame was not written")
	}
	second.Stop()
	if maxOpenSinks.Load() != 1 || openSinks.Load() != 0 {
		t.Fatalf("overlapping sinks detected: maximum=%d remaining=%d", maxOpenSinks.Load(), openSinks.Load())
	}
}

func TestAudioInputPlayerQueueIsBoundedAndEnqueueDoesNotWaitForWrite(t *testing.T) {
	firstWriteStarted := make(chan struct{})
	releaseFirstWrite := make(chan struct{})
	releaseWrite := closeOnce(releaseFirstWrite)
	writes := make(chan recordedWrite, 4)
	var writeCalls atomic.Int32

	sink := &fakeOpusSink{
		writeFn: func(frame []byte, timeout time.Duration, discontinuity bool) error {
			call := writeCalls.Add(1)
			writes <- recordedWrite{
				frame:         append([]byte(nil), frame...),
				timeout:       timeout,
				discontinuity: discontinuity,
			}
			if call == 1 {
				close(firstWriteStarted)
				<-releaseFirstWrite
			}
			return nil
		},
	}
	player := newAudioInputPlayer(func() OpusSink { return sink }, testAudioPlayerConfig(2))
	var session *AudioInputSession
	t.Cleanup(func() {
		releaseWrite()
		session.Stop()
	})
	session = startTestSession(t, player)

	if err := session.Enqueue([]byte{0}); err != nil {
		t.Fatalf("initial Enqueue failed: %v", err)
	}
	select {
	case <-firstWriteStarted:
	case <-time.After(time.Second):
		t.Fatal("first write did not start")
	}

	enqueued := make(chan error, 1)
	go func() {
		for _, value := range []byte{1, 2, 3, 4} {
			if err := session.Enqueue([]byte{value}); err != nil {
				enqueued <- err
				return
			}
		}
		enqueued <- nil
	}()
	select {
	case err := <-enqueued:
		if err != nil {
			t.Fatalf("Enqueue failed while writer was blocked: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Enqueue blocked on the synchronous sink writer")
	}

	releaseWrite()
	got := []recordedWrite{receiveWrite(t, writes), receiveWrite(t, writes), receiveWrite(t, writes)}
	values := []byte{got[0].frame[0], got[1].frame[0], got[2].frame[0]}
	if !reflect.DeepEqual(values, []byte{0, 3, 4}) {
		t.Fatalf("queue did not retain only newest frames: %v", values)
	}
	if got[0].discontinuity || !got[1].discontinuity || got[2].discontinuity {
		t.Fatalf("unexpected discontinuity sequence: %t %t %t",
			got[0].discontinuity, got[1].discontinuity, got[2].discontinuity)
	}
}

func TestAudioInputPlayerCodecErrorKeepsSinkOpen(t *testing.T) {
	writes := make(chan recordedWrite, 2)
	var openCalls atomic.Int32
	var closeCalls atomic.Int32
	var writeCalls atomic.Int32

	sink := &fakeOpusSink{
		openFn: func() error {
			openCalls.Add(1)
			return nil
		},
		writeFn: func(frame []byte, timeout time.Duration, discontinuity bool) error {
			writes <- recordedWrite{frame: append([]byte(nil), frame...), discontinuity: discontinuity}
			if writeCalls.Add(1) == 1 {
				return &common.AudioPlaybackError{Code: common.AudioPlaybackErrCodec}
			}
			return nil
		},
		closeFn: func() { closeCalls.Add(1) },
	}
	player := newAudioInputPlayer(func() OpusSink { return sink }, testAudioPlayerConfig(2))
	session := startTestSession(t, player)
	t.Cleanup(session.Stop)
	if err := session.Enqueue([]byte{1}); err != nil {
		t.Fatalf("first Enqueue failed: %v", err)
	}
	first := receiveWrite(t, writes)
	if err := session.Enqueue([]byte{2}); err != nil {
		t.Fatalf("second Enqueue failed: %v", err)
	}
	second := receiveWrite(t, writes)
	session.Stop()

	if first.discontinuity || !second.discontinuity {
		t.Fatalf("unexpected codec recovery flags: first=%t second=%t", first.discontinuity, second.discontinuity)
	}
	if openCalls.Load() != 1 || closeCalls.Load() != 1 {
		t.Fatalf("codec error reopened sink: opens=%d closes=%d", openCalls.Load(), closeCalls.Load())
	}
}

func TestAudioInputPlayerDeviceErrorClearsQueueAndReopens(t *testing.T) {
	firstWriteStarted := make(chan struct{})
	releaseFirstWrite := make(chan struct{})
	releaseWrite := closeOnce(releaseFirstWrite)
	retryStarted := make(chan time.Duration, 1)
	allowRetry := make(chan struct{})
	releaseRetry := closeOnce(allowRetry)
	writes := make(chan recordedWrite, 3)
	var openCalls atomic.Int32
	var closeCalls atomic.Int32
	var writeCalls atomic.Int32

	sink := &fakeOpusSink{
		openFn: func() error {
			openCalls.Add(1)
			return nil
		},
		writeFn: func(frame []byte, timeout time.Duration, discontinuity bool) error {
			writes <- recordedWrite{frame: append([]byte(nil), frame...), discontinuity: discontinuity}
			if writeCalls.Add(1) == 1 {
				close(firstWriteStarted)
				<-releaseFirstWrite
				return &common.AudioPlaybackError{Code: common.AudioPlaybackErrDeviceIO}
			}
			return nil
		},
		closeFn: func() { closeCalls.Add(1) },
	}
	config := testAudioPlayerConfig(4)
	config.waitBackoff = func(ctx context.Context, delay time.Duration) bool {
		retryStarted <- delay
		select {
		case <-ctx.Done():
			return false
		case <-allowRetry:
			return true
		}
	}
	player := newAudioInputPlayer(func() OpusSink { return sink }, config)
	var session *AudioInputSession
	t.Cleanup(func() {
		releaseWrite()
		releaseRetry()
		session.Stop()
	})
	session = startTestSession(t, player)

	if err := session.Enqueue([]byte{1}); err != nil {
		t.Fatalf("first Enqueue failed: %v", err)
	}
	select {
	case <-firstWriteStarted:
	case <-time.After(time.Second):
		t.Fatal("first write did not start")
	}
	if err := session.Enqueue([]byte{2}); err != nil {
		t.Fatalf("second Enqueue failed: %v", err)
	}
	if err := session.Enqueue([]byte{3}); err != nil {
		t.Fatalf("third Enqueue failed: %v", err)
	}
	releaseWrite()

	select {
	case delay := <-retryStarted:
		if delay != defaultRetryInitial {
			t.Fatalf("unexpected first retry delay: %s", delay)
		}
	case <-time.After(time.Second):
		t.Fatal("writer did not enter reconnect backoff")
	}
	if err := session.Enqueue([]byte{4}); err != nil {
		t.Fatalf("post-error Enqueue failed: %v", err)
	}
	releaseRetry()

	first := receiveWrite(t, writes)
	second := receiveWrite(t, writes)
	if !reflect.DeepEqual(first.frame, []byte{1}) || !reflect.DeepEqual(second.frame, []byte{4}) {
		t.Fatalf("stale frames survived device error: first=%v second=%v", first.frame, second.frame)
	}
	if !second.discontinuity {
		t.Fatal("first frame after reopen was not marked discontinuous")
	}
	if openCalls.Load() != 2 {
		t.Fatalf("unexpected Open count: %d", openCalls.Load())
	}

	session.Stop()
	if closeCalls.Load() != 2 {
		t.Fatalf("unexpected Close count: %d", closeCalls.Load())
	}
}

func TestAudioInputPlayerStopCancelsReconnectBackoff(t *testing.T) {
	retryStarted := make(chan struct{})
	var openCalls atomic.Int32
	var once sync.Once

	sink := &fakeOpusSink{
		openFn: func() error {
			openCalls.Add(1)
			return &common.AudioPlaybackError{Code: common.AudioPlaybackErrUnavailable}
		},
	}
	config := testAudioPlayerConfig(2)
	config.waitBackoff = func(ctx context.Context, delay time.Duration) bool {
		once.Do(func() { close(retryStarted) })
		<-ctx.Done()
		return false
	}
	player := newAudioInputPlayer(func() OpusSink { return sink }, config)
	session := startTestSession(t, player)
	t.Cleanup(session.Stop)
	select {
	case <-retryStarted:
	case <-time.After(time.Second):
		t.Fatal("writer did not enter reconnect backoff")
	}

	stopped := make(chan struct{})
	go func() {
		session.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("Stop did not cancel reconnect backoff")
	}
	if openCalls.Load() != 1 {
		t.Fatalf("Open retried after Stop: %d calls", openCalls.Load())
	}
}

func TestAudioInputPlayerWriteFailuresIncreaseReconnectBackoff(t *testing.T) {
	retryDelays := make(chan time.Duration, 3)
	allowRetry := make(chan struct{}, 2)
	var openCalls atomic.Int32

	sink := &fakeOpusSink{
		openFn: func() error {
			openCalls.Add(1)
			return nil
		},
		writeFn: func(frame []byte, timeout time.Duration, discontinuity bool) error {
			return &common.AudioPlaybackError{Code: common.AudioPlaybackErrDeviceIO}
		},
	}
	config := testAudioPlayerConfig(2)
	config.waitBackoff = func(ctx context.Context, delay time.Duration) bool {
		retryDelays <- delay
		select {
		case <-ctx.Done():
			return false
		case <-allowRetry:
			return true
		}
	}
	player := newAudioInputPlayer(func() OpusSink { return sink }, config)
	session := startTestSession(t, player)
	t.Cleanup(session.Stop)

	wantDelays := []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond}
	for index, want := range wantDelays {
		if err := session.Enqueue([]byte{byte(index + 1)}); err != nil {
			t.Fatalf("Enqueue %d failed: %v", index, err)
		}
		select {
		case got := <-retryDelays:
			if got != want {
				t.Fatalf("retry %d delay=%s, want %s", index, got, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for retry %d", index)
		}
		if index != len(wantDelays)-1 {
			allowRetry <- struct{}{}
		}
	}

	session.Stop()
	if openCalls.Load() != int32(len(wantDelays)) {
		t.Fatalf("unexpected Open count: %d", openCalls.Load())
	}
}

func TestAudioInputPlayerSuccessfulWriteResetsReconnectBackoff(t *testing.T) {
	retryDelays := make(chan time.Duration, 2)
	allowRetry := make(chan struct{}, 1)
	writes := make(chan int32, 3)
	var writeCalls atomic.Int32

	sink := &fakeOpusSink{
		writeFn: func(frame []byte, timeout time.Duration, discontinuity bool) error {
			call := writeCalls.Add(1)
			writes <- call
			if call == 1 || call == 3 {
				return &common.AudioPlaybackError{Code: common.AudioPlaybackErrDeviceIO}
			}
			return nil
		},
	}
	config := testAudioPlayerConfig(2)
	config.waitBackoff = func(ctx context.Context, delay time.Duration) bool {
		retryDelays <- delay
		select {
		case <-ctx.Done():
			return false
		case <-allowRetry:
			return true
		}
	}
	player := newAudioInputPlayer(func() OpusSink { return sink }, config)
	session := startTestSession(t, player)
	t.Cleanup(session.Stop)

	if err := session.Enqueue([]byte{1}); err != nil {
		t.Fatalf("first Enqueue failed: %v", err)
	}
	select {
	case delay := <-retryDelays:
		if delay != defaultRetryInitial {
			t.Fatalf("unexpected first retry delay: %s", delay)
		}
	case <-time.After(time.Second):
		t.Fatal("first failure did not enter backoff")
	}
	if err := session.Enqueue([]byte{2}); err != nil {
		t.Fatalf("second Enqueue failed: %v", err)
	}
	allowRetry <- struct{}{}
	select {
	case call := <-writes:
		if call != 1 {
			t.Fatalf("unexpected first write call: %d", call)
		}
	case <-time.After(time.Second):
		t.Fatal("first write was not observed")
	}
	select {
	case call := <-writes:
		if call != 2 {
			t.Fatalf("unexpected successful write call: %d", call)
		}
	case <-time.After(time.Second):
		t.Fatal("successful write was not observed")
	}

	if err := session.Enqueue([]byte{3}); err != nil {
		t.Fatalf("third Enqueue failed: %v", err)
	}
	select {
	case call := <-writes:
		if call != 3 {
			t.Fatalf("unexpected third write call: %d", call)
		}
	case <-time.After(time.Second):
		t.Fatal("third write was not observed")
	}
	select {
	case delay := <-retryDelays:
		if delay != defaultRetryInitial {
			t.Fatalf("successful write did not reset backoff: %s", delay)
		}
	case <-time.After(time.Second):
		t.Fatal("third failure did not enter backoff")
	}
}

func TestAudioInputPlayerStopDropsQueuedFrames(t *testing.T) {
	firstWriteStarted := make(chan struct{})
	releaseFirstWrite := make(chan struct{})
	releaseWrite := closeOnce(releaseFirstWrite)
	var writeCalls atomic.Int32

	sink := &fakeOpusSink{
		writeFn: func(frame []byte, timeout time.Duration, discontinuity bool) error {
			if writeCalls.Add(1) == 1 {
				close(firstWriteStarted)
				<-releaseFirstWrite
			}
			return nil
		},
	}
	player := newAudioInputPlayer(func() OpusSink { return sink }, testAudioPlayerConfig(4))
	var session *AudioInputSession
	t.Cleanup(func() {
		releaseWrite()
		session.Stop()
	})
	session = startTestSession(t, player)
	if err := session.Enqueue([]byte{1}); err != nil {
		t.Fatalf("first Enqueue failed: %v", err)
	}
	select {
	case <-firstWriteStarted:
	case <-time.After(time.Second):
		t.Fatal("first write did not start")
	}
	for _, value := range []byte{2, 3, 4} {
		if err := session.Enqueue([]byte{value}); err != nil {
			t.Fatalf("queued Enqueue failed: %v", err)
		}
	}

	stopped := make(chan struct{})
	go func() {
		session.Stop()
		close(stopped)
	}()
	deadline := time.Now().Add(time.Second)
	for player.IsActive() {
		if time.Now().After(deadline) {
			t.Fatal("Stop did not deactivate the player")
		}
		time.Sleep(time.Millisecond)
	}
	releaseWrite()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("Stop did not finish after the current bounded write")
	}
	if writeCalls.Load() != 1 {
		t.Fatalf("Stop played stale queued frames: %d writes", writeCalls.Load())
	}
}

func TestAudioInputPlayerPermissionErrorWaitsForStop(t *testing.T) {
	openCalled := make(chan struct{})
	var openCalls atomic.Int32
	sink := &fakeOpusSink{
		openFn: func() error {
			openCalls.Add(1)
			close(openCalled)
			return &common.AudioPlaybackError{Code: common.AudioPlaybackErrPermission}
		},
	}
	player := newAudioInputPlayer(func() OpusSink { return sink }, testAudioPlayerConfig(2))
	session := startTestSession(t, player)
	t.Cleanup(session.Stop)
	select {
	case <-openCalled:
	case <-time.After(time.Second):
		t.Fatal("sink Open was not called")
	}

	if !player.IsActive() {
		t.Fatal("permission failure unexpectedly ended the TrackRemote session")
	}
	session.Stop()
	if openCalls.Load() != 1 {
		t.Fatalf("permission failure was retried: %d calls", openCalls.Load())
	}
}

func TestDurationMilliseconds(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     uint32
		ok       bool
	}{
		{name: "zero", duration: 0, ok: false},
		{name: "negative", duration: -time.Millisecond, ok: false},
		{name: "round up", duration: time.Millisecond + time.Nanosecond, want: 2, ok: true},
		{name: "maximum", duration: time.Duration(common.AudioPlaybackMaxTimeoutMS) * time.Millisecond, want: common.AudioPlaybackMaxTimeoutMS, ok: true},
		{name: "over maximum", duration: time.Duration(common.AudioPlaybackMaxTimeoutMS)*time.Millisecond + time.Nanosecond, ok: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := durationMilliseconds(test.duration)
			if got != test.want || ok != test.ok {
				t.Fatalf("durationMilliseconds(%s) = (%d, %t), want (%d, %t)",
					test.duration, got, ok, test.want, test.ok)
			}
		})
	}
}

func TestNextBackoff(t *testing.T) {
	maximum := 2 * time.Second
	if got := nextBackoff(100*time.Millisecond, maximum); got != 200*time.Millisecond {
		t.Fatalf("unexpected doubled backoff: %s", got)
	}
	if got := nextBackoff(1500*time.Millisecond, maximum); got != maximum {
		t.Fatalf("backoff did not clamp to maximum: %s", got)
	}
	if got := nextBackoff(maximum, maximum); got != maximum {
		t.Fatalf("maximum backoff changed: %s", got)
	}
}
