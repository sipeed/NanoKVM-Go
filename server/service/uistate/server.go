// Package uistate owns the root-local HTTP status socket used by kvm_ui. The
// router serves five read-only status GETs plus exactly one privileged write,
// POST /api/application/region, which reuses application.Service.SetRegion.
// The socket is root-only (0600) and is not reachable through TCP, so the
// filesystem permission remains the sole authorization boundary: the write adds
// no HTTP authentication or browser middleware and does not move region
// ownership out of the application service.
package uistate

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"NanoKVM-Server/service/application"
	"NanoKVM-Server/service/extensions/tailscale"
	"NanoKVM-Server/service/hid"
	"NanoKVM-Server/service/network"
	"NanoKVM-Server/service/ui"
	"NanoKVM-Server/service/vm"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	// SocketPath is deliberately fixed: it is the private transport contract with
	// kvm_ui and is not a user-configurable API surface.
	SocketPath = "/run/kvm/nanokvm-ui.sock"

	socketPermission        = 0o600
	socketReadHeaderTimeout = 5 * time.Second
	socketReadTimeout       = 15 * time.Second
	socketWriteTimeout      = 15 * time.Second
	socketIdleTimeout       = time.Minute
	socketMaxHeaderBytes    = 32 << 10
	socketProbeTimeout      = time.Second
	maxUnixSocketPathBytes  = 107 // Linux sockaddr_un.sun_path, excluding NUL.
)

var (
	defaultServerMu sync.Mutex
	defaultServer   *server
	defaultPath     = SocketPath

	unixListen      = net.Listen
	unixDialTimeout = net.DialTimeout
	socketLstat     = os.Lstat
	socketChmod     = os.Chmod
	socketRemove    = os.Remove
	socketMkdir     = os.Mkdir
)

type routeHandlers struct {
	hdmiCapture gin.HandlerFunc
	wifi        gin.HandlerFunc
	hidMode     gin.HandlerFunc
	hardware    gin.HandlerFunc
	streamer    gin.HandlerFunc
	setRegion   gin.HandlerFunc
}

type server struct {
	mu sync.Mutex

	socketPath string
	handlers   routeHandlers

	httpServer *http.Server
	listener   net.Listener
	done       chan struct{}
	identity   os.FileInfo
	serveErr   error

	stopping bool
	stopDone chan struct{}
}

// StartServer starts the production local status socket. Repeated calls while it
// is running are harmless.
func StartServer() error {
	defaultServerMu.Lock()
	defer defaultServerMu.Unlock()

	if defaultServer != nil {
		return defaultServer.Start()
	}

	s := newServer(defaultPath, defaultRouteHandlers())
	if err := s.Start(); err != nil {
		return err
	}

	defaultServer = s
	return nil
}

// StopServer stops the production local status socket. It is safe to call more
// than once.
func StopServer() error {
	defaultServerMu.Lock()
	defer defaultServerMu.Unlock()

	if defaultServer == nil {
		return nil
	}

	s := defaultServer
	defaultServer = nil
	return s.Stop()
}

func newServer(socketPath string, handlers routeHandlers) *server {
	return &server{
		socketPath: socketPath,
		handlers:   handlers,
	}
}

func defaultRouteHandlers() routeHandlers {
	vmService := vm.NewService()
	networkService := network.NewService()
	hidService := hid.NewService()
	applicationService := application.NewService(tailscale.DefaultService())

	return routeHandlers{
		hdmiCapture: vmService.GetHdmiCapture,
		wifi:        networkService.GetWifi,
		hidMode:     hidService.GetHidMode,
		hardware:    vmService.GetHardware,
		streamer:    ui.GetStreamer,
		setRegion:   applicationService.SetRegion,
	}
}

func (s *server) Start() error {
	s.mu.Lock()
	for s.stopping {
		stopDone := s.stopDone
		s.mu.Unlock()
		if stopDone != nil {
			<-stopDone
		}
		s.mu.Lock()
	}
	defer s.mu.Unlock()

	if s.listener != nil {
		return nil
	}
	if err := validateSocketPath(s.socketPath); err != nil {
		return err
	}
	if err := prepareSocketPath(s.socketPath); err != nil {
		return err
	}

	listener, err := unixListen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen on UI status socket: %w", err)
	}
	disableUnixListenerUnlink(listener)

	info, err := socketLstat(s.socketPath)
	if err != nil {
		// The normal production seam is os.Lstat. A direct retry gives test and
		// transient-error paths one chance to capture a filesystem identity for
		// safe cleanup; a listener fd cannot provide that identity because an
		// AF_UNIX socket fd has a distinct kernel inode from its path entry.
		fallback, fallbackErr := os.Lstat(s.socketPath)
		if fallbackErr == nil && isSocket(fallback) {
			_ = listener.Close()
			cleanupErr := removeSocketIfSame(s.socketPath, fallback)
			if cleanupErr != nil {
				return errors.Join(
					fmt.Errorf("identify UI status socket: %w", err),
					fmt.Errorf("remove UI status socket after identity failure: %w", cleanupErr),
				)
			}
			return fmt.Errorf("identify UI status socket: %w", err)
		}
		_ = listener.Close()
		return fmt.Errorf("identify UI status socket: %w", err)
	}
	if !isSocket(info) {
		_ = listener.Close()
		cleanupErr := removeSocketIfSame(s.socketPath, info)
		if cleanupErr != nil {
			return errors.Join(
				fmt.Errorf("UI status socket is not a socket after listen: %s", s.socketPath),
				fmt.Errorf("remove UI status socket after identity failure: %w", cleanupErr),
			)
		}
		return fmt.Errorf("UI status socket is not a socket after listen: %s", s.socketPath)
	}

	if err := socketChmod(s.socketPath, socketPermission); err != nil {
		_ = listener.Close()
		cleanupErr := removeSocketIfSame(s.socketPath, info)
		if cleanupErr != nil {
			return errors.Join(
				fmt.Errorf("set UI status socket permission: %w", err),
				fmt.Errorf("remove UI status socket after permission failure: %w", cleanupErr),
			)
		}
		return fmt.Errorf("set UI status socket permission: %w", err)
	}

	httpServer := newHTTPServer(newRouter(s.handlers))
	done := make(chan struct{})
	s.httpServer = httpServer
	s.listener = listener
	s.done = done
	s.identity = info
	s.serveErr = nil

	go s.serve(httpServer, listener, done)
	return nil
}

func (s *server) Stop() error {
	s.mu.Lock()
	if s.stopping {
		stopDone := s.stopDone
		s.mu.Unlock()
		if stopDone != nil {
			<-stopDone
		}
		return nil
	}
	if s.httpServer == nil {
		s.mu.Unlock()
		return nil
	}

	httpServer := s.httpServer
	done := s.done
	identity := s.identity
	socketPath := s.socketPath
	s.httpServer = nil
	s.listener = nil
	s.done = nil
	s.identity = nil
	s.stopping = true
	s.stopDone = make(chan struct{})
	stopDone := s.stopDone
	s.mu.Unlock()

	var stopErr error
	if err := httpServer.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
		stopErr = fmt.Errorf("close UI status HTTP server: %w", err)
	}
	if done != nil {
		<-done
	}
	if err := removeSocketIfSame(socketPath, identity); err != nil {
		cleanupErr := fmt.Errorf("remove UI status socket: %w", err)
		if stopErr != nil {
			stopErr = errors.Join(stopErr, cleanupErr)
		} else {
			stopErr = cleanupErr
		}
	}

	s.mu.Lock()
	s.stopping = false
	close(stopDone)
	s.stopDone = nil
	s.mu.Unlock()

	return stopErr
}

// ServeError returns the unexpected error from the listener goroutine, if one
// occurred. Closing through Stop is normal and does not populate this value.
func (s *server) ServeError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.serveErr
}

func (s *server) serve(httpServer *http.Server, listener net.Listener, done chan struct{}) {
	defer close(done)

	if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Errorf("UI status socket server failed: %v", err)

		s.mu.Lock()
		if s.httpServer == httpServer && s.listener == listener {
			s.serveErr = err
		}
		s.mu.Unlock()
	}
}

func newRouter(handlers routeHandlers) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	r.HandleMethodNotAllowed = true

	r.GET("/api/vm/hdmi/capture", handlers.hdmiCapture)
	r.GET("/api/network/wifi", handlers.wifi)
	r.GET("/api/hid/mode", handlers.hidMode)
	r.GET("/api/vm/hardware", handlers.hardware)
	r.GET("/api/streamer/local", handlers.streamer)
	r.POST("/api/application/region", handlers.setRegion)

	return r
}

func newHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: socketReadHeaderTimeout,
		ReadTimeout:       socketReadTimeout,
		WriteTimeout:      socketWriteTimeout,
		IdleTimeout:       socketIdleTimeout,
		MaxHeaderBytes:    socketMaxHeaderBytes,
	}
}

func validateSocketPath(socketPath string) error {
	if socketPath == "" {
		return errors.New("UI status socket path is empty")
	}
	if strings.IndexByte(socketPath, 0) >= 0 {
		return errors.New("UI status socket path contains a NUL byte")
	}
	if !filepath.IsAbs(socketPath) {
		return fmt.Errorf("UI status socket path must be absolute: %s", socketPath)
	}
	if socketPath == string(filepath.Separator) || filepath.Clean(socketPath) != socketPath {
		return fmt.Errorf("UI status socket path is unsuitable: %s", socketPath)
	}
	if len(socketPath) > maxUnixSocketPathBytes {
		return fmt.Errorf("UI status socket path is too long: %d bytes", len(socketPath))
	}
	return nil
}

func prepareSocketPath(socketPath string) error {
	if err := ensureSocketParent(socketPath); err != nil {
		return err
	}

	info, err := socketLstat(socketPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("lstat UI status socket path: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("UI status socket path is a symlink: %s", socketPath)
	}
	if !isSocket(info) {
		return fmt.Errorf("UI status socket path exists and is not a socket: %s", socketPath)
	}

	active, err := socketIsActive(socketPath)
	if err != nil {
		return err
	}
	if active {
		return fmt.Errorf("UI status socket is already active: %s", socketPath)
	}
	if err := removeStaleSocket(socketPath, info); err != nil {
		return err
	}
	return nil
}

func ensureSocketParent(socketPath string) error {
	parent := filepath.Dir(socketPath)
	// Walk each component and create only missing directories. Checking every
	// component (before and after creation) prevents an intermediate symlink
	// from redirecting the socket into another tree. Existing components are
	// never chmod'ed or otherwise modified.
	cleanParent := filepath.Clean(parent)
	current := string(filepath.Separator)
	remainder := strings.TrimPrefix(cleanParent, string(filepath.Separator))
	for _, component := range strings.Split(remainder, string(filepath.Separator)) {
		if component == "" {
			continue
		}
		current = filepath.Join(current, component)
		info, err := socketLstat(current)
		if errors.Is(err, os.ErrNotExist) {
			if mkdirErr := socketMkdir(current, 0o755); mkdirErr != nil && !errors.Is(mkdirErr, os.ErrExist) {
				return fmt.Errorf("create UI status socket parent: %w", mkdirErr)
			}
			info, err = socketLstat(current)
		}
		if err != nil {
			return fmt.Errorf("lstat UI status socket parent: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("UI status socket parent is not a directory: %s", current)
		}
	}
	return nil
}

func socketIsActive(socketPath string) (bool, error) {
	conn, err := unixDialTimeout("unix", socketPath, socketProbeTimeout)
	if err == nil {
		_ = conn.Close()
		return true, nil
	}
	if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ENOENT) {
		return false, nil
	}
	return false, fmt.Errorf("probe UI status socket: %w", err)
}

func removeStaleSocket(socketPath string, staleIdentity os.FileInfo) error {
	current, err := socketLstat(socketPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("recheck UI status socket before stale removal: %w", err)
	}
	if !isSocket(current) || !os.SameFile(staleIdentity, current) {
		return fmt.Errorf("UI status socket changed before stale removal: %s", socketPath)
	}
	if err := socketRemove(socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove stale UI status socket: %w", err)
	}
	return nil
}

func removeSocketIfSame(socketPath string, identity os.FileInfo) error {
	if socketPath == "" || identity == nil {
		return nil
	}

	current, err := socketLstat(socketPath)
	if err != nil {
		// Cleanup is best effort but must still be able to remove our own
		// socket when a replaceable test seam (or a transient lstat failure)
		// reports an error. The fallback remains identity-checked below.
		if fallback, fallbackErr := os.Lstat(socketPath); fallbackErr == nil {
			current, err = fallback, nil
		}
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !isSocket(current) || !os.SameFile(identity, current) {
		return nil
	}
	if err := socketRemove(socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func isSocket(info os.FileInfo) bool {
	return info != nil && info.Mode()&os.ModeSymlink == 0 && info.Mode()&os.ModeSocket != 0
}

func disableUnixListenerUnlink(listener net.Listener) {
	if unixListener, ok := listener.(*net.UnixListener); ok {
		unixListener.SetUnlinkOnClose(false)
	}
}
