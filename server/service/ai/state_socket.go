package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/application"
	"NanoKVM-Server/service/extensions/tailscale"
	"NanoKVM-Server/service/tailscalegate"

	log "github.com/sirupsen/logrus"
)

const (
	StateSocketPath = "/run/kvm/nanokvm-state.sock"

	stateSocketDefaultSource = "screen"
	stateSocketPermission    = 0o660
	stateSocketMaxRequest    = 64 << 10
	stateSocketReadHeader    = 5 * time.Second
	stateSocketReadTimeout   = 15 * time.Second
	stateSocketIdleTimeout   = time.Minute
	stateSocketWriteTimeout  = 5 * time.Second
	stateSocketMaxHeader     = 32 << 10
)

var (
	stateSocketPath          = StateSocketPath
	defaultStateSocketMu     sync.Mutex
	defaultStateSocketServer *StateSocketServer
)

type StateSocketServer struct {
	mu          sync.Mutex
	manager     *ServiceStateManager
	socketPath  string
	server      *http.Server
	listener    net.Listener
	done        chan struct{}
	gateHandler http.Handler
}

type stateSocketSetServiceRequest struct {
	Enabled *bool  `json:"enabled"`
	Source  string `json:"source,omitempty"`
}

type stateSocketErrorResponse struct {
	Error string `json:"error"`
}

type stateSocketServicesResponse struct {
	Revision int64                                 `json:"revision"`
	Services map[string]stateSocketServiceResponse `json:"services"`
}

type stateSocketServiceResponse struct {
	Enabled   bool   `json:"enabled"`
	Supported bool   `json:"supported"`
	Running   *bool  `json:"running,omitempty"`
	Source    string `json:"source,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

func StartStateSocketServer() error {
	defaultStateSocketMu.Lock()
	defer defaultStateSocketMu.Unlock()

	if defaultStateSocketServer != nil {
		return nil
	}

	server := newStateSocketServer(DefaultServiceStateManager(), stateSocketPath)
	if err := server.Start(); err != nil {
		return err
	}

	defaultStateSocketServer = server
	return nil
}

func StopStateSocketServer() error {
	defaultStateSocketMu.Lock()
	server := defaultStateSocketServer
	defaultStateSocketServer = nil
	defaultStateSocketMu.Unlock()

	if server == nil {
		return nil
	}
	return server.Stop()
}

func newStateSocketServer(manager *ServiceStateManager, socketPath string) *StateSocketServer {
	return &StateSocketServer{
		manager:     manager,
		socketPath:  socketPath,
		gateHandler: newGateHandler(),
	}
}

// newGateHandler builds the production Tailscale login-gate handler. It is a
// package-level seam so tests can substitute a hermetic handler; production
// code must never call anything else.
var newGateHandler = productionGateHandler

// newGateDriverForSocket is the driver construction seam: production builds
// the real managed driver over the shared DefaultService singleton, while
// tests substitute a fake so they never read device state under /etc/kvm.
var newGateDriverForSocket = func() (tailscalegate.Driver, error) {
	return tailscale.NewGateDriver(tailscale.DefaultService())
}

// gateLoadRegion is the region-marker read seam for the gate handler.
var gateLoadRegion = application.LoadRegion

// gateLoadFeatureState is the tri-state server-feature manifest read seam for
// the gate handler. It defaults to the production reader, which fails closed,
// so an un-injected handler never advertises the managed mode. Unlike the
// boolean reader it lets the handler tell a missing manifest (eligible for a
// consent-gated sync) apart from one that forbids the feature.
var gateLoadFeatureState = application.LoadServerFeatureState

// gateSyncFeatureManifest is the consent-gated manifest sync seam for the gate
// handler. The handler calls it at most once per start, only after consent is
// resolved and only when the manifest is missing. Production wires the real
// bounded CDN sync; the seam keeps host tests from performing network I/O and
// lets them script the manifest transition.
var gateSyncFeatureManifest = func(ctx context.Context, region proto.ApplicationRegion) error {
	return application.SyncRegionFeatureManifest(ctx, region)
}

// gateLoadConsent is the privacy-terms consent read seam for the gate handler.
// It lazily reaches the shared managed service, so building the handler
// performs no device I/O and host tests that only construct it stay hermetic.
var gateLoadConsent = func() bool {
	return tailscale.DefaultService().SipeedDisclaimerAccepted()
}

// gatePersistConsent is the privacy-terms consent write seam for the gate
// handler. Like the read seam it is invoked only while serving a consented
// start, never at handler construction time.
var gatePersistConsent = func() error {
	return tailscale.DefaultService().GrantSipeedDisclaimer()
}

// productionGateHandler mounts the gate on the real managed Tailscale driver.
// Construction is fail-closed: when the managed mode is not wired, the gate
// answers with the explicit not-implemented driver (503 / NOTIMPL) instead of
// the simulated demo driver, so a misconfigured device can never fake a
// tailnet peer.
func productionGateHandler() http.Handler {
	return gateHandlerForDriver(newGateDriverForSocket())
}

// gateHandlerForDriver assembles the gate handler, mapping a driver
// construction failure onto the not-implemented driver.
func gateHandlerForDriver(driver tailscalegate.Driver, err error) http.Handler {
	if err != nil {
		log.Warnf("tailscale gate driver unavailable: %v", err)
		return tailscalegate.NewHandler(tailscalegate.Deps{
			Driver:              tailscalegate.NewUnimplementedDriver(),
			LoadRegion:          gateLoadRegion,
			LoadFeatureState:    gateLoadFeatureState,
			LoadConsent:         gateLoadConsent,
			PersistConsent:      gatePersistConsent,
			SyncFeatureManifest: gateSyncFeatureManifest,
		})
	}
	return tailscalegate.NewHandler(tailscalegate.Deps{
		Driver:              driver,
		LoadRegion:          gateLoadRegion,
		LoadFeatureState:    gateLoadFeatureState,
		LoadConsent:         gateLoadConsent,
		PersistConsent:      gatePersistConsent,
		SyncFeatureManifest: gateSyncFeatureManifest,
	})
}

func (s *StateSocketServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listener != nil {
		return nil
	}
	if s.manager == nil {
		return errors.New("service state manager is nil")
	}
	if s.socketPath == "" {
		return errors.New("state socket path is empty")
	}

	if err := s.manager.Load(); err != nil {
		return fmt.Errorf("load service state: %w", err)
	}
	if err := prepareStateSocketPath(s.socketPath); err != nil {
		return err
	}

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return err
	}
	if err := os.Chmod(s.socketPath, stateSocketPermission); err != nil {
		_ = listener.Close()
		_ = removeStateSocketFile(s.socketPath)
		return err
	}

	s.listener = listener
	s.server = newStateSocketHTTPServer(s.routes())
	s.done = make(chan struct{})

	go func(server *http.Server, listener net.Listener, done chan struct{}) {
		defer close(done)
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Errorf("state socket server failed: %v", err)
		}
	}(s.server, s.listener, s.done)

	return nil
}

func newStateSocketHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: stateSocketReadHeader,
		ReadTimeout:       stateSocketReadTimeout,
		IdleTimeout:       stateSocketIdleTimeout,
		MaxHeaderBytes:    stateSocketMaxHeader,
		// /v1/events is long-lived. Each event write gets its own deadline.
	}
}

func (s *StateSocketServer) Stop() error {
	s.mu.Lock()
	server := s.server
	socketPath := s.socketPath
	done := s.done
	s.server = nil
	s.listener = nil
	s.done = nil
	s.mu.Unlock()

	var stopErr error
	if server != nil {
		if err := server.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			stopErr = err
		}
	}
	if done != nil {
		<-done
	}
	if err := removeStateSocketFile(socketPath); err != nil && stopErr == nil {
		stopErr = err
	}
	return stopErr
}

func (s *StateSocketServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/services", s.handleServices)
	mux.HandleFunc("/v1/services/", s.handleService)
	mux.HandleFunc("/v1/events", s.handleEvents)

	// The Tailscale login gate shares this root-local socket but owns its own
	// path and JSON error shape. The path is registered exactly, so every other
	// path under this socket keeps answering 404 without reaching the gate.
	// A nil field (for example a literal StateSocketServer{}) falls back to the
	// production handler, because ServeMux panics while registering a nil
	// handler.
	gate := s.gateHandler
	if gate == nil {
		gate = newGateHandler()
	}
	mux.Handle(tailscalegate.LoginCodePath, gate)

	return mux
}

func (s *StateSocketServer) handleServices(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/v1/services" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		writeStateSocketMethodNotAllowed(w, http.MethodGet)
		return
	}

	snapshot, err := s.manager.List()
	if err != nil {
		writeStateSocketError(w, http.StatusInternalServerError, "list services failed")
		return
	}
	response, err := buildStateSocketServicesResponse(snapshot)
	if err != nil {
		writeStateSocketError(w, http.StatusInternalServerError, "build services response failed")
		return
	}

	writeStateSocketJSON(w, http.StatusOK, response)
}

func (s *StateSocketServer) handleService(w http.ResponseWriter, r *http.Request) {
	serviceID := strings.TrimPrefix(r.URL.Path, "/v1/services/")
	if serviceID == "" || strings.Contains(serviceID, "/") || !isValidAIServiceID(serviceID) {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPut {
		writeStateSocketMethodNotAllowed(w, http.MethodPut)
		return
	}

	var req stateSocketSetServiceRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, stateSocketMaxRequest))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeStateSocketError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		writeStateSocketError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Enabled == nil {
		writeStateSocketError(w, http.StatusBadRequest, "enabled is required")
		return
	}

	source := normalizeStateSocketSource(req.Source)
	snapshot, err := s.manager.Set(serviceID, *req.Enabled, source)
	if err != nil {
		writeStateSocketError(w, http.StatusInternalServerError, "set service state failed")
		return
	}
	response, err := buildStateSocketServicesResponse(snapshot)
	if err != nil {
		writeStateSocketError(w, http.StatusInternalServerError, "build services response failed")
		return
	}

	writeStateSocketJSON(w, http.StatusOK, response)
}

func (s *StateSocketServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/v1/events" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		writeStateSocketMethodNotAllowed(w, http.MethodGet)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeStateSocketError(w, http.StatusInternalServerError, "streaming is not supported")
		return
	}

	ch := s.manager.Subscribe()
	defer s.manager.Unsubscribe(ch)

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	controller := http.NewResponseController(w)
	if err := controller.SetWriteDeadline(time.Now().Add(stateSocketWriteTimeout)); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return
	}
	w.WriteHeader(http.StatusOK)
	flusher.Flush()
	_ = controller.SetWriteDeadline(time.Time{})

	encoder := json.NewEncoder(w)
	for {
		select {
		case event := <-ch:
			if err := controller.SetWriteDeadline(time.Now().Add(stateSocketWriteTimeout)); err != nil && !errors.Is(err, http.ErrNotSupported) {
				return
			}
			if err := encoder.Encode(event); err != nil {
				return
			}
			flusher.Flush()
			_ = controller.SetWriteDeadline(time.Time{})
		case <-r.Context().Done():
			return
		}
	}
}

func prepareStateSocketPath(socketPath string) error {
	dir := filepath.Dir(socketPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	info, err := os.Lstat(socketPath)
	if err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return fmt.Errorf("state socket path exists and is not a socket: %s", socketPath)
		}
		return os.Remove(socketPath)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func removeStateSocketFile(socketPath string) error {
	if socketPath == "" {
		return nil
	}

	info, err := os.Lstat(socketPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSocket == 0 {
		return nil
	}
	return os.Remove(socketPath)
}

func buildStateSocketServicesResponse(snapshot ServiceStateSnapshot) (stateSocketServicesResponse, error) {
	memoryMiB, err := readVisibleMemoryMiB()
	if err != nil {
		return stateSocketServicesResponse{}, err
	}

	capabilities := buildAIServices(memoryMiB, memoryTierForChipType(readChipType()))
	supported := make(map[string]bool, len(capabilities.Services))
	for _, service := range capabilities.Services {
		supported[service.ID] = service.Supported
	}

	services := make(map[string]stateSocketServiceResponse, len(snapshot.Services))
	for _, serviceID := range stateSocketServiceIDs() {
		entry := snapshot.Services[serviceID]
		service := stateSocketServiceResponse{
			Enabled:   entry.Enabled,
			Supported: supported[serviceID],
			Source:    entry.Source,
			UpdatedAt: entry.UpdatedAt,
		}
		if serviceID == aiServiceMemoryFabric {
			running, err := newMemoryFabricManager().status()
			if err != nil {
				return stateSocketServicesResponse{}, err
			}
			logMemoryFabricMismatch(entry.Enabled, running)
			service.Running = &running
		}
		services[serviceID] = service
	}

	return stateSocketServicesResponse{
		Revision: snapshot.Revision,
		Services: services,
	}, nil
}

func stateSocketServiceIDs() []string {
	return []string{
		aiServiceMCP,
		aiServiceMemoryFabric,
		aiServiceScreenTimelapse,
	}
}

func normalizeStateSocketSource(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return stateSocketDefaultSource
	}
	return source
}

func writeStateSocketMethodNotAllowed(w http.ResponseWriter, allowed string) {
	w.Header().Set("Allow", allowed)
	writeStateSocketError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeStateSocketError(w http.ResponseWriter, status int, msg string) {
	writeStateSocketJSON(w, status, stateSocketErrorResponse{Error: msg})
}

func writeStateSocketJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
