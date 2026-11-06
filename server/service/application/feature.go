package application

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	// serverFeaturePath is the device-level feature manifest. NanoKVM-Server
	// refreshes it from the shared CDN source when remote-network settings opens
	// and during legacy region changes; deployments may also preinstall a copy.
	serverFeaturePath = "/etc/kvm/server-feature.json"
	// serverFeatureMaxBytes bounds the manifest so a corrupted or hostile file
	// cannot make the per-request reader allocate without limit.
	serverFeatureMaxBytes = 256 << 10
)

// ServerFeatureFlags is the projection of the device feature manifest that the
// server consumes. Only explicitly true flags are exposed.
type ServerFeatureFlags struct {
	TailscaleCnEnabled bool
}

// FeatureState is the tri-state view of the device feature manifest for
// callers that must distinguish "no manifest yet" from "a manifest that
// forbids the feature". The root-local gate uses it to decide whether a fetch
// is allowed to repair the state; the boolean projection cannot express that.
type FeatureState uint8

const (
	// FeatureStateMissing means no acceptable manifest could be read: the file
	// or its parent is absent, unreadable, symlinked, non-regular, or
	// oversized. This is the only state a manifest sync may repair.
	FeatureStateMissing FeatureState = iota
	// FeatureStateDisabled means a manifest was read but does not enable the
	// feature, either because it explicitly sets enabled:false or because its
	// payload is invalid. An invalid payload is deliberately not Missing:
	// corruption is a positive, externally authored policy state, and
	// treating it as absent would turn a damaged file into a fetch trigger.
	FeatureStateDisabled
	// FeatureStateEnabled means a valid manifest explicitly sets enabled:true.
	FeatureStateEnabled
)

// serverFeatureStore reads the manifest from one fixed path. It is a small
// struct so tests can point it at a t.TempDir() fixture.
type serverFeatureStore struct {
	path string
}

var defaultServerFeatureStore = newServerFeatureStore(serverFeaturePath)

func newServerFeatureStore(path string) *serverFeatureStore {
	if strings.TrimSpace(path) == "" {
		path = serverFeaturePath
	}
	return &serverFeatureStore{path: filepath.Clean(path)}
}

// LoadServerFeatureFlags returns the current /etc/kvm/server-feature.json flags.
//
// It is the single read path for other packages. The result inherits the
// store's fail-closed rules: an absent, unreadable, symlinked, non-regular,
// oversized, or invalid file resolves to the zero value (everything disabled),
// the same safe default the region marker uses. The file is re-read on every
// call and never cached, so a permission or content change is reflected by the
// next request instead of a stale "last known good" value.
func LoadServerFeatureFlags() ServerFeatureFlags {
	return defaultServerFeatureStore.load()
}

// LoadServerFeatureState returns the tri-state view of the current
// /etc/kvm/server-feature.json manifest.
//
// It inherits the store's fail-closed rules from LoadServerFeatureFlags: an
// absent, unreadable, symlinked, non-regular, oversized, or invalid file never
// enables the feature. Unlike the boolean projection it distinguishes a
// missing file from a manifest that forbids the feature, so a caller can
// decide whether a sync may repair the state. The file is re-read on every
// call and never cached.
func LoadServerFeatureState() FeatureState {
	return defaultServerFeatureStore.state()
}

// TailscaleCnEnabled reports whether the manifest explicitly enables the
// managed Tailscale mode in any region. The original flag name is retained for
// compatibility with published manifests and existing devices.
func TailscaleCnEnabled() bool {
	return LoadServerFeatureFlags().TailscaleCnEnabled
}

// state classifies the manifest for callers that must tell a missing file
// apart from a manifest that forbids the feature. Both steps delegate to the
// same read and decode helpers as load(), so the tri-state and boolean views
// can never drift apart.
func (store *serverFeatureStore) state() FeatureState {
	payload, ok := store.read()
	if !ok {
		return FeatureStateMissing
	}
	flags, ok := classifyServerFeatureManifest(payload)
	if !ok || !flags.TailscaleCnEnabled {
		// An invalid payload is a positive disabled answer, not a missing one:
		// the bounded CDN publisher validates before publishing, so a corrupt
		// file implies filesystem damage or tampering and must not become a
		// self-healing fetch trigger.
		return FeatureStateDisabled
	}
	return FeatureStateEnabled
}

// load treats an absent, unreadable, oversized, non-regular, or invalid file
// as disabled. A failed read only ever resolves to false and never logs or
// returns file contents. It derives from state() so the boolean and tri-state
// readers cannot disagree about whether the feature is on.
func (store *serverFeatureStore) load() ServerFeatureFlags {
	if store.state() != FeatureStateEnabled {
		return ServerFeatureFlags{}
	}
	return ServerFeatureFlags{TailscaleCnEnabled: true}
}

// read returns the bounded manifest payload. Every structural failure (missing
// file or directory, symlink at either level, non-regular file, oversized
// file, read error) reports false; callers never see contents of a rejected
// file. It never logs.
func (store *serverFeatureStore) read() ([]byte, bool) {
	if store == nil || store.path == "" {
		return nil, false
	}

	directoryInfo, err := os.Lstat(filepath.Dir(store.path))
	if err != nil || directoryInfo.Mode()&os.ModeSymlink != 0 || !directoryInfo.IsDir() {
		return nil, false
	}
	info, err := os.Lstat(store.path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() ||
		info.Size() > serverFeatureMaxBytes {
		return nil, false
	}

	file, err := os.OpenFile(store.path, os.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, false
	}
	defer func() { _ = file.Close() }()

	payload, err := io.ReadAll(io.LimitReader(file, serverFeatureMaxBytes+1))
	if err != nil || len(payload) > serverFeatureMaxBytes {
		return nil, false
	}
	return payload, true
}

// classifyServerFeatureManifest decodes one manifest payload and reports the
// recognized flags plus whether the payload is a valid manifest. Keys are
// matched exactly, like the device-side kvm_ui parser: because encoding/json
// matches struct field names case-insensitively, an object-based decode would
// accept FEATURES/TAILSCALE_CN/ENABLED and enable only one of the two
// consumers. Any syntax error, non-object level, or missing, misnamed, or
// non-boolean enabled key resolves to false. An explicit false is valid and
// yields the zero flags. Unrelated fields such as version and updated_at stay
// ignored because the traversal only reads the recognized path.
func classifyServerFeatureManifest(payload []byte) (ServerFeatureFlags, bool) {
	root, ok := exactObject(payload)
	if !ok {
		return ServerFeatureFlags{}, false
	}
	features, ok := exactObjectField(root, "features")
	if !ok {
		return ServerFeatureFlags{}, false
	}
	tailscale, ok := exactObjectField(features, "tailscale_cn")
	if !ok {
		return ServerFeatureFlags{}, false
	}
	raw, ok := tailscale["enabled"]
	if !ok {
		return ServerFeatureFlags{}, false
	}
	// A pointer distinguishes JSON null (nil, and therefore not a boolean)
	// from an explicit false, which is a valid manifest.
	var enabled *bool
	if err := json.Unmarshal(raw, &enabled); err != nil || enabled == nil {
		return ServerFeatureFlags{}, false
	}
	return ServerFeatureFlags{TailscaleCnEnabled: *enabled}, true
}

// exactObject decodes one JSON object level. A non-object level, including a
// null that decodes without error into a nil map, reports false.
func exactObject(payload []byte) (map[string]json.RawMessage, bool) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(payload, &object); err != nil || object == nil {
		return nil, false
	}
	return object, true
}

// exactObjectField returns one nested object by exact key.
func exactObjectField(object map[string]json.RawMessage, key string) (map[string]json.RawMessage, bool) {
	raw, ok := object[key]
	if !ok {
		return nil, false
	}
	return exactObject(raw)
}
