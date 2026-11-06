package auth

import (
	"net/http"
	"strings"
	"time"

	"NanoKVM-Server/config"
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

var (
	getConfig              = config.GetInstance
	generateJWT            = middleware.GenerateJWT
	invalidParameterDelay  = 3 * time.Second
	invalidCredentialDelay = 2 * time.Second
	tokenFailureDelay      = 1 * time.Second
)

const (
	authCookieName     = "nano-kvm-token"
	invalidAuthMessage = "invalid username, password, 2fa code or recovery code"
)

func (s *Service) Login(c *gin.Context) {
	var req proto.LoginReq
	var rsp proto.Response

	// authentication disabled
	conf := getConfig()
	if conf.Authentication == "disable" {
		rsp.OkRspWithData(c, &proto.LoginRsp{
			Token: "disabled",
		})
		return
	}

	if err := bindSensitiveRequest(c, &req); err != nil || proto.ValidateLoginReqShape(req) != nil {
		time.Sleep(invalidParameterDelay)
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}

	if req.Challenge != "" {
		s.loginWithTwoFactorChallenge(c, req, conf, &rsp)
		return
	}
	if req.Code != "" {
		s.loginWithDirectTwoFactor(c, req, conf, &rsp)
		return
	}

	s.loginWithPassword(c, req, conf, &rsp)
}

func (s *Service) loginWithDirectTwoFactor(c *gin.Context, req proto.LoginReq, conf *config.Config, rsp *proto.Response) {
	ip := c.ClientIP()
	enabled, err := directTwoFactorLoginEnabled()
	if err != nil {
		log.Errorf("failed to read direct two factor login policy: %v", err)
		recordLoginFailure(unknownAuthRateLimitValue, ip)
		auditAuthEvent("auth_login_failed", authAuditFields(c, unknownAuthRateLimitValue, "login_2fa_direct", "policy_unavailable"))
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return
	}
	if !enabled {
		recordLoginFailure(unknownAuthRateLimitValue, ip)
		auditAuthEvent("auth_login_failed", authAuditFields(c, unknownAuthRateLimitValue, "login_2fa_direct", "not_enabled"))
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return
	}

	account, err := GetAccount()
	if err != nil || account == nil {
		log.Errorf("failed to get account for direct two factor login: %v", err)
		time.Sleep(tokenFailureDelay)
		rsp.ErrRsp(c, -3, "get account failed")
		return
	}
	username := strings.TrimSpace(account.Username)
	if until, limited := loginRateLimitedUntil(username, ip); limited {
		auditAuthEvent("auth_login_rate_limited", authAuditFields(c, username, "login_2fa_direct", "rate_limited"))
		writeAuthRateLimitRsp(c, rsp, until)
		return
	}

	secondFactor := verifySecondFactor(req.Code)
	if !secondFactor.OK {
		recordLoginFailure(username, ip)
		auditAuthEvent("auth_login_failed", authAuditFields(c, username, "login_2fa_direct", "second_factor_invalid"))
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return
	}

	if s.issueLoginTokenWithOptions(c, username, conf, rsp, secondFactor.RecoveryCodeUsed) {
		if secondFactor.RecoveryCodeUsed {
			auditRecoveryCodeUsed(c, username, "login_2fa_direct", secondFactor.RecoveryCodeID)
		}
		clearLoginFailures(username, ip)
		log.Debugf("direct two factor login success, username: %s", username)
	}
}

func (s *Service) loginWithPassword(c *gin.Context, req proto.LoginReq, conf *config.Config, rsp *proto.Response) {
	ip := c.ClientIP()
	if until, limited := loginRateLimitedUntil(req.Username, ip); limited {
		auditAuthEvent("auth_login_rate_limited", authAuditFields(c, req.Username, "login", "rate_limited"))
		writeAuthRateLimitRsp(c, rsp, until)
		return
	}

	if ok := CompareAccount(req.Username, req.Password); !ok {
		recordLoginFailure(req.Username, ip)
		auditAuthEvent("auth_login_failed", authAuditFields(c, req.Username, "login", "credential_invalid"))
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return
	}

	enabled, err := isTOTPEnabled()
	if err != nil {
		log.Errorf("failed to read TOTP state during login: %s", err)
		time.Sleep(tokenFailureDelay)
		rsp.ErrRsp(c, -3, "read two factor state failed")
		return
	}
	if enabled {
		challenge, err := createLoginChallenge(req.Username, c.ClientIP(), c.Request.UserAgent())
		if err != nil {
			log.Errorf("failed to create login challenge: %s", err)
			time.Sleep(tokenFailureDelay)
			rsp.ErrRsp(c, -3, "generate challenge failed")
			return
		}

		rsp.OkRspWithData(c, &proto.LoginRsp{
			Requires2FA: true,
			Challenge:   challenge,
		})
		return
	}

	if s.issueLoginToken(c, req.Username, conf, rsp) {
		clearLoginFailures(req.Username, ip)
		log.Debugf("login success, username: %s", req.Username)
	}
}

func (s *Service) loginWithTwoFactorChallenge(c *gin.Context, req proto.LoginReq, conf *config.Config, rsp *proto.Response) {
	ip := c.ClientIP()
	username, ok := validateLoginChallenge(req.Challenge, c.ClientIP(), c.Request.UserAgent())
	if !ok {
		recordLoginFailure(unknownAuthRateLimitValue, ip)
		auditAuthEvent("auth_login_failed", authAuditFields(c, unknownAuthRateLimitValue, "login_2fa", "challenge_invalid"))
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return
	}
	if until, limited := loginRateLimitedUntil(username, ip); limited {
		auditAuthEvent("auth_login_rate_limited", authAuditFields(c, username, "login_2fa", "rate_limited"))
		writeAuthRateLimitRsp(c, rsp, until)
		return
	}
	secondFactor := verifySecondFactor(req.Code)
	if !secondFactor.OK {
		recordLoginFailure(username, ip)
		auditAuthEvent("auth_login_failed", authAuditFields(c, username, "login_2fa", "second_factor_invalid"))
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return
	}
	if consumedUsername, ok := consumeLoginChallenge(req.Challenge, c.ClientIP(), c.Request.UserAgent()); !ok || consumedUsername != username {
		recordLoginFailure(username, ip)
		auditAuthEvent("auth_login_failed", authAuditFields(c, username, "login_2fa", "challenge_invalid"))
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return
	}

	if s.issueLoginTokenWithOptions(c, username, conf, rsp, secondFactor.RecoveryCodeUsed) {
		if secondFactor.RecoveryCodeUsed {
			auditRecoveryCodeUsed(c, username, "login_2fa", secondFactor.RecoveryCodeID)
		}
		clearLoginFailures(username, ip)
		log.Debugf("login success, username: %s", username)
	}
}

func (s *Service) issueLoginToken(c *gin.Context, username string, conf *config.Config, rsp *proto.Response) bool {
	return s.issueLoginTokenWithOptions(c, username, conf, rsp, false)
}

func (s *Service) issueLoginTokenWithOptions(c *gin.Context, username string, conf *config.Config, rsp *proto.Response, recoveryCodeUsed bool) bool {
	token, err := generateJWT(username)
	if err != nil {
		time.Sleep(tokenFailureDelay)
		rsp.ErrRsp(c, -3, "generate token failed")
		return false
	}
	setAuthCookie(c, token, config.JWTRefreshTokenMaxAge(conf.JWT.RefreshTokenDuration))

	rsp.OkRspWithData(c, &proto.LoginRsp{
		Token:            token,
		RecoveryCodeUsed: recoveryCodeUsed,
	})
	return true
}

func (s *Service) Logout(c *gin.Context) {
	conf := config.GetInstance()

	if conf.JWT.RevokeTokensOnLogout {
		if err := config.RegenerateSecretKey(); err != nil {
			log.Errorf("failed to revoke JWT sessions during logout: %v", err)
			clearAuthCookie(c)
			var rsp proto.Response
			rsp.ErrRsp(c, -1, "failed to revoke sessions")
			return
		}
	}
	clearAuthCookie(c)

	var rsp proto.Response
	rsp.OkRsp(c)
}

func setAuthCookie(c *gin.Context, token string, maxAge int) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(authCookieName, token, maxAge, "/", "", isSecureRequest(c), true)
}

func clearAuthCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(authCookieName, "", -1, "/", "", isSecureRequest(c), true)
}

func isSecureRequest(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	if forwardedProto := c.GetHeader("X-Forwarded-Proto"); forwardedProto != "" {
		return strings.EqualFold(strings.TrimSpace(strings.Split(forwardedProto, ",")[0]), "https")
	}
	return false
}

func (s *Service) GetAccount(c *gin.Context) {
	var rsp proto.Response

	account, err := GetAccount()
	if err != nil {
		rsp.ErrRsp(c, -1, "get account failed")
		return
	}

	rsp.OkRspWithData(c, &proto.GetAccountRsp{
		Username: account.Username,
	})
	log.Debugf("get account successful")
}
