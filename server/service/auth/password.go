package auth

import (
	"errors"
	"os"
	"strings"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

func (s *Service) ChangePassword(c *gin.Context) {
	var req proto.ChangePasswordReq
	var rsp proto.Response

	if err := bindSensitiveRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}

	if !s.verifyPasswordChangeChallenge(c, req, &rsp) {
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		rsp.ErrRsp(c, -3, "failed to hash password")
		return
	}

	if err = SetAccount(req.Username, string(hashedPassword)); err != nil {
		rsp.ErrRsp(c, -4, "failed to save password")
		return
	}

	rsp.OkRsp(c)
	log.Debugf("change password success, username: %s", req.Username)
}

func (s *Service) verifyPasswordChangeChallenge(c *gin.Context, req proto.ChangePasswordReq, rsp *proto.Response) bool {
	enabled, err := isTOTPEnabled()
	if err != nil {
		log.Errorf("failed to read TOTP status for password change: %s", err)
		rsp.ErrRsp(c, -3, "get two factor status failed")
		return false
	}
	if !enabled {
		return true
	}

	account, err := GetAccount()
	if err != nil || account == nil {
		log.Errorf("failed to get account for password change: %v", err)
		rsp.ErrRsp(c, -3, "get account failed")
		return false
	}
	username := strings.TrimSpace(account.Username)
	if strings.TrimSpace(req.Challenge) == "" {
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return false
	}
	jwtUsername := currentJWTUsername(c)
	if jwtUsername != "" && jwtUsername != username {
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return false
	}

	sessionKey := sensitiveOperationSessionKey(c)
	if !consumeSensitiveActionChallenge(req.Challenge, username, sensitiveActionChangePassword, sessionKey, c.ClientIP(), c.Request.UserAgent()) {
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return false
	}

	return true
}

func (s *Service) IsPasswordUpdated(c *gin.Context) {
	var rsp proto.Response

	if _, err := os.Stat(AccountFile); err != nil {
		rsp.OkRspWithData(c, &proto.IsPasswordUpdatedRsp{
			IsUpdated: false,
		})
		return
	}

	account, err := GetAccount()
	if err != nil || account == nil {
		rsp.ErrRsp(c, -1, "failed to get password")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(account.Password), []byte("admin"))

	rsp.OkRspWithData(c, &proto.IsPasswordUpdatedRsp{
		IsUpdated: errors.Is(err, bcrypt.ErrMismatchedHashAndPassword),
	})
}
