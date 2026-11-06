package auth

import (
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func (s *Service) VerifySensitiveAction(c *gin.Context) {
	var req proto.VerifySensitiveActionReq
	var rsp proto.Response

	if err := bindSensitiveRequest(c, &req); err != nil || !validSensitiveAction(req.Action) {
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}

	account, err := GetAccount()
	if err != nil || account == nil {
		log.Errorf("failed to get account for sensitive action: %v", err)
		rsp.ErrRsp(c, -3, "get account failed")
		return
	}

	jwtUsername := currentJWTUsername(c)
	username := account.Username
	sessionKey := sensitiveOperationSessionKey(c)
	if req.Action == SensitiveActionOpenWebTerminal && jwtUsername == "" {
		recordSensitiveOperationFailure(username, sessionKey)
		auditAuthEvent("auth_sensitive_action_failed", authAuditFields(c, username, req.Action, "identity_missing"))
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return
	}
	if jwtUsername != "" && jwtUsername != username {
		recordSensitiveOperationFailure(username, sessionKey)
		auditAuthEvent("auth_sensitive_action_failed", authAuditFields(c, username, req.Action, "user_mismatch"))
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return
	}
	if until, limited := sensitiveOperationRateLimitedUntil(username, sessionKey); limited {
		auditAuthEvent("auth_sensitive_action_failed", authAuditFields(c, username, req.Action, "rate_limited"))
		writeAuthRateLimitRsp(c, &rsp, until)
		return
	}

	var secondFactor secondFactorResult
	enabled, err := isTOTPEnabled()
	if err != nil {
		log.Errorf("failed to read TOTP status for sensitive action: %s", err)
		rsp.ErrRsp(c, -3, "get two factor status failed")
		return
	}

	// Advanced protection-gated actions: when TOTP is on, require only second factor.
	// Non-protection actions (and TOTP-off historical gates) still require password.
	codeOnly := enabled && isProtectionSensitiveAction(req.Action)
	if !codeOnly {
		if !CompareAccount(username, req.Password) {
			recordSensitiveOperationFailure(username, sessionKey)
			auditAuthEvent("auth_sensitive_action_failed", authAuditFields(c, username, req.Action, "credential_invalid"))
			time.Sleep(invalidCredentialDelay)
			rsp.ErrRsp(c, -2, invalidAuthMessage)
			return
		}
	}

	// The web terminal never goes password-less, but its second factor follows the
	// webTerminal protection policy; every other action keeps the historical
	// TOTP-on-always behavior. When the factor is not required, a submitted code is
	// ignored entirely so recovery codes are never burned needlessly.
	requireSecondFactor := enabled
	if req.Action == SensitiveActionOpenWebTerminal {
		requireSecondFactor = webTerminalRequiresSecondFactor(enabled)
	}

	if requireSecondFactor {
		secondFactor = verifySecondFactor(req.Code)
		if !secondFactor.OK {
			recordSensitiveOperationFailure(username, sessionKey)
			auditAuthEvent("auth_sensitive_action_failed", authAuditFields(c, username, req.Action, "second_factor_invalid"))
			time.Sleep(invalidCredentialDelay)
			rsp.ErrRsp(c, -2, invalidAuthMessage)
			return
		}
	}

	challenge, err := createSensitiveActionChallenge(username, req.Action, sessionKey, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		log.Errorf("failed to create sensitive action challenge: %s", err)
		rsp.ErrRsp(c, -3, "generate challenge failed")
		return
	}

	rsp.OkRspWithData(c, &proto.VerifySensitiveActionRsp{
		Challenge: challenge,
		ExpiresIn: int(sensitiveActionChallengeTTL.Seconds()),
	})
	clearSensitiveOperationFailures(username, sessionKey)
	if enabled && secondFactor.RecoveryCodeUsed {
		auditRecoveryCodeUsed(c, username, req.Action, secondFactor.RecoveryCodeID)
	}
}
