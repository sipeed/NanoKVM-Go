package tailscalegate

import (
	"context"
	"sync"
)

// callBlock selects how a fake driver method waits before returning.
type callBlock int

const (
	// blockNone returns immediately.
	blockNone callBlock = iota
	// blockDeadline waits for the context to expire and returns its error. It
	// is how the timeout branches are driven without a hard-coded wait.
	blockDeadline
	// blockRelease signals started and waits until release is closed (or the
	// context expires). It is how the BUSY and fire-and-forget branches hold an
	// action in flight without sleeping.
	blockRelease
)

// fakeCallCounts records how often each Driver method ran.
type fakeCallCounts struct {
	snapshot        int
	initialize      int
	deinitialize    int
	loginCode       int
	endLoginSession int
}

// fakeDriver is a scripted, race-safe Driver for handler tests. It performs no
// I/O and never sleeps: blocking modes wait on channels or the context.
//
// A single release channel is closed by the test to unblock every blocked
// method, which is enough because a test drives one blocked action at a time.
type fakeDriver struct {
	mu sync.Mutex

	report Report

	snapshotErr        error
	initializeErr      error
	deinitializeErr    error
	loginCodeErr       error
	endLoginSessionErr error

	loginCodeValue string

	snapshotBlock        callBlock
	initializeBlock      callBlock
	deinitializeBlock    callBlock
	loginCodeBlock       callBlock
	endLoginSessionBlock callBlock

	counts fakeCallCounts

	// started receives one signal when a blockRelease method begins waiting.
	started chan struct{}
	// release is closed by the test to let blockRelease methods return.
	release chan struct{}
	// finished receives one signal after any method returns.
	finished chan struct{}
}

func newFakeDriver() *fakeDriver {
	return &fakeDriver{
		started:  make(chan struct{}, 8),
		release:  make(chan struct{}),
		finished: make(chan struct{}, 64),
	}
}

func (d *fakeDriver) Snapshot(ctx context.Context) (Report, error) {
	d.mu.Lock()
	d.counts.snapshot++
	report := d.report
	err := d.snapshotErr
	mode := d.snapshotBlock
	d.mu.Unlock()
	defer d.signalFinished()

	if blockErr := d.wait(ctx, mode); blockErr != nil {
		return Report{}, blockErr
	}
	if err != nil {
		return Report{}, err
	}
	return report, nil
}

func (d *fakeDriver) Initialize(ctx context.Context) error {
	d.mu.Lock()
	d.counts.initialize++
	err := d.initializeErr
	mode := d.initializeBlock
	d.mu.Unlock()
	defer d.signalFinished()

	if blockErr := d.wait(ctx, mode); blockErr != nil {
		return blockErr
	}
	return err
}

func (d *fakeDriver) Deinitialize(ctx context.Context) error {
	d.mu.Lock()
	d.counts.deinitialize++
	err := d.deinitializeErr
	mode := d.deinitializeBlock
	d.mu.Unlock()
	defer d.signalFinished()

	if blockErr := d.wait(ctx, mode); blockErr != nil {
		return blockErr
	}
	return err
}

func (d *fakeDriver) LoginCode(ctx context.Context) (string, error) {
	d.mu.Lock()
	d.counts.loginCode++
	value := d.loginCodeValue
	err := d.loginCodeErr
	mode := d.loginCodeBlock
	d.mu.Unlock()
	defer d.signalFinished()

	if blockErr := d.wait(ctx, mode); blockErr != nil {
		return "", blockErr
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

func (d *fakeDriver) EndLoginSession(ctx context.Context) error {
	d.mu.Lock()
	d.counts.endLoginSession++
	err := d.endLoginSessionErr
	mode := d.endLoginSessionBlock
	d.mu.Unlock()
	defer d.signalFinished()

	if blockErr := d.wait(ctx, mode); blockErr != nil {
		return blockErr
	}
	return err
}

// wait applies the configured blocking mode.
func (d *fakeDriver) wait(ctx context.Context, mode callBlock) error {
	switch mode {
	case blockDeadline:
		<-ctx.Done()
		return ctx.Err()
	case blockRelease:
		d.signal(d.started)
		select {
		case <-d.release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	default:
		return nil
	}
}

func (d *fakeDriver) signal(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

func (d *fakeDriver) signalFinished() { d.signal(d.finished) }

func (d *fakeDriver) setReport(report Report) {
	d.mu.Lock()
	d.report = report
	d.mu.Unlock()
}

func (d *fakeDriver) setSnapshotErr(err error) {
	d.mu.Lock()
	d.snapshotErr = err
	d.mu.Unlock()
}

func (d *fakeDriver) setInitializeErr(err error) {
	d.mu.Lock()
	d.initializeErr = err
	d.mu.Unlock()
}

func (d *fakeDriver) setDeinitializeErr(err error) {
	d.mu.Lock()
	d.deinitializeErr = err
	d.mu.Unlock()
}

func (d *fakeDriver) setLoginCode(value string) {
	d.mu.Lock()
	d.loginCodeValue = value
	d.mu.Unlock()
}

func (d *fakeDriver) setLoginCodeErr(err error) {
	d.mu.Lock()
	d.loginCodeErr = err
	d.mu.Unlock()
}

func (d *fakeDriver) setEndLoginSessionErr(err error) {
	d.mu.Lock()
	d.endLoginSessionErr = err
	d.mu.Unlock()
}

func (d *fakeDriver) setInitializeBlock(mode callBlock) {
	d.mu.Lock()
	d.initializeBlock = mode
	d.mu.Unlock()
}

func (d *fakeDriver) setDeinitializeBlock(mode callBlock) {
	d.mu.Lock()
	d.deinitializeBlock = mode
	d.mu.Unlock()
}

func (d *fakeDriver) setEndLoginSessionBlock(mode callBlock) {
	d.mu.Lock()
	d.endLoginSessionBlock = mode
	d.mu.Unlock()
}

func (d *fakeDriver) callCounts() fakeCallCounts {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.counts
}

// testMessageError is a driver error that publishes a stable action message.
type testMessageError string

func (e testMessageError) Error() string { return "fake driver error" }

func (e testMessageError) GateMessage() string { return string(e) }
