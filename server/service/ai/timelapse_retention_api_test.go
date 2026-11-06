package ai

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"

	"github.com/gin-gonic/gin"
)

type screenTimelapseRetentionAPIEnvelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

type screenTimelapseRetentionAPITestEnv struct {
	config ScreenTimelapseRetentionConfig
	status ScreenTimelapseRetentionStatus
	root   string
	now    time.Time

	loadCalls           int
	updateCalls         int
	inventoryCalls      int
	planCalls           int
	saveTriggerCalls    int
	manualTriggerCalls  int
	challengeActions    []string
	challengeValues     []string
	requireChallenge    bool
	verifyChallengeErr  error
	updateErr           error
	triggerAvailable    bool
	saveTriggerAccepted bool
}

func withScreenTimelapseRetentionAPITest(t *testing.T) *screenTimelapseRetentionAPITestEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	oldLoadConfig := screenTimelapseRetentionAPILoadConfig
	oldUpdateCleanup := screenTimelapseRetentionAPIUpdateCleanup
	oldInventory := screenTimelapseRetentionAPIInventory
	oldPlan := screenTimelapseRetentionAPIPlan
	oldRoot := screenTimelapseRetentionAPIRoot
	oldNow := screenTimelapseRetentionAPINow
	oldStatus := screenTimelapseRetentionAPIStatus
	oldTrigger := screenTimelapseRetentionAPITrigger
	oldActionRequiresChallenge := screenTimelapseActionRequiresChallenge
	oldVerifyChallenge := verifyScreenTimelapseChallenge

	env := &screenTimelapseRetentionAPITestEnv{
		config:              defaultScreenTimelapseRetentionConfig(),
		root:                t.TempDir(),
		now:                 time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC),
		triggerAvailable:    true,
		saveTriggerAccepted: true,
	}
	screenTimelapseRetentionAPILoadConfig = func() (ScreenTimelapseRetentionConfig, error) {
		env.loadCalls++
		return env.config, nil
	}
	screenTimelapseRetentionAPIUpdateCleanup = func(expectedRevision int64, cleanup ScreenTimelapseRetentionCleanupConfig) (ScreenTimelapseRetentionConfig, bool, bool, bool, error) {
		env.updateCalls++
		if env.updateErr != nil {
			return env.config, false, false, false, env.updateErr
		}
		if expectedRevision != env.config.Revision {
			return env.config, false, true, false, nil
		}
		if env.config.Cleanup == cleanup {
			return env.config, false, false, false, nil
		}
		env.config.Cleanup = cleanup
		env.config.Revision++
		env.saveTriggerCalls++
		return env.config, true, false, env.saveTriggerAccepted, nil
	}
	screenTimelapseRetentionAPIInventory = func(root string, now time.Time) (ScreenTimelapseRetentionInventory, error) {
		env.inventoryCalls++
		return inventoryScreenTimelapseRetention(root, now)
	}
	screenTimelapseRetentionAPIPlan = func(inventory ScreenTimelapseRetentionInventory, now time.Time, config ScreenTimelapseRetentionConfig) (ScreenTimelapseRetentionPlan, error) {
		env.planCalls++
		return planScreenTimelapseRetention(inventory, now, config)
	}
	screenTimelapseRetentionAPIRoot = func() string { return env.root }
	screenTimelapseRetentionAPINow = func() time.Time { return env.now }
	screenTimelapseRetentionAPIStatus = func() ScreenTimelapseRetentionStatus { return env.status }
	screenTimelapseRetentionAPITrigger = func() bool {
		env.manualTriggerCalls++
		return env.triggerAvailable
	}
	screenTimelapseActionRequiresChallenge = func(string) bool { return env.requireChallenge }
	verifyScreenTimelapseChallenge = func(_ *gin.Context, challenge, action string) error {
		env.challengeActions = append(env.challengeActions, action)
		env.challengeValues = append(env.challengeValues, challenge)
		return env.verifyChallengeErr
	}

	t.Cleanup(func() {
		screenTimelapseRetentionAPILoadConfig = oldLoadConfig
		screenTimelapseRetentionAPIUpdateCleanup = oldUpdateCleanup
		screenTimelapseRetentionAPIInventory = oldInventory
		screenTimelapseRetentionAPIPlan = oldPlan
		screenTimelapseRetentionAPIRoot = oldRoot
		screenTimelapseRetentionAPINow = oldNow
		screenTimelapseRetentionAPIStatus = oldStatus
		screenTimelapseRetentionAPITrigger = oldTrigger
		screenTimelapseActionRequiresChallenge = oldActionRequiresChallenge
		verifyScreenTimelapseChallenge = oldVerifyChallenge
	})
	return env
}

func decodeScreenTimelapseRetentionAPIResponse(t *testing.T, recorderBody []byte) screenTimelapseRetentionAPIEnvelope {
	t.Helper()
	var rsp screenTimelapseRetentionAPIEnvelope
	if err := json.Unmarshal(recorderBody, &rsp); err != nil {
		t.Fatalf("decode response: %v body=%s", err, string(recorderBody))
	}
	return rsp
}

func decodeScreenTimelapseRetentionAPIData[T any](t *testing.T, rsp screenTimelapseRetentionAPIEnvelope) T {
	t.Helper()
	var data T
	if err := json.Unmarshal(rsp.Data, &data); err != nil {
		t.Fatalf("decode response data: %v data=%s", err, string(rsp.Data))
	}
	return data
}

func TestGetScreenTimelapseRetentionReturnsCachedStatusWithoutInventory(t *testing.T) {
	env := withScreenTimelapseRetentionAPITest(t)
	env.config.Revision = 12
	env.config.Cleanup = ScreenTimelapseRetentionCleanupConfig{Enabled: true, RetentionDays: 7, MaxStorageMiB: 2048}
	env.status = ScreenTimelapseRetentionStatus{
		ScannedAt:        env.now,
		UsageBytes:       123,
		ManagedDays:      2,
		LastCleanupAt:    env.now.Add(time.Minute),
		LastTrigger:      screenTimelapseRetentionTriggerScheduled,
		LastDeletedDays:  1,
		LastDeletedBytes: 10,
	}

	service := &Service{}
	c, recorder := newTestContext(http.MethodGet, "/api/ai/screen-timelapse/retention", "")
	service.GetScreenTimelapseRetention(c)
	rsp := decodeScreenTimelapseRetentionAPIResponse(t, recorder.Body.Bytes())
	if rsp.Code != 0 || env.inventoryCalls != 0 || env.planCalls != 0 {
		t.Fatalf("GET response=%#v inventory=%d plan=%d", rsp, env.inventoryCalls, env.planCalls)
	}
	data := decodeScreenTimelapseRetentionAPIData[proto.GetScreenTimelapseRetentionRsp](t, rsp)
	if data.Config.Revision != 12 || data.Config.Cleanup.RetentionDays != 7 || data.Status.ScannedAt != env.now.Format(time.RFC3339Nano) ||
		data.Status.LastCleanupAt != env.now.Add(time.Minute).Format(time.RFC3339Nano) || data.Status.UsageBytes != 123 {
		t.Fatalf("unexpected GET data: %#v", data)
	}
}

func TestSetScreenTimelapseRetentionSavesWithRevisionAndWakesWorker(t *testing.T) {
	env := withScreenTimelapseRetentionAPITest(t)
	env.config.Revision = 12

	service := &Service{}
	c, recorder := newTestContext(http.MethodPut, "/api/ai/screen-timelapse/retention", `{"revision":12,"cleanup":{"enabled":true,"retentionDays":7,"maxStorageMiB":2048},"challenge":""}`)
	service.SetScreenTimelapseRetention(c)
	rsp := decodeScreenTimelapseRetentionAPIResponse(t, recorder.Body.Bytes())
	if rsp.Code != 0 {
		t.Fatalf("unexpected save response: %#v", rsp)
	}
	data := decodeScreenTimelapseRetentionAPIData[proto.SetScreenTimelapseRetentionRsp](t, rsp)
	if !data.Changed || !data.CleanupTriggered || data.Config.Revision != 13 || !data.Config.Cleanup.Enabled ||
		data.Config.Cleanup.RetentionDays != 7 || data.Config.Cleanup.MaxStorageMiB != 2048 || env.saveTriggerCalls != 1 {
		t.Fatalf("unexpected save data=%#v env=%#v", data, env)
	}
}

func TestSetScreenTimelapseRetentionRevisionConflictDoesNotVerifyOrOverwrite(t *testing.T) {
	env := withScreenTimelapseRetentionAPITest(t)
	env.config.Revision = 9
	env.requireChallenge = true

	service := &Service{}
	c, recorder := newTestContext(http.MethodPut, "/api/ai/screen-timelapse/retention", `{"revision":8,"cleanup":{"enabled":true,"retentionDays":7,"maxStorageMiB":0},"challenge":"unused"}`)
	service.SetScreenTimelapseRetention(c)
	rsp := decodeScreenTimelapseRetentionAPIResponse(t, recorder.Body.Bytes())
	if rsp.Code != -3 || env.updateCalls != 0 || len(env.challengeActions) != 0 {
		t.Fatalf("conflict response=%#v env=%#v", rsp, env)
	}
	data := decodeScreenTimelapseRetentionAPIData[proto.SetScreenTimelapseRetentionRsp](t, rsp)
	if data.Config.Revision != 9 || data.Changed || data.CleanupTriggered {
		t.Fatalf("conflict data=%#v", data)
	}
}

func TestScreenTimelapseRetentionSensitiveActionsRequireMatchingChallenge(t *testing.T) {
	env := withScreenTimelapseRetentionAPITest(t)
	env.requireChallenge = true
	env.verifyChallengeErr = errors.New("challenge rejected")
	service := &Service{}

	c, recorder := newTestContext(http.MethodPut, "/api/ai/screen-timelapse/retention", `{"revision":0,"cleanup":{"enabled":true,"retentionDays":7,"maxStorageMiB":0},"challenge":"opaque-put"}`)
	service.SetScreenTimelapseRetention(c)
	rsp := decodeScreenTimelapseRetentionAPIResponse(t, recorder.Body.Bytes())
	if rsp.Code != -4 || env.updateCalls != 0 || !stringSlicesEqual(env.challengeActions, []string{authservice.SensitiveActionSetScreenTimelapseRetention}) ||
		!stringSlicesEqual(env.challengeValues, []string{"opaque-put"}) || strings.Contains(recorder.Body.String(), "opaque-put") {
		t.Fatalf("PUT verification response=%#v env=%#v", rsp, env)
	}

	env.verifyChallengeErr = nil
	c, recorder = newTestContext(http.MethodPost, "/api/ai/screen-timelapse/cleanup", `{"dryRun":false,"challenge":"opaque-cleanup"}`)
	service.CleanupScreenTimelapseRetention(c)
	rsp = decodeScreenTimelapseRetentionAPIResponse(t, recorder.Body.Bytes())
	if rsp.Code != 0 || env.manualTriggerCalls != 1 || !stringSlicesEqual(env.challengeActions, []string{
		authservice.SensitiveActionSetScreenTimelapseRetention,
		authservice.SensitiveActionTriggerScreenTimelapseCleanup,
	}) || strings.Contains(recorder.Body.String(), "opaque-cleanup") {
		t.Fatalf("cleanup verification response=%#v env=%#v", rsp, env)
	}
}

func TestScreenTimelapseRetentionRejectsInvalidAndOversizedRequests(t *testing.T) {
	env := withScreenTimelapseRetentionAPITest(t)
	service := &Service{}

	for _, body := range []string{
		`{"revision":0,"cleanup":{"enabled":true,"retentionDays":7,"maxStorageMiB":0},"unknown":true}`,
		`{"revision":0,"cleanup":{"enabled":true,"retentionDays":0,"maxStorageMiB":0}}`,
		`{"revision":0,"cleanup":{"enabled":true,"retentionDays":7,"maxStorageMiB":1.5}}`,
	} {
		c, recorder := newTestContext(http.MethodPut, "/api/ai/screen-timelapse/retention", body)
		service.SetScreenTimelapseRetention(c)
		if rsp := decodeScreenTimelapseRetentionAPIResponse(t, recorder.Body.Bytes()); rsp.Code != -1 {
			t.Fatalf("invalid request body=%s response=%#v", body, rsp)
		}
	}

	oversized := `{"dryRun":true,"challenge":"` + strings.Repeat("x", screenTimelapseRetentionRequestMaxBytes) + `"}`
	c, recorder := newTestContext(http.MethodPost, "/api/ai/screen-timelapse/cleanup", oversized)
	service.CleanupScreenTimelapseRetention(c)
	if rsp := decodeScreenTimelapseRetentionAPIResponse(t, recorder.Body.Bytes()); rsp.Code != -1 {
		t.Fatalf("oversized request response=%#v", rsp)
	}
	if env.updateCalls != 0 || env.inventoryCalls != 0 || env.manualTriggerCalls != 0 {
		t.Fatalf("invalid requests mutated state: %#v", env)
	}
}

func TestScreenTimelapseRetentionDryRunUsesCandidateConfigWithoutMutation(t *testing.T) {
	env := withScreenTimelapseRetentionAPITest(t)
	oldDate := "2026-07-19"
	todayDate := "2026-07-20"
	_, _, oldBytes := writeTimelapseMutationTestDay(t, env.root, oldDate, env.now.Add(-time.Hour))
	_, _, todayBytes := writeTimelapseMutationTestDay(t, env.root, todayDate, env.now)

	service := &Service{}
	c, recorder := newTestContext(http.MethodPost, "/api/ai/screen-timelapse/cleanup", `{"dryRun":true,"config":{"enabled":true,"retentionDays":1,"maxStorageMiB":0}}`)
	service.CleanupScreenTimelapseRetention(c)
	rsp := decodeScreenTimelapseRetentionAPIResponse(t, recorder.Body.Bytes())
	if rsp.Code != 0 || env.loadCalls != 0 || env.updateCalls != 0 || env.manualTriggerCalls != 0 || env.inventoryCalls != 1 || env.planCalls != 1 {
		t.Fatalf("dry run response=%#v env=%#v", rsp, env)
	}
	data := decodeScreenTimelapseRetentionAPIData[proto.ScreenTimelapseRetentionPreviewRsp](t, rsp)
	if !data.DryRun || data.BeforeBytes != oldBytes+todayBytes || data.ProjectedAfterBytes != todayBytes || data.WouldDeleteDays != 1 ||
		data.WouldDeleteBytes != oldBytes || len(data.Candidates) != 1 || data.Candidates[0].Date != oldDate ||
		len(data.Skipped) != 1 || data.Skipped[0].Date != todayDate || data.Skipped[0].Reason != screenTimelapseRetentionSkipToday {
		t.Fatalf("unexpected dry-run data: %#v", data)
	}
}

func TestScreenTimelapseRetentionTriggerUsesOnlyPersistedConfigAndCoalescesRequests(t *testing.T) {
	env := withScreenTimelapseRetentionAPITest(t)
	service := &Service{}

	c, recorder := newTestContext(http.MethodPost, "/api/ai/screen-timelapse/cleanup", `{"dryRun":false,"config":{"enabled":true,"retentionDays":1,"maxStorageMiB":0}}`)
	service.CleanupScreenTimelapseRetention(c)
	if rsp := decodeScreenTimelapseRetentionAPIResponse(t, recorder.Body.Bytes()); rsp.Code != -1 || env.manualTriggerCalls != 0 {
		t.Fatalf("trigger with candidate config response=%#v env=%#v", rsp, env)
	}

	for index := 0; index < 2; index++ {
		c, recorder = newTestContext(http.MethodPost, "/api/ai/screen-timelapse/cleanup", `{"dryRun":false}`)
		service.CleanupScreenTimelapseRetention(c)
		rsp := decodeScreenTimelapseRetentionAPIResponse(t, recorder.Body.Bytes())
		if rsp.Code != 0 {
			t.Fatalf("repeat trigger response=%#v", rsp)
		}
		data := decodeScreenTimelapseRetentionAPIData[proto.ScreenTimelapseRetentionTriggerRsp](t, rsp)
		if !data.Accepted {
			t.Fatalf("repeat trigger was not accepted: %#v", data)
		}
	}
	if env.manualTriggerCalls != 2 || env.loadCalls != 2 {
		t.Fatalf("unexpected repeated trigger env=%#v", env)
	}
}

func TestScreenTimelapseRetentionSaveFailureDoesNotReportOrTriggerCleanup(t *testing.T) {
	env := withScreenTimelapseRetentionAPITest(t)
	env.updateErr = errors.New("simulated save failure")
	service := &Service{}

	c, recorder := newTestContext(http.MethodPut, "/api/ai/screen-timelapse/retention", `{"revision":0,"cleanup":{"enabled":true,"retentionDays":7,"maxStorageMiB":0}}`)
	service.SetScreenTimelapseRetention(c)
	rsp := decodeScreenTimelapseRetentionAPIResponse(t, recorder.Body.Bytes())
	if rsp.Code != -2 || env.saveTriggerCalls != 0 || env.manualTriggerCalls != 0 {
		t.Fatalf("save failure response=%#v env=%#v", rsp, env)
	}
}
