package vm

import (
	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"
	"NanoKVM-Server/utils"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	SSHService = "ssh.service"
	SSHSocket  = "ssh.socket"
)

func (s *Service) GetSSHState(c *gin.Context) {
	var rsp proto.Response

	rsp.OkRspWithData(c, &proto.GetSSHStateRsp{
		Enabled: isSSHRunning(),
	})
}

func requireSSHChallenge(c *gin.Context, rsp *proto.Response, action string) bool {
	if !authservice.SSHActionRequiresChallenge() {
		return true
	}
	challenge := authservice.ReadSensitiveChallenge(c)
	if err := authservice.VerifySensitiveActionChallenge(c, challenge, action); err != nil {
		log.Errorf("verify SSH challenge for %s failed: %s", action, err)
		rsp.ErrRsp(c, -3, "sensitive action verification required")
		return false
	}
	return true
}

func (s *Service) EnableSSH(c *gin.Context) {
	var rsp proto.Response

	if !requireSSHChallenge(c, &rsp, authservice.SensitiveActionEnableSSH) {
		return
	}

	err := utils.StartService(SSHService, true)
	if err != nil {
		log.Errorf("failed to start SSH: %s", err)
		rsp.ErrRsp(c, -1, "operation failed")
		return
	}

	_ = utils.StartService(SSHSocket, true)

	rsp.OkRsp(c)
	log.Debugf("SSH enabled")
}

func (s *Service) DisableSSH(c *gin.Context) {
	var rsp proto.Response

	if !requireSSHChallenge(c, &rsp, authservice.SensitiveActionDisableSSH) {
		return
	}

	err := utils.StopService(SSHService, true)
	if err != nil {
		log.Errorf("failed to stop SSH: %s", err)
		rsp.ErrRsp(c, -1, "operation failed")
		return
	}

	_ = utils.StopService(SSHSocket, true)

	rsp.OkRsp(c)
	log.Debugf("SSH disabled")
}

func isSSHRunning() bool {
	isRunning, err := utils.IsServiceRunning(SSHService)
	if err != nil {
		log.Errorf("failed to get ssh status: %s", err)
	}
	socketRuning, err := utils.IsServiceRunning(SSHSocket)
	if err != nil {
		log.Errorf("check ssh.socket status is failed: %s", err)
	}
	return isRunning || socketRuning
}
