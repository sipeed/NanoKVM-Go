package daemon

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestWithGateLimitsConcurrency(t *testing.T) {
	gate := make(chan struct{}, 1)
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	var running int32
	var maxRunning int32
	task := withGate(gate, func(ctx context.Context, now time.Time) error {
		current := atomic.AddInt32(&running, 1)
		for {
			max := atomic.LoadInt32(&maxRunning)
			if current <= max || atomic.CompareAndSwapInt32(&maxRunning, max, current) {
				break
			}
		}
		started <- struct{}{}
		<-release
		atomic.AddInt32(&running, -1)
		return nil
	})
	done := make(chan error, 2)
	go func() { done <- task(context.Background(), time.Now()) }()
	<-started
	go func() { done <- task(context.Background(), time.Now()) }()
	select {
	case <-started:
		t.Fatal("second task started while gate was full")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	for i := 0; i < 2; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if maxRunning != 1 {
		t.Fatalf("maxRunning=%d", maxRunning)
	}
}

func TestSafeLoopRecoversAndContinuesAfterErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var calls int32
	done := make(chan struct{})
	go func() {
		safeLoop(ctx, "test", 5*time.Millisecond, true, func(ctx context.Context, now time.Time) error {
			call := atomic.AddInt32(&calls, 1)
			if call == 1 {
				panic("boom")
			}
			if call == 2 {
				return errors.New("temporary")
			}
			if call >= 3 {
				cancel()
			}
			return nil
		})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("safeLoop did not exit")
	}
	if calls < 3 {
		t.Fatalf("calls=%d", calls)
	}
}

func TestSecondsAndMinutesClampIntervals(t *testing.T) {
	if seconds(1, 60) != 5*time.Second {
		t.Fatalf("seconds clamp failed")
	}
	if minutes(1, 30) != 5*time.Minute {
		t.Fatalf("minutes clamp failed")
	}
}
