package config

type Config struct {
	Proto               string              `yaml:"proto"`
	Port                Port                `yaml:"port"`
	Cert                Cert                `yaml:"cert"`
	Logger              Logger              `yaml:"logger"`
	Authentication      string              `yaml:"authentication"`
	FingerRobot         FingerRobotMode     `yaml:"fingerRobot"`
	JWT                 JWT                 `yaml:"jwt"`
	TwoFactorAuth       TwoFactorAuth       `yaml:"twoFactorAuth"`
	Security            Security            `yaml:"security"`
	Stun                string              `yaml:"stun"`
	Turn                Turn                `yaml:"turn"`
	HDMICapture         HDMICapture         `yaml:"hdmiCapture"`
	Onboarding          Onboarding          `yaml:"onboarding"`
	SipeedRemoteNetwork SipeedRemoteNetwork `yaml:"sipeedRemoteNetwork"`
	ServerFeatureSync   ServerFeatureSync   `yaml:"serverFeatureSync"`
	Update              Update              `yaml:"update"`
	Hardware            Hardware            `yaml:"-"`
}

type Update struct {
	// Server is the stable application update directory; empty uses the official server.
	Server string `yaml:"server"`
}

// ServerFeatureSync is a user-owned switch over the
// /etc/kvm/server-feature.json pull. Enabled defaults to true; when false a
// sync only repairs a genuinely missing manifest and never overwrites an
// existing valid, invalid, or explicitly disabled one. There is no setter:
// only a manual server.yaml edit, effective on process restart, may change it.
type ServerFeatureSync struct {
	Enabled bool `yaml:"enabled"`
}

// Onboarding records the device-level first-run guide state. CompletedVersion
// is the desktop tour version the user finished or dismissed and
// MobileCompletedVersion is the mobile portrait tour version; empty means the
// matching guide has never completed on this device.
type Onboarding struct {
	CompletedVersion       string `yaml:"completedVersion"`
	MobileCompletedVersion string `yaml:"mobileCompletedVersion"`
}

type SipeedRemoteNetwork struct {
	DisclaimerAccepted bool `yaml:"disclaimerAccepted"`
}

type FingerRobotMode string

const (
	FingerRobotModeNormal FingerRobotMode = "normal"
	FingerRobotModeDebug  FingerRobotMode = "debug"
)

type Logger struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

type Port struct {
	Http  int `yaml:"http"`
	Https int `yaml:"https"`
}

type Cert struct {
	Crt string `yaml:"crt"`
	Key string `yaml:"key"`
}

type JWT struct {
	SecretKey            string `yaml:"secretKey"`
	RefreshTokenDuration uint64 `yaml:"refreshTokenDuration"`
	RevokeTokensOnLogout bool   `yaml:"revokeTokensOnLogout"`
}

type TwoFactorAuth struct {
	SecretFile string `yaml:"secretFile"`
	Issuer     string `yaml:"issuer"`
	Window     int    `yaml:"window"`
}

type Security struct {
	Download DownloadSecurity `yaml:"download"`
	Upload   UploadSecurity   `yaml:"upload"`
	CSRF     CSRF             `yaml:"csrf"`
	CORS     CORSSecurity     `yaml:"cors"`
	Redirect RedirectSecurity `yaml:"redirect"`
}

type UploadSecurity struct {
	MaxImageBytes     uint64 `yaml:"maxImageBytes"`
	ReservedFreeBytes uint64 `yaml:"reservedFreeBytes"`
}

type DownloadSecurity struct {
	AllowPrivate      bool     `yaml:"allowPrivate"`
	AllowedHosts      []string `yaml:"allowedHosts"`
	AllowedCIDRs      []string `yaml:"allowedCIDRs"`
	NoProgressTimeout uint64   `yaml:"noProgressTimeout"`
	MaxImageBytes     uint64   `yaml:"maxImageBytes"`
	ReservedFreeBytes uint64   `yaml:"reservedFreeBytes"`
}

type CSRF struct {
	TrustedOrigins     []string `yaml:"trustedOrigins"`
	AllowMissingOrigin bool     `yaml:"allowMissingOrigin"`
}

type CORSSecurity struct {
	AllowedOrigins []string `yaml:"allowedOrigins"`
}

type RedirectSecurity struct {
	TrustedHosts []string `yaml:"trustedHosts"`
}

type Turn struct {
	TurnAddr string `yaml:"turnAddr"`
	TurnUser string `yaml:"turnUser"`
	TurnCred string `yaml:"turnCred"`
}

type HDMICapture struct {
	Enabled bool `yaml:"enabled"`
}

type Hardware struct {
	Version HWVersion `yaml:"-"`
}
