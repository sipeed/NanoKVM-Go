package ai

import (
	authservice "NanoKVM-Server/service/auth"

	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

var errTestAIRuntime = errors.New("AI runtime failed")

type aiConfigResponse struct {
	Code int `json:"code"`
	Data struct {
		MCPEnabled      bool   `json:"mcpEnabled"`
		MCPAPIKey       string `json:"mcpApiKey"`
		MCPAPIKeyMasked string `json:"mcpApiKeyMasked"`
	} `json:"data"`
}

type screenTimelapseStateResponse struct {
	Code int `json:"code"`
	Data struct {
		Enabled bool `json:"enabled"`
	} `json:"data"`
}

func withAIStateFiles(t *testing.T) (string, string, string) {
	t.Helper()

	dir := t.TempDir()
	oldConfigFilePath := configFilePath
	oldMCPEnableFilePath := mcpEnableFilePath
	oldScreenTimelapseEnableFilePath := screenTimelapseEnableFilePath
	oldServiceStateFilePath := serviceStateFilePath
	oldSetKvmCommAIRuntime := setKvmCommAIRuntime
	oldMCPActionRequiresChallenge := mcpActionRequiresChallenge
	oldVerifyMCPChallenge := verifyMCPChallenge
	oldScreenTimelapseEnableRequiresChallenge := screenTimelapseEnableRequiresChallenge
	oldVerifyScreenTimelapseEnableChallenge := verifyScreenTimelapseEnableChallenge
	oldVerifyScreenTimelapsePlaybackChallenge := verifyScreenTimelapsePlaybackChallenge

	defaultServiceStateManagerMu.Lock()
	oldDefaultServiceStateManager := defaultServiceStateManager
	defaultServiceStateManager = nil
	defaultServiceStateManagerMu.Unlock()

	configFilePath = filepath.Join(dir, "ai.json")
	mcpEnableFilePath = filepath.Join(dir, "mcp_enable")
	screenTimelapseEnableFilePath = filepath.Join(dir, "scr_enable")
	serviceStateFilePath = filepath.Join(dir, "services.json")
	setKvmCommAIRuntime = func(bool, bool, bool, bool, time.Duration) error {
		return nil
	}
	mcpActionRequiresChallenge = func(string) bool { return false }
	verifyMCPChallenge = func(*gin.Context, string, string) error { return nil }
	screenTimelapseEnableRequiresChallenge = func() bool { return false }
	verifyScreenTimelapseEnableChallenge = func(*gin.Context, string, string) error { return nil }
	verifyScreenTimelapsePlaybackChallenge = func(*gin.Context, string) error { return nil }

	t.Cleanup(func() {
		configFilePath = oldConfigFilePath
		mcpEnableFilePath = oldMCPEnableFilePath
		screenTimelapseEnableFilePath = oldScreenTimelapseEnableFilePath
		serviceStateFilePath = oldServiceStateFilePath
		setKvmCommAIRuntime = oldSetKvmCommAIRuntime
		mcpActionRequiresChallenge = oldMCPActionRequiresChallenge
		verifyMCPChallenge = oldVerifyMCPChallenge
		screenTimelapseEnableRequiresChallenge = oldScreenTimelapseEnableRequiresChallenge
		verifyScreenTimelapseEnableChallenge = oldVerifyScreenTimelapseEnableChallenge
		verifyScreenTimelapsePlaybackChallenge = oldVerifyScreenTimelapsePlaybackChallenge
		defaultServiceStateManagerMu.Lock()
		defaultServiceStateManager = oldDefaultServiceStateManager
		defaultServiceStateManagerMu.Unlock()
	})

	return configFilePath, mcpEnableFilePath, screenTimelapseEnableFilePath
}

func newTestContext(method string, target string, body string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	c.Request = req
	return c, recorder
}

func decodeAIConfigResponse(t *testing.T, recorder *httptest.ResponseRecorder) aiConfigResponse {
	t.Helper()

	var rsp aiConfigResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return rsp
}

func assertResponseOmitsMCPAPIKey(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()

	var rsp struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := rsp.Data["mcpApiKey"]; ok {
		t.Fatalf("response must not contain mcpApiKey: %s", recorder.Body.String())
	}
}

type mcpAPIKeyResponse struct {
	Code int `json:"code"`
	Data struct {
		MCPAPIKey string `json:"mcpApiKey"`
	} `json:"data"`
}

func decodeMCPAPIKeyResponse(t *testing.T, recorder *httptest.ResponseRecorder) mcpAPIKeyResponse {
	t.Helper()
	var rsp mcpAPIKeyResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return rsp
}

func decodeScreenTimelapseStateResponse(t *testing.T, recorder *httptest.ResponseRecorder) screenTimelapseStateResponse {
	t.Helper()

	var rsp screenTimelapseStateResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return rsp
}

func writeMCPServiceState(t *testing.T, enabled bool) {
	t.Helper()
	writeAIServiceState(t, aiServiceMCP, enabled)
}

func writeScreenTimelapseServiceState(t *testing.T, enabled bool) {
	t.Helper()
	writeAIServiceState(t, aiServiceScreenTimelapse, enabled)
}

func writeAIServiceState(t *testing.T, serviceID string, enabled bool) {
	t.Helper()

	state := defaultServiceStateSnapshot()
	state.Revision = 1
	state.Services[serviceID] = ServiceStateEntry{
		Enabled:   enabled,
		UpdatedAt: "2026-06-18T00:00:00Z",
		Source:    "test",
	}
	if err := saveServiceStateToPath(serviceStateFilePath, state); err != nil {
		t.Fatalf("write service state: %v", err)
	}

	defaultServiceStateManagerMu.Lock()
	defaultServiceStateManager = nil
	defaultServiceStateManagerMu.Unlock()
}

func setTestAIRuntime(t *testing.T, err error, calls *[]string) {
	t.Helper()

	oldSetKvmCommAIRuntime := setKvmCommAIRuntime
	setKvmCommAIRuntime = func(snapshotEnabled bool, snapshotPeriodicEnabled bool, timelapseEnabled bool, restart bool, restartDelay time.Duration) error {
		if calls != nil {
			*calls = append(*calls, fmt.Sprintf("runtime:%t:%t:%t:%t:%s", snapshotEnabled, snapshotPeriodicEnabled, timelapseEnabled, restart, restartDelay))
		}
		return err
	}

	t.Cleanup(func() {
		setKvmCommAIRuntime = oldSetKvmCommAIRuntime
	})
}

func stringSlicesEqual(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}

	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}

	return true
}

func assertRuntimeAIConfigOmitsMCPEnabled(t *testing.T, configPath string) {
	t.Helper()

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read AI config: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("decode AI config: %v", err)
	}
	if _, ok := raw["mcpEnabled"]; ok {
		t.Fatalf("expected runtime AI config to omit mcpEnabled, got %s", data)
	}
}

func TestStateFileHelpersSetRemoveAndExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state")

	exists, err := stateFileExists(path)
	if err != nil || exists {
		t.Fatalf("expected missing state file, exists=%t err=%v", exists, err)
	}

	if err := setStateFile(path); err != nil {
		t.Fatalf("set state file: %v", err)
	}
	exists, err = stateFileExists(path)
	if err != nil || !exists {
		t.Fatalf("expected existing state file, exists=%t err=%v", exists, err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat state file: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected state file mode 0600, got %o", info.Mode().Perm())
	}

	if err := removeStateFile(path); err != nil {
		t.Fatalf("remove state file: %v", err)
	}
	exists, err = stateFileExists(path)
	if err != nil || exists {
		t.Fatalf("expected removed state file, exists=%t err=%v", exists, err)
	}

	if err := removeStateFile(path); err != nil {
		t.Fatalf("remove missing state file: %v", err)
	}
}

func TestGetConfigUsesServiceState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configPath, mcpPath, _ := withAIStateFiles(t)
	const apiKey = "nag_mcp_KB6-abcdefghijklmnopW7iXkI"
	if err := saveConfigToPath(configPath, Config{MCPEnabled: true, MCPAPIKey: apiKey}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	if err := setStateFile(mcpPath); err != nil {
		t.Fatalf("set stale mcp state file: %v", err)
	}
	writeMCPServiceState(t, false)

	service := &Service{}
	c, recorder := newTestContext(http.MethodGet, "/api/ai/config", "")
	service.GetConfig(c)
	rsp := decodeAIConfigResponse(t, recorder)
	assertResponseOmitsMCPAPIKey(t, recorder)
	if rsp.Code != 0 || rsp.Data.MCPEnabled || rsp.Data.MCPAPIKeyMasked != "" {
		t.Fatalf("expected secret-free disabled service state, got %#v", rsp)
	}

	if _, err := DefaultServiceStateManager().Set(aiServiceMCP, true, "screen"); err != nil {
		t.Fatalf("set mcp service state: %v", err)
	}
	c, recorder = newTestContext(http.MethodGet, "/api/ai/config", "")
	service.GetConfig(c)
	rsp = decodeAIConfigResponse(t, recorder)
	assertResponseOmitsMCPAPIKey(t, recorder)
	if rsp.Code != 0 || !rsp.Data.MCPEnabled || rsp.Data.MCPAPIKeyMasked != "nag_mcp_KB6-...W7iXkI" {
		t.Fatalf("expected masked API key for enabled service state, got %#v", rsp)
	}
}

func TestSetConfigWritesServiceState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configPath, mcpPath, _ := withAIStateFiles(t)
	if err := saveConfigToPath(configPath, Config{}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	service := &Service{}
	c, recorder := newTestContext(http.MethodPost, "/api/ai/config", `{"mcpEnabled":true}`)
	service.SetConfig(c)
	rsp := decodeAIConfigResponse(t, recorder)
	assertResponseOmitsMCPAPIKey(t, recorder)
	if rsp.Code != 0 || !rsp.Data.MCPEnabled || rsp.Data.MCPAPIKeyMasked == "" {
		t.Fatalf("expected secret-free enabled response, got %#v", rsp)
	}
	snapshot, err := DefaultServiceStateManager().List()
	if err != nil {
		t.Fatalf("list service state after enable: %v", err)
	}
	mcp := snapshot.Services[aiServiceMCP]
	if !mcp.Enabled || mcp.Source != "web" {
		t.Fatalf("expected MCP service state enabled from web, got %#v", mcp)
	}
	if exists, err := stateFileExists(mcpPath); err != nil || exists {
		t.Fatalf("expected mcp state file not created, exists=%t err=%v", exists, err)
	}
	assertRuntimeAIConfigOmitsMCPEnabled(t, configPath)

	c, recorder = newTestContext(http.MethodPost, "/api/ai/config", `{"mcpEnabled":false}`)
	service.SetConfig(c)
	rsp = decodeAIConfigResponse(t, recorder)
	assertResponseOmitsMCPAPIKey(t, recorder)
	if rsp.Code != 0 || rsp.Data.MCPEnabled || rsp.Data.MCPAPIKeyMasked != "" {
		t.Fatalf("expected secret-free disabled response, got %#v", rsp)
	}
	snapshot, err = DefaultServiceStateManager().List()
	if err != nil {
		t.Fatalf("list service state after disable: %v", err)
	}
	mcp = snapshot.Services[aiServiceMCP]
	if mcp.Enabled || mcp.Source != "web" {
		t.Fatalf("expected MCP service state disabled from web, got %#v", mcp)
	}
	if exists, err := stateFileExists(mcpPath); err != nil || exists {
		t.Fatalf("expected mcp state file still absent, exists=%t err=%v", exists, err)
	}
	assertRuntimeAIConfigOmitsMCPEnabled(t, configPath)
}

func TestMCPMiddlewareFollowsServiceStateOnEachRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configPath, mcpPath, _ := withAIStateFiles(t)
	if err := saveConfigToPath(configPath, Config{MCPEnabled: true, MCPAPIKey: "secret"}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	if err := setStateFile(mcpPath); err != nil {
		t.Fatalf("set stale mcp state file: %v", err)
	}
	writeMCPServiceState(t, false)

	router := gin.New()
	router.GET("/api/mcp", MCPAPIKeyMiddleware(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/mcp", nil)
	req.Header.Set("Authorization", "Bearer secret")
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected disabled middleware request to fail, got code=%d body=%s", recorder.Code, recorder.Body.String())
	}

	if _, err := DefaultServiceStateManager().Set(aiServiceMCP, true, "screen"); err != nil {
		t.Fatalf("set mcp service state enabled: %v", err)
	}
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/mcp", nil)
	req.Header.Set("Authorization", "Bearer secret")
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected enabled middleware request to pass, got code=%d body=%s", recorder.Code, recorder.Body.String())
	}

	if _, err := DefaultServiceStateManager().Set(aiServiceMCP, false, "screen"); err != nil {
		t.Fatalf("set mcp service state disabled: %v", err)
	}
	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/mcp", nil)
	req.Header.Set("Authorization", "Bearer secret")
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected disabled middleware request to fail again, got code=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestRegenerateMCPAPIKeyUsesServiceState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configPath, _, _ := withAIStateFiles(t)
	if err := saveConfigToPath(configPath, Config{MCPEnabled: true, MCPAPIKey: "old-key"}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	writeMCPServiceState(t, false)

	service := &Service{}
	c, recorder := newTestContext(http.MethodPost, "/api/ai/mcp/key/regenerate", "")
	service.RegenerateMCPAPIKey(c)
	rsp := decodeAIConfigResponse(t, recorder)
	assertResponseOmitsMCPAPIKey(t, recorder)
	if rsp.Code != 0 || rsp.Data.MCPEnabled {
		t.Fatalf("expected secret-free disabled response after regeneration, got %#v", rsp)
	}
	assertRuntimeAIConfigOmitsMCPEnabled(t, configPath)
}

func TestGetMCPAPIKeyUsesDedicatedProtectedEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configPath, _, _ := withAIStateFiles(t)
	if err := saveConfigToPath(configPath, Config{MCPAPIKey: "test-key"}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	var gotAction, gotChallenge string
	mcpActionRequiresChallenge = func(string) bool { return true }
	verifyMCPChallenge = func(_ *gin.Context, challenge, action string) error {
		gotAction = action
		gotChallenge = challenge
		return nil
	}

	service := &Service{}
	c, recorder := newTestContext(http.MethodPost, "/api/ai/mcp/key", `{"challenge":"challenge"}`)
	service.GetMCPAPIKey(c)
	rsp := decodeMCPAPIKeyResponse(t, recorder)
	if rsp.Code != 0 || rsp.Data.MCPAPIKey != "test-key" {
		t.Fatalf("expected dedicated API-key response, got %#v", rsp)
	}
	if gotAction != authservice.SensitiveActionMCPShowKey || gotChallenge != "challenge" {
		t.Fatalf("unexpected protected request: action=%q challenge=%q", gotAction, gotChallenge)
	}

	gotAction = ""
	gotChallenge = ""
	c, recorder = newTestContext(http.MethodPost, "/api/ai/mcp/key/copy", `{"challenge":"challenge"}`)
	service.CopyMCPAPIKey(c)
	rsp = decodeMCPAPIKeyResponse(t, recorder)
	if rsp.Code != 0 || rsp.Data.MCPAPIKey != "test-key" {
		t.Fatalf("expected protected copy API-key response, got %#v", rsp)
	}
	if gotAction != authservice.SensitiveActionMCPCopyKey || gotChallenge != "challenge" {
		t.Fatalf("unexpected protected copy request: action=%q challenge=%q", gotAction, gotChallenge)
	}
}

func TestProtectedMCPOperationsRejectMissingChallengeBeforeMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configPath, _, _ := withAIStateFiles(t)
	if err := saveConfigToPath(configPath, Config{MCPAPIKey: "old-key"}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	mcpActionRequiresChallenge = func(string) bool { return true }
	verifyMCPChallenge = func(*gin.Context, string, string) error {
		return errors.New("challenge missing")
	}

	service := &Service{}
	c, recorder := newTestContext(http.MethodPost, "/api/ai/mcp/key", `{}`)
	service.GetMCPAPIKey(c)
	if rsp := decodeMCPAPIKeyResponse(t, recorder); rsp.Code != -3 || rsp.Data.MCPAPIKey != "" {
		t.Fatalf("expected key display to require challenge, got %#v", rsp)
	}

	c, recorder = newTestContext(http.MethodPost, "/api/ai/mcp/key/copy", `{}`)
	service.CopyMCPAPIKey(c)
	if rsp := decodeMCPAPIKeyResponse(t, recorder); rsp.Code != -3 || rsp.Data.MCPAPIKey != "" {
		t.Fatalf("expected key copy to require challenge, got %#v", rsp)
	}

	c, recorder = newTestContext(http.MethodPost, "/api/ai/mcp/key/regenerate", `{}`)
	service.RegenerateMCPAPIKey(c)
	if rsp := decodeAIConfigResponse(t, recorder); rsp.Code != -3 {
		t.Fatalf("expected key regeneration to require challenge, got %#v", rsp)
	}

	c, recorder = newTestContext(http.MethodPost, "/api/ai/config", `{"mcpEnabled":true}`)
	service.SetConfig(c)
	if rsp := decodeAIConfigResponse(t, recorder); rsp.Code != -3 {
		t.Fatalf("expected MCP enable to require challenge, got %#v", rsp)
	}

	writeMCPServiceState(t, true)
	c, recorder = newTestContext(http.MethodPost, "/api/ai/config", `{"mcpEnabled":false}`)
	service.SetConfig(c)
	if rsp := decodeAIConfigResponse(t, recorder); rsp.Code != -3 {
		t.Fatalf("expected MCP disable to require challenge, got %#v", rsp)
	}
	if enabled, err := getMCPEnabled(); err != nil || !enabled {
		t.Fatalf("protected MCP disable changed state before challenge verification: enabled=%t err=%v", enabled, err)
	}

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.MCPAPIKey != "old-key" {
		t.Fatalf("protected operations changed key before challenge verification: %#v", cfg)
	}
}

func TestSocketMCPStateMatchesWebConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configPath, _, _ := withAIStateFiles(t)
	setTestStateSocketCapabilities(t, 256, chipTypeAX620QF)
	if err := saveConfigToPath(configPath, Config{MCPAPIKey: "test-key"}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	socketPath := filepath.Join(t.TempDir(), "nanokvm-state.sock")
	stubGateHandlerForTest(t)
	socketServer := newStateSocketServer(DefaultServiceStateManager(), socketPath)
	if err := socketServer.Start(); err != nil {
		t.Fatalf("start state socket: %v", err)
	}
	t.Cleanup(func() {
		_ = socketServer.Stop()
	})
	client := newTestUnixSocketClient(t, socketPath)

	service := &Service{}
	resp := doStateSocketRequest(t, client, http.MethodPut, "/v1/services/mcp", []byte(`{"enabled":true,"source":"screen"}`))
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected socket enable status 200, got %d", resp.StatusCode)
	}

	c, recorder := newTestContext(http.MethodGet, "/api/ai/config", "")
	service.GetConfig(c)
	rsp := decodeAIConfigResponse(t, recorder)
	assertResponseOmitsMCPAPIKey(t, recorder)
	if rsp.Code != 0 || !rsp.Data.MCPEnabled {
		t.Fatalf("expected secret-free Web config to show socket-enabled MCP, got %#v", rsp)
	}

	resp = doStateSocketRequest(t, client, http.MethodPut, "/v1/services/mcp", []byte(`{"enabled":false,"source":"screen"}`))
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected socket disable status 200, got %d", resp.StatusCode)
	}

	c, recorder = newTestContext(http.MethodGet, "/api/ai/config", "")
	service.GetConfig(c)
	rsp = decodeAIConfigResponse(t, recorder)
	assertResponseOmitsMCPAPIKey(t, recorder)
	if rsp.Code != 0 || rsp.Data.MCPEnabled {
		t.Fatalf("expected secret-free Web config to show socket-disabled MCP, got %#v", rsp)
	}
}

func TestScreenTimelapseStateUsesServiceState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, _, screenPath := withAIStateFiles(t)
	var runtimeCalls []string
	setTestAIRuntime(t, nil, &runtimeCalls)
	if err := setStateFile(screenPath); err != nil {
		t.Fatalf("set stale screen timelapse state file: %v", err)
	}
	writeScreenTimelapseServiceState(t, false)

	service := &Service{}
	c, recorder := newTestContext(http.MethodGet, "/api/ai/screen-timelapse/state", "")
	service.GetScreenTimelapseState(c)
	rsp := decodeScreenTimelapseStateResponse(t, recorder)
	if rsp.Code != 0 || rsp.Data.Enabled {
		t.Fatalf("expected service state disabled, got %#v", rsp)
	}
	if err := removeStateFile(screenPath); err != nil {
		t.Fatalf("remove stale screen timelapse state file: %v", err)
	}

	c, recorder = newTestContext(http.MethodPost, "/api/ai/screen-timelapse/state", `{"enabled":true}`)
	service.SetScreenTimelapseState(c)
	rsp = decodeScreenTimelapseStateResponse(t, recorder)
	if rsp.Code != 0 || !rsp.Data.Enabled {
		t.Fatalf("expected enabled response, got %#v", rsp)
	}
	if !stringSlicesEqual(runtimeCalls, []string{"runtime:true:true:true:true:0s"}) {
		t.Fatalf("expected AI runtime enable call, got %#v", runtimeCalls)
	}
	snapshot, err := DefaultServiceStateManager().List()
	if err != nil {
		t.Fatalf("list service state after enable: %v", err)
	}
	screenTimelapse := snapshot.Services[aiServiceScreenTimelapse]
	if !screenTimelapse.Enabled || screenTimelapse.Source != "web" {
		t.Fatalf("expected screen timelapse service state enabled from web, got %#v", screenTimelapse)
	}
	if exists, err := stateFileExists(screenPath); err != nil || exists {
		t.Fatalf("expected screen timelapse state file not created, exists=%t err=%v", exists, err)
	}

	c, recorder = newTestContext(http.MethodGet, "/api/ai/screen-timelapse/state", "")
	service.GetScreenTimelapseState(c)
	rsp = decodeScreenTimelapseStateResponse(t, recorder)
	if rsp.Code != 0 || !rsp.Data.Enabled {
		t.Fatalf("expected existing state file response enabled, got %#v", rsp)
	}

	c, recorder = newTestContext(http.MethodPost, "/api/ai/screen-timelapse/state", `{"enabled":false}`)
	service.SetScreenTimelapseState(c)
	rsp = decodeScreenTimelapseStateResponse(t, recorder)
	if rsp.Code != 0 || rsp.Data.Enabled {
		t.Fatalf("expected disabled response, got %#v", rsp)
	}
	if !stringSlicesEqual(runtimeCalls, []string{"runtime:true:true:true:true:0s", "runtime:false:false:false:true:0s"}) {
		t.Fatalf("expected AI runtime disable call, got %#v", runtimeCalls)
	}
	snapshot, err = DefaultServiceStateManager().List()
	if err != nil {
		t.Fatalf("list service state after disable: %v", err)
	}
	screenTimelapse = snapshot.Services[aiServiceScreenTimelapse]
	if screenTimelapse.Enabled || screenTimelapse.Source != "web" {
		t.Fatalf("expected screen timelapse service state disabled from web, got %#v", screenTimelapse)
	}
	if exists, err := stateFileExists(screenPath); err != nil || exists {
		t.Fatalf("expected screen timelapse state file still absent, exists=%t err=%v", exists, err)
	}
}

func TestProtectedScreenTimelapseEnableRequiresChallengeBeforeStateMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withAIStateFiles(t)
	writeScreenTimelapseServiceState(t, false)

	service := &Service{}
	screenTimelapseEnableRequiresChallenge = func() bool { return true }
	verifyScreenTimelapseEnableChallenge = func(*gin.Context, string, string) error {
		return errors.New("challenge required")
	}

	c, recorder := newTestContext(http.MethodPost, "/api/ai/screen-timelapse/state", `{"enabled":true}`)
	service.SetScreenTimelapseState(c)
	if rsp := decodeScreenTimelapseStateResponse(t, recorder); rsp.Code != -3 {
		t.Fatalf("expected protected enable to be rejected, got %#v", rsp)
	}
	if enabled, err := getScreenTimelapseEnabled(); err != nil || enabled {
		t.Fatalf("rejected protected enable changed state: enabled=%t err=%v", enabled, err)
	}

	var gotChallenge, gotAction string
	verifyScreenTimelapseEnableChallenge = func(_ *gin.Context, challenge, action string) error {
		gotChallenge = challenge
		gotAction = action
		return nil
	}
	c, recorder = newTestContext(http.MethodPost, "/api/ai/screen-timelapse/state", `{"enabled":true,"challenge":"screen-timelapse-challenge"}`)
	service.SetScreenTimelapseState(c)
	if rsp := decodeScreenTimelapseStateResponse(t, recorder); rsp.Code != 0 || !rsp.Data.Enabled {
		t.Fatalf("expected authorized enable response, got %#v", rsp)
	}
	if gotChallenge != "screen-timelapse-challenge" || gotAction != authservice.SensitiveActionScreenTimelapseEnable {
		t.Fatalf("unexpected protected enable verification: challenge=%q action=%q", gotChallenge, gotAction)
	}

	verifyScreenTimelapseEnableChallenge = func(*gin.Context, string, string) error {
		t.Fatal("screen timelapse disable must not verify a challenge")
		return nil
	}
	c, recorder = newTestContext(http.MethodPost, "/api/ai/screen-timelapse/state", `{"enabled":false}`)
	service.SetScreenTimelapseState(c)
	if rsp := decodeScreenTimelapseStateResponse(t, recorder); rsp.Code != 0 || rsp.Data.Enabled {
		t.Fatalf("expected disable without challenge to succeed, got %#v", rsp)
	}

	screenTimelapseEnableRequiresChallenge = func() bool { return false }
	c, recorder = newTestContext(http.MethodPost, "/api/ai/screen-timelapse/state", `{"enabled":true}`)
	service.SetScreenTimelapseState(c)
	if rsp := decodeScreenTimelapseStateResponse(t, recorder); rsp.Code != 0 || !rsp.Data.Enabled {
		t.Fatalf("expected unprotected enable without challenge to succeed, got %#v", rsp)
	}
}

func TestScreenTimelapseRuntimeFailureDoesNotWriteServiceState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withAIStateFiles(t)
	var runtimeCalls []string
	setTestAIRuntime(t, errTestAIRuntime, &runtimeCalls)

	if err := DefaultServiceStateManager().Save(); err != nil {
		t.Fatalf("save initial service state: %v", err)
	}
	before, beforeState := readServiceStateFile(t, serviceStateFilePath)

	service := &Service{}
	c, recorder := newTestContext(http.MethodPost, "/api/ai/screen-timelapse/state", `{"enabled":true}`)
	service.SetScreenTimelapseState(c)
	rsp := decodeScreenTimelapseStateResponse(t, recorder)
	if rsp.Code == 0 {
		t.Fatalf("expected failed response, got %#v", rsp)
	}
	if !stringSlicesEqual(runtimeCalls, []string{"runtime:true:true:true:true:0s"}) {
		t.Fatalf("expected AI runtime call, got %#v", runtimeCalls)
	}

	after, afterState := readServiceStateFile(t, serviceStateFilePath)
	if string(before) != string(after) {
		t.Fatalf("expected service state file unchanged after failed hook")
	}
	if afterState.Revision != beforeState.Revision ||
		afterState.Services[aiServiceScreenTimelapse] != beforeState.Services[aiServiceScreenTimelapse] {
		t.Fatalf("expected service state unchanged, before=%#v after=%#v", beforeState, afterState)
	}
}

func TestSocketScreenTimelapseStateMatchesWebState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	withAIStateFiles(t)
	setTestAIRuntime(t, nil, nil)
	setTestStateSocketCapabilities(t, 512, chipTypeAX620QE)

	socketPath := filepath.Join(t.TempDir(), "nanokvm-state.sock")
	stubGateHandlerForTest(t)
	socketServer := newStateSocketServer(DefaultServiceStateManager(), socketPath)
	if err := socketServer.Start(); err != nil {
		t.Fatalf("start state socket: %v", err)
	}
	t.Cleanup(func() {
		_ = socketServer.Stop()
	})
	client := newTestUnixSocketClient(t, socketPath)

	service := &Service{}
	resp := doStateSocketRequest(t, client, http.MethodPut, "/v1/services/screenTimelapse", []byte(`{"enabled":true,"source":"screen"}`))
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected socket enable status 200, got %d", resp.StatusCode)
	}

	c, recorder := newTestContext(http.MethodGet, "/api/ai/screen-timelapse/state", "")
	service.GetScreenTimelapseState(c)
	rsp := decodeScreenTimelapseStateResponse(t, recorder)
	if rsp.Code != 0 || !rsp.Data.Enabled {
		t.Fatalf("expected Web state to show socket-enabled screen timelapse, got %#v", rsp)
	}

	resp = doStateSocketRequest(t, client, http.MethodPut, "/v1/services/screenTimelapse", []byte(`{"enabled":false,"source":"screen"}`))
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected socket disable status 200, got %d", resp.StatusCode)
	}

	c, recorder = newTestContext(http.MethodGet, "/api/ai/screen-timelapse/state", "")
	service.GetScreenTimelapseState(c)
	rsp = decodeScreenTimelapseStateResponse(t, recorder)
	if rsp.Code != 0 || rsp.Data.Enabled {
		t.Fatalf("expected Web state to show socket-disabled screen timelapse, got %#v", rsp)
	}
}
