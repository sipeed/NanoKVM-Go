package vm

import (
	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/hid"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

var (
	usbNetworkRuntimeActive = isVirtualNetworkMounted
	restartUsbNetwork       = restartUsbGadget
)

func (s *Service) GetUsbNetwork(c *gin.Context) {
	var rsp proto.Response

	virtualDeviceUpdateMu.Lock()
	defer virtualDeviceUpdateMu.Unlock()

	config, err := readUsbNetworkConfig(usbNetworkConfigPaths)
	if err != nil {
		log.Errorf("failed to read USB network configuration: %v", err)
		rsp.ErrRsp(c, -1, "read USB network configuration failed")
		return
	}

	rsp.OkRspWithData(c, config)
	log.Debugf("get USB network configuration: %+v", config)
}

func (s *Service) SetUsbNetwork(c *gin.Context) {
	var req proto.SetUsbNetworkReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	config, err := normalizeUsbNetworkConfig(req)
	if err != nil {
		rsp.ErrRsp(c, -2, err.Error())
		return
	}

	virtualDeviceUpdateMu.Lock()
	defer virtualDeviceUpdateMu.Unlock()
	hid.LockUsbGadgetMutation()
	defer hid.UnlockUsbGadgetMutation()

	isCurrent, err := isUsbNetworkConfigCurrent(usbNetworkConfigPaths, config)
	if err != nil {
		log.Errorf("failed to compare USB network configuration: %v", err)
		rsp.ErrRsp(c, -3, "read USB network configuration failed")
		return
	}
	if isCurrent {
		rsp.OkRspWithData(c, config)
		return
	}

	snapshot, err := snapshotUsbFiles(usbNetworkConfigPaths.dhcp, usbNetworkConfigPaths.ipv4, usbNetworkConfigPaths.forward)
	if err != nil {
		log.Errorf("failed to snapshot USB network configuration: %v", err)
		rsp.ErrRsp(c, -3, "read USB network configuration failed")
		return
	}

	if err := writeUsbNetworkConfig(usbNetworkConfigPaths, config); err != nil {
		log.Errorf("failed to write USB network configuration: %v", err)
		if rollbackErr := restoreUsbFiles(snapshot); rollbackErr != nil {
			log.Errorf("failed to roll back USB network configuration: %v", rollbackErr)
		}
		rsp.ErrRsp(c, -4, "write USB network configuration failed")
		return
	}

	if usbNetworkRuntimeActive() {
		if err := restartUsbNetwork(); err != nil {
			log.Errorf("failed to restart USB gadget after updating network configuration: %v", err)
			if rollbackErr := restoreUsbFiles(snapshot); rollbackErr != nil {
				log.Errorf("failed to roll back USB network configuration: %v", rollbackErr)
			} else if restartErr := restartUsbNetwork(); restartErr != nil {
				log.Errorf("failed to restore USB gadget after network configuration rollback: %v", restartErr)
			}
			rsp.ErrRsp(c, -5, "restart USB network failed")
			return
		}
	}

	rsp.OkRspWithData(c, config)
	log.Infof("set USB network configuration: mode=%s ipv4=%q", config.Mode, config.IPv4)
}
