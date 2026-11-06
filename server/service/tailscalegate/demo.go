package tailscalegate

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// This file adds the built-in simulated gate driver.
//
// It exists so an unmodified device can walk through one complete, replayable
// "normal configuration" instead of answering NOTIMPL forever on the state
// socket. The wire contract, dispatch, budgets, feature gate, and retained gate
// are unchanged; only the implementation behind the Driver boundary differs.
//
// The demo is a loop, not a one-way trip: once the terminal stage (matched and
// tunnel online) has been observed, a CycleSeconds countdown starts, and when
// it expires the whole driver returns to the uninitialized stage. Every visit
// to the UI therefore starts from Start again. The countdown is lazy and is
// only evaluated from the injected clock, so there is still no goroutine,
// timer, or sleep.
//
// The demo driver follows the README section 7.4 table exactly: Initialize
// opens a client initialization session and a six-digit code stays stable for
// the whole pairing window; after PairSeconds the peer is reported as matched
// while the tunnel is still offline; OnlineDelay after that instant the tunnel
// comes online and the peer identity appears. Deinitialize is the inverse and
// returns the demo to the uninitialized stage.
//
// The driver is SIMULATED and performs no system access at all. It does not
// read /etc/kvm, /dev, or /sys, does not invoke tailscale, tailscaled, or
// systemd, does not open a network connection, and does not write a file. It
// also never logs the peer name, IP, or verification code. All time-driven
// transitions are evaluated lazily from an injected clock: there is no
// goroutine, timer, or sleep in this file.
//
// The demo is selected by NewHandler when no driver is injected, which covers
// tests and host walkthroughs. Production wiring (service/ai/state_socket.go)
// injects the real managed-network driver instead, and never selects this
// demo. Until that wiring exists on a device, the initialized state plus
// MockKVM/100.64.0.1 is obviously simulated demo data, not a real tailnet
// peer.

const (
	// demoPairSeconds is the default client initialization window, matching the
	// reference mock server default (--pair-seconds 20).
	demoPairSeconds = 20 * time.Second
	// demoOnlineDelay is the delay between the pairing completing and the
	// tunnel reporting online. One heartbeat period keeps the documented
	// "ready but offline" transition observable while still finishing on a
	// successful configuration.
	demoOnlineDelay = 3 * time.Second
	// demoCycleSeconds is how long the terminal stage is held before the whole
	// demo resets to the uninitialized stage, so the flow can be replayed
	// without restarting the server.
	demoCycleSeconds = 60 * time.Second
	// demoName and demoIP are obviously simulated peer values, exactly as in the
	// reference mock server. 100.64.0.0/10 is the CGNAT range Tailscale uses, so
	// the address cannot be mistaken for a public endpoint.
	demoName       = "MockKVM"
	demoIP         = "100.64.0.1"
	demoCodeModulo = 1000000
)

// DemoOptions configures the simulated driver.
//
// The zero value is valid and selects every production default, matching the
// Deps convention: an unset (zero, negative, or nil) field is replaced by its
// default in NewDemoDriver.
type DemoOptions struct {
	// PairSeconds is how long the client initialization window lasts before the
	// peer is reported as matched. Zero or negative selects 20s.
	PairSeconds time.Duration
	// OnlineDelay is how long after the pairing completes the tunnel reports
	// online. Zero or negative selects 3s.
	OnlineDelay time.Duration
	// CycleSeconds is how long the terminal stage lasts before the driver resets
	// itself to the uninitialized stage for the next replay. Zero or negative
	// selects 60s.
	CycleSeconds time.Duration
	// Name and IP are the raw simulated peer values. Empty selects "MockKVM"
	// and "100.64.0.1".
	Name string
	IP   string
	// Now is the clock read at the start of every method. nil selects time.Now.
	// It must be safe for concurrent use.
	Now func() time.Time
	// Code generates one verification code when a pairing session begins. nil
	// selects a six-digit random decimal. It is called while the state lock is
	// held, so it does not need its own synchronization.
	Code func() string
}

// demoDriver is a race-safe, lazily advancing Driver that walks the documented
// gate stages without touching any real Tailscale component. See the file
// comment for the exact simulation boundary.
//
// One instance is shared by the heartbeat and action connections, so every
// state read and write happens under mu. A read or write never blocks on I/O
// while holding the lock.
type demoDriver struct {
	mu sync.Mutex

	pairSeconds  time.Duration
	onlineDelay  time.Duration
	cycleSeconds time.Duration
	name         string
	ip           string
	now          func() time.Time
	code         func() string

	// Gate and tunnel state. The three booleans mirror the heartbeat snapshot
	// fields; pairing tracks the client initialization session that produces the
	// verification code.
	started bool
	matched bool
	online  bool

	pairing       bool
	pairStartedAt time.Time
	codeValue     string

	// matchedAt is the instant the peer was reported as matched. It is zero
	// before the pairing completes and drives the online transition from then
	// on.
	matchedAt time.Time
	// cycleArmedAt is the instant the terminal stage was first observed. It is
	// zero before the demo reaches the terminal stage and drives the replay
	// countdown from then on. Only the reset itself clears it, so an
	// EndLoginSession cannot cancel an armed replay.
	cycleArmedAt time.Time
}

// NewDemoDriver returns the built-in simulated driver. It performs no system
// access; see the file comment. The production handler selects this driver when
// no driver is explicitly injected.
func NewDemoDriver(opts DemoOptions) Driver {
	pairSeconds := opts.PairSeconds
	if pairSeconds <= 0 {
		pairSeconds = demoPairSeconds
	}
	onlineDelay := opts.OnlineDelay
	if onlineDelay <= 0 {
		onlineDelay = demoOnlineDelay
	}
	cycleSeconds := opts.CycleSeconds
	if cycleSeconds <= 0 {
		cycleSeconds = demoCycleSeconds
	}
	name := opts.Name
	if name == "" {
		name = demoName
	}
	ip := opts.IP
	if ip == "" {
		ip = demoIP
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	code := opts.Code
	if code == nil {
		code = randomDemoCode
	}
	return &demoDriver{
		pairSeconds:  pairSeconds,
		onlineDelay:  onlineDelay,
		cycleSeconds: cycleSeconds,
		name:         name,
		ip:           ip,
		now:          now,
		code:         code,
	}
}

// randomDemoCode returns a zero-padded six-digit decimal, matching the
// reference server's _random_code. The math/rand top-level functions are safe
// for concurrent use, and the driver already serializes calls under its lock.
func randomDemoCode() string {
	return fmt.Sprintf("%06d", rand.Intn(demoCodeModulo))
}

// advanceLocked applies the time-driven transitions up to now. The caller must
// hold d.mu.
//
// Every transition is lazy: nothing runs in the background and state only
// changes when a driver method is called. The order is fixed:
//
//  1. a due pairing session completes and reports the peer as matched;
//  2. a matching completed peer that has waited onlineDelay comes online;
//  3. the terminal stage (matched and online) arms the replay countdown the
//     first time it is seen;
//  4. an armed countdown that has reached cycleSeconds resets the whole demo.
//
// A caller that advances the clock far past several transitions still observes
// the same final state, because each step is evaluated in this order. Step 3
// runs before step 4 with the same now, but an instant that arms the countdown
// can never also expire it, because cycleSeconds is always positive.
func (d *demoDriver) advanceLocked(now time.Time) {
	if d.pairing && now.Sub(d.pairStartedAt) >= d.pairSeconds {
		d.matched = true
		d.pairing = false
		// Anchor the match instant to the actual due time rather than the
		// observation time, so a caller that jumps past several transitions at
		// once still observes the same final state (the online delay is measured
		// from when the pairing was due, not from when it was noticed).
		d.matchedAt = d.pairStartedAt.Add(d.pairSeconds)
		d.pairStartedAt = time.Time{}
	}
	if d.matched && !d.online && now.Sub(d.matchedAt) >= d.onlineDelay {
		d.online = true
	}
	if d.matched && d.online && d.cycleArmedAt.IsZero() {
		d.cycleArmedAt = now
	}
	if !d.cycleArmedAt.IsZero() && now.Sub(d.cycleArmedAt) >= d.cycleSeconds {
		d.resetLocked()
	}
}

// resetLocked returns the driver to the uninitialized stage for the next
// replay. It replaces the whole demo state instead of clearing fields one by
// one, so no pairing flag or timestamp can survive the reset. The caller must
// hold d.mu.
func (d *demoDriver) resetLocked() {
	d.started = false
	d.matched = false
	d.online = false
	d.pairing = false
	d.pairStartedAt = time.Time{}
	d.codeValue = ""
	d.matchedAt = time.Time{}
	d.cycleArmedAt = time.Time{}
}

// beginPairingLocked starts a client initialization session when one is needed.
// It returns without changing anything when the peer is already matched or a
// session is already open, so repeated Initialize or LoginCode calls cannot
// restart the pairing timer or regenerate the current code. The caller must
// hold d.mu.
func (d *demoDriver) beginPairingLocked(now time.Time) {
	if d.matched || d.pairing {
		return
	}
	d.pairing = true
	d.pairStartedAt = now
	d.codeValue = d.code()
}

// Snapshot reports the current gate and tunnel state.
//
// Every field is present. name and ip are the raw demo values; hiding them
// while the tunnel is offline is Resolve's job, and handleCode needs the raw
// values to build the pairing directive.
func (d *demoDriver) Snapshot(ctx context.Context) (Report, error) {
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	d.advanceLocked(d.now())

	status := string(StatusOffline)
	if d.online {
		status = string(StatusOnline)
	}
	started := d.started
	matched := d.matched
	name := d.name
	ip := d.ip
	return Report{
		Status: &status,
		Start:  &started,
		Match:  &matched,
		Name:   &name,
		IP:     &ip,
	}, nil
}

// Initialize starts the simulated component and opens a client initialization
// session when the peer is not matched yet. Repeated calls are idempotent.
func (d *demoDriver) Initialize(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	now := d.now()
	d.advanceLocked(now)
	d.started = true
	d.beginPairingLocked(now)
	return nil
}

// Deinitialize is the inverse of Initialize: it resets the whole simulated
// state to the uninitialized stage and clears the pairing session. It takes
// effect immediately.
func (d *demoDriver) Deinitialize(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	d.advanceLocked(d.now())
	d.resetLocked()
	return nil
}

// LoginCode returns the current verification code, opening a pairing session
// first when none is active. The code is stable for the whole pairing window.
func (d *demoDriver) LoginCode(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	now := d.now()
	d.advanceLocked(now)
	d.beginPairingLocked(now)
	return d.codeValue, nil
}

// EndLoginSession ends the client initialization session only. It never changes
// the initialized gate, and the next code request starts a fresh pairing
// session with a new code.
func (d *demoDriver) EndLoginSession(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	d.advanceLocked(d.now())
	d.pairing = false
	d.pairStartedAt = time.Time{}
	return nil
}
