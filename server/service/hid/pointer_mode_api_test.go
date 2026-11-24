package hid

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

type setPointerModeTestResponse struct {
	Code int                     `json:"code"`
	Msg  string                  `json:"msg"`
	Data proto.GetPointerModeRsp `json:"data"`
}

func performSetPointerModeRequest(t *testing.T, service *Service, body string) setPointerModeTestResponse {
	t.Helper()
	return performSetPointerModeRequestWithContext(t, service, body, context.Background())
}

func performSetPointerModeRequestWithContext(t *testing.T, service *Service, body string, requestContext context.Context) setPointerModeTestResponse {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/hid/pointer-mode", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	if requestContext != nil {
		request = request.WithContext(requestContext)
	}
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = request
	service.SetPointerMode(ginContext)
	if recorder.Code != http.StatusOK {
		t.Fatalf("SetPointerMode HTTP status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response setPointerModeTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode SetPointerMode response: %v; body=%s", err, recorder.Body.String())
	}
	return response
}

func serviceForPointerModeAPI(harness *pointerModeTransactionHarness) *Service {
	return &Service{
		pointerModeManager: harness.manager,
		pointerModeSwitch:  harness.transaction,
	}
}

func TestSetPointerModeReturnsFinalConfirmedMode(t *testing.T) {
	marker := "absolute\n"
	harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, ModeNormal)
	response := performSetPointerModeRequest(t, serviceForPointerModeAPI(harness), `{"mode":"digitizer-stylus"}`)

	if response.Code != 0 || response.Msg != "success" {
		t.Fatalf("response = %+v", response)
	}
	if response.Data.Mode != string(PointerModeDigitizerStylus) || response.Data.Switching || response.Data.Degraded {
		t.Fatalf("final response data = %+v", response.Data)
	}
	data, err := os.ReadFile(harness.modeFile)
	if err != nil || string(data) != "digitizer-stylus\n" {
		t.Fatalf("final marker = %q, %v", data, err)
	}
	harness.assertLocksReleased(t)
}

func TestSetPointerModeSameModeIsIdempotent(t *testing.T) {
	marker := "absolute\n"
	harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, ModeNormal)
	response := performSetPointerModeRequest(t, serviceForPointerModeAPI(harness), `{"mode":"absolute"}`)

	if response.Code != 0 || response.Data.Mode != string(PointerModeAbsolute) {
		t.Fatalf("response = %+v", response)
	}
	if events := harness.recorder.snapshot(); strings.Join(events, ",") != "usb-lock,usb-unlock" {
		t.Fatalf("idempotent events = %v", events)
	}
	_, _, scripts, verifies := harness.callSnapshot()
	if len(scripts) != 0 || len(verifies) != 0 {
		t.Fatalf("idempotent rebuilt gadget: scripts=%v verifies=%v", scripts, verifies)
	}
	harness.assertLocksReleased(t)
}

func TestSetPointerModeBusyStoragePreservesHealthyInput(t *testing.T) {
	marker := "absolute\n"
	harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, ModeNormal)
	transfer, err := harness.storageLock.acquire()
	if err != nil {
		t.Fatal(err)
	}
	defer transfer.Close()
	response := performSetPointerModeRequest(t, serviceForPointerModeAPI(harness), `{"mode":"touch-screen"}`)
	if response.Code != pointerModeAPILockCode || response.Data.Mode != string(PointerModeAbsolute) || response.Data.Switching || response.Data.Degraded {
		t.Fatalf("busy switch damaged input state: %+v", response)
	}
	begin, end, scripts, _ := harness.callSnapshot()
	if begin != 0 || end != 0 || len(scripts) != 0 {
		t.Fatalf("busy switch paused input or rebuilt USB: begin=%d end=%d scripts=%v", begin, end, scripts)
	}
}

func TestSetPointerModeRejectsEmptyUnknownAndLegacyModes(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
	}{
		{name: "empty", body: `{"mode":""}`},
		{name: "unknown", body: `{"mode":"relative"}`},
		{name: "legacy", body: `{"mode":"absolute-android"}`},
		{name: "whitespace", body: `{"mode":" absolute"}`},
		{name: "missing", body: `{}`},
		{name: "malformed", body: `{"mode":`},
		{name: "unknown field", body: `{"mode":"absolute","extra":true}`},
		{name: "multiple values", body: `{"mode":"absolute"}{}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			marker := "absolute\n"
			harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, ModeNormal)
			response := performSetPointerModeRequest(t, serviceForPointerModeAPI(harness), test.body)
			if response.Code != pointerModeAPIInvalidArgumentsCode || response.Msg != "invalid pointer mode request" && response.Msg != "invalid pointer mode" {
				t.Fatalf("response = %+v", response)
			}
			if events := harness.recorder.snapshot(); len(events) != 0 {
				t.Fatalf("invalid request mutated state: %v", events)
			}
		})
	}
}

func TestSetPointerModeRejectsWhileSwitching(t *testing.T) {
	marker := "absolute\n"
	harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, ModeNormal)
	harness.manager.mutex.Lock()
	harness.manager.switching = true
	harness.manager.mutex.Unlock()

	response := performSetPointerModeRequest(t, serviceForPointerModeAPI(harness), `{"mode":"touch-screen"}`)
	if response.Code != pointerModeAPIInProgressCode || response.Msg != "pointer mode switch already in progress" {
		t.Fatalf("response = %+v", response)
	}
	if !response.Data.Switching || response.Data.Mode != string(PointerModeAbsolute) {
		t.Fatalf("switching response data = %+v", response.Data)
	}
	if events := harness.recorder.snapshot(); len(events) != 0 {
		t.Fatalf("in-progress request mutated state: %v", events)
	}
}

func TestSetPointerModeConcurrentRequestReturnsInProgress(t *testing.T) {
	marker := "absolute\n"
	harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, ModeNormal)
	service := serviceForPointerModeAPI(harness)
	entered := make(chan struct{})
	release := make(chan struct{})
	harness.scriptHook = func(call int, _ PointerMode) {
		if call != 0 {
			return
		}
		close(entered)
		<-release
	}

	firstDone := make(chan setPointerModeTestResponse, 1)
	go func() {
		firstDone <- performSetPointerModeRequest(t, service, `{"mode":"digitizer-stylus"}`)
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("first API request did not reach the rebuild script")
	}

	second := performSetPointerModeRequest(t, service, `{"mode":"touch-screen"}`)
	if second.Code != pointerModeAPIInProgressCode || !second.Data.Switching {
		t.Fatalf("concurrent response = %+v", second)
	}
	close(release)
	first := <-firstDone
	if first.Code != 0 || first.Data.Mode != string(PointerModeDigitizerStylus) {
		t.Fatalf("first response = %+v", first)
	}
	harness.assertLocksReleased(t)
}

func TestSetPointerModeMapsScriptFailureAndReturnsRolledBackState(t *testing.T) {
	marker := "absolute\n"
	harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, ModeNormal)
	harness.scriptErrors = []error{errors.New("script unavailable"), nil}

	response := performSetPointerModeRequest(t, serviceForPointerModeAPI(harness), `{"mode":"digitizer-stylus"}`)
	if response.Code != pointerModeAPIScriptCode || !strings.Contains(response.Msg, "USB rebuild") {
		t.Fatalf("response = %+v", response)
	}
	if response.Data.Mode != string(PointerModeAbsolute) || response.Data.Degraded || response.Data.Switching {
		t.Fatalf("rolled-back response data = %+v", response.Data)
	}
	data, err := os.ReadFile(harness.modeFile)
	if err != nil || string(data) != marker {
		t.Fatalf("rolled-back marker = %q, %v", data, err)
	}
	harness.assertLocksReleased(t)
}

func TestSetPointerModeReportsDegradedRollbackFailure(t *testing.T) {
	marker := "absolute\n"
	harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, ModeNormal)
	harness.scriptErrors = []error{errors.New("forward failed"), errors.New("rollback failed")}

	response := performSetPointerModeRequest(t, serviceForPointerModeAPI(harness), `{"mode":"digitizer-stylus"}`)
	if response.Code != pointerModeAPIDegradedCode || !strings.Contains(response.Msg, "degraded") {
		t.Fatalf("response = %+v", response)
	}
	if response.Data.Mode != string(PointerModeUnknown) || !response.Data.Degraded || response.Data.Switching || response.Data.Error == "" {
		t.Fatalf("degraded response data = %+v", response.Data)
	}
	if _, err := os.Stat(harness.modeFile); err != nil {
		t.Fatalf("degraded marker was not restored: %v", err)
	}
	if _, end, _, _ := harness.callSnapshot(); end != 0 {
		t.Fatalf("degraded request resumed input unexpectedly: end=%d", end)
	}
	harness.assertLocksReleased(t)
}

func TestSetPointerModeContinuesAfterRequestContextCancellation(t *testing.T) {
	marker := "absolute\n"
	harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, ModeNormal)
	requestContext, cancel := context.WithCancel(context.Background())
	cancel()

	response := performSetPointerModeRequestWithContext(t, serviceForPointerModeAPI(harness), `{"mode":"touch-screen"}`, requestContext)
	if response.Code != 0 || response.Data.Mode != string(PointerModeTouchScreen) || response.Data.Degraded {
		t.Fatalf("response after canceled request context = %+v", response)
	}
	if _, _, scripts, verifies := harness.callSnapshot(); len(scripts) != 1 || len(verifies) != 1 {
		t.Fatalf("canceled request did not complete transaction: scripts=%v verifies=%v", scripts, verifies)
	}
	harness.assertLocksReleased(t)
}

func TestPointerModeAPIErrorMappingPreservesStage(t *testing.T) {
	tests := []struct {
		stage PointerModeSwitchStage
		want  int
	}{
		{stage: PointerModeSwitchStageLock, want: pointerModeAPILockCode},
		{stage: PointerModeSwitchStageQuiesce, want: pointerModeAPIQuiesceCode},
		{stage: PointerModeSwitchStageHIDClose, want: pointerModeAPIQuiesceCode},
		{stage: PointerModeSwitchStageScript, want: pointerModeAPIScriptCode},
		{stage: PointerModeSwitchStageValidation, want: pointerModeAPIValidationCode},
		{stage: PointerModeSwitchStageHIDOpen, want: pointerModeAPIValidationCode},
		{stage: PointerModeSwitchStageResume, want: pointerModeAPIResumeCode},
		{stage: PointerModeSwitchStageSnapshot, want: pointerModeAPIFailureCode},
		{stage: PointerModeSwitchStagePersistence, want: pointerModeAPIFailureCode},
	}
	for _, test := range tests {
		err := &PointerModeSwitchError{Stage: test.stage, Err: errors.New("test")}
		if got, _ := pointerModeAPIError(err, PointerModeStatus{}); got != test.want {
			t.Errorf("stage %s code = %d, want %d", test.stage, got, test.want)
		}
	}
}

func TestSinglePointerAPICommitsRelativeAndAbsoluteLayouts(t *testing.T) {
	marker := "absolute\n"
	harness := newPointerModeTransactionHarness(t, PointerModeAbsolute, &marker, ModeNormal)
	service := serviceForPointerModeAPI(harness)
	for _, method := range []string{"relative", "absolute", "digitizer-stylus", "touch-screen"} {
		response := performSetPointerModeRequest(t, service, `{"singleEndpoint":true,"mode":"`+method+`"}`)
		if response.Code != 0 || response.Data.Mode != method || !response.Data.SingleEndpoint {
			t.Fatalf("%s: %+v", method, response)
		}
	}
}
