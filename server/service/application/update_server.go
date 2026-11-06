package application

import (
	"NanoKVM-Server/config"
	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

var (
	getConfiguredUpdateServer  = config.GetUpdateServer
	saveConfiguredUpdateServer = config.SetUpdateServer
)

func (s *Service) GetUpdateServer(c *gin.Context) {
	var rsp proto.Response
	rsp.OkRspWithData(c, &proto.GetUpdateServerRsp{Server: getConfiguredUpdateServer()})
}

func (s *Service) SetUpdateServer(c *gin.Context) {
	var req proto.SetUpdateServerReq
	var rsp proto.Response
	if err := proto.ParseFormRequest(c, &req); err != nil || req.Server == nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	server, err := config.NormalizeUpdateServer(*req.Server)
	if err != nil {
		rsp.ErrRsp(c, -1, "invalid update server")
		return
	}
	if err := saveConfiguredUpdateServer(server); err != nil {
		log.Errorf("failed to save application update server: %s", err)
		rsp.ErrRsp(c, -2, "save update server failed")
		return
	}
	rsp.OkRspWithData(c, &proto.GetUpdateServerRsp{Server: server})
}

func applicationUpdateBaseURL(server string, preview bool) (string, error) {
	baseURL, err := config.NormalizeUpdateServer(server)
	if err != nil {
		return "", err
	}
	if baseURL == "" {
		if preview {
			return PreviewURL, nil
		}
		return StableURL, nil
	}
	if preview {
		baseURL += "/preview"
	}
	return baseURL, nil
}
