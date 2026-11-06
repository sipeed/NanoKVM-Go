package stream

import (
	"NanoKVM-Server/common"
	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

var getKvmCommConfig = common.GetKvmCommConfig

// GetEncoder returns the normalized persisted settings used by kvm_vin,
// rather than querying the encoder hardware at runtime.
func (s *Service) GetEncoder(c *gin.Context) {
	var rsp proto.Response

	config, err := func() (*common.KvmCommConfig, error) {
		encoderSettingsMu.Lock()
		defer encoderSettingsMu.Unlock()
		return getKvmCommConfig()
	}()
	if err != nil {
		log.Errorf("failed to load encoder settings: %v", err)
		rsp.ErrRsp(c, -1, "failed to get encoder settings")
		return
	}

	rsp.OkRspWithData(c, &proto.GetEncoderRsp{
		RateControl: config.Encoder.RateControl,
		BitRate:     config.Encoder.BitRate,
		GOP:         config.Encoder.GOP,
		FPS:         config.Encoder.FPS,
	})
}
