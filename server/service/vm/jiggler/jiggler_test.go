package jiggler

import (
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"
)

func TestIdleAndRandomDelay(t *testing.T) {
	for _, delay := range []time.Duration{0, 12 * time.Second, 30 * time.Second} {
		t.Run(delay.String(), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var moves, draws int
				j := load(filepath.Join(t.TempDir(), "jiggler"), func(string) { moves++ }, func() time.Duration { draws++; return delay })
				on := true
				if _, err := j.Configure(&on, "", nil); err != nil {
					t.Fatal(err)
				}
				for cycle := 1; cycle <= 2; cycle++ {
					time.Sleep(idleInterval - time.Millisecond)
					synctest.Wait()
					if draws != cycle-1 || moves != cycle-1 {
						t.Fatal("early draw or movement", draws, moves)
					}
					time.Sleep(time.Millisecond)
					synctest.Wait()
					if draws != cycle {
						t.Fatal("missing idle check", draws)
					}
					if delay != 0 {
						time.Sleep(delay - time.Millisecond)
						synctest.Wait()
						if moves != cycle-1 {
							t.Fatal("early movement", moves)
						}
						time.Sleep(time.Millisecond)
						synctest.Wait()
					}
					if moves != cycle {
						t.Fatal("missing movement", moves)
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
		moves := 0
		j := load(filepath.Join(t.TempDir(), "jiggler"), func(string) { moves++ }, func() time.Duration { return 12 * time.Second })
		on := true
		if _, err := j.Configure(&on, "", nil); err != nil {
			t.Fatal(err)
		}
		time.Sleep(35 * time.Second)
		j.Update()
		time.Sleep(42*time.Second - time.Millisecond)
		synctest.Wait()
		if moves != 0 {
			t.Fatal("old countdown fired")
		}
		time.Sleep(time.Millisecond)
		synctest.Wait()
		if moves != 1 {
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
		j := load(path, func(string) {}, delay)
		if state := j.Status(); !state.Enabled || state.Mode != "absolute" || state.MaxHours != 8 {
			t.Fatal(state)
		}
		time.Sleep(7 * time.Hour)
		j.timer.Stop() // Simulate shutting down the service.
		moves := 0
		j = load(path, func(string) { moves++ }, delay)
		if left := j.Status().RemainingMs; left != time.Hour.Milliseconds() {
			t.Fatal("restored remaining ms", left)
		}
		time.Sleep(time.Hour)
		synctest.Wait()
		if j.Status().Enabled {
			t.Fatal("deadline did not stop restored run")
		}
		before := moves
		time.Sleep(time.Minute)
		if moves != before {
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
		j = load(path, func(string) { t.Fatal("restarted expired run") }, delay)
		if state := j.Status(); state.Enabled || state.MaxHours != 1 {
			t.Fatal(state)
		}
	})
}

func TestChangingLimitRestartsCountdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		j := load(filepath.Join(t.TempDir(), "jiggler"), func(string) {}, func() time.Duration { return 0 })
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
