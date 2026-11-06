package tailscalegate

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/application"
)

func TestUnimplementedDriver(t *testing.T) {
	driver := NewUnimplementedDriver()
	ctx := context.Background()

	cases := []struct {
		name string
		call func() error
	}{
		{"snapshot", func() error {
			if _, err := driver.Snapshot(ctx); err != nil {
				return err
			}
			return nil
		}},
		{"initialize", func() error { return driver.Initialize(ctx) }},
		{"deinitialize", func() error { return driver.Deinitialize(ctx) }},
		{"login code", func() error {
			if _, err := driver.LoginCode(ctx); err != nil {
				return err
			}
			return nil
		}},
		{"end login session", func() error { return driver.EndLoginSession(ctx) }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if err == nil {
				t.Fatal("error = nil, want the unimplemented error")
			}
			if !errors.Is(err, ErrUnimplemented) {
				t.Fatalf("errors.Is(err, ErrUnimplemented) = false for %v", err)
			}
			var messageErr MessageError
			if !errors.As(err, &messageErr) {
				t.Fatalf("error %v does not implement MessageError", err)
			}
			if got := messageErr.GateMessage(); got != MessageNotImplemented {
				t.Fatalf("GateMessage() = %q, want %q", got, MessageNotImplemented)
			}
			want := ActionResponse{Status: false, Message: MessageNotImplemented}
			if got := FailureAction(err); got != want {
				t.Fatalf("FailureAction(err) = %+v, want %+v", got, want)
			}
		})
	}
}

func TestUnimplementedSnapshotReturnsNothing(t *testing.T) {
	report, err := NewUnimplementedDriver().Snapshot(context.Background())
	if err == nil {
		t.Fatal("error = nil, want the unimplemented error")
	}
	if report != (Report{}) {
		t.Fatalf("report = %+v, want the zero report", report)
	}
}

func TestNewHandlerDefaults(t *testing.T) {
	handler := NewHandler(Deps{})
	if handler.driver == nil {
		t.Fatal("driver = nil, want the built-in simulated demo driver")
	}
	// The default driver must be the built-in simulated demo driver, proven by
	// behavior rather than only by type: a snapshot succeeds and reports all five
	// fields.
	report, err := handler.driver.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("default driver Snapshot() error = %v, want nil", err)
	}
	if report.Status == nil || report.Start == nil || report.Match == nil ||
		report.Name == nil || report.IP == nil {
		t.Fatalf("default driver Snapshot() = %+v, want all five fields", report)
	}
	if _, ok := handler.driver.(*demoDriver); !ok {
		t.Fatalf("driver = %T, want *demoDriver", handler.driver)
	}
	if handler.loadRegion == nil {
		t.Fatal("loadRegion = nil, want the production region reader")
	}
	if handler.loadFeatureState == nil {
		t.Fatal("loadFeatureState = nil, want the production manifest reader")
	}
	// A nil sync seam is the deliberate not-wired default: a start that needs a
	// sync must fail closed instead of reaching the network.
	if handler.syncFeatureManifest != nil {
		t.Fatal("syncFeatureManifest = non-nil, want the not-wired nil default")
	}
	if handler.loadConsent == nil {
		t.Fatal("loadConsent = nil, want the fail-closed consent reader")
	}
	if handler.loadConsent() {
		t.Fatal("default loadConsent() = true, want the fail-closed false")
	}
	if handler.persistConsent == nil {
		t.Fatal("persistConsent = nil, want the no-op consent writer")
	}
	if handler.readBudget != defaultReadBudget {
		t.Fatalf("readBudget = %v, want %v", handler.readBudget, defaultReadBudget)
	}
	if handler.actionBudget != defaultActionBudget {
		t.Fatalf("actionBudget = %v, want %v", handler.actionBudget, defaultActionBudget)
	}
	if handler.disableBudget != defaultDisableBudget {
		t.Fatalf("disableBudget = %v, want %v", handler.disableBudget, defaultDisableBudget)
	}
	if handler.actionSlot == nil || cap(handler.actionSlot) != 1 {
		t.Fatalf("actionSlot = %v, want a capacity-1 channel", handler.actionSlot)
	}
	// Constructing the handler must not read the region marker or perform any
	// other I/O: the injection seam is present but is not invoked here.

	defaultHandler := NewDefaultHandler()
	if defaultHandler == nil || defaultHandler.driver == nil || defaultHandler.loadRegion == nil ||
		defaultHandler.loadFeatureState == nil {
		t.Fatal("NewDefaultHandler returned an unusable handler")
	}
}

func TestNewHandlerDefaultsToDemoDriver(t *testing.T) {
	// No Driver is injected, so NewHandler must select the built-in simulated
	// demo driver. The injected region reader keeps /etc/kvm/region unread.
	handler := NewHandler(Deps{
		LoadRegion:      func() proto.ApplicationRegion { return proto.ApplicationRegionCN },
		LoadFeatureFlag: func() bool { return true },
	})

	recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true")
	assertStatus(t, recorder, http.StatusOK)
	if got, want := decodeJSON[snapshotResponse](t, recorder), (snapshotResponse{Status: StatusOffline}); got != want {
		t.Fatalf("default heartbeat = %+v, want %+v", got, want)
	}

	recorder = serve(t, handler, http.MethodGet, LoginCodePath+"?start=true&consent=true")
	assertStatus(t, recorder, http.StatusOK)
	if got, want := decodeJSON[ActionResponse](t, recorder), SuccessAction(); got != want {
		t.Fatalf("default start action = %+v, want %+v", got, want)
	}
}

func TestNewHandlerHonoursOverrides(t *testing.T) {
	driver := newFakeDriver()
	region := func() proto.ApplicationRegion { return proto.ApplicationRegionGlobal }
	featureState := func() application.FeatureState { return application.FeatureStateEnabled }
	syncFeatureManifest := func(context.Context, proto.ApplicationRegion) error { return nil }
	handler := NewHandler(Deps{
		Driver:              driver,
		LoadRegion:          region,
		LoadFeatureState:    featureState,
		SyncFeatureManifest: syncFeatureManifest,
		ReadBudget:          7 * time.Millisecond,
		ActionBudget:        9 * time.Millisecond,
		DisableBudget:       11 * time.Millisecond,
	})

	if handler.driver != driver {
		t.Fatal("driver override was not kept")
	}
	if handler.loadRegion == nil {
		t.Fatal("loadRegion override was not kept")
	}
	if handler.loadFeatureState == nil {
		t.Fatal("loadFeatureState override was not kept")
	}
	if got := handler.loadFeatureState(); got != application.FeatureStateEnabled {
		t.Fatalf("loadFeatureState() = %v, want enabled", got)
	}
	if handler.syncFeatureManifest == nil {
		t.Fatal("syncFeatureManifest override was not kept")
	}
	if handler.readBudget != 7*time.Millisecond {
		t.Fatalf("readBudget = %v, want 7ms", handler.readBudget)
	}
	if handler.actionBudget != 9*time.Millisecond {
		t.Fatalf("actionBudget = %v, want 9ms", handler.actionBudget)
	}
	if handler.disableBudget != 11*time.Millisecond {
		t.Fatalf("disableBudget = %v, want 11ms", handler.disableBudget)
	}
}
