package vm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"

	"github.com/gin-gonic/gin"
)

type fingerRobotDebugHTTPResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func TestGetFingerRobotDebugHidesCapabilityOutsideProtectedDebugMode(t *testing.T) {
	setupFingerRobotDebugHandlerTest(t)
	for _, test := range []struct {
		name        string
		modeEnabled bool
		authEnabled bool
	}{
		{name: "normal mode", modeEnabled: false, authEnabled: true},
		{name: "authentication disabled", modeEnabled: true, authEnabled: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			fingerRobotDebugModeEnabled = func() bool { return test.modeEnabled }
			fingerRobotDebugAuthEnabled = func() bool { return test.authEnabled }
			fingerRobotDebugAvailable = func() bool {
				t.Fatal("disabled capability must not query hardware availability")
				return false
			}
			fingerRobotDebugTouchLease = func() bool {
				t.Fatal("disabled capability must not renew a debug lease")
				return false
			}

			rsp := performFingerRobotDebugRequest(t, http.MethodGet, "/api/vm/finger-robot/debug", nil)
			if rsp.Code != 0 {
				t.Fatalf("expected success, got code=%d msg=%q", rsp.Code, rsp.Msg)
			}
			var data map[string]interface{}
			if err := json.Unmarshal(rsp.Data, &data); err != nil {
				t.Fatalf("unmarshal data: %v", err)
			}
			if data["enabled"] != false || data["available"] != false {
				t.Fatalf("expected disabled response, got %#v", data)
			}
			for _, field := range []string{"zeroUs", "releaseUs", "pressUs", "pressSpeedUsPerSecond", "count", "pressMs"} {
				if _, exists := data[field]; exists {
					t.Fatalf("disabled response leaked %s: %#v", field, data)
				}
			}
		})
	}
}

func TestGetFingerRobotDebugReturnsLimitsAndFlatStatus(t *testing.T) {
	setupFingerRobotDebugHandlerTest(t)
	touchedLease := false
	fingerRobotDebugTouchLease = func() bool {
		touchedLease = true
		return true
	}
	fingerRobotDebugStatus = func() FingerRobotStatus {
		return FingerRobotStatus{
			Running: true,
			Current: 9_007_199_254_740_990,
			Total:   9_007_199_254_740_991,
			State:   "pressing",
			Result:  "running",
			debug:   true,
		}
	}

	rsp := performFingerRobotDebugRequest(t, http.MethodGet, "/api/vm/finger-robot/debug", nil)
	if rsp.Code != 0 {
		t.Fatalf("expected success, got code=%d msg=%q", rsp.Code, rsp.Msg)
	}
	var data proto.GetFingerRobotDebugRsp
	if err := json.Unmarshal(rsp.Data, &data); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if !data.Enabled || !data.Available || data.ZeroUS != FingerRobotFactoryZeroUS {
		t.Fatalf("unexpected capability data: %+v", data)
	}
	if !touchedLease || data.LeaseTimeoutMS != 3000 {
		t.Fatalf("expected GET to renew and expose the debug lease, touched=%t data=%+v", touchedLease, data)
	}
	if data.ReleaseUS == nil || data.ReleaseUS.Default != 2050 || data.PressUS == nil || data.PressUS.Default != 899 {
		t.Fatalf("unexpected pulse defaults: %+v", data)
	}
	if data.ReleaseUS.Max != 2050 || data.PressUS.Max != 2000 || data.Count == nil ||
		data.Count.Min != 1 || data.Count.Default != 50 || data.Count.UnlimitedValue != 0 {
		t.Fatalf("unexpected debug ranges: %+v", data)
	}
	var rawData map[string]json.RawMessage
	if err := json.Unmarshal(rsp.Data, &rawData); err != nil {
		t.Fatalf("unmarshal raw data: %v", err)
	}
	var rawCount map[string]interface{}
	if err := json.Unmarshal(rawData["count"], &rawCount); err != nil {
		t.Fatalf("unmarshal raw count contract: %v", err)
	}
	if _, exists := rawCount["max"]; exists {
		t.Fatalf("unbounded count contract must not expose a finite max: %#v", rawCount)
	}
	if data.PressSpeedUSPerSecond == nil || data.PressSpeedUSPerSecond.Min != 200 ||
		data.PressSpeedUSPerSecond.Max != 2000 || data.PressSpeedUSPerSecond.Default != 2000 {
		t.Fatalf("unexpected press speed range: %+v", data)
	}
	if !data.Running || data.Current != 9_007_199_254_740_990 || data.Total != 9_007_199_254_740_991 ||
		data.State != "pressing" || data.Result != "running" {
		t.Fatalf("unexpected flat status: %+v", data)
	}
}

func TestGetFingerRobotDebugHidesNormalPowerControllerStatus(t *testing.T) {
	setupFingerRobotDebugHandlerTest(t)
	fingerRobotDebugStatus = func() FingerRobotStatus {
		return FingerRobotStatus{
			Running: true,
			Current: 1,
			Total:   1,
			State:   FingerRobotStatePressing,
			Result:  FingerRobotResultRunning,
			Error:   "normal finger robot status must stay private",
			debug:   false,
		}
	}

	rsp := performFingerRobotDebugRequest(t, http.MethodGet, "/api/vm/finger-robot/debug", nil)
	if rsp.Code != 0 {
		t.Fatalf("expected success, got code=%d msg=%q", rsp.Code, rsp.Msg)
	}
	var data proto.GetFingerRobotDebugRsp
	if err := json.Unmarshal(rsp.Data, &data); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if data.Running || data.Current != 0 || data.Total != 0 || data.State != string(FingerRobotStateIdle) ||
		data.Result != string(FingerRobotResultIdle) || data.Error != "" {
		t.Fatalf("normal finger robot operation leaked into Debug status: %+v", data)
	}
}

func TestGetFingerRobotDebugIdleResponseKeepsZeroValueStatusFields(t *testing.T) {
	setupFingerRobotDebugHandlerTest(t)
	fingerRobotDebugStatus = func() FingerRobotStatus {
		return FingerRobotStatus{State: FingerRobotStateIdle, Result: FingerRobotResultIdle}
	}

	rsp := performFingerRobotDebugRequest(t, http.MethodGet, "/api/vm/finger-robot/debug", nil)
	if rsp.Code != 0 {
		t.Fatalf("expected success, got code=%d msg=%q", rsp.Code, rsp.Msg)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(rsp.Data, &data); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	for _, field := range []string{"running", "current", "total", "state", "result", "error"} {
		if _, exists := data[field]; !exists {
			t.Fatalf("idle response omitted required status field %q: %#v", field, data)
		}
	}
	if data["running"] != false || data["current"] != float64(0) || data["total"] != float64(0) {
		t.Fatalf("unexpected idle zero-value status: %#v", data)
	}
}

func TestFingerRobotDebugPOSTGuardsRunBeforeController(t *testing.T) {
	setupFingerRobotDebugHandlerTest(t)
	started := false
	stopped := false
	reset := false
	fingerRobotDebugStart = func(FingerRobotDebugRequest) error {
		started = true
		return nil
	}
	fingerRobotDebugReset = func() error {
		reset = true
		return nil
	}
	fingerRobotDebugStop = func() error {
		stopped = true
		return nil
	}

	for _, test := range []struct {
		name        string
		modeEnabled bool
		authEnabled bool
	}{
		{name: "normal mode", modeEnabled: false, authEnabled: true},
		{name: "authentication disabled", modeEnabled: true, authEnabled: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			fingerRobotDebugModeEnabled = func() bool { return test.modeEnabled }
			fingerRobotDebugAuthEnabled = func() bool { return test.authEnabled }
			started = false
			stopped = false
			reset = false

			testRsp := performFingerRobotDebugRequest(t, http.MethodPost, "/api/vm/finger-robot/debug/test", validFingerRobotDebugBody())
			stopRsp := performFingerRobotDebugRequest(t, http.MethodPost, "/api/vm/finger-robot/debug/stop", nil)
			resetRsp := performFingerRobotDebugRequest(t, http.MethodPost, "/api/vm/finger-robot/debug/reset", nil)
			if testRsp.Code == 0 || stopRsp.Code == 0 || resetRsp.Code == 0 {
				t.Fatalf("expected all actions to be rejected, test=%+v stop=%+v reset=%+v", testRsp, stopRsp, resetRsp)
			}
			if started || stopped || reset {
				t.Fatal("guarded debug actions must not reach the controller")
			}
		})
	}
}

func TestFingerRobotDebugTestRejectsUnavailableControllerBeforeChallenge(t *testing.T) {
	setupFingerRobotDebugHandlerTest(t)
	fingerRobotDebugAvailable = func() bool { return false }
	targetPowerActionRequiresChallenge = func() bool { return true }
	verifyTargetPowerActionChallenge = func(*gin.Context, string, string) error {
		t.Fatal("unavailable controller must not consume a challenge")
		return nil
	}
	fingerRobotDebugStart = func(FingerRobotDebugRequest) error {
		t.Fatal("unavailable controller must not start a sequence")
		return nil
	}

	rsp := performFingerRobotDebugRequest(t, http.MethodPost, "/api/vm/finger-robot/debug/test", validFingerRobotDebugBody())
	if rsp.Code != -3 {
		t.Fatalf("expected unavailable code -3, got code=%d msg=%q", rsp.Code, rsp.Msg)
	}
}

func TestFingerRobotDebugTestStrictlyRejectsUnknownAndInvalidInputs(t *testing.T) {
	setupFingerRobotDebugHandlerTest(t)
	started := false
	fingerRobotDebugStart = func(FingerRobotDebugRequest) error {
		started = true
		return nil
	}

	tests := []struct {
		name string
		body string
	}{
		{name: "missing press speed", body: `{"releaseUs":2050,"pressUs":1900,"count":1,"pressMs":800}`},
		{name: "missing count", body: `{"releaseUs":2050,"pressUs":1900,"pressSpeedUsPerSecond":1000,"pressMs":800}`},
		{name: "null count", body: `{"releaseUs":2050,"pressUs":1900,"pressSpeedUsPerSecond":1000,"count":null,"pressMs":800}`},
		{name: "unknown channel", body: `{"releaseUs":2050,"pressUs":1900,"pressSpeedUsPerSecond":1000,"count":1,"pressMs":800,"channel":0}`},
		{name: "attempt to override zero", body: `{"releaseUs":2050,"pressUs":1900,"pressSpeedUsPerSecond":1000,"count":1,"pressMs":800,"zeroUs":1400}`},
		{name: "wrong direction", body: `{"releaseUs":900,"pressUs":900,"pressSpeedUsPerSecond":1000,"count":1,"pressMs":800}`},
		{name: "pulse below safe range", body: `{"releaseUs":2050,"pressUs":888,"pressSpeedUsPerSecond":1000,"count":1,"pressMs":800}`},
		{name: "press speed below minimum", body: `{"releaseUs":2050,"pressUs":1900,"pressSpeedUsPerSecond":199,"count":1,"pressMs":800}`},
		{name: "press speed above maximum", body: `{"releaseUs":2050,"pressUs":1900,"pressSpeedUsPerSecond":2001,"count":1,"pressMs":800}`},
		{name: "negative count", body: `{"releaseUs":2050,"pressUs":1900,"pressSpeedUsPerSecond":1000,"count":-1,"pressMs":800}`},
		{name: "press duration below minimum", body: `{"releaseUs":2050,"pressUs":1900,"pressSpeedUsPerSecond":1000,"count":1,"pressMs":99}`},
		{name: "multiple json values", body: `{"releaseUs":2050,"pressUs":1900,"pressSpeedUsPerSecond":1000,"count":1,"pressMs":800} {}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			started = false
			rsp := performFingerRobotDebugRequest(t, http.MethodPost, "/api/vm/finger-robot/debug/test", []byte(test.body))
			if rsp.Code != -1 {
				t.Fatalf("expected invalid argument code -1, got code=%d msg=%q", rsp.Code, rsp.Msg)
			}
			if started {
				t.Fatal("invalid request must not reach the controller")
			}
		})
	}
}

func TestFingerRobotDebugTestAcceptsUnlimitedAndLargeFiniteCounts(t *testing.T) {
	setupFingerRobotDebugHandlerTest(t)
	var counts []int64
	fingerRobotDebugStart = func(req FingerRobotDebugRequest) error {
		counts = append(counts, req.Count)
		return nil
	}

	for _, count := range []int64{0, 9_007_199_254_740_991} {
		body := []byte(fmt.Sprintf(`{"releaseUs":2050,"pressUs":1900,"pressSpeedUsPerSecond":1000,"count":%d,"pressMs":800}`, count))
		rsp := performFingerRobotDebugRequest(t, http.MethodPost, "/api/vm/finger-robot/debug/test", body)
		if rsp.Code != 0 {
			t.Fatalf("count %d rejected: code=%d msg=%q", count, rsp.Code, rsp.Msg)
		}
	}
	if want := []int64{0, 9_007_199_254_740_991}; !reflect.DeepEqual(counts, want) {
		t.Fatalf("unexpected controller counts: got %v want %v", counts, want)
	}
}

func TestFingerRobotDebugTestVerifiesTargetPowerChallengeThenStarts(t *testing.T) {
	setupFingerRobotDebugHandlerTest(t)
	targetPowerActionRequiresChallenge = func() bool { return true }
	verified := false
	verifyTargetPowerActionChallenge = func(_ *gin.Context, challenge, action string) error {
		if challenge != "debug-challenge" || action != authservice.SensitiveActionTargetPower {
			t.Fatalf("unexpected challenge verification: challenge=%q action=%q", challenge, action)
		}
		verified = true
		return nil
	}
	var started FingerRobotDebugRequest
	fingerRobotDebugStart = func(req FingerRobotDebugRequest) error {
		started = req
		return nil
	}

	body := []byte(`{"releaseUs":2050,"pressUs":1900,"pressSpeedUsPerSecond":1000,"count":3,"pressMs":800,"challenge":"debug-challenge"}`)
	rsp := performFingerRobotDebugRequest(t, http.MethodPost, "/api/vm/finger-robot/debug/test", body)
	if rsp.Code != 0 || !verified {
		t.Fatalf("expected verified success, got code=%d msg=%q verified=%t", rsp.Code, rsp.Msg, verified)
	}
	if started != (FingerRobotDebugRequest{ReleaseUS: 2050, PressUS: 1900, PressSpeedUSPerSecond: 1000, Count: 3, PressMS: 800}) {
		t.Fatalf("unexpected controller snapshot: %+v", started)
	}
}

func TestFingerRobotDebugChallengeFailureAndBusyMapping(t *testing.T) {
	setupFingerRobotDebugHandlerTest(t)
	targetPowerActionRequiresChallenge = func() bool { return true }
	verifyTargetPowerActionChallenge = func(*gin.Context, string, string) error {
		return errors.New("challenge required")
	}
	fingerRobotDebugStart = func(FingerRobotDebugRequest) error {
		t.Fatal("challenge failure must not reach controller")
		return nil
	}

	rsp := performFingerRobotDebugRequest(t, http.MethodPost, "/api/vm/finger-robot/debug/test", validFingerRobotDebugBody())
	if rsp.Code != -3 {
		t.Fatalf("expected challenge failure code -3, got %d", rsp.Code)
	}

	targetPowerActionRequiresChallenge = func() bool { return false }
	fingerRobotDebugStart = func(FingerRobotDebugRequest) error { return ErrFingerRobotBusy }
	rsp = performFingerRobotDebugRequest(t, http.MethodPost, "/api/vm/finger-robot/debug/test", validFingerRobotDebugBody())
	if rsp.Code != -2 {
		t.Fatalf("expected busy code -2, got %d", rsp.Code)
	}
}

func TestFingerRobotDebugResetBypassesChallengeAndAvailability(t *testing.T) {
	setupFingerRobotDebugHandlerTest(t)
	fingerRobotDebugAvailable = func() bool { return false }
	targetPowerActionRequiresChallenge = func() bool { return true }
	verifyTargetPowerActionChallenge = func(*gin.Context, string, string) error {
		t.Fatal("reset must not require a challenge")
		return nil
	}
	reset := false
	fingerRobotDebugReset = func() error {
		reset = true
		return nil
	}

	rsp := performFingerRobotDebugRequest(t, http.MethodPost, "/api/vm/finger-robot/debug/reset", nil)
	if rsp.Code != 0 || !reset {
		t.Fatalf("expected reset success even when unavailable, code=%d reset=%t", rsp.Code, reset)
	}
}

func TestFingerRobotDebugStopBypassesChallengeAndAvailabilityAndMapsBusy(t *testing.T) {
	setupFingerRobotDebugHandlerTest(t)
	fingerRobotDebugAvailable = func() bool { return false }
	targetPowerActionRequiresChallenge = func() bool { return true }
	verifyTargetPowerActionChallenge = func(*gin.Context, string, string) error {
		t.Fatal("stop must not require a challenge")
		return nil
	}
	stopped := false
	fingerRobotDebugStop = func() error {
		stopped = true
		return nil
	}

	rsp := performFingerRobotDebugRequest(t, http.MethodPost, "/api/vm/finger-robot/debug/stop", nil)
	if rsp.Code != 0 || !stopped {
		t.Fatalf("expected stop success even when unavailable, code=%d stopped=%t", rsp.Code, stopped)
	}

	fingerRobotDebugStop = func() error { return ErrFingerRobotBusy }
	rsp = performFingerRobotDebugRequest(t, http.MethodPost, "/api/vm/finger-robot/debug/stop", nil)
	if rsp.Code != -2 {
		t.Fatalf("expected stop busy code -2, got code=%d msg=%q", rsp.Code, rsp.Msg)
	}
}

func setupFingerRobotDebugHandlerTest(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	originalModeEnabled := fingerRobotDebugModeEnabled
	originalAuthEnabled := fingerRobotDebugAuthEnabled
	originalAvailable := fingerRobotDebugAvailable
	originalLimits := fingerRobotDebugLimits
	originalStatus := fingerRobotDebugStatus
	originalTouchLease := fingerRobotDebugTouchLease
	originalStart := fingerRobotDebugStart
	originalStop := fingerRobotDebugStop
	originalReset := fingerRobotDebugReset
	originalRequiresChallenge := targetPowerActionRequiresChallenge
	originalVerifyChallenge := verifyTargetPowerActionChallenge

	fingerRobotDebugModeEnabled = func() bool { return true }
	fingerRobotDebugAuthEnabled = func() bool { return true }
	fingerRobotDebugAvailable = func() bool { return true }
	fingerRobotDebugLimits = fingerRobotDebugTestLimits
	fingerRobotDebugStatus = func() FingerRobotStatus { return FingerRobotStatus{State: "idle"} }
	fingerRobotDebugTouchLease = func() bool { return false }
	fingerRobotDebugStart = func(FingerRobotDebugRequest) error { return nil }
	fingerRobotDebugStop = func() error { return nil }
	fingerRobotDebugReset = func() error { return nil }
	targetPowerActionRequiresChallenge = func() bool { return false }
	verifyTargetPowerActionChallenge = func(*gin.Context, string, string) error { return nil }

	t.Cleanup(func() {
		fingerRobotDebugModeEnabled = originalModeEnabled
		fingerRobotDebugAuthEnabled = originalAuthEnabled
		fingerRobotDebugAvailable = originalAvailable
		fingerRobotDebugLimits = originalLimits
		fingerRobotDebugStatus = originalStatus
		fingerRobotDebugTouchLease = originalTouchLease
		fingerRobotDebugStart = originalStart
		fingerRobotDebugStop = originalStop
		fingerRobotDebugReset = originalReset
		targetPowerActionRequiresChallenge = originalRequiresChallenge
		verifyTargetPowerActionChallenge = originalVerifyChallenge
	})
}

func fingerRobotDebugTestLimits() FingerRobotLimits {
	return FingerRobotLimits{
		ZeroUS:                       FingerRobotFactoryZeroUS,
		ReleaseMinUS:                 889,
		ReleaseMaxUS:                 2050,
		ReleaseDefaultUS:             2050,
		PressMinUS:                   889,
		PressMaxUS:                   2000,
		PressDefaultUS:               899,
		PressSpeedMinUSPerSecond:     200,
		PressSpeedMaxUSPerSecond:     2000,
		PressSpeedDefaultUSPerSecond: 2000,
		CountMin:                     1,
		CountDefault:                 50,
		CountUnlimitedValue:          0,
		PressMinMS:                   100,
		PressMaxMS:                   30000,
		PressDefaultMS:               100,
		ReleaseIntervalMS:            500,
		LeaseTimeoutMS:               3000,
	}
}

func validFingerRobotDebugBody() []byte {
	return []byte(`{"releaseUs":2050,"pressUs":899,"pressSpeedUsPerSecond":2000,"count":50,"pressMs":100}`)
}

func performFingerRobotDebugRequest(t *testing.T, method, path string, body []byte) fingerRobotDebugHTTPResponse {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, path, bytes.NewReader(body))
	if body != nil {
		ctx.Request.Header.Set("Content-Type", "application/json")
	}

	service := NewService()
	switch path {
	case "/api/vm/finger-robot/debug":
		service.GetFingerRobotDebug(ctx)
	case "/api/vm/finger-robot/debug/test":
		service.TestFingerRobotDebug(ctx)
	case "/api/vm/finger-robot/debug/stop":
		service.StopFingerRobotDebug(ctx)
	case "/api/vm/finger-robot/debug/reset":
		service.ResetFingerRobotDebug(ctx)
	default:
		t.Fatalf("unsupported path %q", path)
	}

	var rsp fingerRobotDebugHTTPResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("unmarshal response: %v body=%s", err, recorder.Body.String())
	}
	return rsp
}
