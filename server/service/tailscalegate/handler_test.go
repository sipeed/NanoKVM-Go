package tailscalegate

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/application"
)

func ptr[T any](value T) *T { return &value }

func testDeps(driver Driver, region proto.ApplicationRegion) Deps {
	return Deps{
		Driver:     driver,
		LoadRegion: func() proto.ApplicationRegion { return region },
		// The legacy boolean reader exercises the boolean-to-tri-state fallback
		// for the tests that only need an enabled manifest. The tri-state and
		// sync matrix overrides LoadFeatureState explicitly.
		LoadFeatureFlag: func() bool { return true },
		// Most handler tests describe a device that already accepted the
		// privacy terms, so a flag-less start reaches the driver. The consent
		// matrix overrides these seams explicitly.
		LoadConsent:    func() bool { return true },
		PersistConsent: func() error { return nil },
		ReadBudget:     time.Second,
		ActionBudget:   time.Second,
		DisableBudget:  time.Second,
	}
}

func serve(t *testing.T, handler *Handler, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, target, nil))
	return recorder
}

// consentRecorder is a scripted pair of consent seams for the start-route
// matrix. It records reads and writes and, when events is non-nil, appends an
// ordered marker per call so a test can prove the persist-before-initialize
// order without a sleeping driver.
type consentRecorder struct {
	accepted     bool
	persistErr   error
	loadCalls    int
	persistCalls int
	events       *[]string
}

func (c *consentRecorder) load() bool {
	c.loadCalls++
	if c.events != nil {
		*c.events = append(*c.events, "load-consent")
	}
	return c.accepted
}

func (c *consentRecorder) persist() error {
	c.persistCalls++
	if c.events != nil {
		*c.events = append(*c.events, "persist-consent")
	}
	if c.persistErr != nil {
		return c.persistErr
	}
	c.accepted = true
	return nil
}

// manifestStateRecorder scripts the tri-state manifest state of the start
// matrix and the sync seam the handler may call to repair it.
type manifestStateRecorder struct {
	state       application.FeatureState
	next        application.FeatureState
	err         error
	syncCalls   int
	region      proto.ApplicationRegion
	hasDeadline bool
}

func (m *manifestStateRecorder) load() application.FeatureState { return m.state }

// sync implements the SyncFeatureManifest seam. It records the call count, the
// region and context it was handed, and on a successful call moves the state to
// next; an unset next leaves the state untouched, modelling a sync that
// published nothing.
func (m *manifestStateRecorder) sync(ctx context.Context, region proto.ApplicationRegion) error {
	m.syncCalls++
	m.region = region
	_, m.hasDeadline = ctx.Deadline()
	if m.err != nil {
		return m.err
	}
	m.state = m.next
	return nil
}

// initializeRecordingDriver decorates the scripted fake driver with an ordered
// event marker before each Initialize call. Every other Driver method is
// promoted from the embedded fake.
type initializeRecordingDriver struct {
	*fakeDriver
	events *[]string
}

func (d *initializeRecordingDriver) Initialize(ctx context.Context) error {
	*d.events = append(*d.events, "initialize")
	return d.fakeDriver.Initialize(ctx)
}

func decodeJSON[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(recorder.Body.Bytes(), &value); err != nil {
		t.Fatalf("unmarshal %q: %v", recorder.Body.String(), err)
	}
	return value
}

func assertStatus(t *testing.T, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()
	if recorder.Code != want {
		t.Fatalf("status = %d, want %d (body %s)", recorder.Code, want, recorder.Body.String())
	}
}

// assertBoundedResponse checks the documented 64 KiB ceiling and the exact
// Content-Length delimiting contract on every response, including the 503
// failure paths that also have to keep the connection usable.
func assertBoundedResponse(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	if recorder.Body.Len() > maxResponseBytes {
		t.Fatalf("body = %d bytes, want <= %d", recorder.Body.Len(), maxResponseBytes)
	}
	if got := recorder.Header().Get("Content-Length"); got != strconv.Itoa(recorder.Body.Len()) {
		t.Fatalf("Content-Length = %q, want %d", got, recorder.Body.Len())
	}
}

func waitSignal(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for driver signal")
	}
}

func waitNoSignal(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
		t.Fatal("unexpected driver signal")
	default:
	}
}

// A transient not-ready code fetch is a documented 503 path: the retained
// gate must survive it untouched so a guiding page never falls back to Start.
func TestHandlerCodeNotReadyKeepsRetainedGate(t *testing.T) {
	driver := newFakeDriver()
	driver.setReport(Report{Start: ptr(true), Match: ptr(false)})
	driver.setLoginCodeErr(ErrLoginCodeNotReady)
	handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))

	recorder := serve(t, handler, http.MethodGet, LoginCodePath)
	assertStatus(t, recorder, http.StatusServiceUnavailable)
	assertBoundedResponse(t, recorder)
	if body := decodeJSON[errorResponse](t, recorder); body.Error != errorLoginCodeUnavailable {
		t.Fatalf("error body = %q, want %q", body.Error, errorLoginCodeUnavailable)
	}

	// The retained gate still reflects the last successful snapshot: exit is
	// rejected as NOT_READY (start confirmed, match not), proving the failed
	// code fetch did not erase the gate.
	recorder = serve(t, handler, http.MethodGet, LoginCodePath+"?exit=true")
	assertStatus(t, recorder, http.StatusOK)
	if body := decodeJSON[ActionResponse](t, recorder); body.Status || body.Message != MessageNotReady {
		t.Fatalf("exit after failed code fetch = %+v, want NOT_READY", body)
	}
	if counts := driver.callCounts(); counts.deinitialize != 0 {
		t.Fatalf("deinitialize calls = %d, want 0", counts.deinitialize)
	}
}

func TestServeHTTPMethodAndPathRejection(t *testing.T) {
	cases := []struct {
		name       string
		method     string
		target     string
		wantStatus int
		wantError  string
	}{
		{"post login path", http.MethodPost, LoginCodePath, http.StatusBadRequest, errorMethodNotSupported},
		{"put login path", http.MethodPut, LoginCodePath, http.StatusBadRequest, errorMethodNotSupported},
		{"post unknown path is still a method failure", http.MethodPost, "/api/unknown", http.StatusBadRequest, errorMethodNotSupported},
		{"get unknown path", http.MethodGet, "/api/unknown", http.StatusNotFound, errorNotFound},
		{"get trailing slash", http.MethodGet, LoginCodePath + "/", http.StatusNotFound, errorNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			driver := newFakeDriver()
			handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))

			recorder := serve(t, handler, tc.method, tc.target)
			assertStatus(t, recorder, tc.wantStatus)
			assertBoundedResponse(t, recorder)
			if got := decodeJSON[errorResponse](t, recorder).Error; got != tc.wantError {
				t.Fatalf("error = %q, want %q", got, tc.wantError)
			}
			if counts := driver.callCounts(); counts != (fakeCallCounts{}) {
				t.Fatalf("driver calls = %+v, want none", counts)
			}
		})
	}
}

// TestNewHandlerLegacyFeatureFlagFallback pins the compatibility resolution for
// callers that still inject the boolean reader: true admits the feature as
// Enabled and false behaves like a Disabled manifest, and an explicit tri-state
// reader always wins when both are set.
func TestNewHandlerLegacyFeatureFlagFallback(t *testing.T) {
	t.Run("true maps to enabled", func(t *testing.T) {
		handler := NewHandler(Deps{LoadFeatureFlag: func() bool { return true }})
		if got := handler.loadFeatureState(); got != application.FeatureStateEnabled {
			t.Fatalf("loadFeatureState() = %v, want enabled", got)
		}
	})

	t.Run("false maps to disabled", func(t *testing.T) {
		handler := NewHandler(Deps{LoadFeatureFlag: func() bool { return false }})
		if got := handler.loadFeatureState(); got != application.FeatureStateDisabled {
			t.Fatalf("loadFeatureState() = %v, want disabled", got)
		}
	})

	t.Run("explicit tri-state reader wins over the legacy flag", func(t *testing.T) {
		handler := NewHandler(Deps{
			LoadFeatureState: func() application.FeatureState { return application.FeatureStateEnabled },
			LoadFeatureFlag:  func() bool { return false },
		})
		if got := handler.loadFeatureState(); got != application.FeatureStateEnabled {
			t.Fatalf("loadFeatureState() = %v, want the explicit tri-state reader", got)
		}
	})
}

func TestServeHTTPIgnoresRegionWhenManifestEnabled(t *testing.T) {
	cases := []struct {
		name   string
		region proto.ApplicationRegion
	}{
		{"cn", proto.ApplicationRegionCN},
		{"global", proto.ApplicationRegionGlobal},
		{"empty", proto.ApplicationRegion("")},
		{"uppercase cn", proto.ApplicationRegion("CN")},
		{"other", proto.ApplicationRegion("us")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			driver := newFakeDriver()
			handler := NewHandler(testDeps(driver, tc.region))

			recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true")
			assertStatus(t, recorder, http.StatusOK)
			if counts := driver.callCounts(); counts.snapshot != 1 {
				t.Fatalf("driver calls = %+v, want one snapshot", counts)
			}
		})
	}
}

func TestServeHTTPFeatureManifestGate(t *testing.T) {
	// A missing or disabled manifest closes every read lane with 404 but still
	// lets the start action through: a missing manifest may be repaired by the
	// consent-gated sync, and a disabled one answers the stable DISABLED
	// message, because a 404 there would be indistinguishable from a transport
	// failure on the client and could not render the policy layout. On the read
	// lanes the driver, the sync, and the consent seams must all stay untouched.
	nonStartTargets := []string{
		LoginCodePath,
		LoginCodePath + "?exit=true",
		LoginCodePath + "?disable=true",
		LoginCodePath + "?heartbeat=true",
	}
	closedStates := []struct {
		name  string
		state application.FeatureState
	}{
		{"missing", application.FeatureStateMissing},
		{"disabled", application.FeatureStateDisabled},
	}

	for _, tc := range closedStates {
		for _, target := range nonStartTargets {
			t.Run(tc.name+" "+target, func(t *testing.T) {
				driver := newFakeDriver()
				syncCalls := 0
				consent := &consentRecorder{}
				deps := testDeps(driver, proto.ApplicationRegionCN)
				deps.LoadFeatureState = func() application.FeatureState { return tc.state }
				deps.SyncFeatureManifest = func(context.Context, proto.ApplicationRegion) error {
					syncCalls++
					return nil
				}
				deps.LoadConsent = consent.load
				deps.PersistConsent = consent.persist
				handler := NewHandler(deps)

				recorder := serve(t, handler, http.MethodGet, target)
				assertStatus(t, recorder, http.StatusNotFound)
				assertBoundedResponse(t, recorder)
				if got := decodeJSON[errorResponse](t, recorder).Error; got != errorNotFound {
					t.Fatalf("error = %q, want %q", got, errorNotFound)
				}
				if counts := driver.callCounts(); counts != (fakeCallCounts{}) {
					t.Fatalf("driver calls = %+v, want none", counts)
				}
				if syncCalls != 0 {
					t.Fatalf("sync calls = %d, want 0 on a non-start route", syncCalls)
				}
				if consent.loadCalls != 0 || consent.persistCalls != 0 {
					t.Fatalf("consent calls = load %d persist %d, want none", consent.loadCalls, consent.persistCalls)
				}
			})
		}
	}

	t.Run("start answers DISABLED without consent, sync, or driver work", func(t *testing.T) {
		driver := newFakeDriver()
		syncCalls := 0
		consent := &consentRecorder{}
		deps := testDeps(driver, proto.ApplicationRegionCN)
		deps.LoadFeatureState = func() application.FeatureState { return application.FeatureStateDisabled }
		deps.SyncFeatureManifest = func(context.Context, proto.ApplicationRegion) error {
			syncCalls++
			return nil
		}
		deps.LoadConsent = consent.load
		deps.PersistConsent = consent.persist
		handler := NewHandler(deps)

		recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?start=true&consent=true")
		assertStatus(t, recorder, http.StatusOK)
		assertBoundedResponse(t, recorder)
		want := ActionResponse{Status: false, Message: MessageDisabled}
		if got := decodeJSON[ActionResponse](t, recorder); got != want {
			t.Fatalf("action = %+v, want %+v", got, want)
		}
		if consent.loadCalls != 0 || consent.persistCalls != 0 {
			t.Fatalf("consent calls = load %d persist %d, want none for a disabled manifest", consent.loadCalls, consent.persistCalls)
		}
		if syncCalls != 0 {
			t.Fatalf("sync calls = %d, want 0 for a disabled manifest", syncCalls)
		}
		if counts := driver.callCounts(); counts != (fakeCallCounts{}) {
			t.Fatalf("driver calls = %+v, want none", counts)
		}
	})
}

func TestServeHTTPFeatureManifestGateRejectsBothRegionsIdentically(t *testing.T) {
	driver := newFakeDriver()
	global := testDeps(driver, proto.ApplicationRegionGlobal)
	global.LoadFeatureFlag = func() bool { return false }
	globallyRejected := serve(t, NewHandler(global), http.MethodGet, LoginCodePath+"?heartbeat=true")

	cnDisabled := testDeps(driver, proto.ApplicationRegionCN)
	cnDisabled.LoadFeatureFlag = func() bool { return false }
	manifestRejected := serve(t, NewHandler(cnDisabled), http.MethodGet, LoginCodePath+"?heartbeat=true")

	if globallyRejected.Code != manifestRejected.Code ||
		globallyRejected.Body.String() != manifestRejected.Body.String() {
		t.Fatalf("region rejection %d %q differs from manifest rejection %d %q",
			globallyRejected.Code, globallyRejected.Body.String(),
			manifestRejected.Code, manifestRejected.Body.String())
	}
	if counts := driver.callCounts(); counts != (fakeCallCounts{}) {
		t.Fatalf("driver calls = %+v, want none", counts)
	}
}

func TestHandlerHeartbeatSnapshot(t *testing.T) {
	cases := []struct {
		name   string
		report Report
		want   snapshotResponse
	}{
		{
			name: "full online report",
			report: Report{
				Status: ptr("online"), Start: ptr(true), Match: ptr(true),
				Name: ptr("MockKVM"), IP: ptr("100.64.0.1"),
			},
			want: snapshotResponse{Status: StatusOnline, Start: true, Match: true, Name: "MockKVM", IP: "100.64.0.1"},
		},
		{
			name:   "empty report resolves every field to its default",
			report: Report{},
			want:   snapshotResponse{Status: StatusOffline},
		},
		{
			name:   "only status keeps the other defaults",
			report: Report{Status: ptr("online")},
			want:   snapshotResponse{Status: StatusOnline},
		},
		{
			name:   "status must be exactly online",
			report: Report{Status: ptr("ONLINE ")},
			want:   snapshotResponse{Status: StatusOffline},
		},
		{
			name: "offline hides name and ip",
			report: Report{
				Status: ptr("offline"), Start: ptr(true), Match: ptr(true),
				Name: ptr("MockKVM"), IP: ptr("100.64.0.1"),
			},
			want: snapshotResponse{Status: StatusOffline, Start: true, Match: true},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			driver := newFakeDriver()
			driver.setReport(tc.report)
			handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))

			recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true")
			assertStatus(t, recorder, http.StatusOK)
			assertBoundedResponse(t, recorder)
			if got := decodeJSON[snapshotResponse](t, recorder); got != tc.want {
				t.Fatalf("snapshot = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestHandlerHeartbeatRefreshFailureRetainsGate(t *testing.T) {
	driver := newFakeDriver()
	driver.setReport(Report{
		Status: ptr("online"), Start: ptr(true), Match: ptr(true),
		Name: ptr("MockKVM"), IP: ptr("100.64.0.1"),
	})
	handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))

	recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true")
	assertStatus(t, recorder, http.StatusOK)

	driver.setSnapshotErr(errors.New("snapshot exploded"))
	recorder = serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true")
	assertStatus(t, recorder, http.StatusServiceUnavailable)
	if got := decodeJSON[errorResponse](t, recorder).Error; got != errorStateUnavailable {
		t.Fatalf("error = %q, want %q", got, errorStateUnavailable)
	}
	if strings.Contains(recorder.Body.String(), "exploded") {
		t.Fatalf("driver error text leaked into the response body")
	}

	// A failed refresh keeps the retained gate and never fabricates a snapshot.
	// The retained ready gate therefore still lets an exit reach the driver.
	recorder = serve(t, handler, http.MethodGet, LoginCodePath+"?exit=true")
	assertStatus(t, recorder, http.StatusOK)
	if got := decodeJSON[ActionResponse](t, recorder); got != SuccessAction() {
		t.Fatalf("action = %+v, want success", got)
	}
	if counts := driver.callCounts(); counts.deinitialize != 1 {
		t.Fatalf("deinitialize calls = %d, want 1", counts.deinitialize)
	}

	// Once the component recovers, the next successful poll must be able to
	// replace the retained gate instead of freezing the recovered value.
	driver.setSnapshotErr(nil)
	driver.setReport(Report{Start: ptr(false), Match: ptr(false)})
	recorder = serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true")
	assertStatus(t, recorder, http.StatusOK)
	if got := decodeJSON[snapshotResponse](t, recorder); got.Start || got.Match {
		t.Fatalf("snapshot = %+v, want the recovered all-false gate", got)
	}

	recorder = serve(t, handler, http.MethodGet, LoginCodePath+"?exit=true")
	assertStatus(t, recorder, http.StatusOK)
	if got := decodeJSON[ActionResponse](t, recorder); got != (ActionResponse{Message: MessageNotReady}) {
		t.Fatalf("action = %+v, want NOT_READY from the replaced gate", got)
	}
}

func TestHandlerStartIdempotentWhenConfirmed(t *testing.T) {
	driver := newFakeDriver()
	driver.setReport(Report{Start: ptr(true), Match: ptr(true)})
	handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))

	recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true")
	assertStatus(t, recorder, http.StatusOK)

	recorder = serve(t, handler, http.MethodGet, LoginCodePath+"?start=true")
	assertStatus(t, recorder, http.StatusOK)
	if got := decodeJSON[ActionResponse](t, recorder); got != SuccessAction() {
		t.Fatalf("action = %+v, want success", got)
	}
	if counts := driver.callCounts(); counts.initialize != 0 {
		t.Fatalf("initialize calls = %d, want 0 for an idempotent start", counts.initialize)
	}
}

func TestHandlerStartSuccessReconfirmsSnapshot(t *testing.T) {
	driver := newFakeDriver()
	// The driver already reports start=true, but the gate is not confirmed
	// until the first refresh, so the action still runs.
	driver.setReport(Report{Start: ptr(true)})
	handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))
	before := driver.callCounts()

	recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?start=true")
	assertStatus(t, recorder, http.StatusOK)
	assertBoundedResponse(t, recorder)
	if got := decodeJSON[ActionResponse](t, recorder); got != SuccessAction() {
		t.Fatalf("action = %+v, want success", got)
	}

	after := driver.callCounts()
	if after.initialize != 1 {
		t.Fatalf("initialize calls = %d, want 1", after.initialize)
	}
	if after.snapshot != before.snapshot+1 {
		t.Fatalf("snapshot calls = %d, want %d (one re-confirm)", after.snapshot, before.snapshot+1)
	}

	recorder = serve(t, handler, http.MethodGet, LoginCodePath+"?start=true")
	assertStatus(t, recorder, http.StatusOK)
	if got := decodeJSON[ActionResponse](t, recorder); got != SuccessAction() {
		t.Fatalf("second action = %+v, want success", got)
	}
	if counts := driver.callCounts(); counts.initialize != 1 {
		t.Fatalf("initialize calls = %d, want 1 after the re-confirmed gate", counts.initialize)
	}
}

func TestHandlerStartFailureMessage(t *testing.T) {
	cases := []struct {
		name    string
		message string
		want    string
	}{
		{"stable not implemented", "NOTIMPL", "NOTIMPL"},
		{"daemon missing", "NO_DAEMON", "NO_DAEMON"},
		{"long message is truncated", "LONG_MESSAGE_HERE", "LONG_MESSA"},
		{"non printable bytes are dropped", "NOT\x00IMPL", "NOTIMPL"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			driver := newFakeDriver()
			driver.setReport(Report{Start: ptr(true)})
			driver.setInitializeErr(testMessageError(tc.message))
			handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))

			recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?start=true")
			assertStatus(t, recorder, http.StatusOK)
			want := ActionResponse{Status: false, Message: tc.want}
			if got := decodeJSON[ActionResponse](t, recorder); got != want {
				t.Fatalf("action = %+v, want %+v", got, want)
			}
		})
	}
}

func TestHandlerStartPlainFailureIsFail(t *testing.T) {
	driver := newFakeDriver()
	driver.setReport(Report{Start: ptr(true)})
	driver.setInitializeErr(errors.New("boom"))
	handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))

	recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?start=true")
	assertStatus(t, recorder, http.StatusOK)
	want := ActionResponse{Status: false, Message: MessageFailure}
	if got := decodeJSON[ActionResponse](t, recorder); got != want {
		t.Fatalf("action = %+v, want %+v", got, want)
	}
	if strings.Contains(recorder.Body.String(), "boom") {
		t.Fatalf("driver error text leaked into the response body")
	}
}

func TestHandlerStartTimeout(t *testing.T) {
	driver := newFakeDriver()
	driver.setReport(Report{Start: ptr(true)})
	driver.setInitializeBlock(blockDeadline)

	deps := testDeps(driver, proto.ApplicationRegionCN)
	deps.ActionBudget = 25 * time.Millisecond
	handler := NewHandler(deps)

	recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?start=true")
	assertStatus(t, recorder, http.StatusOK)
	want := ActionResponse{Status: false, Message: MessageTimeout}
	if got := decodeJSON[ActionResponse](t, recorder); got != want {
		t.Fatalf("action = %+v, want %+v", got, want)
	}
}

func TestHandlerStartWithUnimplementedDriver(t *testing.T) {
	handler := NewHandler(testDeps(NewUnimplementedDriver(), proto.ApplicationRegionCN))

	recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?start=true")
	assertStatus(t, recorder, http.StatusOK)
	want := ActionResponse{Status: false, Message: MessageNotImplemented}
	if got := decodeJSON[ActionResponse](t, recorder); got != want {
		t.Fatalf("action = %+v, want %+v", got, want)
	}
}

// TestHandlerStartConsentChannel pins the consent contract of the start route:
// the explicit flag persists consent before any upstream contact, a persisted
// agreement lets a flag-less start proceed, and an un-agreed device answers the
// stable CONSENT message instead of a generic failure.
func TestHandlerStartConsentChannel(t *testing.T) {
	t.Run("fresh device without a flag answers CONSENT", func(t *testing.T) {
		driver := newFakeDriver()
		driver.setReport(Report{Start: ptr(false)})
		consent := &consentRecorder{}
		deps := testDeps(driver, proto.ApplicationRegionCN)
		deps.LoadConsent = consent.load
		deps.PersistConsent = consent.persist
		handler := NewHandler(deps)

		recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?start=true")
		assertStatus(t, recorder, http.StatusOK)
		assertBoundedResponse(t, recorder)
		want := ActionResponse{Status: false, Message: MessageConsent}
		if got := decodeJSON[ActionResponse](t, recorder); got != want {
			t.Fatalf("action = %+v, want %+v", got, want)
		}
		if consent.persistCalls != 0 {
			t.Fatalf("persist calls = %d, want 0", consent.persistCalls)
		}
		if counts := driver.callCounts(); counts != (fakeCallCounts{}) {
			t.Fatalf("driver calls = %+v, want none before consent", counts)
		}
	})

	t.Run("consent flag persists before initialize", func(t *testing.T) {
		fake := newFakeDriver()
		fake.setReport(Report{Start: ptr(false)})
		events := []string{}
		driver := &initializeRecordingDriver{fakeDriver: fake, events: &events}
		consent := &consentRecorder{events: &events}
		deps := testDeps(driver, proto.ApplicationRegionCN)
		deps.LoadConsent = consent.load
		deps.PersistConsent = consent.persist
		handler := NewHandler(deps)

		recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?start=true&consent=true")
		assertStatus(t, recorder, http.StatusOK)
		if got := decodeJSON[ActionResponse](t, recorder); got != SuccessAction() {
			t.Fatalf("action = %+v, want success", got)
		}
		if consent.persistCalls != 1 {
			t.Fatalf("persist calls = %d, want 1", consent.persistCalls)
		}
		if counts := fake.callCounts(); counts.initialize != 1 {
			t.Fatalf("initialize calls = %d, want 1", counts.initialize)
		}
		if got := strings.Join(events, ","); got != "persist-consent,initialize" {
			t.Fatalf("ordered events = %q, want persist before initialize", got)
		}
	})

	t.Run("persisted consent without a flag proceeds", func(t *testing.T) {
		fake := newFakeDriver()
		fake.setReport(Report{Start: ptr(false)})
		events := []string{}
		driver := &initializeRecordingDriver{fakeDriver: fake, events: &events}
		consent := &consentRecorder{accepted: true, events: &events}
		deps := testDeps(driver, proto.ApplicationRegionCN)
		deps.LoadConsent = consent.load
		deps.PersistConsent = consent.persist
		handler := NewHandler(deps)

		recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?start=true")
		assertStatus(t, recorder, http.StatusOK)
		if got := decodeJSON[ActionResponse](t, recorder); got != SuccessAction() {
			t.Fatalf("action = %+v, want success", got)
		}
		if consent.persistCalls != 0 {
			t.Fatalf("persist calls = %d, want 0 for saved consent", consent.persistCalls)
		}
		if counts := fake.callCounts(); counts.initialize != 1 {
			t.Fatalf("initialize calls = %d, want 1", counts.initialize)
		}
		if got := strings.Join(events, ","); got != "load-consent,initialize" {
			t.Fatalf("ordered events = %q, want load before initialize", got)
		}
	})

	t.Run("re-sending the flag is an idempotent re-persist", func(t *testing.T) {
		driver := newFakeDriver()
		driver.setReport(Report{Start: ptr(false)})
		consent := &consentRecorder{}
		deps := testDeps(driver, proto.ApplicationRegionCN)
		deps.LoadConsent = consent.load
		deps.PersistConsent = consent.persist
		handler := NewHandler(deps)

		for attempt := 1; attempt <= 2; attempt++ {
			recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?start=true&consent=true")
			assertStatus(t, recorder, http.StatusOK)
			if got := decodeJSON[ActionResponse](t, recorder); got != SuccessAction() {
				t.Fatalf("attempt %d action = %+v, want success", attempt, got)
			}
		}
		if consent.persistCalls != 2 {
			t.Fatalf("persist calls = %d, want 2 (one re-persist per flag)", consent.persistCalls)
		}
		if counts := driver.callCounts(); counts.initialize != 2 {
			t.Fatalf("initialize calls = %d, want 2", counts.initialize)
		}
	})

	t.Run("persist failure answers FAIL and skips the driver", func(t *testing.T) {
		driver := newFakeDriver()
		driver.setReport(Report{Start: ptr(false)})
		consent := &consentRecorder{persistErr: errors.New("consent store exploded")}
		deps := testDeps(driver, proto.ApplicationRegionCN)
		deps.LoadConsent = consent.load
		deps.PersistConsent = consent.persist
		handler := NewHandler(deps)

		recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?start=true&consent=true")
		assertStatus(t, recorder, http.StatusOK)
		want := ActionResponse{Status: false, Message: MessageFailure}
		if got := decodeJSON[ActionResponse](t, recorder); got != want {
			t.Fatalf("action = %+v, want %+v", got, want)
		}
		if consent.persistCalls != 1 {
			t.Fatalf("persist calls = %d, want 1", consent.persistCalls)
		}
		if counts := driver.callCounts(); counts != (fakeCallCounts{}) {
			t.Fatalf("driver calls = %+v, want none after a persist failure", counts)
		}
	})

	t.Run("consent flag is ignored on non-start routes", func(t *testing.T) {
		driver := newFakeDriver()
		driver.setReport(Report{Start: ptr(true)})
		consent := &consentRecorder{}
		deps := testDeps(driver, proto.ApplicationRegionCN)
		deps.LoadConsent = consent.load
		deps.PersistConsent = consent.persist
		handler := NewHandler(deps)

		recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true&consent=true")
		assertStatus(t, recorder, http.StatusOK)
		if consent.loadCalls != 0 || consent.persistCalls != 0 {
			t.Fatalf("consent calls = load %d persist %d, want none on a heartbeat", consent.loadCalls, consent.persistCalls)
		}
		if counts := driver.callCounts(); counts.snapshot != 1 || counts.initialize != 0 {
			t.Fatalf("driver calls = %+v, want one snapshot and no initialize", counts)
		}
	})
}

// TestHandlerStartManifestSyncMatrix pins the tri-state manifest contract of
// the start action: only a missing manifest triggers the single bounded sync,
// and only after consent is resolved; the sync result decides between
// initialize, DISABLED, and FAIL; and a confirmed gate short-circuits before
// any manifest, consent, or driver work.
func TestHandlerStartManifestSyncMatrix(t *testing.T) {
	t.Run("missing then enabled syncs once and initializes", func(t *testing.T) {
		driver := newFakeDriver()
		driver.setReport(Report{Start: ptr(false)})
		manifest := &manifestStateRecorder{
			state: application.FeatureStateMissing,
			next:  application.FeatureStateEnabled,
		}
		consent := &consentRecorder{}
		deps := testDeps(driver, proto.ApplicationRegionCN)
		deps.LoadFeatureState = manifest.load
		deps.SyncFeatureManifest = manifest.sync
		deps.LoadConsent = consent.load
		deps.PersistConsent = consent.persist
		handler := NewHandler(deps)

		recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?start=true&consent=true")
		assertStatus(t, recorder, http.StatusOK)
		if got := decodeJSON[ActionResponse](t, recorder); got != SuccessAction() {
			t.Fatalf("action = %+v, want success", got)
		}
		if manifest.syncCalls != 1 {
			t.Fatalf("sync calls = %d, want exactly one", manifest.syncCalls)
		}
		if manifest.region != proto.ApplicationRegionCN {
			t.Fatalf("sync region = %q, want %q", manifest.region, proto.ApplicationRegionCN)
		}
		if !manifest.hasDeadline {
			t.Fatal("sync ran without the action-context deadline")
		}
		if counts := driver.callCounts(); counts.initialize != 1 {
			t.Fatalf("initialize calls = %d, want 1 after the sync enabled the manifest", counts.initialize)
		}
	})

	t.Run("sync to disabled answers DISABLED without initializing", func(t *testing.T) {
		driver := newFakeDriver()
		driver.setReport(Report{Start: ptr(false)})
		manifest := &manifestStateRecorder{
			state: application.FeatureStateMissing,
			next:  application.FeatureStateDisabled,
		}
		deps := testDeps(driver, proto.ApplicationRegionCN)
		deps.LoadFeatureState = manifest.load
		deps.SyncFeatureManifest = manifest.sync
		handler := NewHandler(deps)

		recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?start=true")
		assertStatus(t, recorder, http.StatusOK)
		want := ActionResponse{Status: false, Message: MessageDisabled}
		if got := decodeJSON[ActionResponse](t, recorder); got != want {
			t.Fatalf("action = %+v, want %+v", got, want)
		}
		if manifest.syncCalls != 1 {
			t.Fatalf("sync calls = %d, want exactly one", manifest.syncCalls)
		}
		if counts := driver.callCounts(); counts != (fakeCallCounts{}) {
			t.Fatalf("driver calls = %+v, want none after the sync disabled the manifest", counts)
		}
	})

	t.Run("sync error answers FAIL without initializing", func(t *testing.T) {
		driver := newFakeDriver()
		driver.setReport(Report{Start: ptr(false)})
		manifest := &manifestStateRecorder{
			state: application.FeatureStateMissing,
			err:   errors.New("cdn unreachable"),
		}
		deps := testDeps(driver, proto.ApplicationRegionCN)
		deps.LoadFeatureState = manifest.load
		deps.SyncFeatureManifest = manifest.sync
		handler := NewHandler(deps)

		recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?start=true")
		assertStatus(t, recorder, http.StatusOK)
		want := ActionResponse{Status: false, Message: MessageFailure}
		if got := decodeJSON[ActionResponse](t, recorder); got != want {
			t.Fatalf("action = %+v, want %+v", got, want)
		}
		if manifest.syncCalls != 1 {
			t.Fatalf("sync calls = %d, want exactly one", manifest.syncCalls)
		}
		if counts := driver.callCounts(); counts != (fakeCallCounts{}) {
			t.Fatalf("driver calls = %+v, want none after a sync failure", counts)
		}
	})

	t.Run("still missing after a successful sync answers FAIL", func(t *testing.T) {
		driver := newFakeDriver()
		driver.setReport(Report{Start: ptr(false)})
		// next stays Missing: the sync reports success but publishes nothing.
		manifest := &manifestStateRecorder{state: application.FeatureStateMissing}
		deps := testDeps(driver, proto.ApplicationRegionCN)
		deps.LoadFeatureState = manifest.load
		deps.SyncFeatureManifest = manifest.sync
		handler := NewHandler(deps)

		recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?start=true")
		assertStatus(t, recorder, http.StatusOK)
		want := ActionResponse{Status: false, Message: MessageFailure}
		if got := decodeJSON[ActionResponse](t, recorder); got != want {
			t.Fatalf("action = %+v, want %+v", got, want)
		}
		if manifest.syncCalls != 1 {
			t.Fatalf("sync calls = %d, want exactly one (no retry loop)", manifest.syncCalls)
		}
		if counts := driver.callCounts(); counts != (fakeCallCounts{}) {
			t.Fatalf("driver calls = %+v, want none while the manifest is still missing", counts)
		}
	})

	t.Run("nil sync seam answers FAIL without any sync", func(t *testing.T) {
		driver := newFakeDriver()
		driver.setReport(Report{Start: ptr(false)})
		manifest := &manifestStateRecorder{
			state: application.FeatureStateMissing,
			next:  application.FeatureStateEnabled,
		}
		deps := testDeps(driver, proto.ApplicationRegionCN)
		deps.LoadFeatureState = manifest.load
		// SyncFeatureManifest is deliberately left nil (not wired).
		handler := NewHandler(deps)

		recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?start=true")
		assertStatus(t, recorder, http.StatusOK)
		want := ActionResponse{Status: false, Message: MessageFailure}
		if got := decodeJSON[ActionResponse](t, recorder); got != want {
			t.Fatalf("action = %+v, want %+v", got, want)
		}
		if manifest.syncCalls != 0 {
			t.Fatalf("sync calls = %d, want 0 for a nil seam", manifest.syncCalls)
		}
		if counts := driver.callCounts(); counts != (fakeCallCounts{}) {
			t.Fatalf("driver calls = %+v, want none for a nil seam", counts)
		}
	})

	t.Run("missing without consent issues no fetch", func(t *testing.T) {
		driver := newFakeDriver()
		driver.setReport(Report{Start: ptr(false)})
		manifest := &manifestStateRecorder{
			state: application.FeatureStateMissing,
			next:  application.FeatureStateEnabled,
		}
		consent := &consentRecorder{}
		deps := testDeps(driver, proto.ApplicationRegionCN)
		deps.LoadFeatureState = manifest.load
		deps.SyncFeatureManifest = manifest.sync
		deps.LoadConsent = consent.load
		deps.PersistConsent = consent.persist
		handler := NewHandler(deps)

		recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?start=true")
		assertStatus(t, recorder, http.StatusOK)
		want := ActionResponse{Status: false, Message: MessageConsent}
		if got := decodeJSON[ActionResponse](t, recorder); got != want {
			t.Fatalf("action = %+v, want %+v", got, want)
		}
		if manifest.syncCalls != 0 {
			t.Fatalf("sync calls = %d, want 0 before consent", manifest.syncCalls)
		}
		if manifest.state != application.FeatureStateMissing {
			t.Fatalf("manifest state = %v, want it untouched without consent", manifest.state)
		}
		if counts := driver.callCounts(); counts != (fakeCallCounts{}) {
			t.Fatalf("driver calls = %+v, want none without consent", counts)
		}
	})

	t.Run("confirmed gate short-circuits before manifest, consent, and driver work", func(t *testing.T) {
		driver := newFakeDriver()
		driver.setReport(Report{Start: ptr(true), Match: ptr(true)})
		manifest := &manifestStateRecorder{
			state: application.FeatureStateEnabled,
			next:  application.FeatureStateEnabled,
		}
		consent := &consentRecorder{accepted: true}
		deps := testDeps(driver, proto.ApplicationRegionCN)
		deps.LoadFeatureState = manifest.load
		deps.SyncFeatureManifest = manifest.sync
		deps.LoadConsent = consent.load
		deps.PersistConsent = consent.persist
		handler := NewHandler(deps)

		recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true")
		assertStatus(t, recorder, http.StatusOK)

		// Make both inputs hostile: the confirmed gate must still short-circuit
		// without re-deriving either one.
		manifest.state = application.FeatureStateMissing
		consent.accepted = false
		recorder = serve(t, handler, http.MethodGet, LoginCodePath+"?start=true&consent=true")
		assertStatus(t, recorder, http.StatusOK)
		if got := decodeJSON[ActionResponse](t, recorder); got != SuccessAction() {
			t.Fatalf("action = %+v, want the idempotent success", got)
		}
		if manifest.syncCalls != 0 {
			t.Fatalf("sync calls = %d, want 0 for a confirmed gate", manifest.syncCalls)
		}
		if consent.loadCalls != 0 || consent.persistCalls != 0 {
			t.Fatalf("consent calls = load %d persist %d, want none for a confirmed gate", consent.loadCalls, consent.persistCalls)
		}
		if counts := driver.callCounts(); counts.initialize != 0 {
			t.Fatalf("initialize calls = %d, want 0 for a confirmed gate", counts.initialize)
		}
	})
}

func TestHandlerExitNotReady(t *testing.T) {
	cases := []struct {
		name   string
		report Report
		prime  bool
	}{
		{
			name:   "no confirmed snapshot",
			report: Report{Status: ptr("online"), Start: ptr(true), Match: ptr(true)},
		},
		{
			name:   "start confirmed but match missing",
			report: Report{Start: ptr(true), Match: ptr(false)},
			prime:  true,
		},
		{
			name:   "match confirmed but start missing",
			report: Report{Start: ptr(false), Match: ptr(true)},
			prime:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			driver := newFakeDriver()
			driver.setReport(tc.report)
			handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))
			if tc.prime {
				recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true")
				assertStatus(t, recorder, http.StatusOK)
			}

			recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?exit=true")
			assertStatus(t, recorder, http.StatusOK)
			want := ActionResponse{Status: false, Message: MessageNotReady}
			if got := decodeJSON[ActionResponse](t, recorder); got != want {
				t.Fatalf("action = %+v, want %+v", got, want)
			}
			if counts := driver.callCounts(); counts.deinitialize != 0 {
				t.Fatalf("deinitialize calls = %d, want 0 for a not-ready exit", counts.deinitialize)
			}
		})
	}
}

func TestHandlerExitReady(t *testing.T) {
	driver := newFakeDriver()
	driver.setReport(Report{Start: ptr(true), Match: ptr(true)})
	handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))

	recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true")
	assertStatus(t, recorder, http.StatusOK)

	recorder = serve(t, handler, http.MethodGet, LoginCodePath+"?exit=true")
	assertStatus(t, recorder, http.StatusOK)
	if got := decodeJSON[ActionResponse](t, recorder); got != SuccessAction() {
		t.Fatalf("exit action = %+v, want success", got)
	}
	if counts := driver.callCounts(); counts.deinitialize != 1 {
		t.Fatalf("deinitialize calls = %d, want 1", counts.deinitialize)
	}
}

func TestHandlerExitFailure(t *testing.T) {
	driver := newFakeDriver()
	driver.setReport(Report{Start: ptr(true), Match: ptr(true)})
	handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))

	recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true")
	assertStatus(t, recorder, http.StatusOK)

	driver.setDeinitializeErr(testMessageError("NO_DAEMON"))
	recorder = serve(t, handler, http.MethodGet, LoginCodePath+"?exit=true")
	assertStatus(t, recorder, http.StatusOK)
	want := ActionResponse{Status: false, Message: "NO_DAEMON"}
	if got := decodeJSON[ActionResponse](t, recorder); got != want {
		t.Fatalf("action = %+v, want %+v", got, want)
	}
}

func TestHandlerActionBusy(t *testing.T) {
	driver := newFakeDriver()
	driver.setReport(Report{Start: ptr(true)})
	driver.setInitializeBlock(blockRelease)
	handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		done <- serve(t, handler, http.MethodGet, LoginCodePath+"?start=true")
	}()
	waitSignal(t, driver.started)

	recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?start=true")
	assertStatus(t, recorder, http.StatusOK)
	want := ActionResponse{Status: false, Message: MessageBusy}
	if got := decodeJSON[ActionResponse](t, recorder); got != want {
		t.Fatalf("second action = %+v, want %+v", got, want)
	}
	if counts := driver.callCounts(); counts.initialize != 1 {
		t.Fatalf("initialize calls = %d, want 1 (BUSY must not call the driver)", counts.initialize)
	}

	close(driver.release)
	select {
	case first := <-done:
		assertStatus(t, first, http.StatusOK)
		if got := decodeJSON[ActionResponse](t, first); got != SuccessAction() {
			t.Fatalf("first action = %+v, want success", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the first action to finish")
	}
}

func TestHandlerExitActionBusy(t *testing.T) {
	driver := newFakeDriver()
	driver.setReport(Report{Start: ptr(true), Match: ptr(true)})
	driver.setDeinitializeBlock(blockRelease)
	handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))

	// Confirm the ready gate before the first action so the exit passes the
	// readiness check.
	recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true")
	assertStatus(t, recorder, http.StatusOK)

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		done <- serve(t, handler, http.MethodGet, LoginCodePath+"?exit=true")
	}()
	waitSignal(t, driver.started)

	recorder = serve(t, handler, http.MethodGet, LoginCodePath+"?exit=true")
	assertStatus(t, recorder, http.StatusOK)
	if got := decodeJSON[ActionResponse](t, recorder); got != (ActionResponse{Message: MessageBusy}) {
		t.Fatalf("second exit = %+v, want BUSY", got)
	}
	if counts := driver.callCounts(); counts.deinitialize != 1 {
		t.Fatalf("deinitialize calls = %d, want 1 (BUSY must not call the driver)", counts.deinitialize)
	}

	close(driver.release)
	select {
	case first := <-done:
		assertStatus(t, first, http.StatusOK)
		if got := decodeJSON[ActionResponse](t, first); got != SuccessAction() {
			t.Fatalf("first exit = %+v, want success", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the first exit to finish")
	}
}
func TestHandlerLoginCodePlain(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"six digit code", "123456", "123456"},
		{"non printable bytes are dropped", "12\x003456", "123456"},
		{"high bytes are dropped", "abc\xffdef", "abcdef"},
		{"empty stays empty", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			driver := newFakeDriver()
			driver.setReport(Report{Start: ptr(true), Match: ptr(false)})
			driver.setLoginCode(tc.raw)
			handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))

			recorder := serve(t, handler, http.MethodGet, LoginCodePath)
			assertStatus(t, recorder, http.StatusOK)
			assertBoundedResponse(t, recorder)
			if got := decodeJSON[codeResponse](t, recorder).Code; got != tc.want {
				t.Fatalf("code = %q, want %q", got, tc.want)
			}
			if counts := driver.callCounts(); counts.loginCode != 1 {
				t.Fatalf("loginCode calls = %d, want 1", counts.loginCode)
			}
		})
	}
}

func TestHandlerLoginCodeDirectiveUsesRawReport(t *testing.T) {
	cases := []struct {
		name   string
		report Report
		want   string
	}{
		{
			name: "directive uses the raw peer values",
			report: Report{
				Status: ptr("online"), Start: ptr(true), Match: ptr(true),
				Name: ptr("MockKVM"), IP: ptr("100.64.0.1"),
			},
			want: "%online%MockKVM%100.64.0.1%",
		},
		{
			name: "non printable bytes are filtered before the directive",
			report: Report{
				Status: ptr("online"), Start: ptr(true), Match: ptr(true),
				Name: ptr("\x7fMock\x00KVM"), IP: ptr(" 100.64.0.1 "),
			},
			want: "%online%MockKVM% 100.64.0.1 %",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			driver := newFakeDriver()
			driver.setReport(tc.report)
			handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))

			recorder := serve(t, handler, http.MethodGet, LoginCodePath)
			assertStatus(t, recorder, http.StatusOK)
			if got := decodeJSON[codeResponse](t, recorder).Code; got != tc.want {
				t.Fatalf("code = %q, want %q", got, tc.want)
			}
			if counts := driver.callCounts(); counts.loginCode != 0 {
				t.Fatalf("loginCode calls = %d, want 0 for a directive", counts.loginCode)
			}
		})
	}
}

func TestHandlerLoginCodeDirectiveUnavailable(t *testing.T) {
	cases := []struct {
		name   string
		report Report
	}{
		{"missing name", Report{Start: ptr(true), Match: ptr(true), IP: ptr("100.64.0.1")}},
		{"empty name", Report{Start: ptr(true), Match: ptr(true), Name: ptr(""), IP: ptr("100.64.0.1")}},
		{"missing ip", Report{Start: ptr(true), Match: ptr(true), Name: ptr("MockKVM")}},
		{"empty ip", Report{Start: ptr(true), Match: ptr(true), Name: ptr("MockKVM"), IP: ptr("")}},
		{"percent in name", Report{Start: ptr(true), Match: ptr(true), Name: ptr("Mo%ck"), IP: ptr("100.64.0.1")}},
		{"percent in ip", Report{Start: ptr(true), Match: ptr(true), Name: ptr("MockKVM"), IP: ptr("100.%64.0.1")}},
		{"name is not printable", Report{Start: ptr(true), Match: ptr(true), Name: ptr("\x01\x02"), IP: ptr("100.64.0.1")}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			driver := newFakeDriver()
			driver.setReport(tc.report)
			handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))

			recorder := serve(t, handler, http.MethodGet, LoginCodePath)
			assertStatus(t, recorder, http.StatusServiceUnavailable)
			if got := decodeJSON[errorResponse](t, recorder).Error; got != errorPeerIdentityUnavailable {
				t.Fatalf("error = %q, want %q", got, errorPeerIdentityUnavailable)
			}
			if counts := driver.callCounts(); counts.loginCode != 0 {
				t.Fatalf("loginCode calls = %d, want 0 when the directive is invalid", counts.loginCode)
			}
		})
	}
}

func TestHandlerLoginCodeStartFalse(t *testing.T) {
	driver := newFakeDriver()
	driver.setReport(Report{})
	handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))

	recorder := serve(t, handler, http.MethodGet, LoginCodePath)
	assertStatus(t, recorder, http.StatusOK)
	if got := decodeJSON[codeResponse](t, recorder).Code; got != OffCode() {
		t.Fatalf("code = %q, want %q", got, OffCode())
	}
	counts := driver.callCounts()
	if counts.loginCode != 0 {
		t.Fatalf("loginCode calls = %d, want 0 before start", counts.loginCode)
	}
	if counts.snapshot != 1 {
		t.Fatalf("snapshot calls = %d, want 1", counts.snapshot)
	}
}

func TestHandlerLoginCodeRefreshFailureRetainedMatch(t *testing.T) {
	driver := newFakeDriver()
	driver.setReport(Report{
		Status: ptr("online"), Start: ptr(true), Match: ptr(true),
		Name: ptr("MockKVM"), IP: ptr("100.64.0.1"),
	})
	handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))

	recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true")
	assertStatus(t, recorder, http.StatusOK)

	driver.setSnapshotErr(errors.New("snapshot exploded"))
	recorder = serve(t, handler, http.MethodGet, LoginCodePath)
	assertStatus(t, recorder, http.StatusServiceUnavailable)
	if got := decodeJSON[errorResponse](t, recorder).Error; got != errorStateUnavailable {
		t.Fatalf("error = %q, want %q", got, errorStateUnavailable)
	}
	if counts := driver.callCounts(); counts.loginCode != 0 {
		t.Fatalf("loginCode calls = %d, want 0 when a matched peer cannot be identified", counts.loginCode)
	}
}

func TestHandlerLoginCodeRefreshFailureRetainedStartOnly(t *testing.T) {
	driver := newFakeDriver()
	driver.setReport(Report{Start: ptr(true), Match: ptr(false)})
	handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))

	recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true")
	assertStatus(t, recorder, http.StatusOK)

	driver.setSnapshotErr(errors.New("snapshot exploded"))
	driver.setLoginCode("654321")
	recorder = serve(t, handler, http.MethodGet, LoginCodePath)
	assertStatus(t, recorder, http.StatusOK)
	if got := decodeJSON[codeResponse](t, recorder).Code; got != "654321" {
		t.Fatalf("code = %q, want %q", got, "654321")
	}
	if counts := driver.callCounts(); counts.loginCode != 1 {
		t.Fatalf("loginCode calls = %d, want 1", counts.loginCode)
	}
}

func TestHandlerLoginCodeDriverError(t *testing.T) {
	driver := newFakeDriver()
	driver.setReport(Report{Start: ptr(true), Match: ptr(false)})
	driver.setLoginCodeErr(errors.New("code exploded"))
	handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))

	recorder := serve(t, handler, http.MethodGet, LoginCodePath)
	assertStatus(t, recorder, http.StatusServiceUnavailable)
	if got := decodeJSON[errorResponse](t, recorder).Error; got != errorLoginCodeUnavailable {
		t.Fatalf("error = %q, want %q", got, errorLoginCodeUnavailable)
	}
	if strings.Contains(recorder.Body.String(), "exploded") {
		t.Fatalf("driver error text leaked into the response body")
	}
	if counts := driver.callCounts(); counts.loginCode != 1 {
		t.Fatalf("loginCode calls = %d, want 1", counts.loginCode)
	}
}

func TestHandlerDisableRespondsBeforeDriverCompletes(t *testing.T) {
	driver := newFakeDriver()
	driver.setEndLoginSessionBlock(blockRelease)
	handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))

	recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?disable=true")
	assertStatus(t, recorder, http.StatusOK)
	assertBoundedResponse(t, recorder)
	if got := decodeJSON[codeResponse](t, recorder).Code; got != OffCode() {
		t.Fatalf("code = %q, want %q", got, OffCode())
	}

	// The response is complete while the driver is still blocked.
	waitSignal(t, driver.started)
	if counts := driver.callCounts(); counts.endLoginSession != 1 {
		t.Fatalf("endLoginSession calls = %d, want 1", counts.endLoginSession)
	}

	close(driver.release)
	waitSignal(t, driver.finished)
}

func TestHandlerDisableSkipsWhenActionBusy(t *testing.T) {
	driver := newFakeDriver()
	driver.setReport(Report{Start: ptr(true)})
	driver.setInitializeBlock(blockRelease)
	handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		done <- serve(t, handler, http.MethodGet, LoginCodePath+"?start=true")
	}()
	waitSignal(t, driver.started)

	recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?disable=true")
	assertStatus(t, recorder, http.StatusOK)
	if got := decodeJSON[codeResponse](t, recorder).Code; got != OffCode() {
		t.Fatalf("code = %q, want %q", got, OffCode())
	}
	if counts := driver.callCounts(); counts.endLoginSession != 0 {
		t.Fatalf("endLoginSession calls = %d, want 0 while an action is in flight", counts.endLoginSession)
	}
	waitNoSignal(t, driver.started)

	close(driver.release)
	select {
	case first := <-done:
		assertStatus(t, first, http.StatusOK)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the start action to finish")
	}
}

func TestHandlerDisableDriverFailureStillAnswers(t *testing.T) {
	driver := newFakeDriver()
	driver.setEndLoginSessionErr(errors.New("teardown exploded"))
	handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))

	recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?disable=true")
	assertStatus(t, recorder, http.StatusOK)
	if got := decodeJSON[codeResponse](t, recorder).Code; got != OffCode() {
		t.Fatalf("code = %q, want %q", got, OffCode())
	}

	waitSignal(t, driver.finished)
	if counts := driver.callCounts(); counts.endLoginSession != 1 {
		t.Fatalf("endLoginSession calls = %d, want 1", counts.endLoginSession)
	}
}

// TestHandlerResponseBounds spot-checks every route's body size and
// Content-Length contract.
func TestHandlerResponseBounds(t *testing.T) {
	cases := []struct {
		name   string
		target string
		prime  bool
	}{
		{"heartbeat", "?heartbeat=true", false},
		{"code", "", false},
		{"start", "?start=true", false},
		{"exit", "?exit=true", true},
		{"disable", "?disable=true", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			driver := newFakeDriver()
			driver.setReport(Report{
				Status: ptr("online"), Start: ptr(true), Match: ptr(true),
				Name: ptr("MockKVM"), IP: ptr("100.64.0.1"),
			})
			driver.setLoginCode("123456")
			handler := NewHandler(testDeps(driver, proto.ApplicationRegionCN))
			if tc.prime {
				recorder := serve(t, handler, http.MethodGet, LoginCodePath+"?heartbeat=true")
				assertStatus(t, recorder, http.StatusOK)
			}

			recorder := serve(t, handler, http.MethodGet, LoginCodePath+tc.target)
			assertStatus(t, recorder, http.StatusOK)
			assertBoundedResponse(t, recorder)
			if tc.target == "?disable=true" {
				waitSignal(t, driver.finished)
			}
		})
	}
}
