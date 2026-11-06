package application

import (
	"context"

	"NanoKVM-Server/proto"
)

type Service struct {
	regionStore         applicationRegionStore
	restartKVMComm      func() error
	regionCoordinator   RegionChangeCoordinator
	syncFeatureManifest func(ctx context.Context, region proto.ApplicationRegion) error
}

const (
	// StableURL and PreviewURL are the v2 update channels. Every bundle under
	// them is signed by a key that only v2-aware releases pin, so releases that
	// predate the move keep polling the frozen legacy channel below and reach
	// v2 by installing the root-signed bridge release published there.
	StableURL  = "https://cdn.sipeed.com/nanokvm-go/v2"
	PreviewURL = "https://cdn.sipeed.com/nanokvm-go/v2/preview"
	// LegacyBaseURL is the pre-v2 channel root. Its update manifests are frozen
	// on the bridge release; it still hosts the server feature manifest, which
	// devices on both sides of the move share.
	LegacyBaseURL = "https://cdn.sipeed.com/nanokvm-go"

	AppDir                    = "/kvmapp"
	TempDir                   = "/root/.kvmcache"
	latestFileName            = "latest.json"
	packageArch               = "armhf"
	updateKeyRingPath         = "/usr/share/nanokvm/update-keys.json"
	applicationUpdateLockPath = "/run/lock/nanokvm-application-update.lock"
	deviceMaintenanceLockPath = "/run/lock/nanokvm-device-maintenance.lock"
	mainAppName               = "nanokvm-go"
	kvmCommAppName            = "kvmcomm"
)

var appNames = []string{kvmCommAppName, mainAppName}

func NewService(coordinators ...RegionChangeCoordinator) *Service {
	service := &Service{regionStore: defaultApplicationRegionStore}
	if len(coordinators) > 0 {
		service.regionCoordinator = coordinators[0]
	}
	return service
}
