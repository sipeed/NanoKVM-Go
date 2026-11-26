package jiggler

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func testLoad(path string, noteMove func(string), jitter func() time.Duration) *Jiggler {
	return load(path, func(mode string, _ bool) *movement { noteMove(mode); return nil }, jitter)
}

func TestIdleAndRandomDelay(t *testing.T) {
	for _, delay := range []time.Duration{0, 12 * time.Second, 30 * time.Second} {
		t.Run(delay.String(), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var moves, draws atomic.Int64
				j := testLoad(filepath.Join(t.TempDir(), "jiggler"), func(string) { moves.Add(1) }, func() time.Duration { draws.Add(1); return delay })
				on := true
				if _, err := j.Configure(&on, "", nil); err != nil {
					t.Fatal(err)
				}
				for cycle := int64(1); cycle <= 2; cycle++ {
					time.Sleep(idleInterval - time.Millisecond)
					synctest.Wait()
					if draws.Load() != cycle-1 || moves.Load() != cycle-1 {
						t.Fatal("early draw or movement", draws.Load(), moves.Load())
					}
					time.Sleep(time.Millisecond)
					synctest.Wait()
					if draws.Load() != cycle {
						t.Fatal("missing idle check", draws.Load())
					}
					if delay != 0 {
						time.Sleep(delay - time.Millisecond)
						synctest.Wait()
						if moves.Load() != cycle-1 {
							t.Fatal("early movement", moves.Load())
						}
						time.Sleep(time.Millisecond)
						synctest.Wait()
					}
					if moves.Load() != cycle {
						t.Fatal("missing movement", moves.Load())
					}
				}
				on = false
				if _, err := j.Configure(&on, "", nil); err != nil {
					t.Fatal(err)
				}
			})
		})
	}
}

func TestActivityCancelsPendingMove(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var moves atomic.Int64
		j := testLoad(filepath.Join(t.TempDir(), "jiggler"), func(string) { moves.Add(1) }, func() time.Duration { return 12 * time.Second })
		on := true
		if _, err := j.Configure(&on, "", nil); err != nil {
			t.Fatal(err)
		}
		time.Sleep(35 * time.Second)
		j.Update()
		time.Sleep(42*time.Second - time.Millisecond)
		synctest.Wait()
		if moves.Load() != 0 {
			t.Fatal("old countdown fired")
		}
		time.Sleep(time.Millisecond)
		synctest.Wait()
		if moves.Load() != 1 {
			t.Fatal("missing move after renewed idle period")
		}
		on = false
		if _, err := j.Configure(&on, "", nil); err != nil {
			t.Fatal(err)
		}
	})
}

func TestRestoreExpiryAndUnlimited(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "jiggler")
		if err := os.WriteFile(path, []byte("absolute\n"), 0600); err != nil {
			t.Fatal(err)
		}
		delay := func() time.Duration { return 12 * time.Second }
		j := testLoad(path, func(string) {}, delay)
		if state := j.Status(); !state.Enabled || state.Mode != "absolute" || state.MaxHours != 8 {
			t.Fatal(state)
		}
		time.Sleep(7 * time.Hour)
		j.mu.Lock()
		j.timer.Stop() // Simulate shutting down the service.
		j.timer = nil
		j.mu.Unlock()
		var moves atomic.Int64
		j = testLoad(path, func(string) { moves.Add(1) }, delay)
		if left := j.Status().RemainingMs; left != time.Hour.Milliseconds() {
			t.Fatal("restored remaining ms", left)
		}
		time.Sleep(time.Hour)
		synctest.Wait()
		if j.Status().Enabled {
			t.Fatal("deadline did not stop restored run")
		}
		before := moves.Load()
		time.Sleep(time.Minute)
		if moves.Load() != before {
			t.Fatal("moved after expiry")
		}
		zero, on := int32(0), true
		if _, err := j.Configure(&on, "", &zero); err != nil {
			t.Fatal(err)
		}
		time.Sleep(9 * time.Hour)
		if !j.Status().Enabled {
			t.Fatal("unlimited run expired")
		}
		one := int32(1)
		if state, err := j.Configure(nil, "", &one); err != nil || !state.Enabled || state.RemainingMs != time.Hour.Milliseconds() {
			t.Fatal("expected a fresh one-hour countdown", state, err)
		}
		time.Sleep(time.Hour)
		synctest.Wait()
		j = testLoad(path, func(string) { t.Error("restarted expired run") }, delay)
		if state := j.Status(); state.Enabled || state.MaxHours != 1 {
			t.Fatal(state)
		}
	})
}

func TestChangingLimitRestartsCountdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		j := testLoad(filepath.Join(t.TempDir(), "jiggler"), func(string) {}, func() time.Duration { return 0 })
		on := true
		if _, err := j.Configure(&on, "", nil); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Hour)
		j.Update()
		for _, hours := range []int32{6, 0, 6} {
			state, err := j.Configure(nil, "", &hours)
			if err != nil || !state.Enabled || state.ElapsedMs != 0 {
				t.Fatal("changing the limit should start a fresh countdown", state, err)
			}
			if hours == 6 && state.RemainingMs != (6*time.Hour).Milliseconds() {
				t.Fatal("expected six hours remaining", state)
			}
		}
		time.Sleep(time.Hour)
		same := int32(6)
		if state, err := j.Configure(nil, "", &same); err != nil || state.RemainingMs != (5*time.Hour).Milliseconds() {
			t.Fatal("repeating the saved value should preserve the countdown", state, err)
		}
		on = false
		if _, err := j.Configure(&on, "", nil); err != nil {
			t.Fatal(err)
		}
	})
}
