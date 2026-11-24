package vm

import (
	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/vm/jiggler"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func (s *Service) GetMouseJiggler(c *gin.Context) {
	var rsp proto.Response

	rsp.OkRspWithData(c, jiggler.GetJiggler().Status())
}

func (s *Service) SetMouseJiggler(c *gin.Context) {
	var req proto.SetMouseJigglerReq
	var rsp proto.Response

	err := proto.ParseFormRequest(c, &req)
	if err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	state, err := jiggler.GetJiggler().Configure(req.Enabled, req.Mode, req.MaxHours)
	if err != nil {
		log.Warnf("set mouse jiggler: %v", err)
		rsp.ErrRsp(c, -2, "operation failed")
		return
	}
	rsp.OkRspWithData(c, state)
}
