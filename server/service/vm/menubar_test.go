package vm

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

type menuBarTestResponse struct {
	Code int `json:"code"`
	Data *struct {
		DisabledItems []string `json:"disabledItems"`
	} `json:"data"`
}

func TestMissingMenuBarUsesDisabledDefault(t *testing.T) {
	setupMenuBarTest(t)
	menuBarReadFile = func(string) ([]byte, error) { return nil, os.ErrNotExist }

	config, err := readMenuBarConfig()
	if err != nil {
		t.Fatalf("read missing menu-bar config: %v", err)
	}
	if menuBarFingerRobotEnabled(config) {
		t.Fatalf("missing menu-bar config enabled finger robot: %+v", config)
	}
	if !slices.Equal(config.DisabledItems, []string{menuBarPowerItem, "mouseJiggler"}) {
		t.Fatalf("missing menu-bar default = %+v, want power and mouseJiggler", config.DisabledItems)
	}
}

func TestGetMissingMenuBarReturnsDisabledDefault(t *testing.T) {
	setupMenuBarTest(t)
	menuBarReadFile = func(string) ([]byte, error) { return nil, os.ErrNotExist }

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/vm/menubar", nil)

	NewService().GetMenuBarConfig(ctx)

	var rsp menuBarTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("unmarshal menu-bar response: %v body=%s", err, recorder.Body.String())
	}
	if rsp.Code != 0 || rsp.Data == nil {
		t.Fatalf("missing menu-bar GET response = %+v, want code 0 with data", rsp)
	}
	if !slices.Equal(rsp.Data.DisabledItems, []string{menuBarPowerItem, "mouseJiggler"}) {
		t.Fatalf("missing menu-bar GET disabledItems = %v, want power and mouseJiggler", rsp.Data.DisabledItems)
	}
}

func TestFingerRobotSettingFailsClosedWithoutValidMenuBarFile(t *testing.T) {
	setupMenuBarTest(t)

	tests := []struct {
		name string
		read func(string) ([]byte, error)
	}{
		{
			name: "missing",
			read: func(string) ([]byte, error) { return nil, os.ErrNotExist },
		},
		{
			name: "malformed",
			read: func(string) ([]byte, error) { return []byte(`{`), nil },
		},
		{
			name: "unreadable",
			read: func(string) ([]byte, error) { return nil, errors.New("read failed") },
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			menuBarReadFile = test.read
			if IsFingerRobotSettingEnabled() {
				t.Fatal("finger robot must remain disabled when the menu-bar state is unavailable")
			}
		})
	}
}

func TestFingerRobotSettingUsesPowerMenuVisibility(t *testing.T) {
	if menuBarFingerRobotEnabled(nil) {
		t.Fatal("nil menu-bar config must be disabled")
	}
	if menuBarFingerRobotEnabled(&proto.MenuBarConfig{}) {
		t.Fatal("missing disabledItems must fail closed")
	}
	if menuBarFingerRobotEnabled(&proto.MenuBarConfig{DisabledItems: []string{"power"}}) {
		t.Fatal("disabled power menu must disable the finger robot")
	}
	if !menuBarFingerRobotEnabled(&proto.MenuBarConfig{DisabledItems: []string{"keyboard"}}) {
		t.Fatal("visible power menu must enable the finger robot")
	}
	if menuBarFingerRobotEnabled(defaultMenuBarConfig()) {
		t.Fatal("default menu-bar config must keep the finger robot disabled")
	}
}

func TestSetMenuBarRejectsMissingOrNullDisabledItems(t *testing.T) {
	setupMenuBarTest(t)
	menuBarReadFile = func(string) ([]byte, error) {
		t.Fatal("invalid request must not read persisted settings")
		return nil, nil
	}
	menuBarWriteFile = func(string, []byte, os.FileMode) error {
		t.Fatal("invalid request must not write persisted settings")
		return nil
	}
	fingerRobotInitialize = func() error {
		t.Fatal("invalid request must not initialize PWM")
		return nil
	}
	fingerRobotDisable = func() error {
		t.Fatal("invalid request must not disable PWM")
		return nil
	}

	for _, body := range []string{`{}`, `{"disabledItems":null}`} {
		rsp := performSetMenuBarBody(t, []byte(body))
		if rsp.Code != -1 {
			t.Fatalf("body %s response code = %d, want -1", body, rsp.Code)
		}
	}
}

func TestSetMenuBarEnablesFingerRobotBeforePersistingFirstConfig(t *testing.T) {
	setupMenuBarTest(t)
	menuBarReadFile = func(string) ([]byte, error) { return nil, os.ErrNotExist }

	var initialized int
	fingerRobotInitialize = func() error {
		initialized++
		return nil
	}
	fingerRobotDisable = func() error {
		t.Fatal("disable must not run while enabling")
		return nil
	}
	var persisted proto.MenuBarConfig
	menuBarWriteFile = func(path string, data []byte, mode os.FileMode) error {
		if path != MenuBarConfigFile || mode != 0o644 {
			t.Fatalf("unexpected menu-bar write: path=%q mode=%#o", path, mode)
		}
		return json.Unmarshal(data, &persisted)
	}

	rsp := performSetMenuBarRequest(t, []string{})
	if rsp.Code != 0 {
		t.Fatalf("enable response code = %d, want 0", rsp.Code)
	}
	if initialized != 1 {
		t.Fatalf("initialize calls = %d, want 1", initialized)
	}
	if !menuBarFingerRobotEnabled(&persisted) {
		t.Fatalf("persisted config did not enable finger robot: %+v", persisted)
	}
}

func TestSetMenuBarSynchronizesDisabledControllerWhenConfigIsMissing(t *testing.T) {
	setupMenuBarTest(t)
	menuBarReadFile = func(string) ([]byte, error) { return nil, os.ErrNotExist }

	var disabled int
	fingerRobotInitialize = func() error {
		t.Fatal("missing config with an explicit disabled setting must not initialize PWM")
		return nil
	}
	fingerRobotDisable = func() error {
		disabled++
		return nil
	}
	var persisted proto.MenuBarConfig
	menuBarWriteFile = func(_ string, data []byte, _ os.FileMode) error {
		return json.Unmarshal(data, &persisted)
	}

	rsp := performSetMenuBarRequest(t, []string{menuBarPowerItem})
	if rsp.Code != 0 {
		t.Fatalf("disable response code = %d, want 0", rsp.Code)
	}
	if disabled != 1 {
		t.Fatalf("disable calls for missing persisted state = %d, want 1", disabled)
	}
	if menuBarFingerRobotEnabled(&persisted) {
		t.Fatalf("persisted missing-file replacement enabled finger robot: %+v", persisted)
	}
}

func TestSetMenuBarDisablesFingerRobotAtSafeRelease(t *testing.T) {
	setupMenuBarTest(t)
	menuBarReadFile = func(string) ([]byte, error) {
		return []byte(`{"disabledItems":[]}`), nil
	}

	var disabled int
	fingerRobotInitialize = func() error {
		t.Fatal("initialize must not run while disabling")
		return nil
	}
	fingerRobotDisable = func() error {
		disabled++
		return nil
	}
	menuBarWriteFile = func(string, []byte, os.FileMode) error { return nil }

	rsp := performSetMenuBarRequest(t, []string{"power"})
	if rsp.Code != 0 {
		t.Fatalf("disable response code = %d, want 0", rsp.Code)
	}
	if disabled != 1 {
		t.Fatalf("disable calls = %d, want 1", disabled)
	}
}

func TestSetMenuBarForcesDisabledStateWhenPersistedStateIsUnknown(t *testing.T) {
	setupMenuBarTest(t)
	menuBarReadFile = func(string) ([]byte, error) {
		return []byte(`{`), nil
	}

	var disabled int
	fingerRobotInitialize = func() error {
		t.Fatal("unknown persisted state must fail closed")
		return nil
	}
	fingerRobotDisable = func() error {
		disabled++
		return nil
	}
	menuBarWriteFile = func(string, []byte, os.FileMode) error { return nil }

	rsp := performSetMenuBarRequest(t, []string{"power"})
	if rsp.Code != 0 {
		t.Fatalf("repair response code = %d, want 0", rsp.Code)
	}
	if disabled != 1 {
		t.Fatalf("disable calls for unknown state = %d, want 1", disabled)
	}
}

func TestSetMenuBarPersistsEnabledSettingWhileFingerRobotIsUnavailable(t *testing.T) {
	setupMenuBarTest(t)
	menuBarReadFile = func(string) ([]byte, error) { return nil, os.ErrNotExist }
	fingerRobotInitialize = func() error { return errors.New("PWM unavailable") }
	var disabled int
	fingerRobotDisable = func() error {
		disabled++
		return nil
	}
	var written []byte
	menuBarWriteFile = func(_ string, data []byte, _ os.FileMode) error {
		written = append([]byte(nil), data...)
		return nil
	}

	rsp := performSetMenuBarRequest(t, []string{})
	if rsp.Code != 0 {
		t.Fatalf("unavailable enable response code = %d, want 0", rsp.Code)
	}
	if disabled != 0 {
		t.Fatalf("unavailable enable rollback calls = %d, want 0", disabled)
	}
	if string(written) != `{"disabledItems":[]}` {
		t.Fatalf("persisted unavailable enabled setting = %s", written)
	}
}

func TestSetMenuBarRejectsNonRetryableFingerRobotEnableFailure(t *testing.T) {
	setupMenuBarTest(t)
	menuBarReadFile = func(string) ([]byte, error) { return nil, os.ErrNotExist }
	fingerRobotInitialize = func() error { return ErrFingerRobotInvalid }
	var disabled int
	fingerRobotDisable = func() error {
		disabled++
		return nil
	}
	menuBarWriteFile = func(string, []byte, os.FileMode) error {
		t.Fatal("non-retryable initialization failure must not persist")
		return nil
	}

	rsp := performSetMenuBarRequest(t, []string{})
	if rsp.Code != -3 {
		t.Fatalf("non-retryable enable response code = %d, want -3", rsp.Code)
	}
	if disabled != 1 {
		t.Fatalf("non-retryable enable rollback calls = %d, want 1", disabled)
	}
}

func TestSetMenuBarRollsBackFingerRobotWhenPersistenceFails(t *testing.T) {
	setupMenuBarTest(t)
	menuBarReadFile = func(string) ([]byte, error) { return nil, os.ErrNotExist }

	var initialized, disabled int
	fingerRobotInitialize = func() error {
		initialized++
		return nil
	}
	fingerRobotDisable = func() error {
		disabled++
		return nil
	}
	menuBarWriteFile = func(string, []byte, os.FileMode) error {
		return errors.New("write failed")
	}

	rsp := performSetMenuBarRequest(t, []string{})
	if rsp.Code != -2 {
		t.Fatalf("failed write response code = %d, want -2", rsp.Code)
	}
	if initialized != 1 || disabled != 1 {
		t.Fatalf("transition/rollback calls = initialize:%d disable:%d, want 1/1", initialized, disabled)
	}
}

func TestWriteMenuBarFileAtomicallyReplacesTarget(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "menubar")
	if err := os.WriteFile(path, []byte(`{"disabledItems":["power"]}`), 0o600); err != nil {
		t.Fatalf("write original menu-bar file: %v", err)
	}

	want := []byte(`{"disabledItems":[]}`)
	if err := writeMenuBarFileAtomically(path, want, 0o644); err != nil {
		t.Fatalf("atomic menu-bar write: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read replaced menu-bar file: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("replaced menu-bar data = %s, want %s", got, want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat replaced menu-bar file: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("replaced menu-bar mode = %#o, want 0644", info.Mode().Perm())
	}
	matches, err := filepath.Glob(filepath.Join(directory, ".menubar.*"))
	if err != nil {
		t.Fatalf("glob temporary menu-bar files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("atomic write left temporary files: %v", matches)
	}
}

func setupMenuBarTest(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	originalReadFile := menuBarReadFile
	originalWriteFile := menuBarWriteFile
	originalInitialize := fingerRobotInitialize
	originalDisable := fingerRobotDisable
	t.Cleanup(func() {
		menuBarReadFile = originalReadFile
		menuBarWriteFile = originalWriteFile
		fingerRobotInitialize = originalInitialize
		fingerRobotDisable = originalDisable
	})
}

func performSetMenuBarRequest(t *testing.T, disabledItems []string) menuBarTestResponse {
	t.Helper()

	body, err := json.Marshal(proto.SetMenuBarConfigReq{DisabledItems: disabledItems})
	if err != nil {
		t.Fatalf("marshal menu-bar request: %v", err)
	}
	return performSetMenuBarBody(t, body)
}

func performSetMenuBarBody(t *testing.T, body []byte) menuBarTestResponse {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/vm/menubar", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")

	NewService().SetMenuBarConfig(ctx)

	var rsp menuBarTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("unmarshal menu-bar response: %v body=%s", err, recorder.Body.String())
	}
	return rsp
}
