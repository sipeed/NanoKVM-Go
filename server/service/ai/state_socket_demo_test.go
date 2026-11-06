package ai

import (
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/tailscalegate"
)

// This file walks the built-in simulated gate driver across a real unix state
// socket. It proves an explicitly injected demo driver replays one complete
// "normal configuration" and then resets itself, using an injected clock and
// feature-manifest reader. The region reader verifies that availability no
// longer depends on /etc/kvm/region.
//
// Every snapshot asserted here is obviously simulated demo data: the peer is
// MockKVM/100.64.0.1 and no Tailscale component, device path, or network
// request is involved.

// demoSocketCode is the fixed verification code generator result, so the test
// can assert the exact code body across repeated fetches.
const demoSocketCode = "123456"

// Exact snapshot bodies for each stage, including the S0 body that opens,
// closes, and follows every reset of the demo loop.
const (
	demoSocketS0Body = `{"status":"offline","start":false,"match":false,"name":"","ip":""}`
	demoSocketS1Body = `{"status":"offline","start":true,"match":false,"name":"","ip":""}`
	demoSocketS2Body = `{"status":"offline","start":true,"match":true,"name":"","ip":""}`
	demoSocketS3Body = `{"status":"online","start":true,"match":true,"name":"MockKVM","ip":"100.64.0.1"}`
)

// demoSocketClock is a mutex-guarded fake clock. The driver reads it from the
// handler's per-connection goroutines, so it is safe under -race even though
// this test drives requests sequentially. It never advances on its own.
type demoSocketClock struct {
	mu  sync.Mutex
	now time.Time
}

func newDemoSocketClock() *demoSocketClock {
	return &demoSocketClock{now: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)}
}

func (c *demoSocketClock) time() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *demoSocketClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// demoRegionReader counts unexpected region reads and always answers cn, so
// the gate cannot fall through to application.LoadRegion and touch the device.
type demoRegionReader struct {
	mu    sync.Mutex
	calls int
}

func (r *demoRegionReader) load() proto.ApplicationRegion {
	r.mu.Lock()
	r.calls++
	r.mu.Unlock()
	return proto.ApplicationRegionCN
}

func (r *demoRegionReader) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

// demoSocketRequest issues one GET and asserts the transport-level and body
// contract in full: HTTP status, application/json, an exact Content-Length, and
// the complete body. The step label names the failing hop.
func demoSocketRequest(t *testing.T, client *http.Client, step string, path string, wantStatus int, wantBody string) string {
	t.Helper()

	resp := doStateSocketRequest(t, client, http.MethodGet, path, nil)
	body := readStateSocketBody(t, resp)

	if resp.StatusCode != wantStatus {
		t.Fatalf("%s: expected status %d, got %d body=%s", step, wantStatus, resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("%s: expected application/json content type, got %q", step, got)
	}
	if resp.ContentLength != int64(len(body)) {
		t.Fatalf("%s: expected Content-Length %d, got %d", step, len(body), resp.ContentLength)
	}
	if string(body) != wantBody {
		t.Fatalf("%s: expected body %s, got %s", step, wantBody, body)
	}
	return string(body)
}

// TestStateSocketDemoDriverWalksFullFlow drives an injected demo driver over a
// real unix socket through S0 -> S1 -> S2 -> S3 -> cycle reset -> replay -> exit,
// asserting the exact wire body at every step.
func TestStateSocketDemoDriverWalksFullFlow(t *testing.T) {
	clock := newDemoSocketClock()
	region := &demoRegionReader{}
	var featureCalls atomic.Int32

	gate := tailscalegate.NewHandler(tailscalegate.Deps{
		Driver: tailscalegate.NewDemoDriver(tailscalegate.DemoOptions{
			Now:  clock.time,
			Code: func() string { return demoSocketCode },
		}),
		LoadRegion: region.load,
		LoadFeatureFlag: func() bool {
			featureCalls.Add(1)
			return true
		},
		// The demo flow describes a device that already accepted the privacy
		// terms, so the S1 start action reaches the driver without a flag.
		LoadConsent:    func() bool { return true },
		PersistConsent: func() error { return nil },
	})

	env := newTestServiceStateEnv(t)
	_, client, _ := startTestStateSocketServerWithGate(t, env, gate)

	heartbeat := tailscalegate.LoginCodePath + "?heartbeat=true"
	start := tailscalegate.LoginCodePath + "?start=true"
	exit := tailscalegate.LoginCodePath + "?exit=true"

	// S0: the process starts uninitialized, and a code request before
	// initialization answers the %off% compatibility value without a fetch.
	demoSocketRequest(t, client, "S0 heartbeat", heartbeat, http.StatusOK, demoSocketS0Body)
	demoSocketRequest(t, client, "S0 code is the off compatibility value", tailscalegate.LoginCodePath, http.StatusOK, `{"code":"%off%"}`)

	// S1: Initialize succeeds, the snapshot shows start without match, and the
	// pairing window serves one stable code.
	demoSocketRequest(t, client, "S1 start action succeeds", start, http.StatusOK, `{"status":true,"message":""}`)
	demoSocketRequest(t, client, "S1 heartbeat", heartbeat, http.StatusOK, demoSocketS1Body)
	firstCode := demoSocketRequest(t, client, "S1 first code", tailscalegate.LoginCodePath, http.StatusOK, `{"code":"`+demoSocketCode+`"}`)
	secondCode := demoSocketRequest(t, client, "S1 second code", tailscalegate.LoginCodePath, http.StatusOK, `{"code":"`+demoSocketCode+`"}`)
	if firstCode != secondCode {
		t.Fatalf("expected the pairing code to stay stable, got %s then %s", firstCode, secondCode)
	}

	// S2: advancing exactly PairSeconds promotes the pairing lazily on the next
	// observation: the peer is matched but the tunnel is still offline, so the
	// peer identity stays hidden. The code request returns the pairing
	// directive built from the raw report.
	clock.advance(20 * time.Second)
	demoSocketRequest(t, client, "S2 heartbeat after PairSeconds", heartbeat, http.StatusOK, demoSocketS2Body)
	demoSocketRequest(t, client, "S2 pairing directive", tailscalegate.LoginCodePath, http.StatusOK, `{"code":"%online%MockKVM%100.64.0.1%"}`)

	// S3: advancing exactly OnlineDelay brings the tunnel online and publishes
	// the peer identity. This observation is the one that arms the replay
	// countdown.
	clock.advance(3 * time.Second)
	demoSocketRequest(t, client, "S3 heartbeat arms the replay countdown", heartbeat, http.StatusOK, demoSocketS3Body)

	// Reset: advancing exactly CycleSeconds resets the whole demo to S0 on the
	// next observation, and Start succeeds again over the same socket.
	clock.advance(60 * time.Second)
	demoSocketRequest(t, client, "reset heartbeat returns to S0", heartbeat, http.StatusOK, demoSocketS0Body)
	demoSocketRequest(t, client, "reset start action replays the demo", start, http.StatusOK, `{"status":true,"message":""}`)

	// Replay to the online stage, then exit: the deinitialize action resets
	// the gate to S0 immediately (no countdown needed), and a repeated exit
	// before a new initialization answers NOT_READY.
	clock.advance(20*time.Second + 3*time.Second)
	demoSocketRequest(t, client, "replay heartbeat reaches S3", heartbeat, http.StatusOK, demoSocketS3Body)
	demoSocketRequest(t, client, "exit action succeeds", exit, http.StatusOK, `{"status":true,"message":""}`)
	demoSocketRequest(t, client, "exit heartbeat returns to S0", heartbeat, http.StatusOK, demoSocketS0Body)
	demoSocketRequest(t, client, "repeated exit is not ready", exit, http.StatusOK, `{"status":false,"message":"NOT_READY"}`)

	// Availability is controlled by the feature manifest, without reading the
	// legacy region marker.
	if calls := featureCalls.Load(); calls == 0 {
		t.Fatal("expected the injected feature-manifest reader to be used")
	}
	if calls := region.callCount(); calls != 0 {
		t.Fatalf("expected no region reads, got %d", calls)
	}
}
