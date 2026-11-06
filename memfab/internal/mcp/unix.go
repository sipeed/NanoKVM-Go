package mcp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

const (
	socketDirectoryMode = 0o700
	socketFileMode      = 0o600
)

// UnixServer owns one private HTTP-over-Unix listener and removes only the
// socket inode it created. It never opens a TCP listener.
type UnixServer struct {
	path       string
	listener   *net.UnixListener
	server     *http.Server
	socketInfo os.FileInfo
	closeOnce  sync.Once
	closeErr   error
}

// ListenUnix prepares and binds the local tool backend. A live socket is never
// displaced; a refused, stale socket can be recovered, while non-socket paths
// are rejected without modification.
func ListenUnix(socketPath string, handler http.Handler) (*UnixServer, error) {
	if handler == nil {
		return nil, errors.New("tool backend handler is required")
	}
	if socketPath == "" || !filepath.IsAbs(socketPath) || filepath.Clean(socketPath) != socketPath {
		return nil, errors.New("tool socket path must be a clean absolute path")
	}
	if err := prepareSocketDirectory(filepath.Dir(socketPath)); err != nil {
		return nil, err
	}
	if err := prepareSocketPath(socketPath); err != nil {
		return nil, err
	}

	addr := &net.UnixAddr{Name: socketPath, Net: "unix"}
	listener, err := net.ListenUnix("unix", addr)
	if err != nil {
		return nil, fmt.Errorf("bind tool socket: %w", err)
	}
	listener.SetUnlinkOnClose(false)
	cleanup := func() {
		_ = listener.Close()
		_ = os.Remove(socketPath)
	}
	if err := os.Chmod(socketPath, socketFileMode); err != nil {
		cleanup()
		return nil, fmt.Errorf("secure tool socket: %w", err)
	}
	info, err := os.Lstat(socketPath)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("inspect tool socket: %w", err)
	}

	return &UnixServer{
		path:       socketPath,
		listener:   listener,
		socketInfo: info,
		server:     newToolHTTPServer(handler),
	}, nil
}

func newToolHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
}

func prepareSocketDirectory(dir string) error {
	info, err := os.Lstat(dir)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(dir, socketDirectoryMode); err != nil {
			return fmt.Errorf("create tool socket directory: %w", err)
		}
		info, err = os.Lstat(dir)
	}
	if err != nil {
		return fmt.Errorf("inspect tool socket directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("tool socket parent is not a directory")
	}
	if info.Mode().Perm() != socketDirectoryMode {
		return fmt.Errorf("tool socket directory mode is %#o, want %#o", info.Mode().Perm(), os.FileMode(socketDirectoryMode))
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int64(stat.Uid) != int64(os.Geteuid()) {
		return errors.New("tool socket directory is not owned by the service user")
	}
	return nil
}

func prepareSocketPath(socketPath string) error {
	info, err := os.Lstat(socketPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect existing tool socket: %w", err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		return errors.New("refusing to replace non-socket tool path")
	}

	conn, dialErr := net.DialTimeout("unix", socketPath, 200*time.Millisecond)
	if dialErr == nil {
		_ = conn.Close()
		return errors.New("tool socket is already in use")
	}
	if !errors.Is(dialErr, syscall.ECONNREFUSED) {
		return fmt.Errorf("cannot verify existing tool socket is stale: %w", dialErr)
	}
	if err := os.Remove(socketPath); err != nil {
		return fmt.Errorf("remove stale tool socket: %w", err)
	}
	return nil
}

// Serve blocks until Shutdown, Close, or a listener error.
func (s *UnixServer) Serve() error {
	err := s.server.Serve(s.listener)
	_ = s.cleanup()
	if errors.Is(err, http.ErrServerClosed) || errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

// Shutdown gracefully drains finite tool calls and removes the owned socket.
func (s *UnixServer) Shutdown(ctx context.Context) error {
	err := s.server.Shutdown(ctx)
	cleanupErr := s.cleanup()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return cleanupErr
}

// Close immediately stops the listener and removes the owned socket.
func (s *UnixServer) Close() error {
	err := s.server.Close()
	cleanupErr := s.cleanup()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return cleanupErr
}

func (s *UnixServer) SocketPath() string { return s.path }

func (s *UnixServer) cleanup() error {
	s.closeOnce.Do(func() {
		if err := s.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			s.closeErr = err
		}
		current, err := os.Lstat(s.path)
		if os.IsNotExist(err) {
			return
		}
		if err != nil {
			if s.closeErr == nil {
				s.closeErr = err
			}
			return
		}
		if current.Mode()&os.ModeSocket == 0 || !os.SameFile(s.socketInfo, current) {
			return
		}
		if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) && s.closeErr == nil {
			s.closeErr = err
		}
	})
	return s.closeErr
}
