package vm

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"

	"github.com/gin-gonic/gin"
)

type fingerRobotHTTPResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func TestGetFingerRobotReturnsProductContractWithoutRawPWM(t *testing.T) {
	setupFingerRobotHandlerTest(t)
	fingerRobotNormalStatus = func() FingerRobotStatus {
		return FingerRobotStatus{
			Running: true,
			State:   FingerRobotStateHolding,
			Result:  FingerRobotResultRunning,
			Error:   "ramp to private pulse 1450 us failed",
		}
	}

	rsp, raw := performFingerRobotRequest(t, http.MethodGet, "/api/vm/finger-robot", "")
	if rsp.Code != 0 {
		t.Fatalf("GET failed: code=%d msg=%q", rsp.Code, rsp.Msg)
	}
	var data proto.GetFingerRobotRsp
	if err := json.Unmarshal(rsp.Data, &data); err != nil {
		t.Fatalf("decode GET data: %v", err)
	}
	if !data.Enabled || !data.Available || data.Config == nil {
		t.Fatalf("unexpected capability/config: %+v", data)
	}
	if data.Config.Version != FingerRobotConfigVersion || data.Config.Revision != 1 ||
		data.Config.ActiveArm != FingerRobotArmProfileShortID || data.Config.StrengthByArm.Long != 3 ||
		data.Config.StrengthByArm.Short != 3 || data.Config.PressMS != 800 || data.Config.Speed != "slow" {
		t.Fatalf("unexpected public config: %+v", data.Config)
	}
	if len(data.Profiles) != 2 || data.Profiles[0] != (proto.FingerRobotProfileRsp{ID: "long", DisplayName: "长摆臂", DefaultStrength: 3}) ||
		data.Profiles[1] != (proto.FingerRobotProfileRsp{ID: "short", DisplayName: "短摆臂", DefaultStrength: 3}) {
		t.Fatalf("unexpected public profiles: %+v", data.Profiles)
	}
	if data.Strength != (proto.FingerRobotRangeRsp{Min: 1, Max: 5, Default: 3}) ||
		data.PressMS != (proto.FingerRobotRangeRsp{Min: 100, Max: 30000, Default: 800}) ||
		len(data.SpeedOptions) != 2 || data.SpeedOptions[0] != "slow" || data.SpeedOptions[1] != "fast" || data.DefaultSpeed != "slow" {
		t.Fatalf("unexpected public choices: %+v", data)
	}
	if !data.Status.Running || data.Status.State != "holding" || data.Status.Result != "running" || data.Status.Error != "operation failed" {
		t.Fatalf("unexpected public status: %+v", data.Status)
	}
	for _, forbidden := range []string{"pressUs", "releaseUs", "minSafePressUs", "pressUsByStrength", "usPerSecond", "private pulse", "1450 us"} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("GET leaked raw PWM field %q: %s", forbidden, raw)
		}
	}
}

func TestGetFingerRobotAlwaysReportsDisabledAndUnavailableFlags(t *testing.T) {
	setupFingerRobotHandlerTest(t)
	fingerRobotNormalSettingEnabled = func() bool { return false }
	fingerRobotNormalAvailable = func() bool { return false }

	rsp, _ := performFingerRobotRequest(t, http.MethodGet, "/api/vm/finger-robot", "")
	if rsp.Code != 0 {
		t.Fatalf("GET failed: %+v", rsp)
	}
	var data proto.GetFingerRobotRsp
	if err := json.Unmarshal(rsp.Data, &data); err != nil {
		t.Fatal(err)
	}
	if data.Enabled || data.Available || data.Config == nil {
		t.Fatalf("unexpected disabled response: %+v", data)
	}
}

func TestGetFingerRobotConfigFailureReleasesAndFailsClosed(t *testing.T) {
	setupFingerRobotHandlerTest(t)
	loadErr := errors.New("corrupt config")
	fingerRobotNormalLoadConfig = func() (FingerRobotConfig, error) {
		return FingerRobotConfig{}, loadErr
	}
	released := 0
	available := true
	status := FingerRobotStatus{State: FingerRobotStateIdle, Result: FingerRobotResultIdle}
	fingerRobotNormalAvailable = func() bool { return available }
	fingerRobotNormalStatus = func() FingerRobotStatus { return status }
	fingerRobotNormalRelease = func() error {
		released++
		available = false
		status = FingerRobotStatus{
			State:  FingerRobotStateReleasing,
			Result: FingerRobotResultFailed,
			Error:  "raw PWM release at 2050 us failed",
		}
		return errors.New("release failed")
	}

	rsp, raw := performFingerRobotRequest(t, http.MethodGet, "/api/vm/finger-robot", "")
	if rsp.Code != -5 || released != 1 {
		t.Fatalf("config failure response=%+v release calls=%d", rsp, released)
	}
	var data proto.GetFingerRobotRsp
	if err := json.Unmarshal(rsp.Data, &data); err != nil {
		t.Fatal(err)
	}
	if data.Config != nil || len(data.Profiles) != 2 || data.Available ||
		data.Status.Result != "failed" || data.Status.Error != "operation failed" {
		t.Fatalf("failed GET should preserve safe public capability only: %+v", data)
	}
	if strings.Contains(raw, "2050") || strings.Contains(raw, "raw PWM") {
		t.Fatalf("failed GET leaked controller details: %s", raw)
	}
}

func TestSetFingerRobotSavesWithoutChallengeOrHardware(t *testing.T) {
	setupFingerRobotHandlerTest(t)
	fingerRobotNormalSettingEnabled = func() bool { return false }
	fingerRobotNormalAvailable = func() bool { return false }
	targetPowerActionRequiresChallenge = func() bool { return true }
	verifyTargetPowerActionChallenge = func(*gin.Context, string, string) error {
		t.Fatal("configuration save must not consume a challenge")
		return nil
	}
	fingerRobotNormalPress = func(FingerRobotPressRequest) error {
		t.Fatal("configuration update must not press the servo")
		return nil
	}
	fingerRobotNormalRelease = func() error {
		t.Fatal("valid configuration update must not release or move the servo")
		return nil
	}
	updated := false
	fingerRobotNormalUpdateConfig = func(expectedRevision int64, update FingerRobotConfigUpdate) (FingerRobotConfig, bool, error) {
		if expectedRevision != 1 || update.ActiveArm != "long" || update.StrengthByArm.Long != 4 ||
			update.StrengthByArm.Short != 2 || update.PressMS != 1200 || update.Speed != FingerRobotSpeedFast {
			t.Fatalf("unexpected update: revision=%d update=%+v", expectedRevision, update)
		}
		updated = true
		return FingerRobotConfig{
			Version:       FingerRobotConfigVersion,
			Revision:      2,
			ActiveArm:     update.ActiveArm,
			StrengthByArm: update.StrengthByArm,
			PressMS:       update.PressMS,
			Speed:         update.Speed,
		}, true, nil
	}

	body := `{"version":1,"revision":1,"activeArm":"long","strengthByArm":{"long":4,"short":2},"pressMs":1200,"speed":"fast","challenge":"opaque-target-power"}`
	rsp, raw := performFingerRobotRequest(t, http.MethodPut, "/api/vm/finger-robot", body)
	if rsp.Code != 0 || !updated {
		t.Fatalf("update response=%+v updated=%t", rsp, updated)
	}
	var data proto.SetFingerRobotRsp
	if err := json.Unmarshal(rsp.Data, &data); err != nil {
		t.Fatal(err)
	}
	if !data.Changed || data.Config.Revision != 2 || data.Config.ActiveArm != "long" || data.Config.Speed != "fast" {
		t.Fatalf("unexpected update data: %+v", data)
	}
	if strings.Contains(raw, "opaque-target-power") {
		t.Fatalf("response leaked challenge: %s", raw)
	}
}

func TestSetFingerRobotRevisionConflictDoesNotConsumeChallengeOrOverwrite(t *testing.T) {
	setupFingerRobotHandlerTest(t)
	current := DefaultFingerRobotConfig()
	current.Revision = 9
	fingerRobotNormalLoadConfig = func() (FingerRobotConfig, error) { return current, nil }
	targetPowerActionRequiresChallenge = func() bool { return true }
	verifyTargetPowerActionChallenge = func(*gin.Context, string, string) error {
		t.Fatal("stale update must not consume a challenge")
		return nil
	}
	fingerRobotNormalUpdateConfig = func(int64, FingerRobotConfigUpdate) (FingerRobotConfig, bool, error) {
		t.Fatal("stale update must not reach the store")
		return FingerRobotConfig{}, false, nil
	}

	body := validSetFingerRobotBody(8)
	rsp, _ := performFingerRobotRequest(t, http.MethodPut, "/api/vm/finger-robot", body)
	if rsp.Code != -6 || rsp.Msg != "revision conflict" {
		t.Fatalf("unexpected conflict response: %+v", rsp)
	}
	var data proto.SetFingerRobotRsp
	if err := json.Unmarshal(rsp.Data, &data); err != nil {
		t.Fatal(err)
	}
	if data.Config.Revision != 9 || data.Changed {
		t.Fatalf("conflict did not return current config: %+v", data)
	}
}

func TestSetFingerRobotMapsConcurrentRevisionConflict(t *testing.T) {
	setupFingerRobotHandlerTest(t)
	current := DefaultFingerRobotConfig()
	current.Revision = 4
	fingerRobotNormalLoadConfig = func() (FingerRobotConfig, error) { return current, nil }
	targetPowerActionRequiresChallenge = func() bool { return true }
	verifyTargetPowerActionChallenge = func(*gin.Context, string, string) error {
		t.Fatal("concurrent revision conflict must not consume a challenge")
		return nil
	}
	fingerRobotNormalUpdateConfig = func(int64, FingerRobotConfigUpdate) (FingerRobotConfig, bool, error) {
		concurrent := current
		concurrent.Revision = 5
		return concurrent, false, ErrFingerRobotConfigRevisionConflict
	}

	rsp, _ := performFingerRobotRequest(t, http.MethodPut, "/api/vm/finger-robot", validSetFingerRobotBody(4))
	if rsp.Code != -6 {
		t.Fatalf("concurrent conflict response=%+v", rsp)
	}
	var data proto.SetFingerRobotRsp
	if err := json.Unmarshal(rsp.Data, &data); err != nil {
		t.Fatal(err)
	}
	if data.Config.Revision != 5 {
		t.Fatalf("concurrent conflict did not return latest config: %+v", data)
	}
}

func TestSetFingerRobotLoadFailureReleasesSafely(t *testing.T) {
	setupFingerRobotHandlerTest(t)
	targetPowerActionRequiresChallenge = func() bool { return true }
	verifyTargetPowerActionChallenge = func(*gin.Context, string, string) error {
		t.Fatal("unreadable configuration must fail safe without challenge verification")
		return nil
	}
	fingerRobotNormalLoadConfig = func() (FingerRobotConfig, error) {
		return FingerRobotConfig{}, errors.New("read failed")
	}
	released := 0
	fingerRobotNormalRelease = func() error {
		released++
		return nil
	}

	rsp, _ := performFingerRobotRequest(t, http.MethodPut, "/api/vm/finger-robot", validSetFingerRobotBody(1))
	if rsp.Code != -5 || released != 1 {
		t.Fatalf("load failure response=%+v releases=%d", rsp, released)
	}
}

func TestSetFingerRobotStrictlyRejectsIncompleteUnknownAndInvalidInput(t *testing.T) {
	setupFingerRobotHandlerTest(t)
	targetPowerActionRequiresChallenge = func() bool { return true }
	verifyTargetPowerActionChallenge = func(*gin.Context, string, string) error {
		t.Fatal("invalid input must not consume a challenge")
		return nil
	}
	fingerRobotNormalLoadConfig = func() (FingerRobotConfig, error) {
		t.Fatal("invalid input must not load or create configuration")
		return FingerRobotConfig{}, nil
	}
	fingerRobotNormalUpdateConfig = func(int64, FingerRobotConfigUpdate) (FingerRobotConfig, bool, error) {
		t.Fatal("invalid input must not reach the store")
		return FingerRobotConfig{}, false, nil
	}

	valid := validSetFingerRobotBody(1)
	tests := []struct {
		name string
		body string
	}{
		{name: "empty body", body: ""},
		{name: "missing all fields", body: `{}`},
		{name: "wrong version", body: strings.Replace(valid, `"version":1`, `"version":2`, 1)},
		{name: "zero revision", body: strings.Replace(valid, `"revision":1`, `"revision":0`, 1)},
		{name: "missing nested strength", body: strings.Replace(valid, `"long":3,`, "", 1)},
		{name: "unknown root field", body: strings.TrimSuffix(valid, "}") + `,"pressUs":1450}`},
		{name: "unknown nested profile", body: strings.Replace(valid, `"short":3`, `"short":3,"medium":3`, 1)},
		{name: "unknown active arm", body: strings.Replace(valid, `"activeArm":"short"`, `"activeArm":"medium"`, 1)},
		{name: "strength below range", body: strings.Replace(valid, `"long":3`, `"long":0`, 1)},
		{name: "strength above range", body: strings.Replace(valid, `"short":3`, `"short":6`, 1)},
		{name: "press duration below range", body: strings.Replace(valid, `"pressMs":800`, `"pressMs":99`, 1)},
		{name: "press duration above range", body: strings.Replace(valid, `"pressMs":800`, `"pressMs":30001`, 1)},
		{name: "unknown speed", body: strings.Replace(valid, `"speed":"slow"`, `"speed":"medium"`, 1)},
		{name: "multiple JSON values", body: valid + ` {}`},
		{name: "oversized", body: `{"version":1,"challenge":"` + strings.Repeat("x", fingerRobotRequestMaxBytes) + `"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rsp, _ := performFingerRobotRequest(t, http.MethodPut, "/api/vm/finger-robot", test.body)
			if rsp.Code != -1 {
				t.Fatalf("invalid body accepted: code=%d msg=%q body=%s", rsp.Code, rsp.Msg, test.body)
			}
		})
	}
}

func TestPressFingerRobotUsesOnePersistedProductSnapshot(t *testing.T) {
	setupFingerRobotHandlerTest(t)
	config := DefaultFingerRobotConfig()
	config.ActiveArm = FingerRobotArmProfileLongID
	config.StrengthByArm.Long = 4
	config.PressMS = 1200
	config.Speed = FingerRobotSpeedFast
	loadCalls := 0
	fingerRobotNormalLoadConfig = func() (FingerRobotConfig, error) {
		loadCalls++
		return config, nil
	}
	targetPowerActionRequiresChallenge = func() bool { return true }
	verified := false
	verifyTargetPowerActionChallenge = func(_ *gin.Context, challenge, action string) error {
		if challenge != "press-challenge" || action != authservice.SensitiveActionTargetPower {
			t.Fatalf("unexpected challenge=%q action=%q", challenge, action)
		}
		verified = true
		return nil
	}
	var got FingerRobotPressRequest
	fingerRobotNormalPress = func(req FingerRobotPressRequest) error {
		if !verified {
			t.Fatal("press started before challenge verification")
		}
		got = req
		return nil
	}

	rsp, _ := performFingerRobotRequest(t, http.MethodPost, "/api/vm/finger-robot/press", `{"challenge":"press-challenge"}`)
	if rsp.Code != 0 || loadCalls != 1 || got != (FingerRobotPressRequest{ProfileID: "long", Strength: 4, PressMS: 1200, Speed: FingerRobotSpeedFast}) {
		t.Fatalf("press response=%+v loads=%d request=%+v", rsp, loadCalls, got)
	}
}

func TestPressFingerRobotMapsBusyAndConfigFailure(t *testing.T) {
	t.Run("busy", func(t *testing.T) {
		setupFingerRobotHandlerTest(t)
		fingerRobotNormalPress = func(FingerRobotPressRequest) error { return ErrFingerRobotBusy }
		rsp, _ := performFingerRobotRequest(t, http.MethodPost, "/api/vm/finger-robot/press", `{}`)
		if rsp.Code != -2 {
			t.Fatalf("busy response: %+v", rsp)
		}
	})

	t.Run("unavailable", func(t *testing.T) {
		setupFingerRobotHandlerTest(t)
		fingerRobotNormalPress = func(FingerRobotPressRequest) error { return ErrFingerRobotUnavailable }
		rsp, _ := performFingerRobotRequest(t, http.MethodPost, "/api/vm/finger-robot/press", `{}`)
		if rsp.Code != -5 || rsp.Msg != "finger robot is unavailable" {
			t.Fatalf("unavailable response: %+v", rsp)
		}
	})

	t.Run("config failure releases", func(t *testing.T) {
		setupFingerRobotHandlerTest(t)
		targetPowerActionRequiresChallenge = func() bool { return true }
		verifyTargetPowerActionChallenge = func(*gin.Context, string, string) error {
			t.Fatal("corrupt configuration must fail safe before challenge verification")
			return nil
		}
		fingerRobotNormalLoadConfig = func() (FingerRobotConfig, error) {
			return FingerRobotConfig{}, errors.New("read failed")
		}
		released := 0
		fingerRobotNormalRelease = func() error {
			released++
			return nil
		}
		fingerRobotNormalPress = func(FingerRobotPressRequest) error {
			t.Fatal("invalid config must never reach the controller press")
			return nil
		}
		rsp, _ := performFingerRobotRequest(t, http.MethodPost, "/api/vm/finger-robot/press", `{}`)
		if rsp.Code != -5 || released != 1 {
			t.Fatalf("config failure response=%+v releases=%d", rsp, released)
		}
	})
}

func TestFingerRobotActionsStrictlyValidateInput(t *testing.T) {
	for _, action := range []struct {
		name string
		path string
	}{
		{name: "press", path: "/api/vm/finger-robot/press"},
		{name: "release", path: "/api/vm/finger-robot/release"},
		{name: "replace arm", path: "/api/vm/finger-robot/replace-arm"},
	} {
		t.Run(action.name+" unknown field", func(t *testing.T) {
			setupFingerRobotHandlerTest(t)
			targetPowerActionRequiresChallenge = func() bool { return true }
			verifyTargetPowerActionChallenge = func(*gin.Context, string, string) error {
				t.Fatal("invalid request must not consume challenge")
				return nil
			}
			rsp, _ := performFingerRobotRequest(t, http.MethodPost, action.path, `{"channel":0}`)
			if rsp.Code != -1 {
				t.Fatalf("unknown field response: %+v", rsp)
			}
		})
	}
}

func TestPressFingerRobotRequiresTargetPowerChallenge(t *testing.T) {
	setupFingerRobotHandlerTest(t)
	targetPowerActionRequiresChallenge = func() bool { return true }
	verifyTargetPowerActionChallenge = func(_ *gin.Context, challenge, sensitiveAction string) error {
		if challenge != "rejected" || sensitiveAction != authservice.SensitiveActionTargetPower {
			t.Fatalf("challenge=%q action=%q", challenge, sensitiveAction)
		}
		return errors.New("rejected")
	}
	fingerRobotNormalPress = func(FingerRobotPressRequest) error {
		t.Fatal("challenge failure must not press")
		return nil
	}
	rsp, _ := performFingerRobotRequest(t, http.MethodPost, "/api/vm/finger-robot/press", `{"challenge":"rejected"}`)
	if rsp.Code != -3 {
		t.Fatalf("challenge failure response: %+v", rsp)
	}
}

func TestReleaseFingerRobotSucceedsWithoutChallenge(t *testing.T) {
	setupFingerRobotHandlerTest(t)
	targetPowerActionRequiresChallenge = func() bool { return true }
	verifyTargetPowerActionChallenge = func(*gin.Context, string, string) error {
		t.Fatal("safety release must not consume a challenge")
		return nil
	}
	released := 0
	fingerRobotNormalRelease = func() error {
		released++
		return nil
	}
	fingerRobotNormalReplaceArm = func() error {
		t.Fatal("release must not trigger arm replacement")
		return nil
	}

	rsp, _ := performFingerRobotRequest(t, http.MethodPost, "/api/vm/finger-robot/release", `{"challenge":"ignored"}`)
	if rsp.Code != 0 || released != 1 {
		t.Fatalf("ungated release response=%+v releases=%d", rsp, released)
	}
}

func TestReplaceFingerRobotArmRequiresTargetPowerChallenge(t *testing.T) {
	t.Run("missing challenge", func(t *testing.T) {
		setupFingerRobotHandlerTest(t)
		targetPowerActionRequiresChallenge = func() bool { return true }
		verifyTargetPowerActionChallenge = func(_ *gin.Context, challenge, sensitiveAction string) error {
			if challenge != "" || sensitiveAction != authservice.SensitiveActionTargetPower {
				t.Fatalf("challenge=%q action=%q", challenge, sensitiveAction)
			}
			return errors.New("missing challenge")
		}
		fingerRobotNormalReplaceArm = func() error {
			t.Fatal("missing challenge must not move the arm")
			return nil
		}
		rsp, _ := performFingerRobotRequest(t, http.MethodPost, "/api/vm/finger-robot/replace-arm", `{}`)
		if rsp.Code != -3 {
			t.Fatalf("missing challenge response: %+v", rsp)
		}
	})

	t.Run("rejected challenge", func(t *testing.T) {
		setupFingerRobotHandlerTest(t)
		targetPowerActionRequiresChallenge = func() bool { return true }
		verifyTargetPowerActionChallenge = func(_ *gin.Context, challenge, sensitiveAction string) error {
			if challenge != "rejected" || sensitiveAction != authservice.SensitiveActionTargetPower {
				t.Fatalf("challenge=%q action=%q", challenge, sensitiveAction)
			}
			return errors.New("rejected")
		}
		fingerRobotNormalReplaceArm = func() error {
			t.Fatal("challenge failure must not move the arm")
			return nil
		}
		rsp, _ := performFingerRobotRequest(t, http.MethodPost, "/api/vm/finger-robot/replace-arm", `{"challenge":"rejected"}`)
		if rsp.Code != -3 {
			t.Fatalf("challenge failure response: %+v", rsp)
		}
	})

	t.Run("verified challenge actuates after verification", func(t *testing.T) {
		setupFingerRobotHandlerTest(t)
		targetPowerActionRequiresChallenge = func() bool { return true }
		verified := false
		verifyTargetPowerActionChallenge = func(_ *gin.Context, challenge, sensitiveAction string) error {
			if challenge != "replace-arm-challenge" || sensitiveAction != authservice.SensitiveActionTargetPower {
				t.Fatalf("challenge=%q action=%q", challenge, sensitiveAction)
			}
			verified = true
			return nil
		}
		actuated := false
		fingerRobotNormalReplaceArm = func() error {
			if !verified {
				t.Fatal("arm replacement started before challenge verification")
			}
			actuated = true
			return nil
		}
		rsp, _ := performFingerRobotRequest(t, http.MethodPost, "/api/vm/finger-robot/replace-arm", `{"challenge":"replace-arm-challenge"}`)
		if rsp.Code != 0 || !actuated {
			t.Fatalf("gated replace arm response=%+v actuated=%t", rsp, actuated)
		}
	})
}

func TestReplaceFingerRobotArmSucceedsWithoutChallengeWhenPolicyOff(t *testing.T) {
	setupFingerRobotHandlerTest(t)
	targetPowerActionRequiresChallenge = func() bool { return false }
	verifyTargetPowerActionChallenge = func(*gin.Context, string, string) error {
		t.Fatal("policy-off replace arm must not verify a challenge")
		return nil
	}
	actuated := false
	fingerRobotNormalReplaceArm = func() error {
		actuated = true
		return nil
	}

	rsp, _ := performFingerRobotRequest(t, http.MethodPost, "/api/vm/finger-robot/replace-arm", `{}`)
	if rsp.Code != 0 || !actuated {
		t.Fatalf("policy-off replace arm response=%+v actuated=%t", rsp, actuated)
	}
}

func TestReleaseFingerRobotAlwaysCallsIdempotentSafetyRelease(t *testing.T) {
	setupFingerRobotHandlerTest(t)
	fingerRobotNormalSettingEnabled = func() bool { return false }
	fingerRobotNormalAvailable = func() bool { return false }
	targetPowerActionRequiresChallenge = func() bool { return true }
	verifyTargetPowerActionChallenge = func(*gin.Context, string, string) error {
		t.Fatal("safety release must not consume a challenge")
		return nil
	}
	released := 0
	fingerRobotNormalRelease = func() error {
		released++
		return nil
	}

	rsp, _ := performFingerRobotRequest(t, http.MethodPost, "/api/vm/finger-robot/release", `{}`)
	if rsp.Code != 0 || released != 1 {
		t.Fatalf("release response=%+v calls=%d", rsp, released)
	}
}

func TestReplaceFingerRobotArmHoldsFactoryZeroAndReleaseRestoresNormalPosition(t *testing.T) {
	setupFingerRobotHandlerTest(t)
	backend := newFakeFingerRobotBackend()
	controller := newFingerRobotController(backend)
	initializeControllerForTest(t, controller, backend)
	t.Cleanup(func() { _ = controller.Shutdown() })

	originalDebugMode := fingerRobotDebugModeEnabled
	fingerRobotDebugModeEnabled = func() bool { return false }
	t.Cleanup(func() { fingerRobotDebugModeEnabled = originalDebugMode })
	fingerRobotNormalLoadConfig = func() (FingerRobotConfig, error) {
		t.Fatal("arm replacement must not depend on saved settings")
		return FingerRobotConfig{}, nil
	}
	fingerRobotNormalUpdateConfig = func(int64, FingerRobotConfigUpdate) (FingerRobotConfig, bool, error) {
		t.Fatal("arm replacement must not change saved settings")
		return FingerRobotConfig{}, false, nil
	}
	targetPowerActionRequiresChallenge = func() bool { return true }
	verified := false
	verifyTargetPowerActionChallenge = func(_ *gin.Context, challenge, sensitiveAction string) error {
		if verified {
			t.Fatal("arm replacement verifies once; the safety release must not verify")
		}
		if challenge != "replace-arm-challenge" || sensitiveAction != authservice.SensitiveActionTargetPower {
			t.Fatalf("challenge=%q action=%q", challenge, sensitiveAction)
		}
		verified = true
		return nil
	}
	fingerRobotNormalReplaceArm = func() error {
		if !verified {
			t.Fatal("arm replacement started before challenge verification")
		}
		return controller.Reset()
	}
	fingerRobotNormalRelease = controller.Release

	rsp, _ := performFingerRobotRequest(t, http.MethodPost, "/api/vm/finger-robot/replace-arm", `{"challenge":"replace-arm-challenge"}`)
	if rsp.Code != 0 {
		t.Fatalf("replace arm response=%+v", rsp)
	}
	if pulses := backend.pulseSnapshot(); !reflect.DeepEqual(pulses, []int{FingerRobotFactoryZeroUS}) {
		t.Fatalf("arm replacement pulses=%v", pulses)
	}
	if status := controller.Status(); !isFingerRobotFactoryZeroHeld(status) {
		t.Fatalf("arm replacement did not hold the position: %+v", status)
	}

	rsp, _ = performFingerRobotRequest(t, http.MethodPost, "/api/vm/finger-robot/release", `{}`)
	if rsp.Code != 0 {
		t.Fatalf("release after replacement: %+v", rsp)
	}
	if pulses := backend.pulseSnapshot(); !reflect.DeepEqual(pulses, []int{FingerRobotFactoryZeroUS, FingerRobotSafeReleaseUS}) {
		t.Fatalf("release after replacement pulses=%v", pulses)
	}
	if status := controller.Status(); isFingerRobotFactoryZeroHeld(status) || status.Result != FingerRobotResultSuccess {
		t.Fatalf("release did not exit arm replacement: %+v", status)
	}
}

func TestReplaceFingerRobotArmRequiresDetectedServo(t *testing.T) {
	for _, scenario := range []struct {
		name      string
		enabled   bool
		available bool
	}{
		{name: "not detected", enabled: true, available: false},
		{name: "disabled", enabled: false, available: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			setupFingerRobotHandlerTest(t)
			fingerRobotNormalSettingEnabled = func() bool { return scenario.enabled }
			fingerRobotNormalAvailable = func() bool { return scenario.available }
			targetPowerActionRequiresChallenge = func() bool { return true }
			verifyTargetPowerActionChallenge = func(*gin.Context, string, string) error {
				t.Fatal("unavailable servo must not consume a challenge")
				return nil
			}
			fingerRobotNormalReplaceArm = func() error {
				t.Fatal("unavailable servo must not move or retry initialization")
				return nil
			}

			rsp, _ := performFingerRobotRequest(t, http.MethodPost, "/api/vm/finger-robot/replace-arm", `{"challenge":"replace-arm-challenge"}`)
			if rsp.Code != -5 || rsp.Msg != "finger robot is unavailable" {
				t.Fatalf("unavailable response=%+v", rsp)
			}
		})
	}
}

func TestReplaceFingerRobotArmRejectsInvalidInputAndHidesHardwareErrors(t *testing.T) {
	setupFingerRobotHandlerTest(t)
	for _, body := range []string{"", "null", `{"pulseUs":1500}`, `{} {}`, `{"challenge":123}`} {
		fingerRobotNormalReplaceArm = func() error {
			t.Fatalf("invalid input %q must not move the arm", body)
			return nil
		}
		rsp, _ := performFingerRobotRequest(t, http.MethodPost, "/api/vm/finger-robot/replace-arm", body)
		if rsp.Code != -1 {
			t.Fatalf("invalid body %q response=%+v", body, rsp)
		}
	}

	for _, controllerErr := range []error{ErrFingerRobotUnavailable, errors.New("private PWM 1500 us failure")} {
		fingerRobotNormalReplaceArm = func() error { return controllerErr }
		rsp, raw := performFingerRobotRequest(t, http.MethodPost, "/api/vm/finger-robot/replace-arm", `{}`)
		if rsp.Code != -5 || rsp.Msg != "operation failed" || strings.Contains(raw, "1500") {
			t.Fatalf("controller failure exposed an invalid public response: %+v", rsp)
		}
	}
}

func TestSnapshotConfiguredFingerRobotPressUsesSavedConfigAndJoinsSafetyError(t *testing.T) {
	setupFingerRobotHandlerTest(t)
	request, err := snapshotConfiguredFingerRobotPress()
	if err != nil {
		t.Fatal(err)
	}
	if request.PressMS != 800 || request.ProfileID != FingerRobotArmProfileShortID ||
		request.Strength != FingerRobotDefaultStrengthLevel || request.Speed != FingerRobotSpeedSlow {
		t.Fatalf("unexpected request snapshot: %+v", request)
	}

	loadErr := errors.New("load failed")
	releaseErr := errors.New("release failed")
	fingerRobotNormalLoadConfig = func() (FingerRobotConfig, error) { return FingerRobotConfig{}, loadErr }
	fingerRobotNormalRelease = func() error { return releaseErr }
	_, err = snapshotConfiguredFingerRobotPress()
	if !errors.Is(err, loadErr) || !errors.Is(err, releaseErr) {
		t.Fatalf("config safety errors were not joined: %v", err)
	}
}

func setupFingerRobotHandlerTest(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	originalSettingEnabled := fingerRobotNormalSettingEnabled
	originalAvailable := fingerRobotNormalAvailable
	originalStatus := fingerRobotNormalStatus
	originalProfiles := fingerRobotNormalListProfiles
	originalLoad := fingerRobotNormalLoadConfig
	originalUpdate := fingerRobotNormalUpdateConfig
	originalPress := fingerRobotNormalPress
	originalRelease := fingerRobotNormalRelease
	originalReplaceArm := fingerRobotNormalReplaceArm
	originalRequiresChallenge := targetPowerActionRequiresChallenge
	originalVerifyChallenge := verifyTargetPowerActionChallenge
	t.Cleanup(func() {
		fingerRobotNormalSettingEnabled = originalSettingEnabled
		fingerRobotNormalAvailable = originalAvailable
		fingerRobotNormalStatus = originalStatus
		fingerRobotNormalListProfiles = originalProfiles
		fingerRobotNormalLoadConfig = originalLoad
		fingerRobotNormalUpdateConfig = originalUpdate
		fingerRobotNormalPress = originalPress
		fingerRobotNormalRelease = originalRelease
		fingerRobotNormalReplaceArm = originalReplaceArm
		targetPowerActionRequiresChallenge = originalRequiresChallenge
		verifyTargetPowerActionChallenge = originalVerifyChallenge
	})

	fingerRobotNormalSettingEnabled = func() bool { return true }
	fingerRobotNormalAvailable = func() bool { return true }
	fingerRobotNormalStatus = func() FingerRobotStatus {
		return FingerRobotStatus{State: FingerRobotStateIdle, Result: FingerRobotResultIdle}
	}
	fingerRobotNormalListProfiles = ListFingerRobotArmProfiles
	fingerRobotNormalLoadConfig = func() (FingerRobotConfig, error) { return DefaultFingerRobotConfig(), nil }
	fingerRobotNormalUpdateConfig = func(expectedRevision int64, update FingerRobotConfigUpdate) (FingerRobotConfig, bool, error) {
		return FingerRobotConfig{
			Version:       FingerRobotConfigVersion,
			Revision:      expectedRevision + 1,
			ActiveArm:     update.ActiveArm,
			StrengthByArm: update.StrengthByArm,
			PressMS:       update.PressMS,
			Speed:         update.Speed,
		}, true, nil
	}
	fingerRobotNormalPress = func(FingerRobotPressRequest) error { return nil }
	fingerRobotNormalRelease = func() error { return nil }
	fingerRobotNormalReplaceArm = func() error { return nil }
	targetPowerActionRequiresChallenge = func() bool { return false }
	verifyTargetPowerActionChallenge = func(*gin.Context, string, string) error { return nil }
}

func validSetFingerRobotBody(expectedRevision int64) string {
	return `{"version":1,"revision":` + fmtInt64(expectedRevision) + `,"activeArm":"short","strengthByArm":{"long":3,"short":3},"pressMs":800,"speed":"slow","challenge":"challenge"}`
}

func fmtInt64(value int64) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func performFingerRobotRequest(t *testing.T, method, path, body string) (fingerRobotHTTPResponse, string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(method, path, bytes.NewBufferString(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	service := NewService()
	switch {
	case method == http.MethodGet:
		service.GetFingerRobot(ctx)
	case method == http.MethodPut:
		service.SetFingerRobot(ctx)
	case strings.HasSuffix(path, "/press"):
		service.PressFingerRobot(ctx)
	case strings.HasSuffix(path, "/release"):
		service.ReleaseFingerRobot(ctx)
	case strings.HasSuffix(path, "/replace-arm"):
		service.ReplaceFingerRobotArm(ctx)
	default:
		t.Fatalf("unsupported test request: %s %s", method, path)
	}

	var rsp fingerRobotHTTPResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("decode response: %v body=%s", err, recorder.Body.String())
	}
	return rsp, recorder.Body.String()
}
