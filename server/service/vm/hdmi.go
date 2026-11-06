package vm

import (
	"NanoKVM-Server/common"
	"NanoKVM-Server/config"
	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func (s *Service) GetHdmiCapture(c *gin.Context) {
	var rsp proto.Response

	enabled := config.GetHDMICaptureEnabled()

	rsp.OkRspWithData(c, &proto.GetHdmiCaptureRsp{
		Enabled: enabled,
	})
	log.Debugf("get HDMI capture status: %t", enabled)
}

func (s *Service) SetHdmiCapture(c *gin.Context) {
	var req proto.SetHdmiCaptureReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	oldEnabled := config.GetHDMICaptureEnabled()

	if result := common.GetKvmVision().SetHDMI(req.Enabled); result < 0 {
		log.Errorf("failed to set HDMI capture status to %t: %d", req.Enabled, result)
		rsp.ErrRsp(c, -2, "failed to set HDMI capture status")
		return
	}

	if err := config.SetHDMICaptureEnabled(req.Enabled); err != nil {
		log.Errorf("failed to save HDMI capture status %t: %v", req.Enabled, err)

		if result := common.GetKvmVision().SetHDMI(oldEnabled); result < 0 {
			log.Errorf("failed to rollback HDMI capture status to %t: %d", oldEnabled, result)
		}

		rsp.ErrRsp(c, -3, "failed to save HDMI capture status")
		return
	}

	rsp.OkRsp(c)
	log.Debugf("set HDMI capture status: %t", req.Enabled)
}
