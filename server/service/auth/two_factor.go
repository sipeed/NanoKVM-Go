package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	log "github.com/sirupsen/logrus"
)

var sensitiveRequestValidator = validator.New()

func bindSensitiveRequest(c *gin.Context, req interface{}) error {
	if err := c.ShouldBind(req); err != nil {
		log.Errorf("parse request failed, err: %s", err)
		return err
	}
	if err := sensitiveRequestValidator.Struct(req); err != nil {
		log.Errorf("validate request failed, err: %s", err)
		return err
	}
	return nil
}

func sensitiveOperationSessionKey(c *gin.Context) string {
	token := strings.TrimSpace(authTokenFromRequest(c))
	if token != "" {
		return hashAuthRateLimitKey("token:" + token)
	}

	client := strings.TrimSpace(c.ClientIP()) + "\n" + strings.TrimSpace(c.Request.UserAgent())
	return hashAuthRateLimitKey("client:" + client)
}

func authTokenFromRequest(c *gin.Context) string {
	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}
	if cookie, err := c.Cookie(authCookieName); err == nil {
		return strings.TrimSpace(cookie)
	}
	return ""
}

func hashAuthRateLimitKey(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func (s *Service) TwoFactorStatus(c *gin.Context) {
	var rsp proto.Response

	enabled, err := isTOTPEnabled()
	if err != nil {
		log.Errorf("failed to read TOTP status: %s", err)
		rsp.ErrRsp(c, -3, "get two factor status failed")
		return
	}

	recoveryEnabled, recoveryRemaining, err := recoveryCodesStatus()
	if err != nil {
		log.Errorf("failed to read recovery code status: %s", err)
		rsp.ErrRsp(c, -3, "get two factor status failed")
		return
	}

	rsp.OkRspWithData(c, &proto.TwoFactorStatusRsp{
		Enabled:           enabled,
		RecoveryEnabled:   recoveryEnabled,
		RecoveryRemaining: recoveryRemaining,
	})
}

func (s *Service) TwoFactorSetup(c *gin.Context) {
	var req proto.TwoFactorSetupReq
	var rsp proto.Response

	if err := bindSensitiveRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}

	account, err := GetAccount()
	if err != nil || account == nil {
		log.Errorf("failed to get account for TOTP setup: %v", err)
		rsp.ErrRsp(c, -3, "get account failed")
		return
	}
	enabled, err := isTOTPEnabled()
	if err != nil {
		log.Errorf("failed to read TOTP status for setup: %s", err)
		rsp.ErrRsp(c, -3, "get two factor status failed")
		return
	}
	if enabled {
		cancelPendingTOTPSetup(account.Username)
		auditAuthEvent("auth_2fa_operation_failed", authAuditFields(c, account.Username, "setup", "already_enabled"))
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return
	}
	sessionKey := sensitiveOperationSessionKey(c)
	if until, limited := sensitiveOperationRateLimitedUntil(account.Username, sessionKey); limited {
		auditAuthEvent("auth_2fa_operation_failed", authAuditFields(c, account.Username, "setup", "rate_limited"))
		writeAuthRateLimitRsp(c, &rsp, until)
		return
	}
	if !CompareAccount(account.Username, req.Password) {
		recordSensitiveOperationFailure(account.Username, sessionKey)
		auditAuthEvent("auth_2fa_operation_failed", authAuditFields(c, account.Username, "setup", "credential_invalid"))
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return
	}

	secret, uri, err := createPendingTOTPSetup(account.Username)
	if err != nil {
		log.Errorf("failed to create pending TOTP setup: %s", err)
		rsp.ErrRsp(c, -3, "create two factor setup failed")
		return
	}

	rsp.OkRspWithData(c, &proto.TwoFactorSetupRsp{
		URI:    uri,
		Secret: secret,
	})
	clearSensitiveOperationFailures(account.Username, sessionKey)
}

func (s *Service) TwoFactorSetupCancel(c *gin.Context) {
	var rsp proto.Response

	username := currentAccountUsername()
	if username != "" {
		cancelPendingTOTPSetup(username)
	}

	rsp.OkRsp(c)
}

func (s *Service) TwoFactorEnable(c *gin.Context) {
	var req proto.TwoFactorEnableReq
	var rsp proto.Response

	if err := bindSensitiveRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}

	account, err := GetAccount()
	if err != nil || account == nil {
		log.Errorf("failed to get account for TOTP enable: %v", err)
		rsp.ErrRsp(c, -3, "get account failed")
		return
	}
	enabled, err := isTOTPEnabled()
	if err != nil {
		log.Errorf("failed to read TOTP status for enable: %s", err)
		rsp.ErrRsp(c, -3, "get two factor status failed")
		return
	}
	if enabled {
		cancelPendingTOTPSetup(account.Username)
		auditAuthEvent("auth_2fa_operation_failed", authAuditFields(c, account.Username, "enable", "already_enabled"))
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return
	}
	sessionKey := sensitiveOperationSessionKey(c)
	if until, limited := sensitiveOperationRateLimitedUntil(account.Username, sessionKey); limited {
		auditAuthEvent("auth_2fa_operation_failed", authAuditFields(c, account.Username, "enable", "rate_limited"))
		writeAuthRateLimitRsp(c, &rsp, until)
		return
	}
	var invalidationErr error
	err = enablePendingTOTPSetupWithPreCommit(account.Username, req.Code, func() error {
		invalidationErr = invalidateAllAuthSessionsDueTo2FAEnabled()
		return invalidationErr
	})
	if invalidationErr != nil {
		log.Errorf("failed to invalidate authentication sessions before enabling 2FA: %v", invalidationErr)
		auditAuthEvent("auth_2fa_operation_failed", authAuditFields(c, account.Username, "enable", "session_invalidation_failed"))
		rsp.ErrRsp(c, -3, "enable two factor failed")
		return
	}
	if err != nil {
		recordSensitiveOperationFailure(account.Username, sessionKey)
		auditAuthEvent("auth_2fa_operation_failed", authAuditFields(c, account.Username, "enable", "second_factor_invalid"))
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return
	}

	clearAuthCookie(c)
	rsp.OkRsp(c)
	clearSensitiveOperationFailures(account.Username, sessionKey)
	auditAuthEvent("auth_2fa_enabled", authAuditFields(c, account.Username, "enable", "success"))
}

func (s *Service) TwoFactorDisable(c *gin.Context) {
	var req proto.TwoFactorDisableReq
	var rsp proto.Response

	if err := bindSensitiveRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}

	enabled, err := isTOTPEnabled()
	if err != nil {
		log.Errorf("failed to read TOTP status for disable: %s", err)
		rsp.ErrRsp(c, -3, "get two factor status failed")
		return
	}
	if !enabled {
		rsp.OkRsp(c)
		return
	}

	account, err := GetAccount()
	if err != nil || account == nil {
		log.Errorf("failed to get account for TOTP disable: %v", err)
		rsp.ErrRsp(c, -3, "get account failed")
		return
	}
	sessionKey := sensitiveOperationSessionKey(c)
	if until, limited := sensitiveOperationRateLimitedUntil(account.Username, sessionKey); limited {
		auditAuthEvent("auth_2fa_operation_failed", authAuditFields(c, account.Username, "disable", "rate_limited"))
		writeAuthRateLimitRsp(c, &rsp, until)
		return
	}
	if !CompareAccount(account.Username, req.Password) {
		recordSensitiveOperationFailure(account.Username, sessionKey)
		auditAuthEvent("auth_2fa_operation_failed", authAuditFields(c, account.Username, "disable", "credential_invalid"))
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return
	}
	secondFactor := verifySecondFactor(req.Code)
	if !secondFactor.OK {
		recordSensitiveOperationFailure(account.Username, sessionKey)
		auditAuthEvent("auth_2fa_operation_failed", authAuditFields(c, account.Username, "disable", "second_factor_invalid"))
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return
	}
	if err := clearTwoFactorProtectionForTwoFactorDisable(); err != nil {
		log.Errorf("failed to clear two factor protection during disable: %s", err)
		rsp.ErrRsp(c, -3, "disable two factor failed")
		return
	}
	if err := clearRecoveryCodes(); err != nil {
		log.Errorf("failed to clear recovery codes: %s", err)
		rsp.ErrRsp(c, -3, "disable two factor failed")
		return
	}
	cancelPendingTOTPSetup(account.Username)

	rsp.OkRsp(c)
	clearSensitiveOperationFailures(account.Username, sessionKey)
	if secondFactor.RecoveryCodeUsed {
		auditRecoveryCodeUsed(c, account.Username, "disable", secondFactor.RecoveryCodeID)
	}
	auditAuthEvent("auth_2fa_disabled", authAuditFields(c, account.Username, "disable", "success"))
}

func (s *Service) TwoFactorRecoveryEnable(c *gin.Context) {
	s.handleTwoFactorRecovery(c, false)
}

func (s *Service) TwoFactorRecoveryRegenerate(c *gin.Context) {
	s.handleTwoFactorRecovery(c, true)
}

func (s *Service) handleTwoFactorRecovery(c *gin.Context, regenerate bool) {
	var req proto.TwoFactorRecoveryReq
	var rsp proto.Response

	if err := bindSensitiveRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}

	enabled, err := isTOTPEnabled()
	if err != nil {
		log.Errorf("failed to read TOTP status for recovery codes: %s", err)
		rsp.ErrRsp(c, -3, "get two factor status failed")
		return
	}
	if !enabled {
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return
	}

	account, err := GetAccount()
	if err != nil || account == nil {
		log.Errorf("failed to get account for recovery codes: %v", err)
		rsp.ErrRsp(c, -3, "get account failed")
		return
	}
	sessionKey := sensitiveOperationSessionKey(c)
	if until, limited := sensitiveOperationRateLimitedUntil(account.Username, sessionKey); limited {
		auditAuthEvent("auth_2fa_operation_failed", authAuditFields(c, account.Username, recoveryOperationName(regenerate), "rate_limited"))
		writeAuthRateLimitRsp(c, &rsp, until)
		return
	}
	if !CompareAccount(account.Username, req.Password) {
		recordSensitiveOperationFailure(account.Username, sessionKey)
		auditAuthEvent("auth_2fa_operation_failed", authAuditFields(c, account.Username, recoveryOperationName(regenerate), "credential_invalid"))
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return
	}

	secondFactor := verifySecondFactor(req.Code)
	if !secondFactor.OK {
		recordSensitiveOperationFailure(account.Username, sessionKey)
		auditAuthEvent("auth_2fa_operation_failed", authAuditFields(c, account.Username, recoveryOperationName(regenerate), "second_factor_invalid"))
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return
	}

	if !regenerate {
		recoveryEnabled, _, err := recoveryCodesStatus()
		if err != nil {
			log.Errorf("failed to read recovery code status for enable: %s", err)
			rsp.ErrRsp(c, -3, "get recovery code status failed")
			return
		}
		if recoveryEnabled {
			rsp.ErrRsp(c, -2, "recovery codes already enabled")
			return
		}
	}

	codes, err := generateAndSaveRecoveryCodes(defaultRecoveryCodeCount)
	if err != nil {
		log.Errorf("failed to generate recovery codes: %s", err)
		rsp.ErrRsp(c, -3, "generate recovery codes failed")
		return
	}

	rsp.OkRspWithData(c, &proto.TwoFactorRecoveryRsp{RecoveryCodes: codes})
	clearSensitiveOperationFailures(account.Username, sessionKey)
	if secondFactor.RecoveryCodeUsed {
		auditRecoveryCodeUsed(c, account.Username, recoveryOperationName(regenerate), secondFactor.RecoveryCodeID)
	}
	auditAuthEvent("auth_recovery_codes_regenerated", authAuditFields(c, account.Username, recoveryOperationName(regenerate), "success"))
}

func currentAccountUsername() string {
	account, err := GetAccount()
	if err != nil || account == nil {
		return ""
	}
	return strings.TrimSpace(account.Username)
}

func recoveryOperationName(regenerate bool) string {
	if regenerate {
		return "recovery_regenerate"
	}
	return "recovery_enable"
}
