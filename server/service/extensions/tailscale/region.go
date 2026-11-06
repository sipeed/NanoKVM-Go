package tailscale

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/application"
)

const (
	defaultTailscaleRegionPath = "/etc/kvm/region"
	tailscaleRegionMaxBytes    = 64
)

type tailscaleRegionStore struct {
	path string
}

var defaultTailscaleRegionStore = newTailscaleRegionStore(defaultTailscaleRegionPath)

func newTailscaleRegionStore(path string) *tailscaleRegionStore {
	if strings.TrimSpace(path) == "" {
		path = defaultTailscaleRegionPath
	}
	return &tailscaleRegionStore{path: filepath.Clean(path)}
}

// load is deliberately fail-closed: a missing, unreadable, oversized or
// malformed file always means the global region.
func (store *tailscaleRegionStore) load() proto.TailscaleRegion {
	if store == nil || store.path == "" {
		return proto.TailscaleRegionGlobal
	}

	directoryInfo, err := os.Lstat(filepath.Dir(store.path))
	if err != nil || directoryInfo.Mode()&os.ModeSymlink != 0 || !directoryInfo.IsDir() {
		return proto.TailscaleRegionGlobal
	}
	info, err := os.Lstat(store.path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() ||
		info.Size() > tailscaleRegionMaxBytes {
		return proto.TailscaleRegionGlobal
	}

	file, err := os.OpenFile(store.path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return proto.TailscaleRegionGlobal
	}
	defer func() { _ = file.Close() }()

	payload, err := io.ReadAll(io.LimitReader(file, tailscaleRegionMaxBytes+1))
	if err != nil || len(payload) > tailscaleRegionMaxBytes {
		return proto.TailscaleRegionGlobal
	}
	region := proto.TailscaleRegion(strings.TrimSpace(string(payload)))
	if !validTailscaleRegion(region) {
		return proto.TailscaleRegionGlobal
	}
	return region
}

func validTailscaleRegion(region proto.TailscaleRegion) bool {
	return region == proto.TailscaleRegionCN || region == proto.TailscaleRegionGlobal
}

func (s *Service) regionStoreOrDefault() *tailscaleRegionStore {
	if s != nil && s.regionStore != nil {
		return s.regionStore
	}
	return defaultTailscaleRegionStore
}

func (s *Service) currentRegion() proto.TailscaleRegion {
	return s.regionStoreOrDefault().load()
}

func (s *Service) availableModes() []proto.TailscaleMode {
	return availableModesForRegion(s.currentRegion(), s.manifestEnabled())
}

// loadFeatureFlags reads the server-feature manifest through the package seam.
// Keeping the read behind a method lets the service reuse it without a second
// injection point.
func (s *Service) loadFeatureFlags() application.ServerFeatureFlags {
	return loadServerFeatureFlags()
}

// manifestEnabled reports the server-feature manifest switch as it applies to
// the managed Tailscale mode.
func (s *Service) manifestEnabled() bool {
	return s.loadFeatureFlags().TailscaleCnEnabled
}

// sipeedEnabled reports whether the read path may advertise or reflect the
// managed mode. sipeedModeAvailable applies the same conclusion to every
// endpoint/write path.
func (s *Service) sipeedEnabled() bool {
	return sipeedModeAllowed(s.currentRegion(), s.manifestEnabled())
}

// sipeedModeAvailable applies the manifest switch to managed endpoints in
// every region.
func (s *Service) sipeedModeAvailable() bool {
	return s.sipeedEnabled()
}

// PrepareRegionChange holds the Tailscale mutation guard until the application
// service has persisted the legacy region marker and restarted kvmcomm.
// Region changes no longer disconnect or delete the managed Sipeed network.
func (s *Service) PrepareRegionChange(
	_ context.Context,
	region proto.ApplicationRegion,
) (bool, func(), error) {
	if region != proto.ApplicationRegionCN && region != proto.ApplicationRegionGlobal {
		return false, nil, errors.New("invalid application region")
	}

	release, ok := tailscaleMutations.tryBegin(false)
	if !ok {
		return false, nil, errors.New("tailscale operation already in progress")
	}

	return false, release, nil
}
