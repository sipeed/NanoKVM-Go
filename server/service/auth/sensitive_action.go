package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	SensitiveActionChangePassword                = "change_password"
	SensitiveActionRunScript                     = "run_script"
	SensitiveActionUploadScript                  = "upload_script"
	SensitiveActionDeleteScript                  = "delete_script"
	SensitiveActionEnableSSH                     = "enable_ssh"
	SensitiveActionDisableSSH                    = "disable_ssh"
	SensitiveActionNetBirdInstall                = "netbird_install"
	SensitiveActionNetBirdUninstall              = "netbird_uninstall"
	SensitiveActionNetBirdStart                  = "netbird_start"
	SensitiveActionNetBirdStop                   = "netbird_stop"
	SensitiveActionNetBirdRestart                = "netbird_restart"
	SensitiveActionNetBirdLogin                  = "netbird_login"
	SensitiveActionNetBirdLogout                 = "netbird_logout"
	SensitiveActionNetBirdLoginCancel            = "netbird_login_cancel"
	SensitiveActionZeroTierInstall               = "zerotier_install"
	SensitiveActionZeroTierUninstall             = "zerotier_uninstall"
	SensitiveActionZeroTierStart                 = "zerotier_start"
	SensitiveActionZeroTierStop                  = "zerotier_stop"
	SensitiveActionZeroTierRestart               = "zerotier_restart"
	SensitiveActionZeroTierJoin                  = "zerotier_join"
	SensitiveActionZeroTierLeave                 = "zerotier_leave"
	SensitiveActionCloudflareInstall             = "cloudflare_install"
	SensitiveActionCloudflareUninstall           = "cloudflare_uninstall"
	SensitiveActionCloudflareConfigure           = "cloudflare_configure"
	SensitiveActionCloudflareStart               = "cloudflare_start"
	SensitiveActionCloudflareStop                = "cloudflare_stop"
	SensitiveActionCloudflareRestart             = "cloudflare_restart"
	SensitiveActionFRPInstall                    = "frp_install"
	SensitiveActionFRPUninstall                  = "frp_uninstall"
	SensitiveActionFRPConfigure                  = "frp_configure"
	SensitiveActionFRPStart                      = "frp_start"
	SensitiveActionFRPStop                       = "frp_stop"
	SensitiveActionFRPRestart                    = "frp_restart"
	SensitiveActionTailscaleInstall              = "tailscale_install"
	SensitiveActionTailscaleLogin                = "tailscale_login"
	SensitiveActionTailscaleUp                   = "tailscale_up"
	SensitiveActionTailscaleDown                 = "tailscale_down"
	SensitiveActionTailscaleStart                = "tailscale_start"
	SensitiveActionTailscaleStop                 = "tailscale_stop"
	SensitiveActionTailscaleRestart              = "tailscale_restart"
	SensitiveActionTailscaleLogout               = "tailscale_logout"
	SensitiveActionTailscaleUninstall            = "tailscale_uninstall"
	SensitiveActionTailscaleSetExitNode          = "tailscale_set_exit_node"
	SensitiveActionTailscaleSetSubnetRoute       = "tailscale_set_subnet_route"
	SensitiveActionMCPShowKey                    = "mcp_show_key"
	SensitiveActionMCPCopyKey                    = "mcp_copy_key"
	SensitiveActionMCPRegenerateKey              = "mcp_regenerate_key"
	SensitiveActionMCPEnable                     = "mcp_enable"
	SensitiveActionMCPDisable                    = "mcp_disable"
	SensitiveActionTargetPower                   = "target_power"
	SensitiveActionNanoKVMRestart                = "nanokvm_restart"
	SensitiveActionDeleteImage                   = "delete_image"
	SensitiveActionOpenWebTerminal               = "open_web_terminal"
	SensitiveActionScreenTimelapseEnable         = "screen_timelapse_enable"
	SensitiveActionScreenTimelapsePlayback       = "screen_timelapse_playback"
	SensitiveActionSetScreenTimelapseRetention   = "set_screen_timelapse_retention"
	SensitiveActionTriggerScreenTimelapseCleanup = "trigger_screen_timelapse_cleanup"
	SensitiveActionUpdateSignatureVerification   = "update_signature_verification"
	sensitiveActionChallengeTTL                  = 5 * time.Minute
	mcpAuthorizationTTL                          = sensitiveActionChallengeTTL
	screenTimelapsePlaybackAuthorizationTTL      = sensitiveActionChallengeTTL
)

const sensitiveActionChangePassword = SensitiveActionChangePassword

var errInvalidSensitiveAction = errors.New("invalid sensitive action")

type sensitiveActionChallenge struct {
	username        string
	action          string
	createdAt       time.Time
	expiresAt       time.Time
	ip              string
	userAgentDigest string
	sessionKeyHash  string
	used            bool
}

const (
	sensitiveActionAuthorizationScopeMCP                     = "mcp"
	sensitiveActionAuthorizationScopeScreenTimelapsePlayback = "screen-timelapse-playback"
)

type sensitiveActionAuthorization struct {
	scope           string
	username        string
	expiresAt       time.Time
	ip              string
	userAgentDigest string
	sessionKeyHash  string
}

var sensitiveActionChallenges = struct {
	sync.Mutex
	items map[string]sensitiveActionChallenge
}{items: make(map[string]sensitiveActionChallenge)}

var sensitiveActionAuthorizations = struct {
	sync.Mutex
	items map[string]sensitiveActionAuthorization
}{items: make(map[string]sensitiveActionAuthorization)}

func createSensitiveActionChallenge(username, action, sessionKey, ip, userAgent string) (string, error) {
	username = strings.TrimSpace(username)
	action = strings.TrimSpace(action)
	if username == "" || !validSensitiveAction(action) {
		return "", errInvalidSensitiveAction
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	token := base64.RawURLEncoding.EncodeToString(buf)
	now := totpNow()

	sensitiveActionChallenges.Lock()
	defer sensitiveActionChallenges.Unlock()
	cleanupExpiredSensitiveActionChallengesLocked(now)
	sensitiveActionChallenges.items[sensitiveActionChallengeKey(token)] = sensitiveActionChallenge{
		username:        username,
		action:          action,
		createdAt:       now,
		expiresAt:       now.Add(sensitiveActionChallengeTTL),
		ip:              strings.TrimSpace(ip),
		userAgentDigest: userAgentHash(userAgent),
		sessionKeyHash:  sensitiveActionSessionKeyHash(sessionKey),
	}

	return token, nil
}

func consumeSensitiveActionChallenge(token, username, action, sessionKey, ip, userAgent string) bool {
	now := totpNow()
	key := sensitiveActionChallengeKey(token)
	username = strings.TrimSpace(username)
	action = strings.TrimSpace(action)

	sensitiveActionChallenges.Lock()
	defer sensitiveActionChallenges.Unlock()
	cleanupExpiredSensitiveActionChallengesLocked(now)

	challenge, ok := sensitiveActionChallenges.items[key]
	if !ok || challenge.used || !now.Before(challenge.expiresAt) {
		delete(sensitiveActionChallenges.items, key)
		return false
	}
	if challenge.username != username || challenge.action != action {
		return false
	}
	if challenge.ip != strings.TrimSpace(ip) || challenge.userAgentDigest != userAgentHash(userAgent) {
		return false
	}
	if challenge.sessionKeyHash != sensitiveActionSessionKeyHash(sessionKey) {
		return false
	}

	challenge.used = true
	sensitiveActionChallenges.items[key] = challenge
	delete(sensitiveActionChallenges.items, key)
	return true
}

func cleanupExpiredSensitiveActionChallenges(now time.Time) {
	sensitiveActionChallenges.Lock()
	defer sensitiveActionChallenges.Unlock()
	cleanupExpiredSensitiveActionChallengesLocked(now)
}

func cleanupExpiredSensitiveActionChallengesLocked(now time.Time) {
	for key, challenge := range sensitiveActionChallenges.items {
		if !now.Before(challenge.expiresAt) || challenge.used {
			delete(sensitiveActionChallenges.items, key)
		}
	}
}

func validSensitiveAction(action string) bool {
	if IsFRPSensitiveAction(action) || IsCloudflareSensitiveAction(action) || IsZeroTierSensitiveAction(action) || IsNetBirdSensitiveAction(action) {
		return true
	}
	switch strings.TrimSpace(action) {
	case SensitiveActionChangePassword,
		SensitiveActionRunScript,
		SensitiveActionUploadScript,
		SensitiveActionDeleteScript,
		SensitiveActionEnableSSH,
		SensitiveActionDisableSSH,
		SensitiveActionTailscaleInstall,
		SensitiveActionTailscaleLogin,
		SensitiveActionTailscaleUp,
		SensitiveActionTailscaleDown,
		SensitiveActionTailscaleStart,
		SensitiveActionTailscaleStop,
		SensitiveActionTailscaleRestart,
		SensitiveActionTailscaleLogout,
		SensitiveActionTailscaleUninstall,
		SensitiveActionTailscaleSetExitNode,
		SensitiveActionTailscaleSetSubnetRoute,
		SensitiveActionMCPShowKey,
		SensitiveActionMCPCopyKey,
		SensitiveActionMCPRegenerateKey,
		SensitiveActionMCPEnable,
		SensitiveActionMCPDisable,
		SensitiveActionTargetPower,
		SensitiveActionNanoKVMRestart,
		SensitiveActionDeleteImage,
		SensitiveActionOpenWebTerminal,
		SensitiveActionScreenTimelapseEnable,
		SensitiveActionScreenTimelapsePlayback,
		SensitiveActionSetScreenTimelapseRetention,
		SensitiveActionTriggerScreenTimelapseCleanup,
		SensitiveActionUpdateSignatureVerification:
		return true
	default:
		return false
	}
}

// isProtectionSensitiveAction reports whether action is gated by advanced 2FA
// protection. When TOTP is enabled these actions re-auth with code/recovery only
// (no login password). Register future protection-gated actions here.
func isProtectionSensitiveAction(action string) bool {
	if IsFRPSensitiveAction(action) || IsCloudflareSensitiveAction(action) || IsZeroTierSensitiveAction(action) || IsNetBirdSensitiveAction(action) {
		return true
	}
	switch strings.TrimSpace(action) {
	case SensitiveActionRunScript,
		SensitiveActionUploadScript,
		SensitiveActionDeleteScript,
		SensitiveActionEnableSSH,
		SensitiveActionDisableSSH,
		SensitiveActionTailscaleInstall,
		SensitiveActionTailscaleLogin,
		SensitiveActionTailscaleUp,
		SensitiveActionTailscaleDown,
		SensitiveActionTailscaleStart,
		SensitiveActionTailscaleStop,
		SensitiveActionTailscaleRestart,
		SensitiveActionTailscaleLogout,
		SensitiveActionTailscaleUninstall,
		SensitiveActionTailscaleSetExitNode,
		SensitiveActionTailscaleSetSubnetRoute,
		SensitiveActionMCPShowKey,
		SensitiveActionMCPCopyKey,
		SensitiveActionMCPRegenerateKey,
		SensitiveActionMCPEnable,
		SensitiveActionMCPDisable,
		SensitiveActionTargetPower,
		SensitiveActionNanoKVMRestart,
		SensitiveActionDeleteImage,
		SensitiveActionScreenTimelapseEnable,
		SensitiveActionScreenTimelapsePlayback,
		SensitiveActionSetScreenTimelapseRetention,
		SensitiveActionTriggerScreenTimelapseCleanup,
		SensitiveActionUpdateSignatureVerification:
		return true
	default:
		return false
	}
}

// IsTailscaleSensitiveAction reports whether action is one of the Tailscale
// control-plane ops gated by policy.Tailscale (all mutating page actions).
func IsTailscaleSensitiveAction(action string) bool {
	switch strings.TrimSpace(action) {
	case SensitiveActionTailscaleInstall,
		SensitiveActionTailscaleLogin,
		SensitiveActionTailscaleUp,
		SensitiveActionTailscaleDown,
		SensitiveActionTailscaleStart,
		SensitiveActionTailscaleStop,
		SensitiveActionTailscaleRestart,
		SensitiveActionTailscaleLogout,
		SensitiveActionTailscaleUninstall,
		SensitiveActionTailscaleSetExitNode,
		SensitiveActionTailscaleSetSubnetRoute:
		return true
	default:
		return false
	}
}

func IsFRPSensitiveAction(action string) bool {
	switch strings.TrimSpace(action) {
	case SensitiveActionFRPInstall, SensitiveActionFRPUninstall, SensitiveActionFRPConfigure,
		SensitiveActionFRPStart, SensitiveActionFRPStop, SensitiveActionFRPRestart:
		return true
	default:
		return false
	}
}

func IsCloudflareSensitiveAction(action string) bool {
	switch strings.TrimSpace(action) {
	case SensitiveActionCloudflareInstall, SensitiveActionCloudflareUninstall, SensitiveActionCloudflareConfigure,
		SensitiveActionCloudflareStart, SensitiveActionCloudflareStop, SensitiveActionCloudflareRestart:
		return true
	default:
		return false
	}
}

// IsMCPSensitiveAction reports whether action controls access to an MCP API key.
func IsMCPSensitiveAction(action string) bool {
	switch strings.TrimSpace(action) {
	case SensitiveActionMCPShowKey,
		SensitiveActionMCPCopyKey,
		SensitiveActionMCPRegenerateKey,
		SensitiveActionMCPEnable,
		SensitiveActionMCPDisable:
		return true
	default:
		return false
	}
}

// IsScreenTimelapseSensitiveAction reports whether action can persist a
// retention policy or schedule deletion of screen timelapse recordings.
func IsScreenTimelapseSensitiveAction(action string) bool {
	switch strings.TrimSpace(action) {
	case SensitiveActionSetScreenTimelapseRetention,
		SensitiveActionTriggerScreenTimelapseCleanup:
		return true
	default:
		return false
	}
}

// VerifyMCPChallenge accepts a matching one-time challenge or an existing
// MCP-only authorization grant bound to the current authenticated client.
func VerifyMCPChallenge(c *gin.Context, challenge, action string) error {
	if !IsMCPSensitiveAction(action) {
		return errInvalidSensitiveAction
	}

	username, sessionKey, err := sensitiveActionIdentity(c)
	if err != nil {
		return err
	}
	if hasMCPAuthorization(username, sessionKey, c.ClientIP(), c.Request.UserAgent()) {
		return nil
	}
	if strings.TrimSpace(challenge) == "" ||
		!consumeSensitiveActionChallenge(challenge, username, action, sessionKey, c.ClientIP(), c.Request.UserAgent()) {
		return errInvalidSensitiveAction
	}

	grantMCPAuthorization(username, sessionKey, c.ClientIP(), c.Request.UserAgent())
	return nil
}

// VerifyScreenTimelapsePlaybackChallenge accepts a matching one-time challenge or an
// existing Screen Timelapse playback authorization bound to this authenticated client.
func VerifyScreenTimelapsePlaybackChallenge(c *gin.Context, challenge string) error {
	if !ScreenTimelapsePlaybackRequiresChallenge() {
		return nil
	}

	username, sessionKey, err := sensitiveActionIdentity(c)
	if err != nil {
		return err
	}
	if hasScreenTimelapsePlaybackAuthorization(username, sessionKey, c.ClientIP(), c.Request.UserAgent()) {
		return nil
	}
	if strings.TrimSpace(challenge) == "" ||
		!consumeSensitiveActionChallenge(challenge, username, SensitiveActionScreenTimelapsePlayback, sessionKey, c.ClientIP(), c.Request.UserAgent()) {
		return errInvalidSensitiveAction
	}

	grantScreenTimelapsePlaybackAuthorization(username, sessionKey, c.ClientIP(), c.Request.UserAgent())
	return nil
}

func VerifySensitiveActionChallenge(c *gin.Context, challenge, action string) error {
	username, sessionKey, err := sensitiveActionIdentity(c)
	if err != nil {
		return err
	}
	if strings.TrimSpace(challenge) == "" {
		return errInvalidSensitiveAction
	}
	if !consumeSensitiveActionChallenge(challenge, username, action, sessionKey, c.ClientIP(), c.Request.UserAgent()) {
		return errInvalidSensitiveAction
	}

	return nil
}

func sensitiveActionIdentity(c *gin.Context) (string, string, error) {
	if c == nil || c.Request == nil {
		return "", "", errInvalidSensitiveAction
	}

	account, err := GetAccount()
	if err != nil || account == nil {
		return "", "", errors.New("get account failed")
	}

	username := strings.TrimSpace(account.Username)
	if jwtUsername := currentJWTUsername(c); jwtUsername != "" && jwtUsername != username {
		return "", "", errInvalidSensitiveAction
	}
	return username, sensitiveOperationSessionKey(c), nil
}

// ReadSensitiveChallenge extracts an optional sensitive-action challenge from the
// request. Prefers JSON/form body binding; falls back to form/query values.
func ReadSensitiveChallenge(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}

	var req struct {
		Challenge string `json:"challenge" form:"challenge"`
	}
	// Best-effort bind by Content-Type (JSON or form). Errors leave Challenge empty.
	_ = c.ShouldBind(&req)
	if ch := strings.TrimSpace(req.Challenge); ch != "" {
		return ch
	}
	if ch := strings.TrimSpace(c.PostForm("challenge")); ch != "" {
		return ch
	}
	return strings.TrimSpace(c.Query("challenge"))
}

func sensitiveActionChallengeKey(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func sensitiveActionSessionKeyHash(sessionKey string) string {
	return hashAuthRateLimitKey("sensitive-action-session:" + strings.TrimSpace(sessionKey))
}

func sensitiveActionAuthorizationKey(scope, sessionKey string) string {
	return hashAuthRateLimitKey("sensitive-action-authorization:" + scope + ":" + strings.TrimSpace(sessionKey))
}

func grantSensitiveActionAuthorization(scope, username, sessionKey, ip, userAgent string, ttl time.Duration) {
	now := totpNow()
	key := sensitiveActionAuthorizationKey(scope, sessionKey)

	sensitiveActionAuthorizations.Lock()
	defer sensitiveActionAuthorizations.Unlock()
	cleanupExpiredSensitiveActionAuthorizationsLocked(now)
	sensitiveActionAuthorizations.items[key] = sensitiveActionAuthorization{
		scope:           scope,
		username:        strings.TrimSpace(username),
		expiresAt:       now.Add(ttl),
		ip:              strings.TrimSpace(ip),
		userAgentDigest: userAgentHash(userAgent),
		sessionKeyHash:  sensitiveActionSessionKeyHash(sessionKey),
	}
}

func hasSensitiveActionAuthorization(scope, username, sessionKey, ip, userAgent string) bool {
	now := totpNow()
	key := sensitiveActionAuthorizationKey(scope, sessionKey)

	sensitiveActionAuthorizations.Lock()
	defer sensitiveActionAuthorizations.Unlock()
	cleanupExpiredSensitiveActionAuthorizationsLocked(now)

	authorization, ok := sensitiveActionAuthorizations.items[key]
	if !ok || !now.Before(authorization.expiresAt) {
		delete(sensitiveActionAuthorizations.items, key)
		return false
	}
	return authorization.scope == scope &&
		authorization.username == strings.TrimSpace(username) &&
		authorization.ip == strings.TrimSpace(ip) &&
		authorization.userAgentDigest == userAgentHash(userAgent) &&
		authorization.sessionKeyHash == sensitiveActionSessionKeyHash(sessionKey)
}

func sensitiveActionAuthorizationExpiresIn(scope, username, sessionKey, ip, userAgent string) int {
	now := totpNow()
	key := sensitiveActionAuthorizationKey(scope, sessionKey)

	sensitiveActionAuthorizations.Lock()
	defer sensitiveActionAuthorizations.Unlock()
	cleanupExpiredSensitiveActionAuthorizationsLocked(now)

	authorization, ok := sensitiveActionAuthorizations.items[key]
	if !ok || authorization.scope != scope || !now.Before(authorization.expiresAt) || authorization.username != strings.TrimSpace(username) ||
		authorization.ip != strings.TrimSpace(ip) || authorization.userAgentDigest != userAgentHash(userAgent) ||
		authorization.sessionKeyHash != sensitiveActionSessionKeyHash(sessionKey) {
		return 0
	}

	return int(authorization.expiresAt.Sub(now).Seconds())
}

func clearSensitiveActionAuthorizations(scope string) {
	sensitiveActionAuthorizations.Lock()
	defer sensitiveActionAuthorizations.Unlock()
	for key, authorization := range sensitiveActionAuthorizations.items {
		if authorization.scope == scope {
			delete(sensitiveActionAuthorizations.items, key)
		}
	}
}

func cleanupExpiredSensitiveActionAuthorizationsLocked(now time.Time) {
	for key, authorization := range sensitiveActionAuthorizations.items {
		if !now.Before(authorization.expiresAt) {
			delete(sensitiveActionAuthorizations.items, key)
		}
	}
}

func grantMCPAuthorization(username, sessionKey, ip, userAgent string) {
	grantSensitiveActionAuthorization(sensitiveActionAuthorizationScopeMCP, username, sessionKey, ip, userAgent, mcpAuthorizationTTL)
}

func hasMCPAuthorization(username, sessionKey, ip, userAgent string) bool {
	return hasSensitiveActionAuthorization(sensitiveActionAuthorizationScopeMCP, username, sessionKey, ip, userAgent)
}

func mcpAuthorizationExpiresIn(username, sessionKey, ip, userAgent string) int {
	return sensitiveActionAuthorizationExpiresIn(sensitiveActionAuthorizationScopeMCP, username, sessionKey, ip, userAgent)
}

func clearMCPAuthorizations() {
	clearSensitiveActionAuthorizations(sensitiveActionAuthorizationScopeMCP)
}

func resetMCPAuthorizationsForTest() {
	clearMCPAuthorizations()
}

func grantScreenTimelapsePlaybackAuthorization(username, sessionKey, ip, userAgent string) {
	grantSensitiveActionAuthorization(sensitiveActionAuthorizationScopeScreenTimelapsePlayback, username, sessionKey, ip, userAgent, screenTimelapsePlaybackAuthorizationTTL)
}

func hasScreenTimelapsePlaybackAuthorization(username, sessionKey, ip, userAgent string) bool {
	return hasSensitiveActionAuthorization(sensitiveActionAuthorizationScopeScreenTimelapsePlayback, username, sessionKey, ip, userAgent)
}

func screenTimelapsePlaybackAuthorizationExpiresIn(username, sessionKey, ip, userAgent string) int {
	return sensitiveActionAuthorizationExpiresIn(sensitiveActionAuthorizationScopeScreenTimelapsePlayback, username, sessionKey, ip, userAgent)
}

func clearScreenTimelapsePlaybackAuthorizations() {
	clearSensitiveActionAuthorizations(sensitiveActionAuthorizationScopeScreenTimelapsePlayback)
}

func resetScreenTimelapsePlaybackAuthorizationsForTest() {
	clearScreenTimelapsePlaybackAuthorizations()
}

func IsZeroTierSensitiveAction(action string) bool {
	switch strings.TrimSpace(action) {
	case SensitiveActionZeroTierInstall, SensitiveActionZeroTierUninstall, SensitiveActionZeroTierStart, SensitiveActionZeroTierStop, SensitiveActionZeroTierRestart, SensitiveActionZeroTierJoin, SensitiveActionZeroTierLeave:
		return true
	default:
		return false
	}
}

func IsNetBirdSensitiveAction(action string) bool {
	switch strings.TrimSpace(action) {
	case SensitiveActionNetBirdInstall, SensitiveActionNetBirdUninstall, SensitiveActionNetBirdStart, SensitiveActionNetBirdStop, SensitiveActionNetBirdRestart, SensitiveActionNetBirdLogin, SensitiveActionNetBirdLoginCancel, SensitiveActionNetBirdLogout:
		return true
	default:
		return false
	}
}
