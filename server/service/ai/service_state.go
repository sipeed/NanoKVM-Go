package ai

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	ServiceStateFile = "/etc/kvm/services.json"

	serviceStateVersion         = 1
	serviceStateDefaultSource   = "system"
	serviceStateEventChanged    = "service_changed"
	serviceStateSourceMigration = "migration"
)

var (
	serviceStateFilePath         = ServiceStateFile
	defaultServiceStateManagerMu sync.Mutex
	defaultServiceStateManager   *ServiceStateManager
)

type ServiceStateEntry struct {
	Enabled   bool   `json:"enabled"`
	UpdatedAt string `json:"updatedAt"`
	Source    string `json:"source"`
}

type ServiceStateSnapshot struct {
	Version  int                          `json:"version"`
	Revision int64                        `json:"revision"`
	Services map[string]ServiceStateEntry `json:"services"`
}

type ServiceStateEvent struct {
	Type      string `json:"type"`
	Revision  int64  `json:"revision"`
	Service   string `json:"service"`
	Enabled   bool   `json:"enabled"`
	Source    string `json:"source"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type serviceStateRuntimeReconciler func(current ServiceStateSnapshot, next ServiceStateSnapshot, serviceID string, source string) error

type ServiceStateManager struct {
	mu                sync.Mutex
	path              string
	now               func() time.Time
	loaded            bool
	state             ServiceStateSnapshot
	runtimeReconciler serviceStateRuntimeReconciler
	subscribers       map[chan ServiceStateEvent]struct{}
}

func NewServiceStateManager() *ServiceStateManager {
	return newServiceStateManager(serviceStateFilePath, time.Now)
}

func DefaultServiceStateManager() *ServiceStateManager {
	defaultServiceStateManagerMu.Lock()
	defer defaultServiceStateManagerMu.Unlock()

	if defaultServiceStateManager == nil {
		defaultServiceStateManager = NewServiceStateManager()
		defaultServiceStateManager.setRuntimeReconciler(defaultServiceStateRuntimeReconciler)
	}
	return defaultServiceStateManager
}

func newServiceStateManager(path string, now func() time.Time) *ServiceStateManager {
	if now == nil {
		now = time.Now
	}

	return &ServiceStateManager{
		path:  path,
		now:   now,
		state: defaultServiceStateSnapshot(),
	}
}

func (m *ServiceStateManager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, err := m.loadOrMigrateLocked()
	if err != nil {
		return err
	}

	m.state = state
	m.loaded = true
	return nil
}

func (m *ServiceStateManager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureLoadedLocked(); err != nil {
		return err
	}
	return saveServiceStateToPath(m.path, m.state)
}

func (m *ServiceStateManager) Get(serviceID string) (ServiceStateEntry, error) {
	if !isValidAIServiceID(serviceID) {
		return ServiceStateEntry{}, invalidAIServiceIDError(serviceID)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureLoadedLocked(); err != nil {
		return ServiceStateEntry{}, err
	}

	return m.state.Services[serviceID], nil
}

func (m *ServiceStateManager) List() (ServiceStateSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureLoadedLocked(); err != nil {
		return ServiceStateSnapshot{}, err
	}

	return cloneServiceStateSnapshot(m.state), nil
}

func (m *ServiceStateManager) Set(serviceID string, enabled bool, source string) (ServiceStateSnapshot, error) {
	if !isValidAIServiceID(serviceID) {
		return ServiceStateSnapshot{}, invalidAIServiceIDError(serviceID)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.ensureLoadedLocked(); err != nil {
		return ServiceStateSnapshot{}, err
	}

	source = normalizeServiceStateSource(source)
	updatedAt := m.now().UTC().Format(time.RFC3339)

	next := cloneServiceStateSnapshot(m.state)
	next.Revision++
	next.Services[serviceID] = ServiceStateEntry{
		Enabled:   enabled,
		UpdatedAt: updatedAt,
		Source:    source,
	}

	if m.runtimeReconciler != nil {
		current := cloneServiceStateSnapshot(m.state)
		if err := m.runtimeReconciler(current, cloneServiceStateSnapshot(next), serviceID, source); err != nil {
			return ServiceStateSnapshot{}, err
		}
	}

	if err := saveServiceStateToPath(m.path, next); err != nil {
		return ServiceStateSnapshot{}, err
	}

	m.state = next
	m.loaded = true

	event := ServiceStateEvent{
		Type:      serviceStateEventChanged,
		Revision:  next.Revision,
		Service:   serviceID,
		Enabled:   enabled,
		Source:    source,
		UpdatedAt: updatedAt,
	}
	m.broadcastLocked(event)

	return cloneServiceStateSnapshot(m.state), nil
}

func (m *ServiceStateManager) setRuntimeReconciler(reconciler serviceStateRuntimeReconciler) {
	m.runtimeReconciler = reconciler
}

func (m *ServiceStateManager) Subscribe() chan ServiceStateEvent {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.subscribers == nil {
		m.subscribers = make(map[chan ServiceStateEvent]struct{})
	}

	ch := make(chan ServiceStateEvent, 8)
	m.subscribers[ch] = struct{}{}
	return ch
}

func (m *ServiceStateManager) Unsubscribe(ch chan ServiceStateEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.subscribers == nil {
		return
	}
	delete(m.subscribers, ch)
}

func (m *ServiceStateManager) ensureLoadedLocked() error {
	if m.loaded {
		return nil
	}

	state, err := m.loadOrMigrateLocked()
	if err != nil {
		return err
	}

	m.state = state
	m.loaded = true
	return nil
}

func (m *ServiceStateManager) loadOrMigrateLocked() (ServiceStateSnapshot, error) {
	state, err := loadServiceStateFromPath(m.path)
	if err == nil {
		return state, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return ServiceStateSnapshot{}, err
	}

	return m.migrateFromLegacyStateLocked()
}

func (m *ServiceStateManager) migrateFromLegacyStateLocked() (ServiceStateSnapshot, error) {
	cfg, err := loadConfigFromPath(configFilePath)
	if err != nil {
		return ServiceStateSnapshot{}, err
	}

	updatedAt := m.now().UTC().Format(time.RFC3339)
	state := migratedServiceStateSnapshot(cfg.MCPEnabled, updatedAt)
	if err := saveServiceStateToPath(m.path, state); err != nil {
		return ServiceStateSnapshot{}, err
	}
	if err := cleanupLegacyServiceStateFiles(); err != nil {
		return ServiceStateSnapshot{}, err
	}

	return state, nil
}

func (m *ServiceStateManager) broadcastLocked(event ServiceStateEvent) {
	for ch := range m.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}

func loadServiceStateFromPath(path string) (ServiceStateSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ServiceStateSnapshot{}, err
	}

	var state ServiceStateSnapshot
	if err := json.Unmarshal(data, &state); err != nil {
		return ServiceStateSnapshot{}, err
	}

	return normalizeServiceStateSnapshot(state), nil
}

func migratedServiceStateSnapshot(mcpEnabled bool, updatedAt string) ServiceStateSnapshot {
	state := defaultServiceStateSnapshot()
	state.Revision = 1
	state.Services[aiServiceMCP] = ServiceStateEntry{
		Enabled:   mcpEnabled,
		UpdatedAt: updatedAt,
		Source:    serviceStateSourceMigration,
	}
	state.Services[aiServiceMemoryFabric] = ServiceStateEntry{
		Enabled:   false,
		UpdatedAt: updatedAt,
		Source:    serviceStateSourceMigration,
	}
	state.Services[aiServiceScreenTimelapse] = ServiceStateEntry{
		Enabled:   false,
		UpdatedAt: updatedAt,
		Source:    serviceStateSourceMigration,
	}
	return state
}

func cleanupLegacyServiceStateFiles() error {
	if err := removeStateFile(mcpEnableFilePath); err != nil {
		return err
	}
	return removeStateFile(screenTimelapseEnableFilePath)
}

func saveServiceStateToPath(path string, state ServiceStateSnapshot) error {
	state = normalizeServiceStateSnapshot(state)

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, ".services.json.*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}

	dirFile, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer dirFile.Close()
	return dirFile.Sync()
}

func defaultServiceStateSnapshot() ServiceStateSnapshot {
	return ServiceStateSnapshot{
		Version:  serviceStateVersion,
		Revision: 0,
		Services: map[string]ServiceStateEntry{
			aiServiceMCP:             {},
			aiServiceMemoryFabric:    {},
			aiServiceScreenTimelapse: {},
		},
	}
}

func normalizeServiceStateSnapshot(state ServiceStateSnapshot) ServiceStateSnapshot {
	if state.Version == 0 {
		state.Version = serviceStateVersion
	}
	if state.Services == nil {
		state.Services = make(map[string]ServiceStateEntry)
	}

	normalized := ServiceStateSnapshot{
		Version:  state.Version,
		Revision: state.Revision,
		Services: make(map[string]ServiceStateEntry, len(defaultServiceStateSnapshot().Services)),
	}

	for serviceID := range defaultServiceStateSnapshot().Services {
		normalized.Services[serviceID] = state.Services[serviceID]
	}

	return normalized
}

func cloneServiceStateSnapshot(state ServiceStateSnapshot) ServiceStateSnapshot {
	clone := ServiceStateSnapshot{
		Version:  state.Version,
		Revision: state.Revision,
		Services: make(map[string]ServiceStateEntry, len(state.Services)),
	}

	for serviceID, entry := range state.Services {
		clone.Services[serviceID] = entry
	}

	return clone
}

func normalizeServiceStateSource(source string) string {
	if source == "" {
		return serviceStateDefaultSource
	}
	return source
}

func isValidAIServiceID(serviceID string) bool {
	switch serviceID {
	case aiServiceMCP, aiServiceMemoryFabric, aiServiceScreenTimelapse:
		return true
	default:
		return false
	}
}

func invalidAIServiceIDError(serviceID string) error {
	return fmt.Errorf("invalid AI service id %q", serviceID)
}
