package vm

import (
	"errors"
	"time"

	"NanoKVM-Server/common"
	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	commServiceStartupDelay = 3 * time.Second
)

func (s *Service) GetLowPower(c *gin.Context) {
	var rsp proto.Response

	config, err := common.GetKvmCommConfig()
	if err != nil {
		log.Errorf("failed to load vin config: %v", err)
		rsp.ErrRsp(c, -1, "failed to get status")
		return
	}

	enabled := lowPowerEnabled(config)

	data := &proto.GetLowPowerRsp{
		Enabled: enabled,
	}

	rsp.OkRspWithData(c, data)
	log.Debugf("get low power mode status: %v", enabled)
}

func lowPowerEnabled(config *common.KvmCommConfig) bool {
	if config == nil {
		return false
	}
	return config.Device.IdleTimeoutMS > 0
}

func (s *Service) SetLowPower(c *gin.Context) {
	var req proto.SetLowPowerReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	if err := common.SetKvmCommIdleTimeoutEnabled(req.Enable, true, commServiceStartupDelay); err != nil {
		log.Errorf("failed to set vin low power status: %v", err)
		code, message := lowPowerResponseFromKvmCommError(err)
		rsp.ErrRsp(c, code, message)
		return
	}

	rsp.OkRsp(c)
	log.Debugf("set low power mode: %v", req.Enable)
}

func lowPowerResponseFromKvmCommError(err error) (int, string) {
	switch {
	case errors.Is(err, common.ErrKvmCommLoadConfigFailed):
		return -2, "failed to load config"
	case errors.Is(err, common.ErrKvmCommUpdateConfigFailed):
		return -3, "failed to update config"
	case errors.Is(err, common.ErrKvmCommSaveConfigFailed):
		return -3, "failed to save config"
	case errors.Is(err, common.ErrKvmCommRestartAndRestoreFailed):
		return -5, "failed to restart kvm_vin process and restore config"
	case errors.Is(err, common.ErrKvmCommRestartAfterRestoreFailed):
		return -6, "failed to restart kvm_vin process after restoring config"
	case errors.Is(err, common.ErrKvmCommRestartFailed):
		return -4, "failed to restart kvm_vin process"
	default:
		return -3, "failed to update config"
	}
}
