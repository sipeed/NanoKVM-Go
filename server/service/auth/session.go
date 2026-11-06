package auth

import (
	"NanoKVM-Server/config"
	"NanoKVM-Server/service/authsession"
	"NanoKVM-Server/service/media"
	"NanoKVM-Server/service/ws"

	log "github.com/sirupsen/logrus"
)

const (
	sessionExpiredWebSocketCloseCode   = 4001
	twoFactorEnabledWebSocketCloseCode = 4002
	SystemTimeUpdatedSessionCloseCode  = 4003
)

var invalidateAllAuthSessions = func() {
	if err := InvalidateAllSessions(); err != nil {
		log.Errorf("failed to invalidate authentication sessions: %v", err)
	}
}

var (
	regenerateSessionJWTSecretKey = config.RegenerateJWTSecretKey
)

// InvalidateAllSessions rotates the JWT key and closes authenticated streaming
// connections. Callers use this after changing the wall clock so tokens issued
// against an incorrect system time cannot become valid again.
func InvalidateAllSessions() error {
	return invalidateAllSessionsWithCloseReason(sessionExpiredWebSocketCloseCode, "session expired")
}

func invalidateAllSessionsWithCloseReason(code int, reason string) error {
	if err := regenerateSessionJWTSecretKey(); err != nil {
		return err
	}
	finishSessionInvalidation(code, reason)
	return nil
}

// PrepareSystemTimeSessionInvalidation permanently rotates the JWT key before
// the clock changes. The returned function closes remaining capabilities with
// the result-specific reason and performs no fallible I/O.
func PrepareSystemTimeSessionInvalidation() (func(timeChanged bool), error) {
	if err := regenerateSessionJWTSecretKey(); err != nil {
		return nil, err
	}

	return func(timeChanged bool) {
		if timeChanged {
			finishSessionInvalidation(SystemTimeUpdatedSessionCloseCode, "system time updated")
			return
		}
		finishSessionInvalidation(sessionExpiredWebSocketCloseCode, "session expired")
	}, nil
}

func finishSessionInvalidation(code int, reason string) {
	clearEphemeralAuthenticationState()
	media.DefaultManager().CloseAll()
	authsession.CloseAll(code, reason)
	ws.GetManager().CloseAll(code, reason)
}

func clearEphemeralAuthenticationState() {
	loginChallenges.Lock()
	loginChallenges.items = make(map[string]loginChallenge)
	loginChallenges.Unlock()

	sensitiveActionChallenges.Lock()
	sensitiveActionChallenges.items = make(map[string]sensitiveActionChallenge)
	sensitiveActionChallenges.Unlock()

	sensitiveActionAuthorizations.Lock()
	sensitiveActionAuthorizations.items = make(map[string]sensitiveActionAuthorization)
	sensitiveActionAuthorizations.Unlock()

	pendingTOTPSetups.Lock()
	pendingTOTPSetups.items = make(map[string]pendingTOTPSetup)
	pendingTOTPSetups.Unlock()

	clearTwoFactorProtectionAuthorizations()
}

var invalidateAllAuthSessionsDueTo2FAEnabled = func() error {
	if err := regenerateSessionJWTSecretKey(); err != nil {
		return err
	}
	authsession.CloseAll(twoFactorEnabledWebSocketCloseCode, "2FA enabled")
	ws.GetManager().CloseAll(twoFactorEnabledWebSocketCloseCode, "2FA enabled")
	return nil
}
