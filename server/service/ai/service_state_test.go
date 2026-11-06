package ai

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

var fixedServiceStateTime = time.Date(2026, 6, 18, 10, 20, 30, 0, time.UTC)

type serviceStateTestEnv struct {
	manager    *ServiceStateManager
	statePath  string
	configPath string
	mcpPath    string
	screenPath string
}

func newTestServiceStateManager(t *testing.T) (*ServiceStateManager, string) {
	t.Helper()

	env := newTestServiceStateEnv(t)
	return env.manager, env.statePath
}

func newTestServiceStateEnv(t *testing.T) serviceStateTestEnv {
	t.Helper()

	dir := t.TempDir()
	oldConfigFilePath := configFilePath
	oldMCPEnableFilePath := mcpEnableFilePath
	oldScreenTimelapseEnableFilePath := screenTimelapseEnableFilePath

	configFilePath = filepath.Join(dir, "ai.json")
	mcpEnableFilePath = filepath.Join(dir, "mcp_enable")
	screenTimelapseEnableFilePath = filepath.Join(dir, "scr_enable")

	t.Cleanup(func() {
		configFilePath = oldConfigFilePath
		mcpEnableFilePath = oldMCPEnableFilePath
		screenTimelapseEnableFilePath = oldScreenTimelapseEnableFilePath
	})

	path := filepath.Join(dir, "services.json")
	manager := newServiceStateManager(path, func() time.Time {
		return fixedServiceStateTime
	})
	return serviceStateTestEnv{
		manager:    manager,
		statePath:  path,
		configPath: configFilePath,
		mcpPath:    mcpEnableFilePath,
		screenPath: screenTimelapseEnableFilePath,
	}
}

func readServiceStateFile(t *testing.T, path string) ([]byte, ServiceStateSnapshot) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read services state file: %v", err)
	}

	var state ServiceStateSnapshot
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("decode services state file: %v", err)
	}

	return data, state
}

func TestServiceStateManagerDefaultsWhenMissing(t *testing.T) {
	manager, path := newTestServiceStateManager(t)

	if err := manager.Load(); err != nil {
		t.Fatalf("load missing services state: %v", err)
	}

	snapshot, err := manager.List()
	if err != nil {
		t.Fatalf("list services state: %v", err)
	}
	if snapshot.Version != serviceStateVersion || snapshot.Revision != 1 {
		t.Fatalf("unexpected default snapshot header: %#v", snapshot)
	}

	for _, serviceID := range []string{aiServiceMCP, aiServiceMemoryFabric, aiServiceScreenTimelapse} {
		entry, err := manager.Get(serviceID)
		if err != nil {
			t.Fatalf("get default service %s: %v", serviceID, err)
		}
		if entry.Enabled || entry.UpdatedAt != fixedServiceStateTime.Format(time.RFC3339) ||
			entry.Source != serviceStateSourceMigration {
			t.Fatalf("unexpected default entry for %s: %#v", serviceID, entry)
		}
	}

	_, saved := readServiceStateFile(t, path)
	if saved.Revision != 1 {
		t.Fatalf("expected missing services state to be migrated and saved, got %#v", saved)
	}
}

func TestServiceStateManagerMigratesLegacyMCPEnabled(t *testing.T) {
	env := newTestServiceStateEnv(t)
	if err := saveConfigToPath(env.configPath, Config{MCPEnabled: true, MCPAPIKey: "test-key"}); err != nil {
		t.Fatalf("write legacy ai config: %v", err)
	}
	if err := setStateFile(env.mcpPath); err != nil {
		t.Fatalf("create legacy mcp state file: %v", err)
	}
	if err := setStateFile(env.screenPath); err != nil {
		t.Fatalf("create legacy screen state file: %v", err)
	}

	if err := env.manager.Load(); err != nil {
		t.Fatalf("load migrated state: %v", err)
	}

	snapshot, err := env.manager.List()
	if err != nil {
		t.Fatalf("list migrated state: %v", err)
	}
	if snapshot.Revision != 1 {
		t.Fatalf("expected migrated revision 1, got %#v", snapshot)
	}
	mcp := snapshot.Services[aiServiceMCP]
	if !mcp.Enabled || mcp.Source != serviceStateSourceMigration ||
		mcp.UpdatedAt != fixedServiceStateTime.Format(time.RFC3339) {
		t.Fatalf("unexpected migrated mcp state: %#v", mcp)
	}
	if snapshot.Services[aiServiceMemoryFabric].Enabled ||
		snapshot.Services[aiServiceScreenTimelapse].Enabled {
		t.Fatalf("expected non-MCP services to migrate disabled, got %#v", snapshot.Services)
	}

	if exists, err := stateFileExists(env.mcpPath); err != nil || exists {
		t.Fatalf("expected legacy mcp state file removed, exists=%t err=%v", exists, err)
	}
	if exists, err := stateFileExists(env.screenPath); err != nil || exists {
		t.Fatalf("expected legacy screen state file removed, exists=%t err=%v", exists, err)
	}
}

func TestServiceStateManagerExistingStateIgnoresLegacyAIConfig(t *testing.T) {
	env := newTestServiceStateEnv(t)
	if err := saveConfigToPath(env.configPath, Config{MCPEnabled: true, MCPAPIKey: "test-key"}); err != nil {
		t.Fatalf("write legacy ai config: %v", err)
	}

	existing := ServiceStateSnapshot{
		Version:  serviceStateVersion,
		Revision: 4,
		Services: map[string]ServiceStateEntry{
			aiServiceMCP:             {Enabled: false, UpdatedAt: "2026-06-18T00:00:00Z", Source: "web"},
			aiServiceMemoryFabric:    {},
			aiServiceScreenTimelapse: {},
		},
	}
	if err := saveServiceStateToPath(env.statePath, existing); err != nil {
		t.Fatalf("write existing service state: %v", err)
	}

	if err := env.manager.Load(); err != nil {
		t.Fatalf("load existing service state: %v", err)
	}

	entry, err := env.manager.Get(aiServiceMCP)
	if err != nil {
		t.Fatalf("get mcp state: %v", err)
	}
	if entry.Enabled {
		t.Fatalf("expected existing service state to ignore legacy mcpEnabled=true, got %#v", entry)
	}
}

func TestServiceStateManagerInvalidLegacyAIConfigDoesNotCreateServicesState(t *testing.T) {
	env := newTestServiceStateEnv(t)
	if err := os.WriteFile(env.configPath, []byte(`{"mcpEnabled":`), 0o600); err != nil {
		t.Fatalf("write invalid legacy ai config: %v", err)
	}

	if err := env.manager.Load(); err == nil {
		t.Fatalf("expected invalid legacy AI config error")
	}
	if _, err := os.Stat(env.statePath); !os.IsNotExist(err) {
		t.Fatalf("expected services state not created, stat err=%v", err)
	}
}

func TestServiceStateManagerLoadsExistingState(t *testing.T) {
	manager, path := newTestServiceStateManager(t)
	expectedTime := "2026-06-18T10:20:30Z"
	existing := ServiceStateSnapshot{
		Version:  serviceStateVersion,
		Revision: 7,
		Services: map[string]ServiceStateEntry{
			aiServiceMCP: {
				Enabled:   true,
				UpdatedAt: expectedTime,
				Source:    "web",
			},
			aiServiceMemoryFabric: {
				Enabled:   false,
				UpdatedAt: expectedTime,
				Source:    "system",
			},
			aiServiceScreenTimelapse: {
				Enabled:   true,
				UpdatedAt: expectedTime,
				Source:    "screen",
			},
		},
	}

	data, err := json.Marshal(existing)
	if err != nil {
		t.Fatalf("encode existing state: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write existing state: %v", err)
	}

	if err := manager.Load(); err != nil {
		t.Fatalf("load existing state: %v", err)
	}

	snapshot, err := manager.List()
	if err != nil {
		t.Fatalf("list existing state: %v", err)
	}
	if snapshot.Version != existing.Version || snapshot.Revision != existing.Revision {
		t.Fatalf("unexpected loaded snapshot header: %#v", snapshot)
	}
	if snapshot.Services[aiServiceMCP] != existing.Services[aiServiceMCP] ||
		snapshot.Services[aiServiceScreenTimelapse] != existing.Services[aiServiceScreenTimelapse] {
		t.Fatalf("unexpected loaded services: %#v", snapshot.Services)
	}
}

func TestServiceStateManagerSaveWritesValidFile(t *testing.T) {
	manager, path := newTestServiceStateManager(t)

	if err := manager.Save(); err != nil {
		t.Fatalf("save default state: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat saved state: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected state file mode 0600, got %o", info.Mode().Perm())
	}

	data, state := readServiceStateFile(t, path)
	if !bytes.HasSuffix(data, []byte("\n")) {
		t.Fatalf("expected saved JSON to end with newline")
	}
	if state.Version != serviceStateVersion || state.Revision != 1 {
		t.Fatalf("unexpected saved state: %#v", state)
	}
}

func TestServiceStateManagerInvalidJSONDoesNotOverwriteFile(t *testing.T) {
	manager, path := newTestServiceStateManager(t)
	original := []byte(`{"version":`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatalf("write invalid state: %v", err)
	}

	if err := manager.Load(); err == nil {
		t.Fatalf("expected invalid JSON load error")
	}
	if _, err := manager.Set(aiServiceMCP, true, "web"); err == nil {
		t.Fatalf("expected invalid JSON set error")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read invalid state after set: %v", err)
	}
	if !bytes.Equal(data, original) {
		t.Fatalf("expected invalid file unchanged, got %q", data)
	}
}

func TestServiceStateManagerSetUpdatesRevisionStateAndSource(t *testing.T) {
	manager, path := newTestServiceStateManager(t)

	snapshot, err := manager.Set(aiServiceMCP, true, "web")
	if err != nil {
		t.Fatalf("set mcp state: %v", err)
	}

	entry := snapshot.Services[aiServiceMCP]
	if snapshot.Revision != 2 || !entry.Enabled || entry.Source != "web" {
		t.Fatalf("unexpected set snapshot: %#v", snapshot)
	}
	if entry.UpdatedAt != fixedServiceStateTime.Format(time.RFC3339) {
		t.Fatalf("unexpected updatedAt: %q", entry.UpdatedAt)
	}

	_, saved := readServiceStateFile(t, path)
	if saved.Revision != 2 || saved.Services[aiServiceMCP] != entry {
		t.Fatalf("unexpected saved set state: %#v", saved)
	}
}

func TestServiceStateManagerSetUsesDefaultSource(t *testing.T) {
	manager, _ := newTestServiceStateManager(t)

	snapshot, err := manager.Set(aiServiceScreenTimelapse, true, "")
	if err != nil {
		t.Fatalf("set state with default source: %v", err)
	}

	entry := snapshot.Services[aiServiceScreenTimelapse]
	if entry.Source != serviceStateDefaultSource {
		t.Fatalf("expected default source %q, got %q", serviceStateDefaultSource, entry.Source)
	}
}

func TestServiceStateManagerRejectsInvalidServiceID(t *testing.T) {
	manager, path := newTestServiceStateManager(t)
	if err := manager.Save(); err != nil {
		t.Fatalf("save default state: %v", err)
	}
	before, beforeState := readServiceStateFile(t, path)

	if _, err := manager.Set("unknown", true, "web"); err == nil {
		t.Fatalf("expected invalid service id error")
	}
	if _, err := manager.Get("unknown"); err == nil {
		t.Fatalf("expected invalid service id get error")
	}

	after, afterState := readServiceStateFile(t, path)
	if !bytes.Equal(before, after) {
		t.Fatalf("expected state file unchanged after invalid service id")
	}
	if afterState.Revision != beforeState.Revision {
		t.Fatalf("expected revision unchanged, before=%d after=%d", beforeState.Revision, afterState.Revision)
	}
}

func TestServiceStateManagerConcurrentSetKeepsValidState(t *testing.T) {
	manager, path := newTestServiceStateManager(t)
	const setCount = 25

	var wg sync.WaitGroup
	errs := make(chan error, setCount)
	services := []string{aiServiceMCP, aiServiceMemoryFabric, aiServiceScreenTimelapse}
	for i := 0; i < setCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := manager.Set(services[i%len(services)], i%2 == 0, "web")
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent set failed: %v", err)
		}
	}

	snapshot, err := manager.List()
	if err != nil {
		t.Fatalf("list state after concurrent set: %v", err)
	}
	if snapshot.Revision != setCount+1 {
		t.Fatalf("expected revision %d, got %d", setCount+1, snapshot.Revision)
	}

	_, saved := readServiceStateFile(t, path)
	if saved.Revision != setCount+1 {
		t.Fatalf("expected saved revision %d, got %d", setCount+1, saved.Revision)
	}
}

func TestServiceStateManagerSubscribeAndUnsubscribe(t *testing.T) {
	manager, _ := newTestServiceStateManager(t)
	ch := manager.Subscribe()

	if _, err := manager.Set(aiServiceMCP, true, "web"); err != nil {
		t.Fatalf("set state with subscriber: %v", err)
	}

	select {
	case event := <-ch:
		if event.Type != serviceStateEventChanged || event.Service != aiServiceMCP || !event.Enabled ||
			event.Source != "web" || event.Revision != 2 {
			t.Fatalf("unexpected event: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for service state event")
	}

	manager.Unsubscribe(ch)

	if _, err := manager.Set(aiServiceMCP, false, "screen"); err != nil {
		t.Fatalf("set state after unsubscribe: %v", err)
	}

	select {
	case event := <-ch:
		t.Fatalf("unexpected event after unsubscribe: %#v", event)
	case <-time.After(100 * time.Millisecond):
	}
}
