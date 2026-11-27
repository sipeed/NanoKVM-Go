package hid

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

type hidResetTestResponse struct {
	Code int                     `json:"code"`
	Msg  string                  `json:"msg"`
	Data proto.GetPointerModeRsp `json:"data"`
}

func performHIDReset(t *testing.T, service *Service) hidResetTestResponse {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/hid/reset", bytes.NewReader(nil))
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = request

	service.Reset(ginContext)
	if recorder.Code != http.StatusOK {
		t.Fatalf("Reset HTTP status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response hidResetTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode Reset response: %v; body=%s", err, recorder.Body.String())
	}
	return response
}

func degradedResetService(harness *pointerModeTransactionHarness) *Service {
	harness.manager.markDegraded(errors.New("previous rollback failed"), harness.epoch)
	return &Service{
		pointerModeManager: harness.manager,
		pointerModeSwitch:  harness.transaction,
	}
}

func assertGlobalUSBMutationLockReleased(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := LockUsbGadgetMutationContext(ctx); err != nil {
		t.Fatalf("global USB mutation lock was not released: %v", err)
	}
	UnlockUsbGadgetMutation()
}

func TestResetRecoversDegradedPointerMode(t *testing.T) {
	tests := []struct {
		name           string
		marker         *string
		usbMode        string
		wantMode       PointerMode
		wantScriptArg  string
		wantMarkerData string
	}{
		{
			name:           "normal absolute",
			marker:         pointerModeMarkerValue("absolute\n"),
			usbMode:        ModeNormal,
			wantMode:       PointerModeAbsolute,
			wantScriptArg:  "restart",
			wantMarkerData: "absolute\n",
		},
		{
			name:           "HID-only touchscreen",
			marker:         pointerModeMarkerValue("touch-screen\n"),
			usbMode:        ModeHidOnly,
			wantMode:       PointerModeTouchScreen,
			wantScriptArg:  "hid-only",
			wantMarkerData: "touch-screen\n",
		},
		{
			name:           "legacy marker canonicalized",
			marker:         pointerModeMarkerValue("absolute-android\n"),
			usbMode:        ModeNormal,
			wantMode:       PointerModeDigitizerStylus,
			wantScriptArg:  "restart",
			wantMarkerData: "digitizer-stylus\n",
		},
		{
			name:           "missing marker defaults to single-absolute",
			marker:         nil,
			usbMode:        ModeNormal,
			wantMode:       PointerModeSingleAbsolute,
			wantScriptArg:  "restart",
			wantMarkerData: "single-absolute\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			harness := newPointerModeTransactionHarness(t, PointerModeUnknown, test.marker, test.usbMode)
			response := performHIDReset(t, degradedResetService(harness))
			if response.Code != 0 || response.Msg != "success" {
				t.Fatalf("Reset response = %+v", response)
			}

			status := harness.manager.status()
			if status.Mode != test.wantMode || status.Switching || status.Degraded || status.Error != "" {
				t.Fatalf("recovered status = %+v, want healthy %s", status, test.wantMode)
			}
			begin, end, scripts, verifies := harness.callSnapshot()
			if begin != 0 || end != 1 || fmt.Sprint(scripts) != fmt.Sprint([]string{test.wantScriptArg}) ||
				fmt.Sprint(verifies) != fmt.Sprint([]PointerMode{test.wantMode}) {
				t.Fatalf("calls begin=%d end=%d scripts=%v verifies=%v", begin, end, scripts, verifies)
			}
			data, err := os.ReadFile(harness.modeFile)
			if err != nil || string(data) != test.wantMarkerData {
				t.Fatalf("recovered marker = %q, %v; want %q", data, err, test.wantMarkerData)
			}
			harness.assertLocksReleased(t)
			assertGlobalUSBMutationLockReleased(t)
		})
	}
}

func TestResetAcceptsVerifiedRollbackAsDegradedRecovery(t *testing.T) {
	marker := "digitizer-stylus\n"
	harness := newPointerModeTransactionHarness(t, PointerModeUnknown, &marker, ModeNormal)
	harness.scriptErrors = []error{errors.New("first rebuild failed"), nil}

	response := performHIDReset(t, degradedResetService(harness))
	if response.Code != 0 || response.Msg != "success" {
		t.Fatalf("Reset response = %+v", response)
	}
	status := harness.manager.status()
	if status.Mode != PointerModeDigitizerStylus || status.Switching || status.Degraded {
		t.Fatalf("verified rollback status = %+v", status)
	}
	begin, end, scripts, verifies := harness.callSnapshot()
	if begin != 0 || end != 1 || fmt.Sprint(scripts) != fmt.Sprint([]string{"restart", "restart"}) ||
		fmt.Sprint(verifies) != fmt.Sprint([]PointerMode{PointerModeDigitizerStylus}) {
		t.Fatalf("calls begin=%d end=%d scripts=%v verifies=%v", begin, end, scripts, verifies)
	}
	harness.assertLocksReleased(t)
	assertGlobalUSBMutationLockReleased(t)
}

func TestResetFailureKeepsPointerModeDegradedAndQuiesced(t *testing.T) {
	tests := []struct {
		name          string
		configure     func(*pointerModeTransactionHarness)
		wantScriptNum int
		wantVerifyNum int
	}{
		{
			name: "script and rollback script fail",
			configure: func(harness *pointerModeTransactionHarness) {
				harness.scriptErrors = []error{errors.New("rebuild failed"), errors.New("rollback rebuild failed")}
			},
			wantScriptNum: 2,
		},
		{
			name: "validation and rollback validation fail",
			configure: func(harness *pointerModeTransactionHarness) {
				harness.verifyErrors = []error{errors.New("profile validation failure"), errors.New("rollback profile validation failure")}
			},
			wantScriptNum: 2,
			wantVerifyNum: 2,
		},
		{
			name: "input epoch cannot resume",
			configure: func(harness *pointerModeTransactionHarness) {
				harness.endErrors = []error{errors.New("epoch resume failed")}
			},
			wantScriptNum: 1,
			wantVerifyNum: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			marker := "touch-screen\n"
			harness := newPointerModeTransactionHarness(t, PointerModeUnknown, &marker, ModeNormal)
			test.configure(harness)
			response := performHIDReset(t, degradedResetService(harness))
			if response.Code != -3 || response.Msg != "failed to recover degraded HID state" {
				t.Fatalf("Reset response = %+v", response)
			}
			if response.Data.Mode != string(PointerModeUnknown) || response.Data.Switching ||
				!response.Data.Degraded || response.Data.Error == "" {
				t.Fatalf("degraded response data = %+v", response.Data)
			}
			status := harness.manager.status()
			if status.Mode != PointerModeUnknown || status.Switching || !status.Degraded || status.Error == "" {
				t.Fatalf("manager status = %+v", status)
			}
			begin, end, scripts, verifies := harness.callSnapshot()
			if begin != 0 || len(scripts) != test.wantScriptNum || len(verifies) != test.wantVerifyNum {
				t.Fatalf("calls begin=%d end=%d scripts=%v verifies=%v", begin, end, scripts, verifies)
			}
			if test.name != "input epoch cannot resume" && end != 0 {
				t.Fatalf("failed recovery resumed input: end=%d", end)
			}
			if test.name == "input epoch cannot resume" && end != 1 {
				t.Fatalf("resume failure end calls = %d, want 1", end)
			}
			harness.assertLocksReleased(t)
			assertGlobalUSBMutationLockReleased(t)
		})
	}
}

func pointerModeMarkerValue(value string) *string {
	return &value
}
