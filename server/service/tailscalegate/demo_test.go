package tailscalegate

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// demoClock is a manually advanced clock. It never reads wall time and has no
// timer, so every demo-driver test is fully deterministic and never sleeps.
type demoClock struct {
	mu  sync.Mutex
	now time.Time
}

func newDemoClock() *demoClock {
	return &demoClock{now: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}
}

func (c *demoClock) time() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *demoClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// accessor returns the clock as the injectable DemoOptions.Now function.
func (c *demoClock) accessor() func() time.Time { return c.time }

// fixedDemoCode returns a code generator that always yields value.
func fixedDemoCode(value string) func() string {
	return func() string { return value }
}

// sequentialDemoCode returns a code generator that yields a new code on every
// call, which makes a pairing restart observable.
func sequentialDemoCode() func() string {
	next := 0
	return func() string {
		next++
		return fmt.Sprintf("%06d", next)
	}
}

// newTestDemoDriver builds a demo driver on a manual clock. Unset timing and
// peer fields keep their production defaults; only the clock and code
// generator are injected.
func newTestDemoDriver(clock *demoClock, code func() string) *demoDriver {
	return NewDemoDriver(DemoOptions{Now: clock.accessor(), Code: code}).(*demoDriver)
}

// readDemoSnapshot asserts a successful, complete report and returns its
// resolved wire snapshot.
func readDemoSnapshot(t *testing.T, driver Driver, ctx context.Context) Snapshot {
	t.Helper()
	report, err := driver.Snapshot(ctx)
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if report.Status == nil || report.Start == nil || report.Match == nil ||
		report.Name == nil || report.IP == nil {
		t.Fatalf("Snapshot() = %+v, want all five fields present", report)
	}
	return Resolve(report)
}

// assertDemoSnapshot asserts the resolved snapshot of one Snapshot call.
func assertDemoSnapshot(t *testing.T, driver Driver, ctx context.Context, want Snapshot) {
	t.Helper()
	if got := readDemoSnapshot(t, driver, ctx); got != want {
		t.Fatalf("resolved snapshot = %+v, want %+v", got, want)
	}
}

func TestDemoDriverForwardFlow(t *testing.T) {
	clock := newDemoClock()
	driver := newTestDemoDriver(clock, fixedDemoCode("123456"))
	ctx := context.Background()

	// S0: uninitialized, every other field at its default.
	assertDemoSnapshot(t, driver, ctx, Snapshot{Status: StatusOffline})

	// S1: initialized, pairing open, peer not matched yet.
	if err := driver.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	assertDemoSnapshot(t, driver, ctx, Snapshot{Status: StatusOffline, Start: true})

	// The code is stable for the whole pairing window: a second fetch must not
	// regenerate it.
	first, err := driver.LoginCode(ctx)
	if err != nil {
		t.Fatalf("LoginCode() error = %v", err)
	}
	second, err := driver.LoginCode(ctx)
	if err != nil {
		t.Fatalf("LoginCode() error = %v", err)
	}
	if first != "123456" || second != first {
		t.Fatalf("login codes = %q then %q, want a stable %q", first, second, "123456")
	}

	// S2: advancing past PairSeconds completes the pairing on the next read,
	// while the tunnel is still offline.
	clock.advance(demoPairSeconds)
	assertDemoSnapshot(t, driver, ctx, Snapshot{Status: StatusOffline, Start: true, Match: true})

	// Initialize is idempotent: it must not restart the pairing timer or
	// regenerate the code after the peer is matched.
	if err := driver.Initialize(ctx); err != nil {
		t.Fatalf("second Initialize() error = %v", err)
	}
	assertDemoSnapshot(t, driver, ctx, Snapshot{Status: StatusOffline, Start: true, Match: true})
	third, err := driver.LoginCode(ctx)
	if err != nil {
		t.Fatalf("LoginCode() error = %v", err)
	}
	if third != first {
		t.Fatalf("login code after idempotent Initialize = %q, want %q", third, first)
	}

	// S3: one OnlineDelay after the match instant the tunnel is online and the
	// peer identity appears.
	clock.advance(demoOnlineDelay)
	assertDemoSnapshot(t, driver, ctx, Snapshot{
		Status: StatusOnline, Start: true, Match: true,
		Name: demoName, IP: demoIP,
	})
}

func TestDemoDriverDeinitializeResets(t *testing.T) {
	clock := newDemoClock()
	driver := newTestDemoDriver(clock, fixedDemoCode("123456"))
	ctx := context.Background()

	driveToOnline(t, driver, clock, ctx)

	// Deinitialize is the inverse of Initialize: the whole demo returns to the
	// uninitialized stage and the pairing session is cleared.
	if err := driver.Deinitialize(ctx); err != nil {
		t.Fatalf("Deinitialize() error = %v", err)
	}
	assertDemoSnapshot(t, driver, ctx, Snapshot{Status: StatusOffline})

	// Repeated Deinitialize calls are idempotent.
	if err := driver.Deinitialize(ctx); err != nil {
		t.Fatalf("second Deinitialize() error = %v", err)
	}
	assertDemoSnapshot(t, driver, ctx, Snapshot{Status: StatusOffline})

	// A fresh Initialize after the reset opens a new pairing session.
	if err := driver.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() after reset error = %v", err)
	}
	assertDemoSnapshot(t, driver, ctx, Snapshot{Status: StatusOffline, Start: true})
}

func TestDemoDriverReportKeepsRawPeerValues(t *testing.T) {
	clock := newDemoClock()
	driver := newTestDemoDriver(clock, fixedDemoCode("123456"))
	ctx := context.Background()

	if err := driver.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	clock.advance(demoPairSeconds)

	// A matched-but-offline peer must keep raw name/ip in the report, because
	// handleCode builds the pairing directive from the raw report.
	report, err := driver.Snapshot(ctx)
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if report.Match == nil || !*report.Match {
		t.Fatalf("report.Match = %v, want true", report.Match)
	}
	if report.Name == nil || *report.Name != demoName {
		t.Fatalf("report.Name = %v, want %q", report.Name, demoName)
	}
	if report.IP == nil || *report.IP != demoIP {
		t.Fatalf("report.IP = %v, want %q", report.IP, demoIP)
	}

	// Resolve is what hides the peer while the tunnel is offline.
	resolved := Resolve(report)
	if resolved.Status != StatusOffline || resolved.Name != "" || resolved.IP != "" {
		t.Fatalf("resolved snapshot = %+v, want the offline peer hidden", resolved)
	}

	// Once the tunnel comes online the identity is visible again.
	clock.advance(demoOnlineDelay)
	report, err = driver.Snapshot(ctx)
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	resolved = Resolve(report)
	if resolved.Name != demoName || resolved.IP != demoIP {
		t.Fatalf("resolved snapshot = %+v, want the online peer visible", resolved)
	}
}

func TestDemoDriverEndLoginSessionRestartsPairing(t *testing.T) {
	clock := newDemoClock()
	driver := newTestDemoDriver(clock, sequentialDemoCode())
	ctx := context.Background()

	if err := driver.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	first, err := driver.LoginCode(ctx)
	if err != nil {
		t.Fatalf("LoginCode() error = %v", err)
	}

	// Ending the session interrupts the pairing but must not unset the
	// initialized gate.
	if err := driver.EndLoginSession(ctx); err != nil {
		t.Fatalf("EndLoginSession() error = %v", err)
	}
	assertDemoSnapshot(t, driver, ctx, Snapshot{Status: StatusOffline, Start: true})

	// The next fetch opens a fresh pairing session with a new code.
	second, err := driver.LoginCode(ctx)
	if err != nil {
		t.Fatalf("LoginCode() error = %v", err)
	}
	if second == first {
		t.Fatalf("login code after EndLoginSession = %q, want a fresh code", second)
	}

	// The restarted session still completes after its own PairSeconds.
	clock.advance(demoPairSeconds)
	assertDemoSnapshot(t, driver, ctx, Snapshot{Status: StatusOffline, Start: true, Match: true})
}

func TestDemoDriverExpiredContextDoesNotMutate(t *testing.T) {
	t.Run("all methods reject", func(t *testing.T) {
		clock := newDemoClock()
		driver := newTestDemoDriver(clock, fixedDemoCode("123456"))
		expired, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()

		calls := []struct {
			name string
			call func() error
		}{
			{"snapshot", func() error {
				_, err := driver.Snapshot(expired)
				return err
			}},
			{"initialize", func() error { return driver.Initialize(expired) }},
			{"deinitialize", func() error { return driver.Deinitialize(expired) }},
			{"login code", func() error {
				_, err := driver.LoginCode(expired)
				return err
			}},
			{"end login session", func() error { return driver.EndLoginSession(expired) }},
		}
		for _, tc := range calls {
			t.Run(tc.name, func(t *testing.T) {
				if err := tc.call(); !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("error = %v, want context.DeadlineExceeded", err)
				}
			})
		}

		// None of the rejected calls may have changed the gate: still S0.
		assertDemoSnapshot(t, driver, context.Background(), Snapshot{Status: StatusOffline})
	})

	t.Run("pairing not promoted", func(t *testing.T) {
		clock := newDemoClock()
		driver := newTestDemoDriver(clock, fixedDemoCode("123456"))
		if err := driver.Initialize(context.Background()); err != nil {
			t.Fatalf("Initialize() error = %v", err)
		}
		// The pairing window is already due, but an expired context must not be
		// allowed to perform the lazy promotion.
		clock.advance(demoPairSeconds)

		expired, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()
		if _, err := driver.Snapshot(expired); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Snapshot() error = %v, want context.DeadlineExceeded", err)
		}

		// Read the state directly: any valid driver call would itself promote the
		// due pairing, so a method call cannot prove the rejected one did nothing.
		driver.mu.Lock()
		started, pairing, matched := driver.started, driver.pairing, driver.matched
		driver.mu.Unlock()
		if !started || !pairing || matched {
			t.Fatalf("state after expired Snapshot = started %v / pairing %v / matched %v, "+
				"want the pairing window untouched", started, pairing, matched)
		}
	})
}

func TestDemoDriverDefaults(t *testing.T) {
	driver := NewDemoDriver(DemoOptions{}).(*demoDriver)
	if driver.pairSeconds != demoPairSeconds {
		t.Fatalf("pairSeconds = %v, want %v", driver.pairSeconds, demoPairSeconds)
	}
	if driver.onlineDelay != demoOnlineDelay {
		t.Fatalf("onlineDelay = %v, want %v", driver.onlineDelay, demoOnlineDelay)
	}
	if driver.cycleSeconds != demoCycleSeconds {
		t.Fatalf("cycleSeconds = %v, want %v", driver.cycleSeconds, demoCycleSeconds)
	}
	if driver.name != demoName {
		t.Fatalf("name = %q, want %q", driver.name, demoName)
	}
	if driver.ip != demoIP {
		t.Fatalf("ip = %q, want %q", driver.ip, demoIP)
	}
	if driver.now == nil {
		t.Fatal("now = nil, want time.Now")
	}
	if driver.code == nil {
		t.Fatal("code = nil, want the default six-digit generator")
	}

	// An explicitly set zero duration still means "use the default", matching
	// the Deps convention.
	zero := NewDemoDriver(DemoOptions{PairSeconds: 0, OnlineDelay: 0, CycleSeconds: 0}).(*demoDriver)
	if zero.pairSeconds != demoPairSeconds || zero.onlineDelay != demoOnlineDelay {
		t.Fatalf("zero options = pair %v / online %v, want the production defaults",
			zero.pairSeconds, zero.onlineDelay)
	}
	if zero.cycleSeconds != demoCycleSeconds {
		t.Fatalf("zero options cycleSeconds = %v, want %v", zero.cycleSeconds, demoCycleSeconds)
	}
}

func TestRandomDemoCodeFormat(t *testing.T) {
	for i := 0; i < 64; i++ {
		code := randomDemoCode()
		if len(code) != 6 {
			t.Fatalf("randomDemoCode() = %q, want six digits", code)
		}
		if strings.Trim(code, "0123456789") != "" {
			t.Fatalf("randomDemoCode() = %q, want only decimal digits", code)
		}
	}
}

// newCycleDemoDriver builds a demo driver on a manual clock with a short,
// explicit replay window so cycle tests never depend on the 60s default.
func newCycleDemoDriver(clock *demoClock, cycleSeconds time.Duration) *demoDriver {
	return NewDemoDriver(DemoOptions{
		Now:          clock.accessor(),
		Code:         fixedDemoCode("123456"),
		CycleSeconds: cycleSeconds,
	}).(*demoDriver)
}

func TestDemoDriverCycleResetsAfterTerminal(t *testing.T) {
	clock := newDemoClock()
	driver := newCycleDemoDriver(clock, 10*time.Second)
	ctx := context.Background()

	// driveToOnline reaches the terminal stage and its final Snapshot
	// observation arms the countdown at that instant.
	driveToOnline(t, driver, clock, ctx)

	// Strictly inside the window the demo stays on the terminal stage.
	clock.advance(10*time.Second - time.Second)
	assertDemoSnapshot(t, driver, ctx, Snapshot{
		Status: StatusOnline, Start: true, Match: true,
		Name: demoName, IP: demoIP,
	})

	// Reaching the window resets the whole demo to the S0 defaults.
	clock.advance(time.Second)
	assertDemoSnapshot(t, driver, ctx, Snapshot{Status: StatusOffline})
}

func TestDemoDriverCycleReplaysAfterReset(t *testing.T) {
	clock := newDemoClock()
	driver := newCycleDemoDriver(clock, 10*time.Second)
	ctx := context.Background()

	driveToOnline(t, driver, clock, ctx)
	clock.advance(10 * time.Second)
	assertDemoSnapshot(t, driver, ctx, Snapshot{Status: StatusOffline})

	// The reset left no pairing gate or timestamp behind, so a fresh
	// Initialize walks the exact same stages again.
	if err := driver.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() after reset error = %v", err)
	}
	assertDemoSnapshot(t, driver, ctx, Snapshot{Status: StatusOffline, Start: true})

	clock.advance(demoPairSeconds)
	assertDemoSnapshot(t, driver, ctx, Snapshot{Status: StatusOffline, Start: true, Match: true})

	clock.advance(demoOnlineDelay)
	assertDemoSnapshot(t, driver, ctx, Snapshot{
		Status: StatusOnline, Start: true, Match: true,
		Name: demoName, IP: demoIP,
	})
}

func TestDemoDriverCycleSurvivesEndLoginSession(t *testing.T) {
	clock := newDemoClock()
	driver := newCycleDemoDriver(clock, 10*time.Second)
	ctx := context.Background()

	driveToOnline(t, driver, clock, ctx) // arms the countdown at the terminal stage

	if err := driver.EndLoginSession(ctx); err != nil {
		t.Fatalf("EndLoginSession() error = %v", err)
	}
	assertDemoSnapshot(t, driver, ctx, Snapshot{
		Status: StatusOnline, Start: true, Match: true,
		Name: demoName, IP: demoIP,
	})

	// The armed countdown is not cancelled by ending the pairing session: the
	// demo still returns to S0 once the window expires.
	clock.advance(10 * time.Second)
	assertDemoSnapshot(t, driver, ctx, Snapshot{Status: StatusOffline})
}

func TestDemoDriverCycleNotArmedWithoutTerminal(t *testing.T) {
	clock := newDemoClock()
	// A long online delay keeps the demo at the matched-but-offline stage for
	// the whole test, so the terminal stage is never reached and the countdown
	// is never armed.
	driver := NewDemoDriver(DemoOptions{
		Now:          clock.accessor(),
		Code:         fixedDemoCode("123456"),
		CycleSeconds: 10 * time.Second,
		OnlineDelay:  time.Hour,
	}).(*demoDriver)
	ctx := context.Background()

	if err := driver.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	clock.advance(demoPairSeconds)
	assertDemoSnapshot(t, driver, ctx, Snapshot{Status: StatusOffline, Start: true, Match: true})

	// Far past the cycle window the state must still be the matched-offline
	// stage, because the countdown was never armed.
	clock.advance(10 * time.Minute)
	assertDemoSnapshot(t, driver, ctx, Snapshot{Status: StatusOffline, Start: true, Match: true})
}

func TestDemoDriverRace(t *testing.T) {
	clock := newDemoClock()
	driver := newTestDemoDriver(clock, fixedDemoCode("123456"))
	ctx := context.Background()

	const iterations = 200
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			if _, err := driver.Snapshot(ctx); err != nil {
				t.Errorf("Snapshot() error = %v", err)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			if err := driver.Deinitialize(ctx); err != nil {
				t.Errorf("Deinitialize() error = %v", err)
				return
			}
		}
	}()

	wg.Wait()

	// A final Deinitialize leaves the demo at the uninitialized stage.
	if err := driver.Deinitialize(ctx); err != nil {
		t.Fatalf("final Deinitialize() error = %v", err)
	}
	assertDemoSnapshot(t, driver, ctx, Snapshot{Status: StatusOffline})
}

// driveToOnline walks a driver from S0 to the online terminal stage using only
// clock advancement, so callers can start from a known steady state.
func driveToOnline(t *testing.T, driver *demoDriver, clock *demoClock, ctx context.Context) {
	t.Helper()
	if err := driver.Initialize(ctx); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	clock.advance(demoPairSeconds)
	clock.advance(demoOnlineDelay)
	assertDemoSnapshot(t, driver, ctx, Snapshot{
		Status: StatusOnline, Start: true, Match: true,
		Name: demoName, IP: demoIP,
	})
}
