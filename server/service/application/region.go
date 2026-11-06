package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/utils"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	applicationRegionPath      = "/etc/kvm/region"
	applicationRegionMaxBytes  = 64
	applicationRegionBodyLimit = 1 << 10
	applicationRegionService   = "kvmcomm.service"
)

// RegionChangeCoordinator serializes a region transition with the remote-network
// provider before the persisted region changes. The returned release function
// remains held through the shared
// feature-manifest sync, the region file write, and the kvmcomm restart.
type RegionChangeCoordinator interface {
	PrepareRegionChange(
		ctx context.Context,
		region proto.ApplicationRegion,
	) (cloudCleanupFailed bool, release func(), err error)
}

// Region changes serialize with one another. Production also injects the
// remote-network coordinator, whose mutation guard is acquired while this lock
// is held and released after the feature-manifest sync, the region write, and
// the kvmcomm restart complete.
var regionMutationMu sync.Mutex

type applicationRegionStore interface {
	load() proto.ApplicationRegion
	save(proto.ApplicationRegion) error
}

type fileApplicationRegionStore struct {
	path string
}

var defaultApplicationRegionStore applicationRegionStore = newApplicationRegionStore(applicationRegionPath)

func newApplicationRegionStore(path string) *fileApplicationRegionStore {
	if strings.TrimSpace(path) == "" {
		path = applicationRegionPath
	}
	return &fileApplicationRegionStore{path: filepath.Clean(path)}
}

// LoadRegion returns the current /etc/kvm/region marker value.
//
// It is the single read path for other packages that need the marker. The
// result inherits the store's fail-closed rules: an absent, unreadable,
// symlinked, non-regular, oversized, or invalid file resolves to
// proto.ApplicationRegionGlobal, the same safe default the region API uses.
func LoadRegion() proto.ApplicationRegion { return defaultApplicationRegionStore.load() }

// load treats an absent, unreadable, oversized, non-regular, or invalid file
// as the safe default. This also keeps startup behavior deterministic when an
// older image has no region marker yet.
func (store *fileApplicationRegionStore) load() proto.ApplicationRegion {
	if store == nil || store.path == "" {
		return proto.ApplicationRegionGlobal
	}

	directoryInfo, err := os.Lstat(filepath.Dir(store.path))
	if err != nil || directoryInfo.Mode()&os.ModeSymlink != 0 || !directoryInfo.IsDir() {
		return proto.ApplicationRegionGlobal
	}
	info, err := os.Lstat(store.path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() ||
		info.Size() > applicationRegionMaxBytes {
		return proto.ApplicationRegionGlobal
	}

	file, err := os.OpenFile(store.path, os.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return proto.ApplicationRegionGlobal
	}
	defer func() { _ = file.Close() }()

	payload, err := io.ReadAll(io.LimitReader(file, applicationRegionMaxBytes+1))
	if err != nil || len(payload) > applicationRegionMaxBytes {
		return proto.ApplicationRegionGlobal
	}
	region := proto.ApplicationRegion(strings.TrimSpace(string(payload)))
	if !validApplicationRegion(region) {
		return proto.ApplicationRegionGlobal
	}
	return region
}

// save publishes a region through the shared atomic file helper. The helper
// keeps the same-directory temporary file and directory sync so the rename is
// durable across a power loss on filesystems used by the device. The region
// path therefore still yields a ".region-*.tmp" temporary file.
func (store *fileApplicationRegionStore) save(region proto.ApplicationRegion) error {
	if store == nil || store.path == "" {
		return errors.New("application region path is empty")
	}
	if !validApplicationRegion(region) {
		return errors.New("invalid application region")
	}
	return publishAtomicFile(store.path, []byte(string(region)+"\n"), 0o644)
}

func validApplicationRegion(region proto.ApplicationRegion) bool {
	return region == proto.ApplicationRegionCN || region == proto.ApplicationRegionGlobal
}

func (s *Service) applicationRegionStore() applicationRegionStore {
	if s != nil && s.regionStore != nil {
		return s.regionStore
	}
	return defaultApplicationRegionStore
}

func (s *Service) restartKVMCommService() error {
	if s != nil && s.restartKVMComm != nil {
		return s.restartKVMComm()
	}
	return utils.RestartService(applicationRegionService)
}

func (s *Service) syncRegionFeatureManifest(ctx context.Context, region proto.ApplicationRegion) error {
	if s != nil && s.syncFeatureManifest != nil {
		return s.syncFeatureManifest(ctx, region)
	}
	return SyncRegionFeatureManifest(ctx, region)
}

func (s *Service) GetRegion(c *gin.Context) {
	var rsp proto.Response
	rsp.OkRspWithData(c, &proto.GetApplicationRegionRsp{Region: s.applicationRegionStore().load()})
}

func (s *Service) SetRegion(c *gin.Context) {
	var rsp proto.Response
	request, err := parseApplicationRegionRequest(c)
	if err != nil {
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}

	regionMutationMu.Lock()
	defer regionMutationMu.Unlock()

	store := s.applicationRegionStore()
	previous := store.load()
	if request.Region == previous {
		rsp.OkRspWithData(c, &proto.GetApplicationRegionRsp{Region: previous})
		return
	}

	cloudCleanupFailed := false
	if s != nil && s.regionCoordinator != nil {
		var release func()
		cloudCleanupFailed, release, err = s.regionCoordinator.PrepareRegionChange(
			c.Request.Context(),
			request.Region,
		)
		if err != nil {
			log.Errorf("prepare application region change failed: %s", err)
			rsp.ErrRspWithData(c, -1, "set region failed", &proto.GetApplicationRegionRsp{
				Region:             previous,
				CloudCleanupFailed: cloudCleanupFailed,
			})
			return
		}
		if release != nil {
			defer release()
		}
	}

	// Both regions share the same manifest. A sync failure leaves the previously
	// published manifest in use and does not prevent changing the legacy marker.
	if err := s.syncRegionFeatureManifest(c.Request.Context(), request.Region); err != nil {
		log.Warnf("sync feature manifest before region change failed: %s", err)
	}

	if err := store.save(request.Region); err != nil {
		log.Errorf("save application region failed: %s", err)
		// Re-publish the old value if a failed write left any partial state.
		if restoreErr := store.save(previous); restoreErr != nil {
			log.Errorf("restore application region after save failure failed: %s", restoreErr)
		}
		rsp.ErrRspWithData(c, -1, "set region failed", &proto.GetApplicationRegionRsp{
			Region:             store.load(),
			CloudCleanupFailed: cloudCleanupFailed,
		})
		return
	}

	if err := s.restartKVMCommService(); err != nil {
		log.Errorf("restart %s after application region change failed: %s", applicationRegionService, err)
		restoreErr := store.save(previous)
		if restoreErr != nil {
			log.Errorf("restore application region after restart failure failed: %s", restoreErr)
		} else if rollbackRestartErr := s.restartKVMCommService(); rollbackRestartErr != nil {
			log.Errorf("restart %s after application region rollback failed: %s", applicationRegionService, rollbackRestartErr)
		}
		rsp.ErrRspWithData(c, -1, "set region failed", &proto.GetApplicationRegionRsp{
			Region:             store.load(),
			CloudCleanupFailed: cloudCleanupFailed,
		})
		return
	}

	rsp.OkRspWithData(c, &proto.GetApplicationRegionRsp{
		Region:             request.Region,
		CloudCleanupFailed: cloudCleanupFailed,
	})
}

func parseApplicationRegionRequest(c *gin.Context) (*proto.SetApplicationRegionReq, error) {
	if c == nil || c.Request == nil || c.Request.Body == nil {
		return nil, errors.New("missing request body")
	}
	payload, err := io.ReadAll(io.LimitReader(c.Request.Body, applicationRegionBodyLimit+1))
	if err != nil || len(payload) > applicationRegionBodyLimit {
		return nil, errors.New("invalid request body")
	}

	var request proto.SetApplicationRegionReq
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return nil, errors.New("invalid request body")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, errors.New("invalid request body")
	}
	if !validApplicationRegion(request.Region) {
		return nil, errors.New("invalid application region")
	}
	return &request, nil
}
