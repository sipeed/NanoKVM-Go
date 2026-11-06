package application

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

const (
	validEnabledFeatureManifest  = `{"version":1,"updated_at":"2026-09-22T10:00:00Z","features":{"tailscale_cn":{"enabled":true}}}`
	validDisabledFeatureManifest = `{"features":{"tailscale_cn":{"enabled":false}}}`
)

// useTestServerFeatureSource points the sync URL and client seams at a test
// source and restores the production defaults with t.Cleanup.
func useTestServerFeatureSource(t *testing.T, baseURL string, client *http.Client) {
	t.Helper()
	previousBaseURL := serverFeatureBaseURL
	previousClient := serverFeatureHTTPClient
	serverFeatureBaseURL = baseURL
	if client == nil {
		client = newServerFeatureHTTPClient()
	}
	serverFeatureHTTPClient = client
	t.Cleanup(func() {
		serverFeatureBaseURL = previousBaseURL
		serverFeatureHTTPClient = previousClient
	})
}

// useServerFeatureSyncEnabled replaces the pull-toggle seam for one test and
// restores the production getter with t.Cleanup. Combined with
// useTestServerFeatureStore (which pins the seam to true) the last call wins, so
// a toggle test only needs to call this helper after the store helper.
func useServerFeatureSyncEnabled(t *testing.T, enabled bool) {
	t.Helper()
	previous := serverFeatureSyncEnabled
	serverFeatureSyncEnabled = func() bool { return enabled }
	t.Cleanup(func() { serverFeatureSyncEnabled = previous })
}

func newFeatureManifestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func assertManifestBytes(t *testing.T, path, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read manifest %s: %v", path, err)
	}
	if string(content) != want {
		t.Fatalf("manifest %s = %q, want %q", path, content, want)
	}
}

func assertNoTemporaryManifestFiles(t *testing.T, directory string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(directory, ".*-*.tmp"))
	if err != nil {
		t.Fatalf("glob temporary files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files remain: %v", matches)
	}
}

func manifestInode(t *testing.T, path string) uint64 {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("stat %s: unexpected Sys() type %T", path, info.Sys())
	}
	return stat.Ino
}

// featureManifestRequestCounter counts requests made by manifest sync tests.
type featureManifestRequestCounter struct {
	mu       sync.Mutex
	requests int
}

func (counter *featureManifestRequestCounter) handler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		counter.mu.Lock()
		counter.requests++
		counter.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}
}

func (counter *featureManifestRequestCounter) count() int {
	counter.mu.Lock()
	defer counter.mu.Unlock()
	return counter.requests
}

// TestServerFeatureManifestStaysOnLegacyRoot pins the manifest to the pre-v2
// channel root: devices on both sides of the v2 move must read one copy.
func TestServerFeatureManifestStaysOnLegacyRoot(t *testing.T) {
	got, err := serverFeatureManifestURL(proto.ApplicationRegionCN)
	if err != nil {
		t.Fatalf("serverFeatureManifestURL(cn) error = %v", err)
	}
	if want := "https://cdn.sipeed.com/nanokvm-go/config/cn/features.json"; got != want {
		t.Fatalf("serverFeatureManifestURL(cn) = %q, want %q", got, want)
	}
	if !strings.HasPrefix(StableURL, LegacyBaseURL+"/v2") || !strings.HasPrefix(PreviewURL, StableURL+"/") {
		t.Fatalf("update channels %q, %q must live under %q/v2", StableURL, PreviewURL, LegacyBaseURL)
	}
}

func TestServerFeatureManifestURL(t *testing.T) {
	previous := serverFeatureBaseURL
	serverFeatureBaseURL = "https://cdn.example"
	t.Cleanup(func() { serverFeatureBaseURL = previous })

	got, err := serverFeatureManifestURL(proto.ApplicationRegionCN)
	if err != nil {
		t.Fatalf("serverFeatureManifestURL(cn) error = %v", err)
	}
	if want := "https://cdn.example/config/cn/features.json"; got != want {
		t.Fatalf("serverFeatureManifestURL(cn) = %q, want %q", got, want)
	}
	if global, err := serverFeatureManifestURL(proto.ApplicationRegionGlobal); err != nil || global != got {
		t.Fatalf("global manifest source = %q, %v; want the shared source %q", global, err, got)
	}
	if _, err := serverFeatureManifestURL(proto.ApplicationRegion("invalid")); err == nil {
		t.Fatal("invalid region unexpectedly has a manifest source")
	}
}

func TestFeatureManifestSyncWritesValidManifest(t *testing.T) {
	var (
		mu        sync.Mutex
		gotPath   string
		gotMethod string
		gotAccept string
	)
	server := newFeatureManifestServer(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotAccept = r.Header.Get("Accept")
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, validEnabledFeatureManifest)
	})

	directory := t.TempDir()
	path := filepath.Join(directory, "server-feature.json")
	useTestServerFeatureStore(t, path)
	useTestServerFeatureSource(t, server.URL, server.Client())

	if err := SyncRegionFeatureManifest(context.Background(), proto.ApplicationRegionCN); err != nil {
		t.Fatalf("SyncRegionFeatureManifest() error = %v", err)
	}

	assertManifestBytes(t, path, validEnabledFeatureManifest)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat manifest: %v", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o644); got != want {
		t.Fatalf("manifest mode = %o, want %o", got, want)
	}
	assertNoTemporaryManifestFiles(t, directory)

	mu.Lock()
	defer mu.Unlock()
	if gotMethod != http.MethodGet {
		t.Fatalf("request method = %q, want GET", gotMethod)
	}
	if gotPath != "/config/cn/features.json" {
		t.Fatalf("request path = %q, want /config/cn/features.json", gotPath)
	}
	if gotAccept != "application/json" {
		t.Fatalf("Accept header = %q, want application/json", gotAccept)
	}
}

func TestFeatureManifestSyncWritesDisabledManifest(t *testing.T) {
	server := newFeatureManifestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, validDisabledFeatureManifest)
	})

	directory := t.TempDir()
	path := filepath.Join(directory, "server-feature.json")
	useTestServerFeatureStore(t, path)
	useTestServerFeatureSource(t, server.URL, server.Client())

	if err := SyncRegionFeatureManifest(context.Background(), proto.ApplicationRegionCN); err != nil {
		t.Fatalf("SyncRegionFeatureManifest() error = %v", err)
	}

	assertManifestBytes(t, path, validDisabledFeatureManifest)
	flags, ok := classifyServerFeatureManifest([]byte(validDisabledFeatureManifest))
	if !ok || flags.TailscaleCnEnabled {
		t.Fatalf("disabled manifest classified as flags=%+v ok=%v, want valid and disabled", flags, ok)
	}
	assertNoTemporaryManifestFiles(t, directory)
}

// TestFeatureManifestSyncSkipsIdenticalPayload covers the toggle-enabled
// (production default) identical-payload fast path: with the pull toggle on,
// the sync still fetches once but leaves the inode and mtime untouched.
func TestFeatureManifestSyncSkipsIdenticalPayload(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "server-feature.json")
	writeServerFeatureManifest(t, path, validEnabledFeatureManifest)

	pinned := time.Unix(1_000_000, 0)
	if err := os.Chtimes(path, pinned, pinned); err != nil {
		t.Fatalf("pin mtime: %v", err)
	}
	beforeInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat before: %v", err)
	}
	beforeInode := manifestInode(t, path)

	server := newFeatureManifestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, validEnabledFeatureManifest)
	})
	useTestServerFeatureStore(t, path)
	useTestServerFeatureSource(t, server.URL, server.Client())

	if err := SyncRegionFeatureManifest(context.Background(), proto.ApplicationRegionCN); err != nil {
		t.Fatalf("SyncRegionFeatureManifest() error = %v", err)
	}

	afterInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after: %v", err)
	}
	if !afterInfo.ModTime().Equal(beforeInfo.ModTime()) {
		t.Fatalf("mtime changed from %v to %v on an identical payload", beforeInfo.ModTime(), afterInfo.ModTime())
	}
	if afterInode := manifestInode(t, path); afterInode != beforeInode {
		t.Fatalf("inode changed from %d to %d on an identical payload", beforeInode, afterInode)
	}
	assertManifestBytes(t, path, validEnabledFeatureManifest)
	assertNoTemporaryManifestFiles(t, directory)
}

func TestFeatureManifestSyncRejectsNon200(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := newFeatureManifestServer(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				_, _ = io.WriteString(w, validEnabledFeatureManifest)
			})

			directory := t.TempDir()
			path := filepath.Join(directory, "server-feature.json")
			writeServerFeatureManifest(t, path, validDisabledFeatureManifest)
			useTestServerFeatureStore(t, path)
			useTestServerFeatureSource(t, server.URL, server.Client())

			if err := SyncRegionFeatureManifest(context.Background(), proto.ApplicationRegionCN); err == nil {
				t.Fatalf("SyncRegionFeatureManifest() error = nil, want status %d failure", status)
			}
			assertManifestBytes(t, path, validDisabledFeatureManifest)
			assertNoTemporaryManifestFiles(t, directory)
		})
	}
}

func TestFeatureManifestSyncRejectsOversizedBody(t *testing.T) {
	server := newFeatureManifestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, strings.Repeat("x", serverFeatureMaxBytes+1))
	})

	directory := t.TempDir()
	path := filepath.Join(directory, "server-feature.json")
	writeServerFeatureManifest(t, path, validEnabledFeatureManifest)
	useTestServerFeatureStore(t, path)
	useTestServerFeatureSource(t, server.URL, server.Client())

	if err := SyncRegionFeatureManifest(context.Background(), proto.ApplicationRegionCN); err == nil {
		t.Fatal("SyncRegionFeatureManifest() error = nil, want oversized failure")
	}
	assertManifestBytes(t, path, validEnabledFeatureManifest)
	assertNoTemporaryManifestFiles(t, directory)
}

func TestFeatureManifestSyncRejectsInvalidManifests(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "json syntax error", body: `{"features":`},
		{name: "top level array", body: `[true]`},
		{name: "features not an object", body: `{"features":[]}`},
		{name: "enabled string", body: `{"features":{"tailscale_cn":{"enabled":"true"}}}`},
		{name: "enabled number", body: `{"features":{"tailscale_cn":{"enabled":1}}}`},
		{name: "enabled null", body: `{"features":{"tailscale_cn":{"enabled":null}}}`},
		{name: "enabled array", body: `{"features":{"tailscale_cn":{"enabled":[]}}}`},
		{name: "enabled missing", body: `{"features":{"tailscale_cn":{}}}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := newFeatureManifestServer(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, tc.body)
			})

			directory := t.TempDir()
			path := filepath.Join(directory, "server-feature.json")
			writeServerFeatureManifest(t, path, validEnabledFeatureManifest)
			useTestServerFeatureStore(t, path)
			useTestServerFeatureSource(t, server.URL, server.Client())

			if err := SyncRegionFeatureManifest(context.Background(), proto.ApplicationRegionCN); err == nil {
				t.Fatalf("SyncRegionFeatureManifest() error = nil for %s", tc.name)
			}
			assertManifestBytes(t, path, validEnabledFeatureManifest)
			assertNoTemporaryManifestFiles(t, directory)
		})
	}
}

func TestFeatureManifestSyncSurfacesTransportFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	baseURL := server.URL
	client := server.Client()
	server.Close()

	directory := t.TempDir()
	path := filepath.Join(directory, "server-feature.json")
	writeServerFeatureManifest(t, path, validEnabledFeatureManifest)
	useTestServerFeatureStore(t, path)
	useTestServerFeatureSource(t, baseURL, client)

	if err := SyncRegionFeatureManifest(context.Background(), proto.ApplicationRegionCN); err == nil {
		t.Fatal("SyncRegionFeatureManifest() error = nil, want transport failure")
	}
	assertManifestBytes(t, path, validEnabledFeatureManifest)
	assertNoTemporaryManifestFiles(t, directory)
}

func TestFeatureManifestSyncRejectsCrossHostRedirect(t *testing.T) {
	t.Run("redirect policy refuses before the hop", func(t *testing.T) {
		counter := &featureManifestRequestCounter{}
		second := newFeatureManifestServer(t, counter.handler(validEnabledFeatureManifest))
		redirector := newFeatureManifestServer(t, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, second.URL+"/config/cn/features.json", http.StatusMovedPermanently)
		})

		directory := t.TempDir()
		path := filepath.Join(directory, "server-feature.json")
		writeServerFeatureManifest(t, path, validDisabledFeatureManifest)
		useTestServerFeatureStore(t, path)
		useTestServerFeatureSource(t, redirector.URL, newServerFeatureHTTPClient())

		if err := SyncRegionFeatureManifest(context.Background(), proto.ApplicationRegionCN); err == nil {
			t.Fatal("SyncRegionFeatureManifest() error = nil, want cross-host redirect failure")
		}
		assertManifestBytes(t, path, validDisabledFeatureManifest)
		assertNoTemporaryManifestFiles(t, directory)
		if count := counter.count(); count != 0 {
			t.Fatalf("redirect target received %d requests, want 0", count)
		}
	})

	t.Run("permissive client is caught by the final host check", func(t *testing.T) {
		counter := &featureManifestRequestCounter{}
		second := newFeatureManifestServer(t, counter.handler(validEnabledFeatureManifest))
		redirector := newFeatureManifestServer(t, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, second.URL+"/config/cn/features.json", http.StatusMovedPermanently)
		})

		directory := t.TempDir()
		path := filepath.Join(directory, "server-feature.json")
		writeServerFeatureManifest(t, path, validDisabledFeatureManifest)
		useTestServerFeatureStore(t, path)
		useTestServerFeatureSource(t, redirector.URL, redirector.Client())

		if err := SyncRegionFeatureManifest(context.Background(), proto.ApplicationRegionCN); err == nil {
			t.Fatal("SyncRegionFeatureManifest() error = nil, want final-host mismatch failure")
		}
		assertManifestBytes(t, path, validDisabledFeatureManifest)
		assertNoTemporaryManifestFiles(t, directory)
		if count := counter.count(); count != 1 {
			t.Fatalf("redirect target received %d requests, want 1", count)
		}
	})
}

func TestServerFeatureHTTPClientRedirectPolicy(t *testing.T) {
	client := newServerFeatureHTTPClient()
	if client.Timeout != serverFeatureSyncTimeout {
		t.Fatalf("client timeout = %v, want %v", client.Timeout, serverFeatureSyncTimeout)
	}
	if client.CheckRedirect == nil {
		t.Fatal("client CheckRedirect = nil, want a redirect policy")
	}

	original, err := http.NewRequest(http.MethodGet, "https://cdn.sipeed.com/nanokvm-go/config/cn/features.json", nil)
	if err != nil {
		t.Fatal(err)
	}
	newVia := func(count int) []*http.Request {
		via := make([]*http.Request, 0, count)
		for i := 0; i < count; i++ {
			request, err := http.NewRequest(http.MethodGet, "https://cdn.sipeed.com/hop", nil)
			if err != nil {
				t.Fatal(err)
			}
			via = append(via, request)
		}
		return via
	}

	cases := []struct {
		name    string
		target  string
		via     []*http.Request
		refused bool
	}{
		{
			name:   "same scheme and host allowed",
			target: "https://cdn.sipeed.com/nanokvm-go/config/cn/features.json",
			via:    []*http.Request{original},
		},
		{
			name:    "https to http downgrade refused",
			target:  "http://cdn.sipeed.com/nanokvm-go/config/cn/features.json",
			via:     []*http.Request{original},
			refused: true,
		},
		{
			name:    "cross host refused",
			target:  "https://evil.example/nanokvm-go/config/cn/features.json",
			via:     []*http.Request{original},
			refused: true,
		},
		{
			name:   "hop count at limit allowed",
			target: "https://cdn.sipeed.com/hop",
			via:    newVia(serverFeatureMaxRedirects),
		},
		{
			name:    "hop count above limit refused",
			target:  "https://cdn.sipeed.com/hop",
			via:     newVia(serverFeatureMaxRedirects + 1),
			refused: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target, err := http.NewRequest(http.MethodGet, tc.target, nil)
			if err != nil {
				t.Fatal(err)
			}
			err = client.CheckRedirect(target, tc.via)
			if tc.refused {
				if !errors.Is(err, http.ErrUseLastResponse) {
					t.Fatalf("CheckRedirect() error = %v, want ErrUseLastResponse", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("CheckRedirect() error = %v, want nil", err)
			}
		})
	}
}

func TestFeatureManifestSyncRejectsUnsafeTargets(t *testing.T) {
	server := newFeatureManifestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, validEnabledFeatureManifest)
	})

	t.Run("symlink target", func(t *testing.T) {
		directory := t.TempDir()
		target := filepath.Join(directory, "target.json")
		writeServerFeatureManifest(t, target, validDisabledFeatureManifest)
		path := filepath.Join(directory, "server-feature.json")
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
		useTestServerFeatureStore(t, path)
		useTestServerFeatureSource(t, server.URL, server.Client())

		if err := SyncRegionFeatureManifest(context.Background(), proto.ApplicationRegionCN); err == nil {
			t.Fatal("SyncRegionFeatureManifest() error = nil, want symlink rejection")
		}
		assertManifestBytes(t, target, validDisabledFeatureManifest)
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Fatal("symlink target was replaced by a regular file")
		}
		assertNoTemporaryManifestFiles(t, directory)
	})

	t.Run("directory target", func(t *testing.T) {
		directory := t.TempDir()
		path := filepath.Join(directory, "server-feature.json")
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
		useTestServerFeatureStore(t, path)
		useTestServerFeatureSource(t, server.URL, server.Client())

		if err := SyncRegionFeatureManifest(context.Background(), proto.ApplicationRegionCN); err == nil {
			t.Fatal("SyncRegionFeatureManifest() error = nil, want directory rejection")
		}
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		if !info.IsDir() {
			t.Fatal("directory target was replaced")
		}
		assertNoTemporaryManifestFiles(t, directory)
	})
}

func TestFeatureManifestSyncCreatesMissingParentDirectory(t *testing.T) {
	// MkdirAll would otherwise be masked by the process umask, so pin it to
	// assert the requested 0o755.
	previousUmask := syscall.Umask(0)
	defer syscall.Umask(previousUmask)

	server := newFeatureManifestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, validEnabledFeatureManifest)
	})

	root := t.TempDir()
	directory := filepath.Join(root, "nested", "kvm")
	path := filepath.Join(directory, "server-feature.json")
	useTestServerFeatureStore(t, path)
	useTestServerFeatureSource(t, server.URL, server.Client())

	if err := SyncRegionFeatureManifest(context.Background(), proto.ApplicationRegionCN); err != nil {
		t.Fatalf("SyncRegionFeatureManifest() error = %v", err)
	}
	assertManifestBytes(t, path, validEnabledFeatureManifest)
	info, err := os.Stat(directory)
	if err != nil {
		t.Fatalf("stat created directory: %v", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o755); got != want {
		t.Fatalf("created directory mode = %o, want %o", got, want)
	}
	assertNoTemporaryManifestFiles(t, directory)
}

func TestFeatureManifestSyncSkipsInvalidRegions(t *testing.T) {
	counter := &featureManifestRequestCounter{}
	server := newFeatureManifestServer(t, counter.handler(validEnabledFeatureManifest))

	cases := []struct {
		name   string
		region proto.ApplicationRegion
	}{
		{name: "unknown", region: proto.ApplicationRegion("unknown")},
		{name: "empty", region: proto.ApplicationRegion("")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			path := filepath.Join(directory, "server-feature.json")
			writeServerFeatureManifest(t, path, validDisabledFeatureManifest)
			useTestServerFeatureStore(t, path)
			useTestServerFeatureSource(t, server.URL, server.Client())

			if err := SyncRegionFeatureManifest(context.Background(), tc.region); err != nil {
				t.Fatalf("SyncRegionFeatureManifest(%q) error = %v, want nil", tc.region, err)
			}
			assertManifestBytes(t, path, validDisabledFeatureManifest)
			assertNoTemporaryManifestFiles(t, directory)
		})
	}

	if count := counter.count(); count != 0 {
		t.Fatalf("invalid regions issued %d HTTP requests, want 0", count)
	}
}

func TestFeatureManifestSyncToggleDisabledSkipsExistingManifest(t *testing.T) {
	cases := []struct {
		name    string
		content string
		state   FeatureState
	}{
		{name: "valid enabled", content: validEnabledFeatureManifest, state: FeatureStateEnabled},
		{name: "valid disabled", content: validDisabledFeatureManifest, state: FeatureStateDisabled},
		{name: "corrupt", content: `{"features":`, state: FeatureStateDisabled},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			counter := &featureManifestRequestCounter{}
			server := newFeatureManifestServer(t, counter.handler(validEnabledFeatureManifest))

			directory := t.TempDir()
			path := filepath.Join(directory, "server-feature.json")
			writeServerFeatureManifest(t, path, tc.content)
			useTestServerFeatureStore(t, path)
			useTestServerFeatureSource(t, server.URL, server.Client())
			useServerFeatureSyncEnabled(t, false)
			before := manifestInode(t, path)

			if err := SyncRegionFeatureManifest(context.Background(), proto.ApplicationRegionCN); err != nil {
				t.Fatalf("SyncRegionFeatureManifest() error = %v, want nil skip", err)
			}
			if count := counter.count(); count != 0 {
				t.Fatalf("disabled sync issued %d HTTP requests, want 0", count)
			}
			if got := LoadServerFeatureState(); got != tc.state {
				t.Fatalf("local manifest state = %v, want %v", got, tc.state)
			}
			assertManifestBytes(t, path, tc.content)
			if after := manifestInode(t, path); after != before {
				t.Fatalf("manifest inode changed from %d to %d on a skipped sync", before, after)
			}
			assertNoTemporaryManifestFiles(t, directory)
		})
	}
}

func TestFeatureManifestSyncToggleDisabledRepairsMissingManifest(t *testing.T) {
	counter := &featureManifestRequestCounter{}
	server := newFeatureManifestServer(t, counter.handler(validEnabledFeatureManifest))

	directory := t.TempDir()
	path := filepath.Join(directory, "server-feature.json")
	useTestServerFeatureStore(t, path)
	useTestServerFeatureSource(t, server.URL, server.Client())
	useServerFeatureSyncEnabled(t, false)

	if err := SyncRegionFeatureManifest(context.Background(), proto.ApplicationRegionCN); err != nil {
		t.Fatalf("SyncRegionFeatureManifest() error = %v", err)
	}
	if count := counter.count(); count != 1 {
		t.Fatalf("disabled sync issued %d HTTP requests, want exactly 1 repair", count)
	}
	if got := LoadServerFeatureState(); got != FeatureStateEnabled {
		t.Fatalf("repaired manifest state = %v, want enabled", got)
	}
	assertManifestBytes(t, path, validEnabledFeatureManifest)
	assertNoTemporaryManifestFiles(t, directory)
}

func TestFeatureManifestSyncToggleDisabledCreatesMissingParentDirectory(t *testing.T) {
	// MkdirAll would otherwise be masked by the process umask, so pin it to
	// assert the requested 0o755.
	previousUmask := syscall.Umask(0)
	defer syscall.Umask(previousUmask)

	counter := &featureManifestRequestCounter{}
	server := newFeatureManifestServer(t, counter.handler(validEnabledFeatureManifest))

	root := t.TempDir()
	directory := filepath.Join(root, "nested", "kvm")
	path := filepath.Join(directory, "server-feature.json")
	useTestServerFeatureStore(t, path)
	useTestServerFeatureSource(t, server.URL, server.Client())
	useServerFeatureSyncEnabled(t, false)

	if err := SyncRegionFeatureManifest(context.Background(), proto.ApplicationRegionCN); err != nil {
		t.Fatalf("SyncRegionFeatureManifest() error = %v", err)
	}
	if count := counter.count(); count != 1 {
		t.Fatalf("disabled sync issued %d HTTP requests, want exactly 1 repair", count)
	}
	assertManifestBytes(t, path, validEnabledFeatureManifest)
	info, err := os.Stat(directory)
	if err != nil {
		t.Fatalf("stat created directory: %v", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o755); got != want {
		t.Fatalf("created directory mode = %o, want %o", got, want)
	}
	assertNoTemporaryManifestFiles(t, directory)
}

func TestFeatureManifestSyncToggleEnabledOverwritesExistingManifest(t *testing.T) {
	counter := &featureManifestRequestCounter{}
	server := newFeatureManifestServer(t, counter.handler(validEnabledFeatureManifest))

	directory := t.TempDir()
	path := filepath.Join(directory, "server-feature.json")
	writeServerFeatureManifest(t, path, validDisabledFeatureManifest)
	useTestServerFeatureStore(t, path)
	useTestServerFeatureSource(t, server.URL, server.Client())
	useServerFeatureSyncEnabled(t, true)

	if err := SyncRegionFeatureManifest(context.Background(), proto.ApplicationRegionCN); err != nil {
		t.Fatalf("SyncRegionFeatureManifest() error = %v", err)
	}
	if count := counter.count(); count != 1 {
		t.Fatalf("enabled sync issued %d HTTP requests, want 1", count)
	}
	assertManifestBytes(t, path, validEnabledFeatureManifest)
	assertNoTemporaryManifestFiles(t, directory)
}

func TestFeatureManifestSyncToggleDisabledInvalidRegionLeavesNoFile(t *testing.T) {
	counter := &featureManifestRequestCounter{}
	server := newFeatureManifestServer(t, counter.handler(validEnabledFeatureManifest))

	directory := t.TempDir()
	path := filepath.Join(directory, "server-feature.json")
	useTestServerFeatureStore(t, path)
	useTestServerFeatureSource(t, server.URL, server.Client())
	useServerFeatureSyncEnabled(t, false)

	if err := SyncRegionFeatureManifest(context.Background(), proto.ApplicationRegion("invalid")); err != nil {
		t.Fatalf("SyncRegionFeatureManifest(invalid) error = %v, want nil", err)
	}
	if count := counter.count(); count != 0 {
		t.Fatalf("invalid region issued %d HTTP requests, want 0", count)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("manifest stat error = %v, want os.ErrNotExist", err)
	}
}

func TestFeatureManifestSyncToggleDisabledKeepsManifestWrittenDuringFetch(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "server-feature.json")
	useTestServerFeatureStore(t, path)
	useServerFeatureSyncEnabled(t, false)

	server := newFeatureManifestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		// Simulate a developer write that lands during the fetch window. The
		// post-fetch re-check must observe it and refuse to publish over it.
		if err := os.WriteFile(path, []byte(validDisabledFeatureManifest), 0o644); err != nil {
			t.Errorf("write manifest during fetch: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, validEnabledFeatureManifest)
	})
	useTestServerFeatureSource(t, server.URL, server.Client())

	if err := SyncRegionFeatureManifest(context.Background(), proto.ApplicationRegionCN); err != nil {
		t.Fatalf("SyncRegionFeatureManifest() error = %v, want nil skip", err)
	}
	assertManifestBytes(t, path, validDisabledFeatureManifest)
	if got := LoadServerFeatureState(); got != FeatureStateDisabled {
		t.Fatalf("manifest written during fetch was overwritten: state = %v, want disabled", got)
	}
	assertNoTemporaryManifestFiles(t, directory)
}

func newFeatureSyncContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/application/features/sync", nil)
	return c, recorder
}

func assertFeatureSyncResponse(t *testing.T, recorder *httptest.ResponseRecorder, wantCode int) {
	t.Helper()
	var response proto.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode feature sync response: %v", err)
	}
	if recorder.Code != http.StatusOK || response.Code != wantCode {
		t.Fatalf("feature sync response = HTTP %d / %+v, want code %d", recorder.Code, response, wantCode)
	}
}

func TestSyncServerFeaturesPublishesForBothRegions(t *testing.T) {
	for _, region := range []proto.ApplicationRegion{proto.ApplicationRegionCN, proto.ApplicationRegionGlobal} {
		for _, enabled := range []bool{false, true} {
			name := string(region) + "/disabled"
			manifest := validDisabledFeatureManifest
			previous := validEnabledFeatureManifest
			if enabled {
				name = string(region) + "/enabled"
				manifest, previous = previous, manifest
			}
			t.Run(name, func(t *testing.T) {
				counter := &featureManifestRequestCounter{}
				server := newFeatureManifestServer(t, counter.handler(manifest))
				directory := t.TempDir()
				path := filepath.Join(directory, "server-feature.json")
				writeServerFeatureManifest(t, path, previous)
				useTestServerFeatureStore(t, path)
				useTestServerFeatureSource(t, server.URL, server.Client())
				store := &fakeApplicationRegionStore{region: region}
				service := &Service{regionStore: store}

				c, recorder := newFeatureSyncContext()
				service.SyncServerFeatures(c)

				assertFeatureSyncResponse(t, recorder, 0)
				assertManifestBytes(t, path, manifest)
				assertNoTemporaryManifestFiles(t, directory)
				if LoadServerFeatureFlags().TailscaleCnEnabled != enabled {
					t.Fatalf("status readers did not see enabled=%v after the response", enabled)
				}
				if counter.count() != 1 {
					t.Fatalf("feature sync issued %d CDN requests, want 1", counter.count())
				}
				if store.load() != region || len(store.saves) != 0 {
					t.Fatal("feature sync changed the region marker")
				}
			})
		}
	}
}

func TestSyncServerFeaturesFailurePreservesLocalManifest(t *testing.T) {
	server := newFeatureManifestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})
	path := filepath.Join(t.TempDir(), "server-feature.json")
	writeServerFeatureManifest(t, path, validEnabledFeatureManifest)
	useTestServerFeatureStore(t, path)
	useTestServerFeatureSource(t, server.URL, server.Client())
	service := &Service{regionStore: &fakeApplicationRegionStore{region: proto.ApplicationRegionGlobal}}

	c, recorder := newFeatureSyncContext()
	service.SyncServerFeatures(c)

	assertFeatureSyncResponse(t, recorder, -1)
	assertManifestBytes(t, path, validEnabledFeatureManifest)
	if !LoadServerFeatureFlags().TailscaleCnEnabled {
		t.Fatal("failed refresh discarded the existing feature flags")
	}
}

// TestSyncServerFeaturesSkipsExistingManifestWhenToggleDisabled proves the web
// settings trigger inherits the pull toggle through the shared choke point: a
// skipped sync is intentional, so the endpoint still answers success while the
// local manifest stays byte-identical and no CDN request is issued.
func TestSyncServerFeaturesSkipsExistingManifestWhenToggleDisabled(t *testing.T) {
	counter := &featureManifestRequestCounter{}
	server := newFeatureManifestServer(t, counter.handler(validEnabledFeatureManifest))
	directory := t.TempDir()
	path := filepath.Join(directory, "server-feature.json")
	writeServerFeatureManifest(t, path, validDisabledFeatureManifest)
	useTestServerFeatureStore(t, path)
	useTestServerFeatureSource(t, server.URL, server.Client())
	useServerFeatureSyncEnabled(t, false)
	store := &fakeApplicationRegionStore{region: proto.ApplicationRegionCN}
	service := &Service{regionStore: store}
	before := manifestInode(t, path)

	c, recorder := newFeatureSyncContext()
	service.SyncServerFeatures(c)

	assertFeatureSyncResponse(t, recorder, 0)
	if count := counter.count(); count != 0 {
		t.Fatalf("disabled sync issued %d CDN requests, want 0", count)
	}
	assertManifestBytes(t, path, validDisabledFeatureManifest)
	if after := manifestInode(t, path); after != before {
		t.Fatalf("manifest inode changed from %d to %d on a skipped sync", before, after)
	}
	if got := LoadServerFeatureState(); got != FeatureStateDisabled {
		t.Fatalf("local manifest state = %v, want disabled", got)
	}
	if store.load() != proto.ApplicationRegionCN || len(store.saves) != 0 {
		t.Fatal("feature sync changed the region marker")
	}
	assertNoTemporaryManifestFiles(t, directory)
}

func TestSyncServerFeaturesHonorsRequestCancellation(t *testing.T) {
	counter := &featureManifestRequestCounter{}
	server := newFeatureManifestServer(t, counter.handler(validDisabledFeatureManifest))
	path := filepath.Join(t.TempDir(), "server-feature.json")
	writeServerFeatureManifest(t, path, validEnabledFeatureManifest)
	useTestServerFeatureStore(t, path)
	useTestServerFeatureSource(t, server.URL, server.Client())
	service := &Service{regionStore: &fakeApplicationRegionStore{region: proto.ApplicationRegionCN}}

	c, recorder := newFeatureSyncContext()
	ctx, cancel := context.WithCancel(c.Request.Context())
	cancel()
	c.Request = c.Request.WithContext(ctx)
	service.SyncServerFeatures(c)

	assertFeatureSyncResponse(t, recorder, -1)
	assertManifestBytes(t, path, validEnabledFeatureManifest)
	if counter.count() != 0 {
		t.Fatal("canceled feature refresh reached the CDN")
	}
}
