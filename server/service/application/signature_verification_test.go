package application

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func withUpdateSignatureVerificationTestState(t *testing.T) string {
	t.Helper()
	directory := applicationStatusTestDirectory(t)
	oldFlag := updateSignatureVerificationFlagPath
	oldLock := applicationUpdatePolicyLockPath
	oldBusy := applicationUpdatePolicyUnitBusy
	oldVerify := verifyUpdateSignatureChallenge
	updateSignatureVerificationFlagPath = filepath.Join(directory, "signature-verification-disabled")
	applicationUpdatePolicyLockPath = filepath.Join(directory, "application-update.lock")
	applicationUpdatePolicyUnitBusy = func() (bool, error) { return false, nil }
	verifyUpdateSignatureChallenge = oldVerify
	t.Cleanup(func() {
		updateSignatureVerificationFlagPath = oldFlag
		applicationUpdatePolicyLockPath = oldLock
		applicationUpdatePolicyUnitBusy = oldBusy
		verifyUpdateSignatureChallenge = oldVerify
	})
	return updateSignatureVerificationFlagPath
}

func TestUpdateSignatureVerificationDefaultsEnabledAndPersistsDisabled(t *testing.T) {
	flag := withUpdateSignatureVerificationTestState(t)

	enabled, err := updateSignatureVerificationEnabled()
	if err != nil || !enabled {
		t.Fatalf("default enabled = %t, err = %v", enabled, err)
	}
	if err := setUpdateSignatureVerificationEnabled(false); err != nil {
		t.Fatalf("disable signature verification: %v", err)
	}
	enabled, err = updateSignatureVerificationEnabled()
	if err != nil || enabled {
		t.Fatalf("persisted enabled = %t, err = %v", enabled, err)
	}
	info, err := os.Lstat(flag)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		t.Fatalf("marker mode = %v", info.Mode())
	}
	if err := setUpdateSignatureVerificationEnabled(true); err != nil {
		t.Fatalf("enable signature verification: %v", err)
	}
	if _, err := os.Lstat(flag); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("enabled marker still exists: %v", err)
	}
}

func TestUpdateSignatureVerificationMalformedMarkerFailsClosed(t *testing.T) {
	for _, test := range []struct {
		name  string
		write func(*testing.T, string)
	}{
		{
			name: "invalid contents",
			write: func(t *testing.T, flag string) {
				if err := os.WriteFile(flag, []byte("enabled\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "oversized",
			write: func(t *testing.T, flag string) {
				if err := os.WriteFile(flag, []byte(updateSignatureVerificationDisabledMarker+"x"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "unsafe permissions",
			write: func(t *testing.T, flag string) {
				if err := os.WriteFile(flag, []byte(updateSignatureVerificationDisabledMarker), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "symlink",
			write: func(t *testing.T, flag string) {
				target := flag + ".target"
				if err := os.WriteFile(target, []byte(updateSignatureVerificationDisabledMarker), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, flag); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "directory",
			write: func(t *testing.T, flag string) {
				if err := os.Mkdir(flag, 0o700); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			flag := withUpdateSignatureVerificationTestState(t)
			test.write(t, flag)
			enabled, err := updateSignatureVerificationEnabled()
			if err == nil || !enabled {
				t.Fatalf("malformed marker enabled = %t, err = %v", enabled, err)
			}
		})
	}
}

func TestDisablingSignatureVerificationAtomicallyReplacesUnsafeMarker(t *testing.T) {
	flag := withUpdateSignatureVerificationTestState(t)
	target := flag + ".target"
	if err := os.WriteFile(target, []byte("must remain unchanged\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, flag); err != nil {
		t.Fatal(err)
	}

	if err := setUpdateSignatureVerificationEnabled(false); err != nil {
		t.Fatalf("disable signature verification: %v", err)
	}
	enabled, err := updateSignatureVerificationEnabled()
	if err != nil || enabled {
		t.Fatalf("persisted enabled = %t, err = %v", enabled, err)
	}
	contents, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "must remain unchanged\n" {
		t.Fatalf("symlink target was changed: %q", contents)
	}
}

func TestSetSignatureVerificationRequiresAndConsumesChallengeBeforeMutation(t *testing.T) {
	withUpdateSignatureVerificationTestState(t)
	gin.SetMode(gin.TestMode)
	oldVerify := verifyUpdateSignatureChallenge
	t.Cleanup(func() { verifyUpdateSignatureChallenge = oldVerify })

	verifiedChallenge := ""
	verifiedAction := ""
	verifyUpdateSignatureChallenge = func(_ *gin.Context, challenge, action string) error {
		verifiedChallenge = challenge
		verifiedAction = action
		if challenge != "one-time-proof" {
			return errors.New("invalid challenge")
		}
		return nil
	}

	response := performSignatureVerificationRequest(t, `{"enabled":false}`)
	if response.Code != -3 {
		t.Fatalf("missing challenge code = %d, want -3", response.Code)
	}
	enabled, err := updateSignatureVerificationEnabled()
	if err != nil || !enabled {
		t.Fatalf("failed auth changed setting: enabled=%t err=%v", enabled, err)
	}

	response = performSignatureVerificationRequest(t, `{"enabled":false,"challenge":"one-time-proof"}`)
	if response.Code != 0 {
		t.Fatalf("authorized response = %+v", response)
	}
	if verifiedChallenge != "one-time-proof" || verifiedAction != "update_signature_verification" {
		t.Fatalf("challenge verification = %q/%q", verifiedChallenge, verifiedAction)
	}
	enabled, err = updateSignatureVerificationEnabled()
	if err != nil || enabled {
		t.Fatalf("authorized setting enabled=%t err=%v", enabled, err)
	}
}

func TestSetSignatureVerificationRejectsBusyUpdate(t *testing.T) {
	withUpdateSignatureVerificationTestState(t)
	gin.SetMode(gin.TestMode)
	oldVerify := verifyUpdateSignatureChallenge
	verifyUpdateSignatureChallenge = func(*gin.Context, string, string) error { return nil }
	applicationUpdatePolicyUnitBusy = func() (bool, error) { return true, nil }
	t.Cleanup(func() { verifyUpdateSignatureChallenge = oldVerify })

	response := performSignatureVerificationRequest(t, `{"enabled":false,"challenge":"proof"}`)
	if response.Code != -2 {
		t.Fatalf("busy response = %+v", response)
	}
	enabled, err := updateSignatureVerificationEnabled()
	if err != nil || !enabled {
		t.Fatalf("busy change mutated setting: enabled=%t err=%v", enabled, err)
	}
}

func performSignatureVerificationRequest(t *testing.T, body string) testApplicationResponse {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/application/signature-verification", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	NewService().SetSignatureVerification(ctx)

	var response testApplicationResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v body=%s", err, recorder.Body.String())
	}
	return response
}

type testApplicationResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}
