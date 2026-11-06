package uistate

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestNewHTTPServerUsesFiniteBounds(t *testing.T) {
	httpServer := newHTTPServer(http.NotFoundHandler())
	if httpServer.ReadHeaderTimeout != socketReadHeaderTimeout ||
		httpServer.ReadTimeout != socketReadTimeout ||
		httpServer.WriteTimeout != socketWriteTimeout ||
		httpServer.IdleTimeout != socketIdleTimeout {
		t.Fatalf("unexpected UI status server timeouts: %#v", httpServer)
	}
	if httpServer.MaxHeaderBytes != socketMaxHeaderBytes {
		t.Fatalf("MaxHeaderBytes = %d, want %d", httpServer.MaxHeaderBytes, socketMaxHeaderBytes)
	}
}

func TestStartCreatesPrivateSocketAndPreservesParentPermissions(t *testing.T) {
	setGinTestMode(t)
	parent := filepath.Join(t.TempDir(), "run")
	if err := os.Mkdir(parent, 0o700); err != nil {
		t.Fatalf("create parent: %v", err)
	}
	if err := os.Chmod(parent, 0o750); err != nil {
		t.Fatalf("set parent mode: %v", err)
	}
	socketPath := filepath.Join(parent, "nanokvm-ui.sock")
	s := newServer(socketPath, testRouteHandlers())

	if err := s.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Stop(); err != nil {
			t.Errorf("stop: %v", err)
		}
	})

	info, err := os.Lstat(socketPath)
	if err != nil {
		t.Fatalf("lstat socket: %v", err)
	}
	if !isSocket(info) {
		t.Fatalf("socket mode = %v, want unix socket", info.Mode())
	}
	if got := info.Mode().Perm(); got != socketPermission {
		t.Fatalf("socket mode = %#o, want %#o", got, socketPermission)
	}

	parentInfo, err := os.Lstat(parent)
	if err != nil {
		t.Fatalf("lstat parent: %v", err)
	}
	if got := parentInfo.Mode().Perm(); got != 0o750 {
		t.Fatalf("parent mode = %#o, want %#o", got, 0o750)
	}

	if err := s.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if _, err := os.Lstat(socketPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("socket remains after stop: %v", err)
	}
}

func TestStartReplacesOnlyStaleSocket(t *testing.T) {
	setGinTestMode(t)
	socketPath := filepath.Join(t.TempDir(), "run", "nanokvm-ui.sock")
	createStaleSocket(t, socketPath)

	s := newServer(socketPath, testRouteHandlers())
	if err := s.Start(); err != nil {
		t.Fatalf("start over stale socket: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Stop(); err != nil {
			t.Errorf("stop: %v", err)
		}
	})

	info, err := os.Lstat(socketPath)
	if err != nil {
		t.Fatalf("lstat replacement socket: %v", err)
	}
	if !isSocket(info) {
		t.Fatalf("replacement mode = %v, want socket", info.Mode())
	}
}

func TestStartRejectsActiveSocket(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "run", "nanokvm-ui.sock")
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o755); err != nil {
		t.Fatalf("create parent: %v", err)
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		t.Fatalf("listen active fixture: %v", err)
	}
	listener.SetUnlinkOnClose(false)
	t.Cleanup(func() {
		_ = listener.Close()
		_ = os.Remove(socketPath)
	})

	s := newServer(socketPath, testRouteHandlers())
	if err := s.Start(); err == nil {
		t.Fatal("active socket was accepted")
	}
	info, err := os.Lstat(socketPath)
	if err != nil {
		t.Fatalf("active socket was removed: %v", err)
	}
	if !isSocket(info) {
		t.Fatalf("active path mode = %v, want socket", info.Mode())
	}
}

func TestStartRejectsUnsafeSocketTargets(t *testing.T) {
	for _, test := range []struct {
		name   string
		create func(t *testing.T, path string)
		verify func(t *testing.T, info os.FileInfo)
	}{
		{
			name: "symlink",
			create: func(t *testing.T, path string) {
				t.Helper()
				target := filepath.Join(filepath.Dir(path), "target")
				if err := os.WriteFile(target, []byte("target"), 0o600); err != nil {
					t.Fatalf("write symlink target: %v", err)
				}
				if err := os.Symlink(target, path); err != nil {
					t.Fatalf("create symlink: %v", err)
				}
			},
			verify: func(t *testing.T, info os.FileInfo) {
				t.Helper()
				if info.Mode()&os.ModeSymlink == 0 {
					t.Fatalf("mode = %v, want symlink", info.Mode())
				}
			},
		},
		{
			name: "regular file",
			create: func(t *testing.T, path string) {
				t.Helper()
				if err := os.WriteFile(path, []byte("preserve"), 0o600); err != nil {
					t.Fatalf("write regular fixture: %v", err)
				}
			},
			verify: func(t *testing.T, info os.FileInfo) {
				t.Helper()
				if !info.Mode().IsRegular() {
					t.Fatalf("mode = %v, want regular file", info.Mode())
				}
			},
		},
		{
			name: "directory",
			create: func(t *testing.T, path string) {
				t.Helper()
				if err := os.Mkdir(path, 0o700); err != nil {
					t.Fatalf("create directory fixture: %v", err)
				}
			},
			verify: func(t *testing.T, info os.FileInfo) {
				t.Helper()
				if !info.IsDir() {
					t.Fatalf("mode = %v, want directory", info.Mode())
				}
			},
		},
		{
			name: "fifo",
			create: func(t *testing.T, path string) {
				t.Helper()
				if err := syscall.Mkfifo(path, 0o600); err != nil {
					t.Fatalf("create FIFO fixture: %v", err)
				}
			},
			verify: func(t *testing.T, info os.FileInfo) {
				t.Helper()
				if info.Mode()&os.ModeNamedPipe == 0 {
					t.Fatalf("mode = %v, want FIFO", info.Mode())
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			socketPath := filepath.Join(t.TempDir(), "run", "nanokvm-ui.sock")
			if err := os.MkdirAll(filepath.Dir(socketPath), 0o755); err != nil {
				t.Fatalf("create parent: %v", err)
			}
			test.create(t, socketPath)

			s := newServer(socketPath, testRouteHandlers())
			if err := s.Start(); err == nil {
				t.Fatalf("%s target was accepted", test.name)
			}
			info, err := os.Lstat(socketPath)
			if err != nil {
				t.Fatalf("%s target was removed: %v", test.name, err)
			}
			test.verify(t, info)
		})
	}
}

func TestStartRejectsUnsafeSocketParent(t *testing.T) {
	for _, test := range []struct {
		name   string
		parent func(t *testing.T, root string) string
	}{
		{
			name: "symlink",
			parent: func(t *testing.T, root string) string {
				t.Helper()
				realParent := filepath.Join(root, "real")
				if err := os.Mkdir(realParent, 0o755); err != nil {
					t.Fatalf("create real parent: %v", err)
				}
				linkParent := filepath.Join(root, "link")
				if err := os.Symlink(realParent, linkParent); err != nil {
					t.Fatalf("create parent symlink: %v", err)
				}
				return linkParent
			},
		},
		{
			name: "regular file",
			parent: func(t *testing.T, root string) string {
				t.Helper()
				parent := filepath.Join(root, "not-a-directory")
				if err := os.WriteFile(parent, []byte("preserve"), 0o600); err != nil {
					t.Fatalf("create parent file: %v", err)
				}
				return parent
			},
		},
		{
			name: "intermediate symlink",
			parent: func(t *testing.T, root string) string {
				t.Helper()
				realParent := filepath.Join(root, "real")
				if err := os.Mkdir(realParent, 0o755); err != nil {
					t.Fatalf("create real parent: %v", err)
				}
				linkParent := filepath.Join(root, "link")
				if err := os.Symlink(realParent, linkParent); err != nil {
					t.Fatalf("create intermediate symlink: %v", err)
				}
				return filepath.Join(linkParent, "nested")
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			parent := test.parent(t, t.TempDir())
			socketPath := filepath.Join(parent, "nanokvm-ui.sock")
			s := newServer(socketPath, testRouteHandlers())
			if err := s.Start(); err == nil {
				t.Fatalf("%s parent was accepted", test.name)
			}
		})
	}
}

func TestStartCleansSocketWhenPostBindIdentityCheckFails(t *testing.T) {
	setGinTestMode(t)
	socketPath := filepath.Join(t.TempDir(), "run", "nanokvm-ui.sock")
	originalLstat := socketLstat
	t.Cleanup(func() { socketLstat = originalLstat })
	want := errors.New("identity unavailable")
	targetLstatCalls := 0
	socketLstat = func(path string) (os.FileInfo, error) {
		if path == socketPath {
			targetLstatCalls++
			// The first lookup is the pre-bind stale-target check. Fail the
			// second lookup instead so this test exercises the post-bind identity
			// failure and its ownership-safe cleanup path.
			if targetLstatCalls == 2 {
				return nil, want
			}
		}
		return originalLstat(path)
	}

	err := newServer(socketPath, testRouteHandlers()).Start()
	if !errors.Is(err, want) {
		t.Fatalf("start error = %v, want %v", err, want)
	}
	if targetLstatCalls < 2 {
		t.Fatalf("target lstat calls = %d, want post-bind identity lookup", targetLstatCalls)
	}
	if _, statErr := os.Lstat(socketPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("socket remains after identity failure: %v", statErr)
	}
}

func TestStartFailuresAreObservableAndLeaveNoOwnedSocket(t *testing.T) {
	t.Run("listen", func(t *testing.T) {
		restoreSocketSeams(t)
		want := errors.New("listen failed")
		unixListen = func(string, string) (net.Listener, error) {
			return nil, want
		}

		socketPath := filepath.Join(t.TempDir(), "run", "nanokvm-ui.sock")
		err := newServer(socketPath, testRouteHandlers()).Start()
		if !errors.Is(err, want) {
			t.Fatalf("start error = %v, want wrapped %v", err, want)
		}
		if _, err := os.Lstat(socketPath); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("socket exists after listen failure: %v", err)
		}
	})

	t.Run("chmod", func(t *testing.T) {
		restoreSocketSeams(t)
		want := errors.New("chmod failed")
		socketChmod = func(string, os.FileMode) error {
			return want
		}

		socketPath := filepath.Join(t.TempDir(), "run", "nanokvm-ui.sock")
		err := newServer(socketPath, testRouteHandlers()).Start()
		if !errors.Is(err, want) {
			t.Fatalf("start error = %v, want wrapped %v", err, want)
		}
		if _, err := os.Lstat(socketPath); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("socket exists after chmod failure: %v", err)
		}
	})
}

func TestServerStartStopIsIdempotentAndPreservesReplacement(t *testing.T) {
	setGinTestMode(t)
	socketPath := filepath.Join(t.TempDir(), "run", "nanokvm-ui.sock")
	s := newServer(socketPath, testRouteHandlers())
	if err := s.Start(); err != nil {
		t.Fatalf("first start: %v", err)
	}
	s.mu.Lock()
	firstListener := s.listener
	s.mu.Unlock()
	if err := s.Start(); err != nil {
		t.Fatalf("second start: %v", err)
	}
	s.mu.Lock()
	secondListener := s.listener
	s.mu.Unlock()
	if secondListener != firstListener {
		t.Fatal("second start replaced the running listener")
	}

	if err := os.Remove(socketPath); err != nil {
		t.Fatalf("remove owned socket for replacement: %v", err)
	}
	const replacement = "replacement must survive"
	if err := os.WriteFile(socketPath, []byte(replacement), 0o600); err != nil {
		t.Fatalf("write replacement: %v", err)
	}

	if err := s.Stop(); err != nil {
		t.Fatalf("stop with replacement: %v", err)
	}
	contents, err := os.ReadFile(socketPath)
	if err != nil {
		t.Fatalf("replacement was removed: %v", err)
	}
	if string(contents) != replacement {
		t.Fatalf("replacement contents = %q", contents)
	}
	if err := s.Stop(); err != nil {
		t.Fatalf("second stop: %v", err)
	}
}

func TestServerCanRestartAfterStop(t *testing.T) {
	setGinTestMode(t)
	socketPath := filepath.Join(t.TempDir(), "run", "nanokvm-ui.sock")
	s := newServer(socketPath, testRouteHandlers())

	if err := s.Start(); err != nil {
		t.Fatalf("first start: %v", err)
	}
	if err := s.Stop(); err != nil {
		t.Fatalf("first stop: %v", err)
	}
	if err := s.Start(); err != nil {
		t.Fatalf("restart: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Stop(); err != nil {
			t.Errorf("final stop: %v", err)
		}
	})

	info, err := os.Lstat(socketPath)
	if err != nil {
		t.Fatalf("lstat restarted socket: %v", err)
	}
	if !isSocket(info) {
		t.Fatalf("restarted socket mode = %v, want unix socket", info.Mode())
	}
}

func TestStartWaitsForConcurrentStop(t *testing.T) {
	setGinTestMode(t)
	restoreSocketSeams(t)

	socketPath := filepath.Join(t.TempDir(), "run", "nanokvm-ui.sock")
	s := newServer(socketPath, testRouteHandlers())
	if err := s.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	originalRemove := socketRemove
	removeStarted := make(chan struct{})
	allowRemove := make(chan struct{})
	socketRemove = func(path string) error {
		close(removeStarted)
		<-allowRemove
		return originalRemove(path)
	}

	stopDone := make(chan error, 1)
	go func() {
		stopDone <- s.Stop()
	}()
	select {
	case <-removeStarted:
	case <-time.After(time.Second):
		t.Fatal("stop did not reach ownership-checked cleanup")
	}

	startDone := make(chan error, 1)
	go func() {
		startDone <- s.Start()
	}()
	select {
	case err := <-startDone:
		t.Fatalf("start completed while stop was still cleaning up: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(allowRemove)
	if err := <-stopDone; err != nil {
		t.Fatalf("stop: %v", err)
	}
	if err := <-startDone; err != nil {
		t.Fatalf("start after stop: %v", err)
	}

	// Restore the seam before the final cleanup so it cannot block on the closed
	// coordination channel a second time.
	socketRemove = originalRemove
	t.Cleanup(func() {
		if err := s.Stop(); err != nil {
			t.Errorf("final stop: %v", err)
		}
	})
}

func TestStopClosesIdleAndInFlightConnectionsAndWaitsForServe(t *testing.T) {
	setGinTestMode(t)
	slowStarted := make(chan struct{})
	slowFinished := make(chan struct{})
	handlers := testRouteHandlers()
	handlers.hdmiCapture = func(c *gin.Context) {
		close(slowStarted)
		<-c.Request.Context().Done()
		close(slowFinished)
	}

	socketPath := filepath.Join(t.TempDir(), "run", "nanokvm-ui.sock")
	s := newServer(socketPath, handlers)
	if err := s.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	s.mu.Lock()
	done := s.done
	s.mu.Unlock()

	idleConnection := dialSocket(t, socketPath)
	defer idleConnection.Close()
	activeConnection := dialSocket(t, socketPath)
	defer activeConnection.Close()
	if _, err := activeConnection.Write([]byte("GET /api/vm/hdmi/capture HTTP/1.1\r\nHost: localhost\r\n\r\n")); err != nil {
		t.Fatalf("write active request: %v", err)
	}
	select {
	case <-slowStarted:
	case <-time.After(time.Second):
		t.Fatal("slow request did not start")
	}

	if err := s.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stop did not wait for the serve goroutine")
	}
	select {
	case <-slowFinished:
	case <-time.After(time.Second):
		t.Fatal("in-flight request was not canceled")
	}

	if err := idleConnection.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set idle read deadline: %v", err)
	}
	if _, err := idleConnection.Read(make([]byte, 1)); err == nil {
		t.Fatal("idle connection remained open after stop")
	}
	if err := activeConnection.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set active read deadline: %v", err)
	}
	if _, err := activeConnection.Read(make([]byte, 1)); err == nil {
		t.Fatal("active connection remained open after stop")
	}
}

func TestUnexpectedServeFailureIsObservable(t *testing.T) {
	setGinTestMode(t)
	socketPath := filepath.Join(t.TempDir(), "run", "nanokvm-ui.sock")
	s := newServer(socketPath, testRouteHandlers())
	if err := s.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	s.mu.Lock()
	listener := s.listener
	done := s.done
	s.mu.Unlock()

	if err := listener.Close(); err != nil {
		t.Fatalf("close listener unexpectedly: %v", err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("serve goroutine did not complete")
	}
	if err := s.ServeError(); err == nil {
		t.Fatal("unexpected serve error was not retained")
	}
	if err := s.Stop(); err != nil {
		t.Fatalf("stop after unexpected serve failure: %v", err)
	}
}

func TestRouterExposesOnlyWhitelistedStatusRoutes(t *testing.T) {
	setGinTestMode(t)
	var mu sync.Mutex
	calls := make(map[string]int)
	handlers := routeHandlers{}
	for _, name := range []string{"hdmi", "wifi", "hid", "hardware", "streamer", "region"} {
		name := name
		handler := func(c *gin.Context) {
			mu.Lock()
			calls[name]++
			mu.Unlock()
			c.String(http.StatusOK, name)
		}
		switch name {
		case "hdmi":
			handlers.hdmiCapture = handler
		case "wifi":
			handlers.wifi = handler
		case "hid":
			handlers.hidMode = handler
		case "hardware":
			handlers.hardware = handler
		case "streamer":
			handlers.streamer = handler
		case "region":
			handlers.setRegion = handler
		}
	}

	router := newRouter(handlers)
	gotRoutes := router.Routes()
	if len(gotRoutes) != 6 {
		t.Fatalf("route count = %d, want 6: %#v", len(gotRoutes), gotRoutes)
	}
	gotRouteNames := make([]string, 0, len(gotRoutes))
	for _, route := range gotRoutes {
		gotRouteNames = append(gotRouteNames, route.Method+" "+route.Path)
	}
	sort.Strings(gotRouteNames)
	wantRouteNames := []string{
		"GET /api/hid/mode",
		"GET /api/network/wifi",
		"GET /api/streamer/local",
		"GET /api/vm/hardware",
		"GET /api/vm/hdmi/capture",
		"POST /api/application/region",
	}
	if !equalStrings(gotRouteNames, wantRouteNames) {
		t.Fatalf("routes = %#v, want %#v", gotRouteNames, wantRouteNames)
	}

	for _, test := range []struct {
		path string
		name string
	}{
		{path: "/api/vm/hdmi/capture", name: "hdmi"},
		{path: "/api/network/wifi", name: "wifi"},
		{path: "/api/hid/mode", name: "hid"},
		{path: "/api/vm/hardware", name: "hardware"},
		{path: "/api/streamer/local", name: "streamer"},
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want %d", test.path, recorder.Code, http.StatusOK)
		}
		if recorder.Body.String() != test.name {
			t.Fatalf("GET %s body = %q, want %q", test.path, recorder.Body.String(), test.name)
		}
	}

	postRecorder := httptest.NewRecorder()
	router.ServeHTTP(postRecorder, httptest.NewRequest(http.MethodPost, "/api/application/region", nil))
	if postRecorder.Code != http.StatusOK {
		t.Fatalf("POST /api/application/region status = %d, want %d", postRecorder.Code, http.StatusOK)
	}
	if postRecorder.Body.String() != "region" {
		t.Fatalf("POST /api/application/region body = %q, want %q", postRecorder.Body.String(), "region")
	}

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		for _, path := range []string{
			"/api/vm/hdmi/capture",
			"/api/network/wifi",
			"/api/hid/mode",
			"/api/vm/hardware",
			"/api/streamer/local",
		} {
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(method, path, nil))
			if recorder.Code != http.StatusMethodNotAllowed && recorder.Code != http.StatusNotFound {
				t.Fatalf("%s %s status = %d, want 404 or 405", method, path, recorder.Code)
			}
		}
	}
	for _, path := range []string{"/api/unknown", "/api/vm/hardware/"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("GET %s status = %d, want %d", path, recorder.Code, http.StatusNotFound)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	for _, name := range []string{"hdmi", "wifi", "hid", "hardware", "streamer", "region"} {
		if calls[name] != 1 {
			t.Fatalf("%s calls = %d, want 1", name, calls[name])
		}
	}
}

func TestRegionWriteRoutePassesThroughHandlerResponse(t *testing.T) {
	setGinTestMode(t)
	var gotMethod, gotPath, gotBody string
	handlers := testRouteHandlers()
	handlers.setRegion = func(c *gin.Context) {
		gotMethod = c.Request.Method
		gotPath = c.Request.URL.Path
		payload, err := io.ReadAll(c.Request.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		gotBody = string(payload)
		c.Header("X-Whitelisted-Write", "set-region")
		c.Data(http.StatusOK, "application/json", []byte(`{"code":0,"msg":"success"}`))
	}

	router := newRouter(handlers)
	request := httptest.NewRequest(http.MethodPost, "/api/application/region", strings.NewReader(`{"region":"global"}`))
	request.Header.Set("Content-Type", "application/json")
	// Deliberately no Authorization header: the root-local socket owns
	// authorization instead of CheckToken or LocalAuth.
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if gotMethod != http.MethodPost || gotPath != "/api/application/region" {
		t.Fatalf("handler saw %s %s, want POST /api/application/region", gotMethod, gotPath)
	}
	if gotBody != `{"region":"global"}` {
		t.Fatalf("handler body = %q, want %q", gotBody, `{"region":"global"}`)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.String() != `{"code":0,"msg":"success"}` {
		t.Fatalf("body = %q, want the handler-owned payload", recorder.Body.String())
	}
	if header := recorder.Header().Get("X-Whitelisted-Write"); header != "set-region" {
		t.Fatalf("handler response header = %q, want %q", header, "set-region")
	}
}

func TestLocalRouterRejectsNonTargetMutations(t *testing.T) {
	setGinTestMode(t)
	invoked := false
	handlers := testRouteHandlers()
	handlers.setRegion = func(c *gin.Context) {
		invoked = true
		c.String(http.StatusOK, "region")
	}

	router := newRouter(handlers)
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(method, "/api/application/region", nil))
		if recorder.Code != http.StatusMethodNotAllowed && recorder.Code != http.StatusNotFound {
			t.Fatalf("%s /api/application/region status = %d, want 404 or 405", method, recorder.Code)
		}
	}
	for _, path := range []string{
		"/api/vm/hardware",
		"/api/network/wifi",
		"/api/hid/mode",
		"/api/vm/hdmi/capture",
		"/api/streamer/local",
		"/api/unknown",
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, path, nil))
		if recorder.Code != http.StatusMethodNotAllowed && recorder.Code != http.StatusNotFound {
			t.Fatalf("POST %s status = %d, want 404 or 405", path, recorder.Code)
		}
	}
	if invoked {
		t.Fatal("setRegion handler was invoked for a rejected mutation")
	}
}

func TestDefaultRouteHandlersWireEveryWhitelistedRoute(t *testing.T) {
	handlers := defaultRouteHandlers()
	for _, test := range []struct {
		name    string
		handler gin.HandlerFunc
	}{
		{name: "hdmiCapture", handler: handlers.hdmiCapture},
		{name: "wifi", handler: handlers.wifi},
		{name: "hidMode", handler: handlers.hidMode},
		{name: "hardware", handler: handlers.hardware},
		{name: "streamer", handler: handlers.streamer},
		{name: "setRegion", handler: handlers.setRegion},
	} {
		if test.handler == nil {
			t.Fatalf("defaultRouteHandlers left %s unset", test.name)
		}
	}
}

func TestServerAllowsKeepAliveAndIsolatesSlowRequests(t *testing.T) {
	setGinTestMode(t)
	t.Run("keep alive", func(t *testing.T) {
		socketPath := filepath.Join(t.TempDir(), "run", "nanokvm-ui.sock")
		s := newServer(socketPath, testRouteHandlers())
		if err := s.Start(); err != nil {
			t.Fatalf("start: %v", err)
		}
		t.Cleanup(func() {
			if err := s.Stop(); err != nil {
				t.Errorf("stop: %v", err)
			}
		})

		connection := dialSocket(t, socketPath)
		defer connection.Close()
		reader := bufio.NewReader(connection)
		for i := 0; i < 2; i++ {
			request, err := http.NewRequest(http.MethodGet, "http://localhost/api/vm/hdmi/capture", nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			if err := request.Write(connection); err != nil {
				t.Fatalf("write request %d: %v", i, err)
			}
			response, err := http.ReadResponse(reader, request)
			if err != nil {
				t.Fatalf("read response %d: %v", i, err)
			}
			body, err := io.ReadAll(response.Body)
			if closeErr := response.Body.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				t.Fatalf("read response body %d: %v", i, err)
			}
			if response.StatusCode != http.StatusOK || string(body) != "hdmi" {
				t.Fatalf("response %d = status %d body %q", i, response.StatusCode, body)
			}
			if response.Close {
				t.Fatalf("response %d forced connection close", i)
			}
		}
	})

	t.Run("slow request", func(t *testing.T) {
		slowStarted := make(chan struct{})
		slowFinished := make(chan struct{})
		handlers := testRouteHandlers()
		handlers.hdmiCapture = func(c *gin.Context) {
			close(slowStarted)
			<-c.Request.Context().Done()
			close(slowFinished)
		}
		handlers.hardware = func(c *gin.Context) {
			c.String(http.StatusOK, "fast")
		}

		socketPath := filepath.Join(t.TempDir(), "run", "nanokvm-ui.sock")
		s := newServer(socketPath, handlers)
		if err := s.Start(); err != nil {
			t.Fatalf("start: %v", err)
		}
		defer s.Stop()

		slowConnection := dialSocket(t, socketPath)
		defer slowConnection.Close()
		if _, err := slowConnection.Write([]byte("GET /api/vm/hdmi/capture HTTP/1.1\r\nHost: localhost\r\n\r\n")); err != nil {
			t.Fatalf("write slow request: %v", err)
		}
		select {
		case <-slowStarted:
		case <-time.After(time.Second):
			t.Fatal("slow request did not start")
		}

		fastConnection := dialSocket(t, socketPath)
		defer fastConnection.Close()
		fastConnection.SetReadDeadline(time.Now().Add(time.Second))
		fastRequest, err := http.NewRequest(http.MethodGet, "http://localhost/api/vm/hardware", nil)
		if err != nil {
			t.Fatalf("new fast request: %v", err)
		}
		if err := fastRequest.Write(fastConnection); err != nil {
			t.Fatalf("write fast request: %v", err)
		}
		fastResponse, err := http.ReadResponse(bufio.NewReader(fastConnection), fastRequest)
		if err != nil {
			t.Fatalf("slow request blocked independent connection: %v", err)
		}
		body, err := io.ReadAll(fastResponse.Body)
		_ = fastResponse.Body.Close()
		if err != nil {
			t.Fatalf("read fast response: %v", err)
		}
		if fastResponse.StatusCode != http.StatusOK || string(body) != "fast" {
			t.Fatalf("fast response = status %d body %q", fastResponse.StatusCode, body)
		}

		stopDone := make(chan error, 1)
		go func() {
			stopDone <- s.Stop()
		}()
		select {
		case err := <-stopDone:
			if err != nil {
				t.Fatalf("stop: %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("stop was blocked by slow request")
		}
		select {
		case <-slowFinished:
		case <-time.After(time.Second):
			t.Fatal("slow request was not canceled by stop")
		}
	})
}

func createStaleSocket(t *testing.T, socketPath string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(socketPath), 0o755); err != nil {
		t.Fatalf("create stale socket parent: %v", err)
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		t.Fatalf("listen stale socket fixture: %v", err)
	}
	listener.SetUnlinkOnClose(false)
	if err := listener.Close(); err != nil {
		t.Fatalf("close stale socket fixture: %v", err)
	}
}

func dialSocket(t *testing.T, socketPath string) net.Conn {
	t.Helper()
	connection, err := net.DialTimeout("unix", socketPath, time.Second)
	if err != nil {
		t.Fatalf("dial socket: %v", err)
	}
	return connection
}

func testRouteHandlers() routeHandlers {
	return routeHandlers{
		hdmiCapture: markerHandler("hdmi"),
		wifi:        markerHandler("wifi"),
		hidMode:     markerHandler("hid"),
		hardware:    markerHandler("hardware"),
		streamer:    markerHandler("streamer"),
		setRegion:   markerHandler("region"),
	}
}

func markerHandler(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(http.StatusOK, name)
	}
}

func setGinTestMode(t *testing.T) {
	t.Helper()
	originalMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() {
		gin.SetMode(originalMode)
	})
}

func restoreSocketSeams(t *testing.T) {
	t.Helper()
	originalListen := unixListen
	originalDialTimeout := unixDialTimeout
	originalLstat := socketLstat
	originalChmod := socketChmod
	originalRemove := socketRemove
	originalMkdir := socketMkdir
	t.Cleanup(func() {
		unixListen = originalListen
		unixDialTimeout = originalDialTimeout
		socketLstat = originalLstat
		socketChmod = originalChmod
		socketRemove = originalRemove
		socketMkdir = originalMkdir
	})
}

func equalStrings(got []string, want []string) bool {
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
