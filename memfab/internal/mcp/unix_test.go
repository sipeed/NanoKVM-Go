package mcp

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestUnixServerLifecyclePermissionsAndRequest(t *testing.T) {
	db := mcpTestDB(t)
	runtimeDir := filepath.Join(t.TempDir(), "memfab")
	if err := os.Mkdir(runtimeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	socketPath := filepath.Join(runtimeDir, "tools.sock")
	server, err := ListenUnix(socketPath, Handler(db, testConfig()))
	if err != nil {
		skipIfSocketsDenied(t, err)
		t.Fatal(err)
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve() }()

	assertMode(t, runtimeDir, 0o700)
	assertMode(t, socketPath, 0o600)
	client := unixHTTPClient(socketPath)
	response, err := client.Get("http://memfab/v1/tools")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), `"version":1`) {
		t.Fatalf("status=%d body=%s", response.StatusCode, body)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-serveDone; err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("socket remains after shutdown: %v", err)
	}
}

func TestUnixServerRejectsInsecureRuntimeDirectory(t *testing.T) {
	runtimeDir := filepath.Join(t.TempDir(), "memfab")
	if err := os.Mkdir(runtimeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := ListenUnix(filepath.Join(runtimeDir, "tools.sock"), http.NewServeMux())
	if err == nil || !strings.Contains(err.Error(), "mode") {
		t.Fatalf("err=%v", err)
	}
	assertMode(t, runtimeDir, 0o755)
}

func TestUnixServerRejectsNonSocketAndLiveSocket(t *testing.T) {
	t.Run("non-socket", func(t *testing.T) {
		path := socketTestPath(t)
		if err := os.WriteFile(path, []byte("do not delete"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := ListenUnix(path, http.NewServeMux()); err == nil || !strings.Contains(err.Error(), "non-socket") {
			t.Fatalf("err=%v", err)
		}
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "do not delete" {
			t.Fatalf("path changed: data=%q err=%v", data, err)
		}
	})

	t.Run("live socket", func(t *testing.T) {
		path := socketTestPath(t)
		listener, err := net.Listen("unix", path)
		if err != nil {
			skipIfSocketsDenied(t, err)
			t.Fatal(err)
		}
		defer listener.Close()
		if _, err := ListenUnix(path, http.NewServeMux()); err == nil || !strings.Contains(err.Error(), "already in use") {
			t.Fatalf("err=%v", err)
		}
		if _, err := os.Lstat(path); err != nil {
			t.Fatalf("live socket removed: %v", err)
		}
	})
}

func TestUnixServerRecoversStaleSocket(t *testing.T) {
	path := socketTestPath(t)
	addr := &net.UnixAddr{Name: path, Net: "unix"}
	stale, err := net.ListenUnix("unix", addr)
	if err != nil {
		skipIfSocketsDenied(t, err)
		t.Fatal(err)
	}
	stale.SetUnlinkOnClose(false)
	if err := stale.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(path); err != nil {
		t.Fatalf("stale socket missing: %v", err)
	}

	server, err := ListenUnix(path, http.NewServeMux())
	if err != nil {
		skipIfSocketsDenied(t, err)
		t.Fatal(err)
	}
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestUnixServerDoesNotRemoveReplacementSocket(t *testing.T) {
	path := socketTestPath(t)
	server, err := ListenUnix(path, http.NewServeMux())
	if err != nil {
		skipIfSocketsDenied(t, err)
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	replacement, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer replacement.Close()

	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(path); err != nil {
		t.Fatalf("replacement socket removed: %v", err)
	}
}

func TestUnixServerRequiresCleanAbsolutePath(t *testing.T) {
	for _, path := range []string{"", "tools.sock", "/tmp/../tmp/tools.sock"} {
		if _, err := ListenUnix(path, http.NewServeMux()); err == nil {
			t.Fatalf("path %q accepted", path)
		}
	}
}

func TestToolHTTPServerHasFiniteResourceBounds(t *testing.T) {
	server := newToolHTTPServer(http.NewServeMux())
	if server.ReadHeaderTimeout <= 0 || server.ReadTimeout <= 0 || server.WriteTimeout <= 0 || server.IdleTimeout <= 0 {
		t.Fatalf("unbounded tool server timeouts: %+v", server)
	}
	if server.MaxHeaderBytes <= 0 || server.MaxHeaderBytes > 64<<10 {
		t.Fatalf("MaxHeaderBytes=%d", server.MaxHeaderBytes)
	}
}

func unixHTTPClient(socketPath string) *http.Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}
	return &http.Client{Transport: transport, Timeout: 2 * time.Second}
}

func socketTestPath(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "memfab")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "tools.sock")
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode=%#o, want %#o", path, got, want)
	}
}

func skipIfSocketsDenied(t *testing.T, err error) {
	t.Helper()
	if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
		t.Skipf("sandbox does not permit socket creation: %v", err)
	}
}
