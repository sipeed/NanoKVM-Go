package ai

import (
	"fmt"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/utils"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	memoryFabricUnitName    = "memfab.service"
	memoryFabricOCRUnitName = "nano_ocr.service"
)

var newMemoryFabricManager = func() memoryFabricManager {
	return memoryFabricManager{
		unitName:     memoryFabricUnitName,
		ocrUnitName:  memoryFabricOCRUnitName,
		isRunning:    utils.IsServiceRunning,
		startService: utils.StartService,
		stopService:  utils.StopService,
	}
}

type memoryFabricManager struct {
	unitName     string
	ocrUnitName  string
	isRunning    func(string) (bool, error)
	startService func(string, bool) error
	stopService  func(string, bool) error
}

func (m memoryFabricManager) status() (bool, error) {
	running, err := m.isRunning(m.unitName)
	if err != nil {
		return false, fmt.Errorf("get %s status: %w", m.unitName, err)
	}

	ocrRunning, err := m.isRunning(m.ocrUnitName)
	if err != nil {
		return false, fmt.Errorf("get %s status: %w", m.ocrUnitName, err)
	}

	return running && ocrRunning, nil
}

func (m memoryFabricManager) setEnabled(enabled bool) error {
	if enabled {
		if err := m.startService(m.unitName, true); err != nil {
			return fmt.Errorf("start %s: %w", m.unitName, err)
		}

		if err := m.startService(m.ocrUnitName, true); err != nil {
			if rollbackErr := m.stopService(m.unitName, true); rollbackErr != nil {
				log.Warnf("failed to rollback %s after %s start failure: %v", m.unitName, m.ocrUnitName, rollbackErr)
			}
			return fmt.Errorf("start %s: %w", m.ocrUnitName, err)
		}

		return nil
	}

	if err := m.stopService(m.unitName, true); err != nil {
		return fmt.Errorf("stop %s: %w", m.unitName, err)
	}

	if err := m.stopService(m.ocrUnitName, true); err != nil {
		if rollbackErr := m.startService(m.unitName, true); rollbackErr != nil {
			log.Warnf("failed to rollback %s after %s stop failure: %v", m.unitName, m.ocrUnitName, rollbackErr)
		}
		return fmt.Errorf("stop %s: %w", m.ocrUnitName, err)
	}

	return nil
}

func memoryFabricServiceStateHook(enabled bool, source string) error {
	return newMemoryFabricManager().setEnabled(enabled)
}

func (s *Service) GetMemoryFabric(c *gin.Context) {
	var rsp proto.Response

	state, err := DefaultServiceStateManager().Get(aiServiceMemoryFabric)
	if err != nil {
		log.Errorf("failed to get Memory Fabric service state: %v", err)
		rsp.ErrRsp(c, -1, "get memory fabric status failed")
		return
	}

	running, err := newMemoryFabricManager().status()
	if err != nil {
		log.Errorf("failed to get Memory Fabric running status: %v", err)
		rsp.ErrRsp(c, -1, "get memory fabric status failed")
		return
	}
	logMemoryFabricMismatch(state.Enabled, running)

	rsp.OkRspWithData(c, &proto.GetMemoryFabricRsp{
		Enabled: state.Enabled,
		Running: running,
	})
}

func (s *Service) SetMemoryFabric(c *gin.Context) {
	var req proto.SetMemoryFabricReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	snapshot, err := DefaultServiceStateManager().Set(aiServiceMemoryFabric, *req.Enabled, "web")
	if err != nil {
		log.Errorf("failed to set Memory Fabric enabled=%v: %v", *req.Enabled, err)
		rsp.ErrRsp(c, -2, "operation failed")
		return
	}
	state := snapshot.Services[aiServiceMemoryFabric]

	running, err := newMemoryFabricManager().status()
	if err != nil {
		log.Errorf("failed to get Memory Fabric running status after set: %v", err)
		rsp.ErrRsp(c, -2, "operation failed")
		return
	}
	logMemoryFabricMismatch(state.Enabled, running)

	rsp.OkRspWithData(c, &proto.GetMemoryFabricRsp{
		Enabled: state.Enabled,
		Running: running,
	})
}

func logMemoryFabricMismatch(enabled bool, running bool) {
	if enabled && !running {
		log.Warn("Memory Fabric expected enabled but memfab.service or nano_ocr.service is not running")
	}
}
