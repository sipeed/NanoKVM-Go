package vm

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"NanoKVM-Server/config"

	"github.com/gin-gonic/gin"
)

type onboardingTestResponse struct {
	Code int `json:"code"`
	Data *struct {
		CompletedVersion       string `json:"completedVersion"`
		MobileCompletedVersion string `json:"mobileCompletedVersion"`
	} `json:"data"`
}

func setupOnboardingTest(t *testing.T) {
	t.Helper()

	originalGet := getOnboardingCompletedVersion
	originalSet := setOnboardingCompletedVersion
	originalMobileGet := getMobileOnboardingCompletedVersion
	originalMobileSet := setMobileOnboardingCompletedVersion
	t.Cleanup(func() {
		getOnboardingCompletedVersion = originalGet
		setOnboardingCompletedVersion = originalSet
		getMobileOnboardingCompletedVersion = originalMobileGet
		setMobileOnboardingCompletedVersion = originalMobileSet
	})
}

func TestGetOnboardingReturnsBothCompletedVersions(t *testing.T) {
	setupOnboardingTest(t)
	getOnboardingCompletedVersion = func() string { return "v1" }
	getMobileOnboardingCompletedVersion = func() string { return "mobile-v1" }

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/vm/onboarding", nil)

	NewService().GetOnboarding(ctx)

	var rsp onboardingTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("unmarshal onboarding response: %v body=%s", err, recorder.Body.String())
	}
	if rsp.Code != 0 || rsp.Data == nil {
		t.Fatalf("GET onboarding response = %+v, want code 0 with data", rsp)
	}
	if rsp.Data.CompletedVersion != "v1" {
		t.Fatalf("GET onboarding completedVersion = %q, want %q", rsp.Data.CompletedVersion, "v1")
	}
	if rsp.Data.MobileCompletedVersion != "mobile-v1" {
		t.Fatalf("GET onboarding mobileCompletedVersion = %q, want %q",
			rsp.Data.MobileCompletedVersion, "mobile-v1")
	}
}

func performSetOnboardingBody(t *testing.T, body []byte) onboardingTestResponse {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/vm/onboarding", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	NewService().SetOnboarding(ctx)

	var rsp onboardingTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("unmarshal onboarding response: %v body=%s", err, recorder.Body.String())
	}
	return rsp
}

func TestSetOnboardingRejectsMissingOrEmptyVersions(t *testing.T) {
	setupOnboardingTest(t)
	setOnboardingCompletedVersion = func(string) error {
		t.Fatal("invalid request must not reach the desktop config writer")
		return nil
	}
	setMobileOnboardingCompletedVersion = func(string) error {
		t.Fatal("invalid request must not reach the mobile config writer")
		return nil
	}

	for _, body := range []string{
		`{}`,
		`{"completedVersion":""}`,
		`{"mobileCompletedVersion":""}`,
		`{"completedVersion":"","mobileCompletedVersion":""}`,
		`{"completedVersion":"` + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" + `"}`, // 33 chars, exceeds max=32
		`{"mobileCompletedVersion":"` + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" + `"}`,
	} {
		rsp := performSetOnboardingBody(t, []byte(body))
		if rsp.Code != -1 {
			t.Fatalf("body %s response code = %d, want -1", body, rsp.Code)
		}
	}
}

func TestSetOnboardingValidatesAllProvidedFieldsBeforeWriting(t *testing.T) {
	setupOnboardingTest(t)
	setOnboardingCompletedVersion = func(string) error {
		t.Fatal("an invalid mobile version must not reach the desktop config writer")
		return nil
	}
	setMobileOnboardingCompletedVersion = func(string) error {
		t.Fatal("invalid request must not reach the mobile config writer")
		return nil
	}

	rsp := performSetOnboardingBody(t, []byte(`{"completedVersion":"v1","mobileCompletedVersion":"INVALID!!"}`))
	if rsp.Code != -1 {
		t.Fatalf("invalid mobile version response code = %d, want -1", rsp.Code)
	}
}

func TestSetOnboardingMapsInvalidVersionToInvalidArguments(t *testing.T) {
	setupOnboardingTest(t)
	setOnboardingCompletedVersion = func(string) error { return config.ErrInvalidOnboardingVersion }

	rsp := performSetOnboardingBody(t, []byte(`{"completedVersion":"INVALID!!"}`))
	if rsp.Code != -1 {
		t.Fatalf("invalid version response code = %d, want -1", rsp.Code)
	}
}

func TestSetOnboardingReportsPersistenceFailure(t *testing.T) {
	setupOnboardingTest(t)
	setOnboardingCompletedVersion = func(string) error { return errors.New("disk full") }

	rsp := performSetOnboardingBody(t, []byte(`{"completedVersion":"v1"}`))
	if rsp.Code != -2 {
		t.Fatalf("persistence failure response code = %d, want -2", rsp.Code)
	}
}

func TestSetOnboardingReportsMobilePersistenceFailure(t *testing.T) {
	setupOnboardingTest(t)
	setMobileOnboardingCompletedVersion = func(string) error { return errors.New("disk full") }

	rsp := performSetOnboardingBody(t, []byte(`{"mobileCompletedVersion":"mobile-v1"}`))
	if rsp.Code != -2 {
		t.Fatalf("mobile persistence failure response code = %d, want -2", rsp.Code)
	}
}

func TestSetOnboardingAcceptsValidDesktopVersion(t *testing.T) {
	setupOnboardingTest(t)
	var written string
	setOnboardingCompletedVersion = func(version string) error {
		written = version
		return nil
	}
	setMobileOnboardingCompletedVersion = func(string) error {
		t.Fatal("desktop-only request must not reach the mobile config writer")
		return nil
	}

	rsp := performSetOnboardingBody(t, []byte(`{"completedVersion":"v1"}`))
	if rsp.Code != 0 {
		t.Fatalf("valid version response code = %d, want 0", rsp.Code)
	}
	if written != "v1" {
		t.Fatalf("config writer received %q, want %q", written, "v1")
	}
}

func TestSetOnboardingAcceptsValidMobileVersion(t *testing.T) {
	setupOnboardingTest(t)
	var written string
	setOnboardingCompletedVersion = func(string) error {
		t.Fatal("mobile-only request must not reach the desktop config writer")
		return nil
	}
	setMobileOnboardingCompletedVersion = func(version string) error {
		written = version
		return nil
	}

	rsp := performSetOnboardingBody(t, []byte(`{"mobileCompletedVersion":"mobile-v1"}`))
	if rsp.Code != 0 {
		t.Fatalf("valid mobile version response code = %d, want 0", rsp.Code)
	}
	if written != "mobile-v1" {
		t.Fatalf("mobile config writer received %q, want %q", written, "mobile-v1")
	}
}

func TestSetOnboardingAcceptsBothVersions(t *testing.T) {
	setupOnboardingTest(t)
	var desktopWritten, mobileWritten string
	setOnboardingCompletedVersion = func(version string) error {
		desktopWritten = version
		return nil
	}
	setMobileOnboardingCompletedVersion = func(version string) error {
		mobileWritten = version
		return nil
	}

	rsp := performSetOnboardingBody(t, []byte(`{"completedVersion":"v1","mobileCompletedVersion":"mobile-v1"}`))
	if rsp.Code != 0 {
		t.Fatalf("dual version response code = %d, want 0", rsp.Code)
	}
	if desktopWritten != "v1" {
		t.Fatalf("desktop config writer received %q, want %q", desktopWritten, "v1")
	}
	if mobileWritten != "mobile-v1" {
		t.Fatalf("mobile config writer received %q, want %q", mobileWritten, "mobile-v1")
	}
}
