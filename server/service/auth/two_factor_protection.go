package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	defaultTwoFactorProtectionFilePath     = "/etc/kvm/two_factor_protection.json"
	twoFactorProtectionFileVersion         = 2
	twoFactorProtectionAuthorizationTTL    = 5 * time.Minute
	twoFactorProtectionAuthorizationNeeded = -4
)

var (
	twoFactorProtectionFilePath    = defaultTwoFactorProtectionFilePath
	twoFactorProtectionMu          sync.Mutex
	errTwoFactorProtectionDisabled = errors.New("two factor authentication is disabled")
)

type twoFactorProtectionPolicy struct {
	Cloudflare      bool
	ZeroTier        bool
	NetBird         bool
	FRP             bool
	Login           bool
	WebTerminal     bool
	SSH             bool
	Tailscale       bool
	MCP             bool
	TargetPower     bool
	NanoKVMRestart  bool
	Scripts         bool
	ScreenTimelapse bool
	DeleteImage     bool
}

type twoFactorProtectionFile struct {
	Cloudflare      *bool `json:"cloudflare"`
	ZeroTier        *bool `json:"zerotier"`
	NetBird         *bool `json:"netbird"`
	FRP             *bool `json:"frp"`
	Version         int   `json:"version"`
	Login           *bool `json:"login"`
	WebTerminal     *bool `json:"webTerminal"`
	SSH             *bool `json:"ssh"`
	Tailscale       *bool `json:"tailscale"`
	MCP             *bool `json:"mcp"`
	TargetPower     *bool `json:"targetPower"`
	NanoKVMRestart  *bool `json:"nanoKVMRestart"`
	Scripts         *bool `json:"scripts"`
	ScreenTimelapse *bool `json:"screenTimelapse"`
	DeleteImage     *bool `json:"deleteImage"`
}

type twoFactorProtectionAuthorization struct {
	username        string
	expiresAt       time.Time
	ip              string
	userAgentDigest string
	sessionKeyHash  string
}

var twoFactorProtectionAuthorizations = struct {
	sync.Mutex
	items map[string]twoFactorProtectionAuthorization
}{items: make(map[string]twoFactorProtectionAuthorization)}

func defaultTwoFactorProtectionPolicy() twoFactorProtectionPolicy {
	return twoFactorProtectionPolicy{
		Login:           false,
		WebTerminal:     false,
		SSH:             false,
		Tailscale:       false,
		MCP:             false,
		TargetPower:     false,
		NanoKVMRestart:  false,
		Scripts:         true,
		ScreenTimelapse: false,
		DeleteImage:     false,
	}
}

func (p twoFactorProtectionPolicy) response() proto.TwoFactorProtectionPolicyRsp {
	return proto.TwoFactorProtectionPolicyRsp{
		Cloudflare:      p.Cloudflare,
		ZeroTier:        p.ZeroTier,
		NetBird:         p.NetBird,
		FRP:             p.FRP,
		Login:           p.Login,
		WebTerminal:     p.WebTerminal,
		SSH:             p.SSH,
		Tailscale:       p.Tailscale,
		MCP:             p.MCP,
		TargetPower:     p.TargetPower,
		NanoKVMRestart:  p.NanoKVMRestart,
		Scripts:         p.Scripts,
		ScreenTimelapse: p.ScreenTimelapse,
		DeleteImage:     p.DeleteImage,
	}
}

func directTwoFactorLoginEnabled() (bool, error) {
	enabled, err := isTOTPEnabled()
	if err != nil {
		return false, err
	}
	if !enabled {
		return false, nil
	}

	policy, err := loadTwoFactorProtectionPolicy()
	if err != nil {
		return false, err
	}
	return policy.Login, nil
}

func twoFactorProtectionPolicyFromRequest(req proto.SetTwoFactorProtectionReq) (twoFactorProtectionPolicy, error) {
	if req.Login == nil || req.WebTerminal == nil || req.SSH == nil || req.Tailscale == nil || req.MCP == nil ||
		req.TargetPower == nil || req.NanoKVMRestart == nil || req.Scripts == nil ||
		req.ScreenTimelapse == nil || req.DeleteImage == nil {
		return twoFactorProtectionPolicy{}, errors.New("incomplete two factor protection policy")
	}

	return twoFactorProtectionPolicy{
		Login:           *req.Login,
		Cloudflare:      req.Cloudflare != nil && *req.Cloudflare,
		ZeroTier:        req.ZeroTier != nil && *req.ZeroTier,
		NetBird:         req.NetBird != nil && *req.NetBird,
		FRP:             req.FRP != nil && *req.FRP,
		WebTerminal:     *req.WebTerminal,
		SSH:             *req.SSH,
		Tailscale:       *req.Tailscale,
		MCP:             *req.MCP,
		TargetPower:     *req.TargetPower,
		NanoKVMRestart:  *req.NanoKVMRestart,
		Scripts:         *req.Scripts,
		ScreenTimelapse: *req.ScreenTimelapse,
		DeleteImage:     *req.DeleteImage,
	}, nil
}

// deleteImageFromStoredField defaults missing deleteImage on older on-disk policies
// to enabled, matching other destructive advanced-protection defaults.
func deleteImageFromStoredField(value *bool) *bool {
	if value == nil {
		return boolPointer(true)
	}
	return value
}

func (f twoFactorProtectionFile) policy() (twoFactorProtectionPolicy, error) {
	if f.Version == 1 {
		return twoFactorProtectionPolicyFromRequest(proto.SetTwoFactorProtectionReq{
			Cloudflare:      f.Cloudflare,
			ZeroTier:        f.ZeroTier,
			NetBird:         f.NetBird,
			FRP:             f.FRP,
			Login:           boolPointer(false),
			WebTerminal:     f.WebTerminal,
			SSH:             f.SSH,
			Tailscale:       f.Tailscale,
			MCP:             f.MCP,
			TargetPower:     f.TargetPower,
			NanoKVMRestart:  f.NanoKVMRestart,
			Scripts:         f.Scripts,
			ScreenTimelapse: f.ScreenTimelapse,
			DeleteImage:     deleteImageFromStoredField(f.DeleteImage),
		})
	}
	if f.Version != twoFactorProtectionFileVersion {
		return twoFactorProtectionPolicy{}, fmt.Errorf("unsupported two factor protection version %d", f.Version)
	}

	return twoFactorProtectionPolicyFromRequest(proto.SetTwoFactorProtectionReq{
		Cloudflare:      f.Cloudflare,
		ZeroTier:        f.ZeroTier,
		NetBird:         f.NetBird,
		FRP:             f.FRP,
		Login:           f.Login,
		WebTerminal:     f.WebTerminal,
		SSH:             f.SSH,
		Tailscale:       f.Tailscale,
		MCP:             f.MCP,
		TargetPower:     f.TargetPower,
		NanoKVMRestart:  f.NanoKVMRestart,
		Scripts:         f.Scripts,
		ScreenTimelapse: f.ScreenTimelapse,
		DeleteImage:     deleteImageFromStoredField(f.DeleteImage),
	})
}

func twoFactorProtectionFileFromPolicy(policy twoFactorProtectionPolicy) twoFactorProtectionFile {
	return twoFactorProtectionFile{
		Cloudflare:      boolPointer(policy.Cloudflare),
		ZeroTier:        boolPointer(policy.ZeroTier),
		NetBird:         boolPointer(policy.NetBird),
		FRP:             boolPointer(policy.FRP),
		Version:         twoFactorProtectionFileVersion,
		Login:           boolPointer(policy.Login),
		WebTerminal:     boolPointer(policy.WebTerminal),
		SSH:             boolPointer(policy.SSH),
		Tailscale:       boolPointer(policy.Tailscale),
		MCP:             boolPointer(policy.MCP),
		TargetPower:     boolPointer(policy.TargetPower),
		NanoKVMRestart:  boolPointer(policy.NanoKVMRestart),
		Scripts:         boolPointer(policy.Scripts),
		ScreenTimelapse: boolPointer(policy.ScreenTimelapse),
		DeleteImage:     boolPointer(policy.DeleteImage),
	}
}

func boolPointer(value bool) *bool {
	return &value
}

func loadTwoFactorProtectionPolicy() (twoFactorProtectionPolicy, error) {
	twoFactorProtectionMu.Lock()
	defer twoFactorProtectionMu.Unlock()

	return loadTwoFactorProtectionPolicyLocked()
}

func loadTwoFactorProtectionPolicyLocked() (twoFactorProtectionPolicy, error) {
	data, err := os.ReadFile(twoFactorProtectionFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaultTwoFactorProtectionPolicy(), nil
		}
		return twoFactorProtectionPolicy{}, err
	}

	var file twoFactorProtectionFile
	if err := json.Unmarshal(data, &file); err != nil {
		return twoFactorProtectionPolicy{}, err
	}

	return file.policy()
}

func saveTwoFactorProtectionPolicy(policy twoFactorProtectionPolicy) error {
	twoFactorProtectionMu.Lock()
	defer twoFactorProtectionMu.Unlock()

	enabled, err := isTOTPEnabled()
	if err != nil {
		return err
	}
	if !enabled {
		return errTwoFactorProtectionDisabled
	}

	return saveTwoFactorProtectionPolicyLocked(policy)
}

func saveTwoFactorProtectionPolicyLocked(policy twoFactorProtectionPolicy) error {
	data, err := json.MarshalIndent(twoFactorProtectionFileFromPolicy(policy), "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(twoFactorProtectionFilePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".two_factor_protection.json.*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, twoFactorProtectionFilePath); err != nil {
		return err
	}
	if err := os.Chmod(twoFactorProtectionFilePath, 0o600); err != nil {
		return err
	}

	dirFile, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer dirFile.Close()
	return dirFile.Sync()
}

// clearTwoFactorProtectionForTwoFactorDisable serializes policy removal with the TOTP
// secret transition. A save that began before this function cannot recreate a policy after
// the secret has been removed, because saveTwoFactorProtectionPolicy rechecks TOTP under
// the same lock.
func clearTwoFactorProtectionForTwoFactorDisable() error {
	twoFactorProtectionMu.Lock()
	defer twoFactorProtectionMu.Unlock()

	if err := clearTwoFactorProtectionPolicyLocked(); err != nil {
		return err
	}
	clearTwoFactorProtectionAuthorizations()
	clearMCPAuthorizations()
	clearScreenTimelapsePlaybackAuthorizations()
	return clearActiveTOTPSecret()
}

func clearTwoFactorProtectionPolicyLocked() error {
	if err := os.Remove(twoFactorProtectionFilePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	dirFile, err := os.Open(filepath.Dir(twoFactorProtectionFilePath))
	if err != nil {
		return err
	}
	defer dirFile.Close()
	return dirFile.Sync()
}

func twoFactorProtectionAuthorizationKey(sessionKey string) string {
	return hashAuthRateLimitKey("two-factor-protection:" + strings.TrimSpace(sessionKey))
}

func grantTwoFactorProtectionAuthorization(username, sessionKey, ip, userAgent string) {
	now := totpNow()
	key := twoFactorProtectionAuthorizationKey(sessionKey)

	twoFactorProtectionAuthorizations.Lock()
	defer twoFactorProtectionAuthorizations.Unlock()
	cleanupExpiredTwoFactorProtectionAuthorizationsLocked(now)
	twoFactorProtectionAuthorizations.items[key] = twoFactorProtectionAuthorization{
		username:        strings.TrimSpace(username),
		expiresAt:       now.Add(twoFactorProtectionAuthorizationTTL),
		ip:              strings.TrimSpace(ip),
		userAgentDigest: userAgentHash(userAgent),
		sessionKeyHash:  sensitiveActionSessionKeyHash(sessionKey),
	}
}

func hasTwoFactorProtectionAuthorization(username, sessionKey, ip, userAgent string) bool {
	now := totpNow()
	key := twoFactorProtectionAuthorizationKey(sessionKey)

	twoFactorProtectionAuthorizations.Lock()
	defer twoFactorProtectionAuthorizations.Unlock()
	cleanupExpiredTwoFactorProtectionAuthorizationsLocked(now)

	authorization, ok := twoFactorProtectionAuthorizations.items[key]
	if !ok || !now.Before(authorization.expiresAt) {
		delete(twoFactorProtectionAuthorizations.items, key)
		return false
	}
	if authorization.username != strings.TrimSpace(username) || authorization.ip != strings.TrimSpace(ip) ||
		authorization.userAgentDigest != userAgentHash(userAgent) ||
		authorization.sessionKeyHash != sensitiveActionSessionKeyHash(sessionKey) {
		return false
	}

	return true
}

func twoFactorProtectionAuthorizationExpiresIn(username, sessionKey, ip, userAgent string) int {
	now := totpNow()
	key := twoFactorProtectionAuthorizationKey(sessionKey)

	twoFactorProtectionAuthorizations.Lock()
	defer twoFactorProtectionAuthorizations.Unlock()
	cleanupExpiredTwoFactorProtectionAuthorizationsLocked(now)

	authorization, ok := twoFactorProtectionAuthorizations.items[key]
	if !ok || !now.Before(authorization.expiresAt) || authorization.username != strings.TrimSpace(username) ||
		authorization.ip != strings.TrimSpace(ip) || authorization.userAgentDigest != userAgentHash(userAgent) ||
		authorization.sessionKeyHash != sensitiveActionSessionKeyHash(sessionKey) {
		return 0
	}

	return int(authorization.expiresAt.Sub(now).Seconds())
}

func clearTwoFactorProtectionAuthorizations() {
	twoFactorProtectionAuthorizations.Lock()
	defer twoFactorProtectionAuthorizations.Unlock()
	twoFactorProtectionAuthorizations.items = make(map[string]twoFactorProtectionAuthorization)
}

func resetTwoFactorProtectionAuthorizationsForTest() {
	clearTwoFactorProtectionAuthorizations()
}

func cleanupExpiredTwoFactorProtectionAuthorizationsLocked(now time.Time) {
	for key, authorization := range twoFactorProtectionAuthorizations.items {
		if !now.Before(authorization.expiresAt) {
			delete(twoFactorProtectionAuthorizations.items, key)
		}
	}
}

func (s *Service) GetTwoFactorProtection(c *gin.Context) {
	var rsp proto.Response

	username, sessionKey, ok := twoFactorProtectionSession(c, &rsp)
	if !ok {
		return
	}

	policy, err := loadTwoFactorProtectionPolicy()
	if err != nil {
		log.Errorf("failed to load two factor protection policy: %v", err)
		rsp.ErrRsp(c, -3, "get two factor protection failed")
		return
	}

	rsp.OkRspWithData(c, &proto.GetTwoFactorProtectionRsp{
		Policy:                                policy.response(),
		AuthorizationExpiresIn:                twoFactorProtectionAuthorizationExpiresIn(username, sessionKey, c.ClientIP(), c.Request.UserAgent()),
		MCPAuthorizationExpiresIn:             mcpAuthorizationExpiresIn(username, sessionKey, c.ClientIP(), c.Request.UserAgent()),
		ScreenTimelapseAuthorizationExpiresIn: screenTimelapsePlaybackAuthorizationExpiresIn(username, sessionKey, c.ClientIP(), c.Request.UserAgent()),
	})
}

// GetDirectTwoFactorLoginStatus returns whether unauthenticated clients may use the
// direct code-only login form. It intentionally exposes no account or policy details.
func (s *Service) GetDirectTwoFactorLoginStatus(c *gin.Context) {
	var rsp proto.Response

	enabled, err := directTwoFactorLoginEnabled()
	if err != nil {
		log.Errorf("failed to read direct two factor login status: %v", err)
		enabled = false
	}

	rsp.OkRspWithData(c, &proto.DirectTwoFactorLoginStatusRsp{Enabled: enabled})
}

func (s *Service) AuthorizeTwoFactorProtection(c *gin.Context) {
	var req proto.AuthorizeTwoFactorProtectionReq
	var rsp proto.Response

	if err := bindSensitiveRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}

	enabled, err := isTOTPEnabled()
	if err != nil {
		log.Errorf("failed to read TOTP status for two factor protection authorization: %v", err)
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
		log.Errorf("failed to get account for two factor protection authorization: %v", err)
		rsp.ErrRsp(c, -3, "get account failed")
		return
	}
	username := strings.TrimSpace(account.Username)
	sessionKey := sensitiveOperationSessionKey(c)
	if jwtUsername := currentJWTUsername(c); jwtUsername != "" && jwtUsername != username {
		recordSensitiveOperationFailure(username, sessionKey)
		auditAuthEvent("auth_2fa_protection_authorization_failed", authAuditFields(c, username, "protection_authorize", "user_mismatch"))
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return
	}
	if until, limited := sensitiveOperationRateLimitedUntil(username, sessionKey); limited {
		auditAuthEvent("auth_2fa_protection_authorization_failed", authAuditFields(c, username, "protection_authorize", "rate_limited"))
		writeAuthRateLimitRsp(c, &rsp, until)
		return
	}

	// Protection policy edits re-auth with second factor only (no login password).
	secondFactor := verifySecondFactor(req.Code)
	if !secondFactor.OK {
		recordSensitiveOperationFailure(username, sessionKey)
		auditAuthEvent("auth_2fa_protection_authorization_failed", authAuditFields(c, username, "protection_authorize", "second_factor_invalid"))
		time.Sleep(invalidCredentialDelay)
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return
	}

	grantTwoFactorProtectionAuthorization(username, sessionKey, c.ClientIP(), c.Request.UserAgent())
	rsp.OkRspWithData(c, &proto.AuthorizeTwoFactorProtectionRsp{
		ExpiresIn: int(twoFactorProtectionAuthorizationTTL.Seconds()),
	})
	clearSensitiveOperationFailures(username, sessionKey)
	if secondFactor.RecoveryCodeUsed {
		auditRecoveryCodeUsed(c, username, "protection_authorize", secondFactor.RecoveryCodeID)
	}
	auditAuthEvent("auth_2fa_protection_authorized", authAuditFields(c, username, "protection_authorize", "success"))
}

func (s *Service) SetTwoFactorProtection(c *gin.Context) {
	var req proto.SetTwoFactorProtectionReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}
	policy, err := twoFactorProtectionPolicyFromRequest(req)
	if err != nil {
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}

	username, sessionKey, ok := twoFactorProtectionSession(c, &rsp)
	if !ok {
		return
	}
	if !hasTwoFactorProtectionAuthorization(username, sessionKey, c.ClientIP(), c.Request.UserAgent()) {
		auditAuthEvent("auth_2fa_protection_update_failed", authAuditFields(c, username, "protection_update", "authorization_required"))
		rsp.ErrRsp(c, twoFactorProtectionAuthorizationNeeded, "two factor protection authorization required")
		return
	}

	// Preserve choices when older clients omit extension switches.
	if req.FRP == nil || req.Cloudflare == nil || req.ZeroTier == nil || req.NetBird == nil {
		stored, err := loadTwoFactorProtectionPolicy()
		if err != nil {
			rsp.ErrRsp(c, -3, "get two factor protection failed")
			return
		}
		if req.FRP == nil {
			policy.FRP = stored.FRP
		}
		if req.Cloudflare == nil {
			policy.Cloudflare = stored.Cloudflare
		}
		if req.ZeroTier == nil {
			policy.ZeroTier = stored.ZeroTier
		}
		if req.NetBird == nil {
			policy.NetBird = stored.NetBird
		}
	}
	if err := saveTwoFactorProtectionPolicy(policy); err != nil {
		if errors.Is(err, errTwoFactorProtectionDisabled) {
			rsp.ErrRsp(c, -2, "two factor authentication is disabled")
			return
		}
		log.Errorf("failed to save two factor protection policy: %v", err)
		rsp.ErrRsp(c, -3, "save two factor protection failed")
		return
	}

	rsp.OkRspWithData(c, &proto.GetTwoFactorProtectionRsp{
		Policy:                 policy.response(),
		AuthorizationExpiresIn: twoFactorProtectionAuthorizationExpiresIn(username, sessionKey, c.ClientIP(), c.Request.UserAgent()),
	})
	auditAuthEvent("auth_2fa_protection_updated", authAuditFields(c, username, "protection_update", "success"))
}

// WebTerminalRequiresSecondFactor reports whether opening the web terminal must
// present a TOTP/recovery code in addition to the account password.
// TOTP off → false (historical password-only gate; ignores any stale on-disk true).
// TOTP on → policy.WebTerminal; TOTP-status or policy read errors fail closed (true).
func WebTerminalRequiresSecondFactor() bool {
	enabled, err := isTOTPEnabled()
	if err != nil {
		return true
	}
	return webTerminalRequiresSecondFactor(enabled)
}

// webTerminalRequiresSecondFactor is the inner decision with the caller's
// already-read TOTP state, avoiding a second TOTP read (TOCTOU).
func webTerminalRequiresSecondFactor(totpEnabled bool) bool {
	if !totpEnabled {
		return false
	}

	policy, err := loadTwoFactorProtectionPolicy()
	if err != nil {
		return true
	}
	return policy.WebTerminal
}

// ScriptActionRequiresChallenge evaluates advanced script protection for a sensitive action.
// TOTP-off keeps historical run-vs-upload/delete behavior; TOTP-on uses the single scripts policy.
// Errors fail closed when TOTP is enabled or status is unknown.
func ScriptActionRequiresChallenge(action string) bool {
	enabled, err := isTOTPEnabled()
	if err != nil {
		return true
	}
	if !enabled {
		// Historical: only script execution stays password-gated when 2FA is off.
		return action == SensitiveActionRunScript
	}

	policy, err := loadTwoFactorProtectionPolicy()
	if err != nil {
		return true
	}
	return policy.Scripts
}

// ScriptUploadRequiresChallenge reports whether uploading a script must present a challenge.
func ScriptUploadRequiresChallenge() bool {
	return ScriptActionRequiresChallenge(SensitiveActionUploadScript)
}

// ScriptExecutionRequiresChallenge reports whether running a script must present a challenge.
func ScriptExecutionRequiresChallenge() bool {
	return ScriptActionRequiresChallenge(SensitiveActionRunScript)
}

// ScriptDeleteRequiresChallenge reports whether deleting a script must present a challenge.
func ScriptDeleteRequiresChallenge() bool {
	return ScriptActionRequiresChallenge(SensitiveActionDeleteScript)
}

// SSHActionRequiresChallenge reports whether enabling or disabling device SSH
// must present a sensitive-action challenge. TOTP off → false (historical).
// TOTP on → policy.SSH; TOTP/policy errors fail closed (require challenge).
// When policy.SSH is false, behavior matches TOTP off (no challenge).
func SSHActionRequiresChallenge() bool {
	enabled, err := isTOTPEnabled()
	if err != nil {
		return true
	}
	if !enabled {
		return false
	}

	policy, err := loadTwoFactorProtectionPolicy()
	if err != nil {
		return true
	}
	return policy.SSH
}

// SSHEnableRequiresChallenge is an alias of SSHActionRequiresChallenge.
func SSHEnableRequiresChallenge() bool {
	return SSHActionRequiresChallenge()
}

// SSHDisableRequiresChallenge is an alias of SSHActionRequiresChallenge.
func SSHDisableRequiresChallenge() bool {
	return SSHActionRequiresChallenge()
}

// TargetPowerActionRequiresChallenge reports whether a physical target-machine
// power-button action must present a sensitive-action challenge. TOTP off →
// false (historical behavior). TOTP/policy errors fail closed.
func TargetPowerActionRequiresChallenge() bool {
	enabled, err := isTOTPEnabled()
	if err != nil {
		return true
	}
	if !enabled {
		return false
	}

	policy, err := loadTwoFactorProtectionPolicy()
	if err != nil {
		return true
	}
	return policy.TargetPower
}

// NanoKVMRestartActionRequiresChallenge reports whether restarting NanoKVM
// itself must present a sensitive-action challenge. TOTP off → false
// (historical behavior). TOTP/policy errors fail closed.
func NanoKVMRestartActionRequiresChallenge() bool {
	enabled, err := isTOTPEnabled()
	if err != nil {
		return true
	}
	if !enabled {
		return false
	}

	policy, err := loadTwoFactorProtectionPolicy()
	if err != nil {
		return true
	}
	return policy.NanoKVMRestart
}

// DeleteImageActionRequiresChallenge reports whether permanently deleting a
// stored ISO/IMG image must present a sensitive-action challenge. TOTP off →
// false (historical behavior). TOTP/policy errors fail closed.
func DeleteImageActionRequiresChallenge() bool {
	enabled, err := isTOTPEnabled()
	if err != nil {
		return true
	}
	if !enabled {
		return false
	}

	policy, err := loadTwoFactorProtectionPolicy()
	if err != nil {
		return true
	}
	return policy.DeleteImage
}

// ScreenTimelapseEnableRequiresChallenge reports whether enabling Screen
// Timelapse must present a sensitive-action challenge. TOTP off → false
// (historical behavior). TOTP/policy errors fail closed.
func ScreenTimelapseEnableRequiresChallenge() bool {
	return screenTimelapseRequiresChallenge()
}

// ScreenTimelapsePlaybackRequiresChallenge reports whether browsing or deleting
// Screen Timelapse recordings must present a sensitive-action challenge.
func ScreenTimelapsePlaybackRequiresChallenge() bool {
	return screenTimelapseRequiresChallenge()
}

func screenTimelapseRequiresChallenge() bool {
	enabled, err := isTOTPEnabled()
	if err != nil {
		return true
	}
	if !enabled {
		return false
	}

	policy, err := loadTwoFactorProtectionPolicy()
	if err != nil {
		return true
	}
	return policy.ScreenTimelapse
}

// TailscaleActionRequiresChallenge reports whether a Tailscale mutating action
// must present a challenge. TOTP off → false. TOTP on → policy.Tailscale for
// listed actions; TOTP/policy errors fail closed.
// When policy.Tailscale is false, behavior matches TOTP off (no challenge).
// Covers all Tailscale settings-page ops (install/login/up/down/start/stop/
// restart/logout/uninstall/exit-node/subnet-route). Status reads are not gated.
func TailscaleActionRequiresChallenge(action string) bool {
	enabled, err := isTOTPEnabled()
	if err != nil {
		return true
	}
	if !enabled {
		return false
	}

	policy, err := loadTwoFactorProtectionPolicy()
	if err != nil {
		return true
	}
	// Unknown action under TOTP still fails closed so call-site mistakes cannot bypass.
	if !IsTailscaleSensitiveAction(action) {
		return true
	}
	return policy.Tailscale
}

// FRP mutations use their own advanced-protection switch. Old policies default
// it to off, just like Tailscale; unreadable state and unknown actions fail closed.
func FRPActionRequiresChallenge(action string) bool {
	enabled, err := isTOTPEnabled()
	if err != nil {
		return true
	}
	if !enabled {
		return false
	}
	policy, err := loadTwoFactorProtectionPolicy()
	if err != nil || !IsFRPSensitiveAction(action) {
		return true
	}
	return policy.FRP
}

// Cloudflare uses an independent protection policy, with the same fail-closed rules.
func CloudflareActionRequiresChallenge(action string) bool {
	enabled, err := isTOTPEnabled()
	if err != nil {
		return true
	}
	if !enabled {
		return false
	}
	policy, err := loadTwoFactorProtectionPolicy()
	if err != nil || !IsCloudflareSensitiveAction(action) {
		return true
	}
	return policy.Cloudflare
}

// MCPActionRequiresChallenge reports whether an MCP credential action must
// present a sensitive-action challenge. TOTP off → false. TOTP on → policy.MCP;
// TOTP/policy errors and unrecognized actions fail closed.
func MCPActionRequiresChallenge(action string) bool {
	enabled, err := isTOTPEnabled()
	if err != nil {
		return true
	}
	if !enabled {
		return false
	}

	policy, err := loadTwoFactorProtectionPolicy()
	if err != nil {
		return true
	}
	if !IsMCPSensitiveAction(action) {
		return true
	}
	return policy.MCP
}

// ScreenTimelapseActionRequiresChallenge reports whether changing retention
// policy or requesting an immediate cleanup must present a sensitive-action
// challenge. TOTP off keeps the historical ungated behavior; TOTP/policy
// failures and unrecognized actions fail closed.
func ScreenTimelapseActionRequiresChallenge(action string) bool {
	enabled, err := isTOTPEnabled()
	if err != nil {
		return true
	}
	if !enabled {
		return false
	}

	policy, err := loadTwoFactorProtectionPolicy()
	if err != nil {
		return true
	}
	if !IsScreenTimelapseSensitiveAction(action) {
		return true
	}
	return policy.ScreenTimelapse
}

func twoFactorProtectionSession(c *gin.Context, rsp *proto.Response) (string, string, bool) {
	enabled, err := isTOTPEnabled()
	if err != nil {
		log.Errorf("failed to read TOTP status for two factor protection: %v", err)
		rsp.ErrRsp(c, -3, "get two factor status failed")
		return "", "", false
	}
	if !enabled {
		rsp.ErrRsp(c, -2, "two factor authentication is disabled")
		return "", "", false
	}

	account, err := GetAccount()
	if err != nil || account == nil {
		log.Errorf("failed to get account for two factor protection: %v", err)
		rsp.ErrRsp(c, -3, "get account failed")
		return "", "", false
	}
	username := strings.TrimSpace(account.Username)
	if jwtUsername := currentJWTUsername(c); jwtUsername != "" && jwtUsername != username {
		rsp.ErrRsp(c, -2, invalidAuthMessage)
		return "", "", false
	}

	return username, sensitiveOperationSessionKey(c), true
}

func ZeroTierActionRequiresChallenge(action string) bool {
	enabled, err := isTOTPEnabled()
	if err != nil {
		return true
	}
	if !enabled {
		return false
	}
	policy, err := loadTwoFactorProtectionPolicy()
	if err != nil || !IsZeroTierSensitiveAction(action) {
		return true
	}
	return policy.ZeroTier
}

func NetBirdActionRequiresChallenge(action string) bool {
	enabled, err := isTOTPEnabled()
	if err != nil {
		return true
	}
	if !enabled {
		return false
	}
	policy, err := loadTwoFactorProtectionPolicy()
	if err != nil || !IsNetBirdSensitiveAction(action) {
		return true
	}
	return policy.NetBird
}
