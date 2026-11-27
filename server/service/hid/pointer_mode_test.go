package hid

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

func TestReadPointerModeFile(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		create   bool
		want     PointerMode
	}{
		{name: "missing", want: PointerModeSingleAbsolute},
		{name: "absolute", contents: "absolute\n", create: true, want: PointerModeAbsolute},
		{name: "digitizer stylus", contents: "digitizer-stylus\n", create: true, want: PointerModeDigitizerStylus},
		{name: "legacy absolute android", contents: "absolute-android\n", create: true, want: PointerModeDigitizerStylus},
		{name: "touch screen", contents: "touch-screen\n", create: true, want: PointerModeTouchScreen},
		{name: "trailing newlines", contents: "touch-screen\n\n", create: true, want: PointerModeTouchScreen},
		{name: "empty", contents: "", create: true, want: PointerModeAbsolute},
		{name: "unknown", contents: "relative\n", create: true, want: PointerModeAbsolute},
		{name: "leading whitespace", contents: " absolute\n", create: true, want: PointerModeAbsolute},
		{name: "trailing whitespace", contents: "absolute \n", create: true, want: PointerModeAbsolute},
		{name: "carriage return", contents: "absolute\r\n", create: true, want: PointerModeAbsolute},
		{name: "embedded newline", contents: "absolute\ntouch-screen\n", create: true, want: PointerModeAbsolute},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			modeFile := filepath.Join(t.TempDir(), "usb.hid2_mode")
			if test.create {
				if err := os.WriteFile(modeFile, []byte(test.contents), 0o600); err != nil {
					t.Fatalf("write mode file: %v", err)
				}
			}

			if got := readPointerModeFile(modeFile); got != test.want {
				t.Fatalf("readPointerModeFile() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestParsePointerModeIsStrict(t *testing.T) {
	tests := []struct {
		value string
		want  PointerMode
		ok    bool
	}{
		{value: "absolute", want: PointerModeAbsolute, ok: true},
		{value: "digitizer-stylus", want: PointerModeDigitizerStylus, ok: true},
		{value: "absolute-android", want: PointerModeDigitizerStylus, ok: true},
		{value: "touch-screen", want: PointerModeTouchScreen, ok: true},
		{value: "", want: PointerModeAbsolute},
		{value: "relative", want: PointerModeAbsolute},
		{value: " absolute", want: PointerModeAbsolute},
		{value: "absolute ", want: PointerModeAbsolute},
		{value: "Absolute", want: PointerModeAbsolute},
	}

	for _, test := range tests {
		got, ok := parsePointerMode(test.value)
		if got != test.want || ok != test.ok {
			t.Errorf("parsePointerMode(%q) = (%q, %t), want (%q, %t)", test.value, got, ok, test.want, test.ok)
		}
	}
}

func TestValidatePointerModeReportID(t *testing.T) {
	tests := []struct {
		mode PointerMode
		id   byte
		ok   bool
	}{
		{mode: PointerModeAbsolute, id: AbsoluteReportIDMouse, ok: true},
		{mode: PointerModeDigitizerStylus, id: AbsoluteReportIDStylus, ok: true},
		{mode: PointerModeTouchScreen, id: TouchScreenReportID, ok: true},
		{mode: PointerModeAbsolute, id: AbsoluteReportIDStylus},
		{mode: PointerModeAbsolute, id: TouchScreenReportID},
		{mode: PointerModeDigitizerStylus, id: AbsoluteReportIDMouse},
		{mode: PointerModeTouchScreen, id: AbsoluteReportIDMouse},
		{mode: PointerMode("unknown"), id: AbsoluteReportIDMouse},
	}

	for _, test := range tests {
		err := ValidatePointerModeReportID(test.mode, test.id)
		if test.ok && err != nil {
			t.Errorf("ValidatePointerModeReportID(%q, %d): %v", test.mode, test.id, err)
		}
		if !test.ok && !errors.Is(err, ErrPointerModeMismatch) {
			t.Errorf("ValidatePointerModeReportID(%q, %d) error = %v, want pointer mode mismatch", test.mode, test.id, err)
		}
	}
}

func TestReadPointerModeFileFallsBackForNonRegularInput(t *testing.T) {
	if got := readPointerModeFile(t.TempDir()); got != PointerModeAbsolute {
		t.Fatalf("readPointerModeFile(directory) = %q, want %q", got, PointerModeAbsolute)
	}
}

func TestPointerModeManagerInitializesFromFileOnce(t *testing.T) {
	modeFile := filepath.Join(t.TempDir(), "usb.hid2_mode")
	if err := os.WriteFile(modeFile, []byte("digitizer-stylus\n"), 0o600); err != nil {
		t.Fatalf("write initial mode: %v", err)
	}

	manager := newPointerModeManager(modeFile)
	if err := os.WriteFile(modeFile, []byte("touch-screen\n"), 0o600); err != nil {
		t.Fatalf("replace persisted mode: %v", err)
	}

	status := manager.status()
	if status.Mode != PointerModeDigitizerStylus {
		t.Fatalf("active mode = %q, want %q", status.Mode, PointerModeDigitizerStylus)
	}
	if status.Switching {
		t.Fatal("new pointer mode manager unexpectedly reports switching")
	}
}

func TestGetPointerModeReturnsRuntimeStateWithoutTouchingMarker(t *testing.T) {
	modeFile := filepath.Join(t.TempDir(), "usb.hid2_mode")
	if err := os.WriteFile(modeFile, []byte("digitizer-stylus\n"), 0o600); err != nil {
		t.Fatalf("write initial mode: %v", err)
	}

	manager := newPointerModeManager(modeFile)
	manager.switching = true
	if err := os.WriteFile(modeFile, []byte("touch-screen\n"), 0o600); err != nil {
		t.Fatalf("replace persisted mode: %v", err)
	}

	response := performGetPointerMode(t, &Service{pointerModeManager: manager})
	if response.Code != 0 || response.Msg != "success" {
		t.Fatalf("response = code %d, message %q", response.Code, response.Msg)
	}
	if response.Data.Mode != string(PointerModeDigitizerStylus) {
		t.Fatalf("response mode = %q, want %q", response.Data.Mode, PointerModeDigitizerStylus)
	}
	if !response.Data.Switching {
		t.Fatal("response omitted the runtime switching state")
	}

	contents, err := os.ReadFile(modeFile)
	if err != nil {
		t.Fatalf("read marker after GET: %v", err)
	}
	if string(contents) != "touch-screen\n" {
		t.Fatalf("GET changed marker to %q", contents)
	}
}

func TestGetPointerModeDoesNotAcquireUsbGadgetMutationLock(t *testing.T) {
	service := &Service{pointerModeManager: &pointerModeManager{activeMode: PointerModeAbsolute}}
	done := make(chan struct{})

	LockUsbGadgetMutation()
	locked := true
	defer func() {
		if locked {
			UnlockUsbGadgetMutation()
		}
	}()

	go func() {
		request := httptest.NewRequest(http.MethodGet, "/api/hid/pointer-mode", nil)
		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		context.Request = request
		service.GetPointerMode(context)
		close(done)
	}()

	select {
	case <-done:
		UnlockUsbGadgetMutation()
		locked = false
	case <-time.After(time.Second):
		UnlockUsbGadgetMutation()
		locked = false
		select {
		case <-done:
		case <-time.After(time.Second):
		}
		t.Fatal("GET /api/hid/pointer-mode waited for the USB gadget mutation lock")
	}
}

func TestNewServiceInitializesPointerModeManager(t *testing.T) {
	service := NewService()
	if service.pointerModeManager == nil {
		t.Fatal("NewService pointerModeManager is nil")
	}
	if service.pointerModeSwitch == nil {
		t.Fatal("NewService pointerModeSwitch is nil")
	}
}

func TestGetPointerModeReturnsDegradedRuntimeState(t *testing.T) {
	manager := &pointerModeManager{
		activeMode: PointerModeUnknown,
		degraded:   true,
		lastError:  "rollback failed",
	}
	response := performGetPointerMode(t, &Service{pointerModeManager: manager})
	if response.Data.Mode != string(PointerModeUnknown) || !response.Data.Degraded || response.Data.Error != "rollback failed" {
		t.Fatalf("degraded response = %+v", response.Data)
	}
}

func performGetPointerMode(t *testing.T, service *Service) struct {
	Code int                     `json:"code"`
	Msg  string                  `json:"msg"`
	Data proto.GetPointerModeRsp `json:"data"`
} {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, "/api/hid/pointer-mode", nil)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request

	service.GetPointerMode(context)
	if recorder.Code != http.StatusOK {
		t.Fatalf("GetPointerMode HTTP status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Code int                     `json:"code"`
		Msg  string                  `json:"msg"`
		Data proto.GetPointerModeRsp `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode GetPointerMode response: %v", err)
	}
	return response
}
