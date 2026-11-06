package vm

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	authservice "NanoKVM-Server/service/auth"

	"github.com/gin-gonic/gin"
)

type rebootTestResponse struct {
	Code int `json:"code"`
}

func TestRebootRejectsMissingNanoKVMRestartChallengeBeforeSystemReboot(t *testing.T) {
	setupRebootHandlerTest(t)
	nanoKVMRestartActionRequiresChallenge = func() bool { return true }
	verifyNanoKVMRestartActionChallenge = func(*gin.Context, string, string) error {
		return errors.New("challenge required")
	}
	rebootFunc = func() error {
		t.Fatal("system reboot must not run when challenge verification fails")
		return nil
	}

	rsp := performRebootRequest(t, "")
	if rsp.Code != -3 {
		t.Fatalf("expected challenge rejection code -3, got %d", rsp.Code)
	}
}

func TestRebootVerifiesAndConsumesNanoKVMRestartChallenge(t *testing.T) {
	setupRebootHandlerTest(t)
	nanoKVMRestartActionRequiresChallenge = func() bool { return true }
	verified := false
	verifyNanoKVMRestartActionChallenge = func(_ *gin.Context, challenge, action string) error {
		if challenge != "nanokvm-restart-challenge" {
			t.Fatalf("expected NanoKVM restart challenge, got %q", challenge)
		}
		if action != authservice.SensitiveActionNanoKVMRestart {
			t.Fatalf("expected %s action, got %q", authservice.SensitiveActionNanoKVMRestart, action)
		}
		verified = true
		return nil
	}
	rebooted := false
	rebootFunc = func() error {
		rebooted = true
		return nil
	}

	rsp := performRebootRequest(t, "nanokvm-restart-challenge")
	if rsp.Code != 0 {
		t.Fatalf("expected successful system reboot request, got %d", rsp.Code)
	}
	if !verified {
		t.Fatal("expected NanoKVM restart challenge to be verified")
	}
	if !rebooted {
		t.Fatal("expected system reboot after successful challenge verification")
	}
}

func TestRebootSkipsNanoKVMRestartChallengeWhenPolicyDisabled(t *testing.T) {
	setupRebootHandlerTest(t)
	nanoKVMRestartActionRequiresChallenge = func() bool { return false }
	verifyNanoKVMRestartActionChallenge = func(*gin.Context, string, string) error {
		t.Fatal("NanoKVM restart challenge verifier must not run when policy is disabled")
		return nil
	}
	rebooted := false
	rebootFunc = func() error {
		rebooted = true
		return nil
	}

	rsp := performRebootRequest(t, "")
	if rsp.Code != 0 {
		t.Fatalf("expected successful system reboot request, got %d", rsp.Code)
	}
	if !rebooted {
		t.Fatal("expected system reboot when policy is disabled")
	}
}

func TestRebootToFlashingRejectsMissingNanoKVMRestartChallenge(t *testing.T) {
	setupRebootHandlerTest(t)
	nanoKVMRestartActionRequiresChallenge = func() bool { return true }
	verifyNanoKVMRestartActionChallenge = func(*gin.Context, string, string) error {
		return errors.New("challenge required")
	}
	rebootToFlashingFunc = func() error {
		t.Fatal("flashing mode reboot must not run when challenge verification fails")
		return nil
	}

	rsp := performRebootToFlashingRequest(t, "")
	if rsp.Code != -3 {
		t.Fatalf("expected challenge rejection code -3, got %d", rsp.Code)
	}
}

func TestRebootToFlashingUsesNanoKVMRestartChallenge(t *testing.T) {
	setupRebootHandlerTest(t)
	nanoKVMRestartActionRequiresChallenge = func() bool { return true }
	verified := false
	verifyNanoKVMRestartActionChallenge = func(_ *gin.Context, challenge, action string) error {
		if challenge != "nanokvm-restart-challenge" {
			t.Fatalf("expected NanoKVM restart challenge, got %q", challenge)
		}
		if action != authservice.SensitiveActionNanoKVMRestart {
			t.Fatalf("expected %s action, got %q", authservice.SensitiveActionNanoKVMRestart, action)
		}
		verified = true
		return nil
	}
	rebootedToFlashing := false
	rebootToFlashingFunc = func() error {
		rebootedToFlashing = true
		return nil
	}

	rsp := performRebootToFlashingRequest(t, "nanokvm-restart-challenge")
	if rsp.Code != 0 {
		t.Fatalf("expected successful flashing mode reboot request, got %d", rsp.Code)
	}
	if !verified {
		t.Fatal("expected NanoKVM restart challenge to be verified")
	}
	if !rebootedToFlashing {
		t.Fatal("expected flashing mode reboot after successful challenge verification")
	}
}

func TestRebootToFlashingCreatesFlagBeforeReboot(t *testing.T) {
	flagPath := filepath.Join(t.TempDir(), "rec")
	rebooted := false

	err := rebootToFlashing(flagPath, func() error {
		if _, err := os.Stat(flagPath); err != nil {
			t.Fatalf("expected flashing mode flag before reboot: %v", err)
		}
		rebooted = true
		return nil
	})
	if err != nil {
		t.Fatalf("reboot to flashing mode: %v", err)
	}
	if !rebooted {
		t.Fatal("expected system reboot to run")
	}
}

func TestRebootToFlashingRemovesNewFlagWhenRebootFails(t *testing.T) {
	flagPath := filepath.Join(t.TempDir(), "rec")
	rebootErr := errors.New("reboot failed")

	err := rebootToFlashing(flagPath, func() error { return rebootErr })
	if !errors.Is(err, rebootErr) {
		t.Fatalf("expected reboot error, got %v", err)
	}
	if _, err := os.Stat(flagPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("new flashing mode flag must be removed after reboot failure, got %v", err)
	}
}

func TestRebootToFlashingPreservesExistingFlagWhenRebootFails(t *testing.T) {
	flagPath := filepath.Join(t.TempDir(), "rec")
	if err := os.WriteFile(flagPath, nil, 0o644); err != nil {
		t.Fatalf("create existing flashing mode flag: %v", err)
	}
	rebootErr := errors.New("reboot failed")

	err := rebootToFlashing(flagPath, func() error { return rebootErr })
	if !errors.Is(err, rebootErr) {
		t.Fatalf("expected reboot error, got %v", err)
	}
	if _, err := os.Stat(flagPath); err != nil {
		t.Fatalf("existing flashing mode flag must be preserved: %v", err)
	}
}

func setupRebootHandlerTest(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	originalRebootFunc := rebootFunc
	originalRebootToFlashingFunc := rebootToFlashingFunc
	originalNanoKVMRestartActionRequiresChallenge := nanoKVMRestartActionRequiresChallenge
	originalVerifyNanoKVMRestartActionChallenge := verifyNanoKVMRestartActionChallenge

	t.Cleanup(func() {
		rebootFunc = originalRebootFunc
		rebootToFlashingFunc = originalRebootToFlashingFunc
		nanoKVMRestartActionRequiresChallenge = originalNanoKVMRestartActionRequiresChallenge
		verifyNanoKVMRestartActionChallenge = originalVerifyNanoKVMRestartActionChallenge
	})
}

func performRebootRequest(t *testing.T, challenge string) rebootTestResponse {
	return performRestartRequest(t, "/api/vm/system/reboot", challenge, NewService().Reboot)
}

func performRebootToFlashingRequest(t *testing.T, challenge string) rebootTestResponse {
	return performRestartRequest(
		t,
		"/api/vm/system/reboot-to-flashing",
		challenge,
		NewService().RebootToFlashing,
	)
}

func performRestartRequest(
	t *testing.T,
	path string,
	challenge string,
	handler func(*gin.Context),
) rebootTestResponse {
	t.Helper()

	body, err := json.Marshal(map[string]string{"challenge": challenge})
	if err != nil {
		t.Fatalf("Marshal request: %v", err)
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	handler(ctx)

	var rsp rebootTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("Unmarshal response: %v body=%s", err, recorder.Body.String())
	}
	return rsp
}
