package application

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// useTestServerFeatureStore points the default manifest store at a test path
// and restores the production default with t.Cleanup.
//
// It also pins the serverFeatureSyncEnabled seam to its production default
// (true). Sync now consults the server.yaml pull toggle, and without this pin
// every pre-existing sync test would call config.GetInstance(), which attempts
// to load or create /etc/kvm/server.yaml. Tests that exercise the other toggle
// value call useServerFeatureSyncEnabled after this helper.
func useTestServerFeatureStore(t *testing.T, path string) *serverFeatureStore {
	t.Helper()
	previous := defaultServerFeatureStore
	store := newServerFeatureStore(path)
	defaultServerFeatureStore = store
	previousEnabled := serverFeatureSyncEnabled
	serverFeatureSyncEnabled = func() bool { return true }
	t.Cleanup(func() {
		defaultServerFeatureStore = previous
		serverFeatureSyncEnabled = previousEnabled
	})
	return store
}

func writeServerFeatureManifest(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestServerFeatureStoreAcceptsOnlyExplicitBoolean(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "explicit true enables",
			content: `{"features":{"tailscale_cn":{"enabled":true}}}`,
			want:    true,
		},
		{
			name:    "explicit false disables",
			content: `{"features":{"tailscale_cn":{"enabled":false}}}`,
			want:    false,
		},
		{
			name:    "missing enabled disables",
			content: `{"features":{"tailscale_cn":{}}}`,
			want:    false,
		},
		{
			name:    "missing tailscale_cn disables",
			content: `{"features":{}}`,
			want:    false,
		},
		{
			name:    "missing features disables",
			content: `{}`,
			want:    false,
		},
		{
			name:    "features not an object disables",
			content: `{"features":[]}`,
			want:    false,
		},
		{
			name:    "tailscale_cn not an object disables",
			content: `{"features":{"tailscale_cn":5}}`,
			want:    false,
		},
		{
			name:    "quoted true disables",
			content: `{"features":{"tailscale_cn":{"enabled":"true"}}}`,
			want:    false,
		},
		{
			name:    "numeric one disables",
			content: `{"features":{"tailscale_cn":{"enabled":1}}}`,
			want:    false,
		},
		{
			name:    "null disables",
			content: `{"features":{"tailscale_cn":{"enabled":null}}}`,
			want:    false,
		},
		{
			name:    "array disables",
			content: `{"features":{"tailscale_cn":{"enabled":[]}}}`,
			want:    false,
		},
		{
			name:    "non-object top level disables",
			content: `[true]`,
			want:    false,
		},
		{
			name:    "boolean top level disables",
			content: `true`,
			want:    false,
		},
		{
			name:    "empty file disables",
			content: ``,
			want:    false,
		},
		{
			name:    "invalid JSON disables",
			content: `{"features":`,
			want:    false,
		},
		{
			name:    "unrelated fields are ignored",
			content: `{"version":3,"updated_at":"2026-09-23T00:00:00Z","features":{"tailscale_cn":{"enabled":true,"note":"ok"}}}`,
			want:    true,
		},
		{
			// encoding/json folds struct field names case-insensitively while the
			// device-side kvm_ui parser matches keys exactly, so a case-different
			// manifest must disable both consumers instead of enabling one.
			name:    "case-different keys disable",
			content: `{"FEATURES":{"TAILSCALE_CN":{"ENABLED":true}}}`,
			want:    false,
		},
		{
			name:    "misnamed leaf disables",
			content: `{"features":{"tailscale_cn":{"enable":true}}}`,
			want:    false,
		},
		{
			name:    "misnamed container disables",
			content: `{"feature":{"tailscale_cn":{"enabled":true}}}`,
			want:    false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "server-feature.json")
			writeServerFeatureManifest(t, path, tc.content)
			store := newServerFeatureStore(path)
			if got := store.load().TailscaleCnEnabled; got != tc.want {
				t.Fatalf("TailscaleCnEnabled = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestServerFeatureStoreStateDistinguishesMissingFromDisabled(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    FeatureState
	}{
		{
			name:    "explicit true is enabled",
			content: `{"features":{"tailscale_cn":{"enabled":true}}}`,
			want:    FeatureStateEnabled,
		},
		{
			name:    "explicit false is disabled",
			content: `{"features":{"tailscale_cn":{"enabled":false}}}`,
			want:    FeatureStateDisabled,
		},
		{
			name:    "missing features is disabled",
			content: `{}`,
			want:    FeatureStateDisabled,
		},
		{
			name:    "missing tailscale_cn is disabled",
			content: `{"features":{}}`,
			want:    FeatureStateDisabled,
		},
		{
			name:    "missing enabled is disabled",
			content: `{"features":{"tailscale_cn":{}}}`,
			want:    FeatureStateDisabled,
		},
		{
			// A readable but malformed manifest is a positive disabled policy;
			// only a rejected read is Missing, so corruption cannot self-heal
			// into a fetch trigger.
			name:    "invalid JSON is disabled not missing",
			content: `{"features":`,
			want:    FeatureStateDisabled,
		},
		{
			name:    "non-object top level is disabled",
			content: `[true]`,
			want:    FeatureStateDisabled,
		},
		{
			name:    "case-different keys are disabled",
			content: `{"FEATURES":{"TAILSCALE_CN":{"ENABLED":true}}}`,
			want:    FeatureStateDisabled,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "server-feature.json")
			writeServerFeatureManifest(t, path, tc.content)
			store := newServerFeatureStore(path)
			if got := store.state(); got != tc.want {
				t.Fatalf("state() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestServerFeatureStoreStateIsMissingWhenNoManifestCanBeRead(t *testing.T) {
	t.Run("absent file", func(t *testing.T) {
		store := newServerFeatureStore(filepath.Join(t.TempDir(), "missing.json"))
		if got := store.state(); got != FeatureStateMissing {
			t.Fatalf("state() = %v, want %v", got, FeatureStateMissing)
		}
	})

	t.Run("missing parent directory", func(t *testing.T) {
		store := newServerFeatureStore(filepath.Join(t.TempDir(), "absent", "server-feature.json"))
		if got := store.state(); got != FeatureStateMissing {
			t.Fatalf("state() = %v, want %v", got, FeatureStateMissing)
		}
	})

	t.Run("oversized file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "server-feature.json")
		writeServerFeatureManifest(t, path, strings.Repeat("x", serverFeatureMaxBytes+1))
		store := newServerFeatureStore(path)
		if got := store.state(); got != FeatureStateMissing {
			t.Fatalf("oversized manifest state = %v, want %v", got, FeatureStateMissing)
		}
	})

	t.Run("file is a symlink", func(t *testing.T) {
		directory := t.TempDir()
		target := filepath.Join(directory, "target.json")
		writeServerFeatureManifest(t, target, `{"features":{"tailscale_cn":{"enabled":true}}}`)
		path := filepath.Join(directory, "server-feature.json")
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
		store := newServerFeatureStore(path)
		if got := store.state(); got != FeatureStateMissing {
			t.Fatalf("symlinked manifest state = %v, want %v", got, FeatureStateMissing)
		}
	})

	t.Run("unreadable file", func(t *testing.T) {
		if os.Geteuid() == 0 {
			// root bypasses the permission bits, so the fixture cannot make
			// OpenFile fail; the branch stays covered on developer and CI hosts
			// that run unprivileged.
			t.Skip("running as root: 0o000 does not deny the open")
		}
		path := filepath.Join(t.TempDir(), "server-feature.json")
		writeServerFeatureManifest(t, path, `{"features":{"tailscale_cn":{"enabled":true}}}`)
		if err := os.Chmod(path, 0o000); err != nil {
			t.Fatal(err)
		}
		store := newServerFeatureStore(path)
		if got := store.state(); got != FeatureStateMissing {
			t.Fatalf("unreadable manifest state = %v, want %v", got, FeatureStateMissing)
		}
	})
}

func TestServerFeatureStoreFailsClosedOnUnsafeFiles(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		store := newServerFeatureStore(filepath.Join(t.TempDir(), "missing.json"))
		if store.load().TailscaleCnEnabled {
			t.Fatal("missing manifest enabled the flag")
		}
	})

	t.Run("missing parent directory", func(t *testing.T) {
		store := newServerFeatureStore(filepath.Join(t.TempDir(), "absent", "server-feature.json"))
		if store.load().TailscaleCnEnabled {
			t.Fatal("manifest in a missing directory enabled the flag")
		}
	})

	t.Run("oversized file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "server-feature.json")
		writeServerFeatureManifest(t, path, strings.Repeat("x", serverFeatureMaxBytes+1))
		store := newServerFeatureStore(path)
		if store.load().TailscaleCnEnabled {
			t.Fatal("oversized manifest enabled the flag")
		}
	})

	t.Run("path is a directory", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "server-feature.json")
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
		store := newServerFeatureStore(path)
		if store.load().TailscaleCnEnabled {
			t.Fatal("directory manifest enabled the flag")
		}
	})

	t.Run("file is a symlink", func(t *testing.T) {
		directory := t.TempDir()
		target := filepath.Join(directory, "target.json")
		writeServerFeatureManifest(t, target, `{"features":{"tailscale_cn":{"enabled":true}}}`)
		path := filepath.Join(directory, "server-feature.json")
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
		store := newServerFeatureStore(path)
		if store.load().TailscaleCnEnabled {
			t.Fatal("symlinked manifest enabled the flag")
		}
	})

	t.Run("parent directory is a symlink", func(t *testing.T) {
		directory := t.TempDir()
		real := filepath.Join(directory, "real")
		writeServerFeatureManifest(t, filepath.Join(real, "server-feature.json"), `{"features":{"tailscale_cn":{"enabled":true}}}`)
		link := filepath.Join(directory, "link")
		if err := os.Symlink(real, link); err != nil {
			t.Fatal(err)
		}
		store := newServerFeatureStore(filepath.Join(link, "server-feature.json"))
		if store.load().TailscaleCnEnabled {
			t.Fatal("manifest behind a symlinked directory enabled the flag")
		}
	})

	t.Run("unreadable file", func(t *testing.T) {
		if os.Geteuid() == 0 {
			// root bypasses the permission bits, so the fixture cannot make
			// OpenFile fail; the branch stays covered on developer and CI hosts
			// that run unprivileged.
			t.Skip("running as root: 0o000 does not deny the open")
		}
		path := filepath.Join(t.TempDir(), "server-feature.json")
		writeServerFeatureManifest(t, path, `{"features":{"tailscale_cn":{"enabled":true}}}`)
		if err := os.Chmod(path, 0o000); err != nil {
			t.Fatal(err)
		}
		store := newServerFeatureStore(path)
		if store.load().TailscaleCnEnabled {
			t.Fatal("unreadable manifest enabled the flag")
		}
	})

	t.Run("parent path is a regular file", func(t *testing.T) {
		directory := t.TempDir()
		parent := filepath.Join(directory, "parent")
		if err := os.WriteFile(parent, []byte("not a directory"), 0o644); err != nil {
			t.Fatal(err)
		}
		store := newServerFeatureStore(filepath.Join(parent, "server-feature.json"))
		if store.load().TailscaleCnEnabled {
			t.Fatal("manifest with a non-directory parent enabled the flag")
		}
	})
}

func TestClassifyServerFeatureManifest(t *testing.T) {
	cases := []struct {
		name      string
		content   string
		wantFlags ServerFeatureFlags
		wantOK    bool
	}{
		{
			name:      "explicit true is valid and enables",
			content:   `{"features":{"tailscale_cn":{"enabled":true}}}`,
			wantFlags: ServerFeatureFlags{TailscaleCnEnabled: true},
			wantOK:    true,
		},
		{
			name:    "explicit false is valid and stays disabled",
			content: `{"features":{"tailscale_cn":{"enabled":false}}}`,
			wantOK:  true,
		},
		{
			name:      "cdn-shaped manifest with metadata",
			content:   `{"version":1,"updated_at":"2026-09-22T10:00:00Z","features":{"tailscale_cn":{"enabled":true}}}`,
			wantFlags: ServerFeatureFlags{TailscaleCnEnabled: true},
			wantOK:    true,
		},
		{name: "invalid JSON", content: `{"features":`, wantOK: false},
		{name: "top level array", content: `[true]`, wantOK: false},
		{name: "top level null", content: `null`, wantOK: false},
		{name: "features is not an object", content: `{"features":[]}`, wantOK: false},
		{name: "tailscale_cn is not an object", content: `{"features":{"tailscale_cn":5}}`, wantOK: false},
		{name: "enabled missing", content: `{"features":{"tailscale_cn":{}}}`, wantOK: false},
		{name: "enabled is a string", content: `{"features":{"tailscale_cn":{"enabled":"true"}}}`, wantOK: false},
		{name: "enabled is a number", content: `{"features":{"tailscale_cn":{"enabled":1}}}`, wantOK: false},
		{name: "enabled is null", content: `{"features":{"tailscale_cn":{"enabled":null}}}`, wantOK: false},
		{name: "enabled is an array", content: `{"features":{"tailscale_cn":{"enabled":[]}}}`, wantOK: false},
		{
			// encoding/json folds struct field names case-insensitively while the
			// device-side kvm_ui parser matches keys exactly, so a case-different
			// manifest must be invalid instead of enabling one consumer.
			name:    "case-different FEATURES",
			content: `{"FEATURES":{"tailscale_cn":{"enabled":true}}}`,
			wantOK:  false,
		},
		{
			name:    "case-different TAILSCALE_CN",
			content: `{"features":{"TAILSCALE_CN":{"enabled":true}}}`,
			wantOK:  false,
		},
		{
			name:    "case-different ENABLED",
			content: `{"features":{"tailscale_cn":{"ENABLED":true}}}`,
			wantOK:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			flags, ok := classifyServerFeatureManifest([]byte(tc.content))
			if ok != tc.wantOK {
				t.Fatalf("classifyServerFeatureManifest(%q) ok = %v, want %v", tc.content, ok, tc.wantOK)
			}
			if flags != tc.wantFlags {
				t.Fatalf("classifyServerFeatureManifest(%q) flags = %+v, want %+v", tc.content, flags, tc.wantFlags)
			}
		})
	}
}

func TestServerFeatureStoreReadFailsClosed(t *testing.T) {
	valid := `{"features":{"tailscale_cn":{"enabled":true}}}`

	t.Run("valid file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "server-feature.json")
		writeServerFeatureManifest(t, path, valid)
		store := newServerFeatureStore(path)
		payload, ok := store.read()
		if !ok {
			t.Fatal("valid manifest reported a failed read")
		}
		if string(payload) != valid {
			t.Fatalf("read payload = %q, want %q", payload, valid)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		store := newServerFeatureStore(filepath.Join(t.TempDir(), "missing.json"))
		if _, ok := store.read(); ok {
			t.Fatal("missing manifest reported a successful read")
		}
	})

	t.Run("missing parent directory", func(t *testing.T) {
		store := newServerFeatureStore(filepath.Join(t.TempDir(), "absent", "server-feature.json"))
		if _, ok := store.read(); ok {
			t.Fatal("manifest in a missing directory reported a successful read")
		}
	})

	t.Run("oversized file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "server-feature.json")
		writeServerFeatureManifest(t, path, strings.Repeat("x", serverFeatureMaxBytes+1))
		store := newServerFeatureStore(path)
		if _, ok := store.read(); ok {
			t.Fatal("oversized manifest reported a successful read")
		}
	})

	t.Run("path is a directory", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "server-feature.json")
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
		store := newServerFeatureStore(path)
		if _, ok := store.read(); ok {
			t.Fatal("directory manifest reported a successful read")
		}
	})

	t.Run("file is a symlink", func(t *testing.T) {
		directory := t.TempDir()
		target := filepath.Join(directory, "target.json")
		writeServerFeatureManifest(t, target, valid)
		path := filepath.Join(directory, "server-feature.json")
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
		store := newServerFeatureStore(path)
		if _, ok := store.read(); ok {
			t.Fatal("symlinked manifest reported a successful read")
		}
	})

	t.Run("parent directory is a symlink", func(t *testing.T) {
		directory := t.TempDir()
		real := filepath.Join(directory, "real")
		writeServerFeatureManifest(t, filepath.Join(real, "server-feature.json"), valid)
		link := filepath.Join(directory, "link")
		if err := os.Symlink(real, link); err != nil {
			t.Fatal(err)
		}
		store := newServerFeatureStore(filepath.Join(link, "server-feature.json"))
		if _, ok := store.read(); ok {
			t.Fatal("manifest behind a symlinked directory reported a successful read")
		}
	})

	t.Run("unreadable file", func(t *testing.T) {
		if os.Geteuid() == 0 {
			// root bypasses the permission bits, so the fixture cannot make
			// OpenFile fail; the branch stays covered on developer and CI hosts
			// that run unprivileged.
			t.Skip("running as root: 0o000 does not deny the open")
		}
		path := filepath.Join(t.TempDir(), "server-feature.json")
		writeServerFeatureManifest(t, path, valid)
		if err := os.Chmod(path, 0o000); err != nil {
			t.Fatal(err)
		}
		store := newServerFeatureStore(path)
		if _, ok := store.read(); ok {
			t.Fatal("unreadable manifest reported a successful read")
		}
	})
}

func TestLoadServerFeatureFlagsDelegatesToDefaultStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server-feature.json")
	writeServerFeatureManifest(t, path, `{"features":{"tailscale_cn":{"enabled":true}}}`)
	useTestServerFeatureStore(t, path)

	if flags := LoadServerFeatureFlags(); !flags.TailscaleCnEnabled {
		t.Fatal("LoadServerFeatureFlags did not read the injected store")
	}
	if !TailscaleCnEnabled() {
		t.Fatal("TailscaleCnEnabled did not read the injected store")
	}

	// The reader must not cache: a content change is visible on the next call.
	writeServerFeatureManifest(t, path, `{"features":{"tailscale_cn":{"enabled":false}}}`)
	if TailscaleCnEnabled() {
		t.Fatal("TailscaleCnEnabled served a cached value after the manifest changed")
	}
}

func TestLoadServerFeatureStateDelegatesToDefaultStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server-feature.json")
	writeServerFeatureManifest(t, path, `{"features":{"tailscale_cn":{"enabled":true}}}`)
	useTestServerFeatureStore(t, path)

	if got := LoadServerFeatureState(); got != FeatureStateEnabled {
		t.Fatalf("LoadServerFeatureState() = %v, want %v", got, FeatureStateEnabled)
	}
	// The tri-state and boolean readers must agree on the same manifest.
	if !TailscaleCnEnabled() {
		t.Fatal("TailscaleCnEnabled disagreed with an enabled feature state")
	}

	// The reader must not cache: a content change is visible on the next call.
	writeServerFeatureManifest(t, path, `{"features":{"tailscale_cn":{"enabled":false}}}`)
	if got := LoadServerFeatureState(); got != FeatureStateDisabled {
		t.Fatalf("LoadServerFeatureState() after change = %v, want %v", got, FeatureStateDisabled)
	}
	if TailscaleCnEnabled() {
		t.Fatal("TailscaleCnEnabled served a cached value after the manifest changed")
	}
}
