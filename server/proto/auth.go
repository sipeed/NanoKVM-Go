package proto

import "errors"

type LoginReq struct {
	Username  string `json:"username,omitempty" form:"username"`
	Password  string `json:"password,omitempty" form:"password"`
	Challenge string `json:"challenge,omitempty" form:"challenge"`
	Code      string `json:"code,omitempty" form:"code"`
}

type LoginRsp struct {
	Token            string `json:"token,omitempty"`
	Requires2FA      bool   `json:"requires2FA,omitempty"`
	Challenge        string `json:"challenge,omitempty"`
	RecoveryCodeUsed bool   `json:"recoveryCodeUsed,omitempty"`
}

type DirectTwoFactorLoginStatusRsp struct {
	Enabled bool `json:"enabled"`
}

type AuthRateLimitRsp struct {
	RateLimited bool  `json:"rateLimited"`
	RetryAfter  int   `json:"retryAfter"`
	RetryAt     int64 `json:"retryAt"`
}

func ValidateLoginReqShape(req LoginReq) error {
	passwordLogin := req.Username != "" || req.Password != ""
	challengeLogin := req.Challenge != "" || req.Code != ""

	if passwordLogin == challengeLogin {
		return errors.New("login request must contain either username/password or challenge/code")
	}
	if passwordLogin && (req.Username == "" || req.Password == "") {
		return errors.New("username and password are required")
	}

	return nil
}

type GetAccountRsp struct {
	Username string `json:"username"`
}

type ChangePasswordReq struct {
	Username  string `validate:"required"`
	Password  string `validate:"required"`
	Challenge string `json:"challenge,omitempty" form:"challenge"`
}

type VerifySensitiveActionReq struct {
	Action   string `validate:"required" json:"action" form:"action"`
	Password string `json:"password" form:"password"` // required for non-protection actions; optional for protection code-only
	Code     string `json:"code" form:"code"`
}

type VerifySensitiveActionRsp struct {
	Challenge string `json:"challenge"`
	ExpiresIn int    `json:"expiresIn"`
}

// SensitiveChallengeReq carries a sensitive-action challenge for an endpoint
// that performs the authorized operation.
type SensitiveChallengeReq struct {
	Challenge string `json:"challenge,omitempty" form:"challenge"`
}

type IsPasswordUpdatedRsp struct {
	IsUpdated bool `json:"isUpdated"`
}

type TwoFactorStatusRsp struct {
	Enabled           bool `json:"enabled"`
	RecoveryEnabled   bool `json:"recoveryEnabled"`
	RecoveryRemaining int  `json:"recoveryRemaining"`
}

type TwoFactorSetupReq struct {
	Password string `validate:"required" json:"password" form:"password"`
}

type TwoFactorSetupRsp struct {
	URI    string `json:"uri"`
	Secret string `json:"secret"`
}

type TwoFactorEnableReq struct {
	Code string `validate:"required" json:"code" form:"code"`
}

type TwoFactorDisableReq struct {
	Password string `validate:"required" json:"password" form:"password"`
	Code     string `validate:"required" json:"code" form:"code"`
}

type TwoFactorRecoveryReq struct {
	Password string `validate:"required" json:"password" form:"password"`
	Code     string `validate:"required" json:"code" form:"code"`
}

type TwoFactorRecoveryRsp struct {
	RecoveryCodes []string `json:"recoveryCodes"`
}

type TwoFactorProtectionPolicyRsp struct {
	Cloudflare      bool `json:"cloudflare"`
	ZeroTier        bool `json:"zerotier"`
	NetBird         bool `json:"netbird"`
	FRP             bool `json:"frp"`
	Login           bool `json:"login"`
	WebTerminal     bool `json:"webTerminal"`
	SSH             bool `json:"ssh"`
	Tailscale       bool `json:"tailscale"`
	MCP             bool `json:"mcp"`
	TargetPower     bool `json:"targetPower"`
	NanoKVMRestart  bool `json:"nanoKVMRestart"`
	Scripts         bool `json:"scripts"`
	ScreenTimelapse bool `json:"screenTimelapse"`
	DeleteImage     bool `json:"deleteImage"`
}

type GetTwoFactorProtectionRsp struct {
	Policy                                TwoFactorProtectionPolicyRsp `json:"policy"`
	AuthorizationExpiresIn                int                          `json:"authorizationExpiresIn"`
	MCPAuthorizationExpiresIn             int                          `json:"mcpAuthorizationExpiresIn"`
	ScreenTimelapseAuthorizationExpiresIn int                          `json:"screenTimelapseAuthorizationExpiresIn"`
}

type AuthorizeTwoFactorProtectionReq struct {
	// Password is ignored; kept optional for backward-compatible clients.
	Password string `json:"password" form:"password"`
	Code     string `validate:"required" json:"code" form:"code"`
}

type AuthorizeTwoFactorProtectionRsp struct {
	ExpiresIn int `json:"expiresIn"`
}

type SetTwoFactorProtectionReq struct {
	Cloudflare      *bool `json:"cloudflare" form:"cloudflare"`
	ZeroTier        *bool `json:"zerotier" form:"zerotier"`
	NetBird         *bool `json:"netbird" form:"netbird"`
	FRP             *bool `json:"frp" form:"frp"`
	Login           *bool `validate:"required" json:"login" form:"login"`
	WebTerminal     *bool `validate:"required" json:"webTerminal" form:"webTerminal"`
	SSH             *bool `validate:"required" json:"ssh" form:"ssh"`
	Tailscale       *bool `validate:"required" json:"tailscale" form:"tailscale"`
	MCP             *bool `validate:"required" json:"mcp" form:"mcp"`
	TargetPower     *bool `validate:"required" json:"targetPower" form:"targetPower"`
	NanoKVMRestart  *bool `validate:"required" json:"nanoKVMRestart" form:"nanoKVMRestart"`
	Scripts         *bool `validate:"required" json:"scripts" form:"scripts"`
	ScreenTimelapse *bool `validate:"required" json:"screenTimelapse" form:"screenTimelapse"`
	DeleteImage     *bool `validate:"required" json:"deleteImage" form:"deleteImage"`
}

type ConnectWifiReq struct {
	Ssid     string `validate:"required" json:"ssid"`
	Password string `validate:"omitempty" json:"password"`
}
