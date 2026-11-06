package config

import "log"

var defaultConfig = &Config{
	Proto: "https",
	Port: Port{
		Http:  80,
		Https: 443,
	},
	Cert: Cert{
		Crt: "/etc/kvm/server.crt",
		Key: "/etc/kvm/server.key",
	},
	Logger: Logger{
		Level: "info",
		File:  "stdout",
	},
	FingerRobot: FingerRobotModeNormal,
	JWT: JWT{
		SecretKey:            "",
		RefreshTokenDuration: DefaultJWTRefreshTokenDurationSeconds,
		RevokeTokensOnLogout: true,
	},
	TwoFactorAuth: TwoFactorAuth{
		SecretFile: "/etc/kvm/totp.secret",
		Issuer:     "NanoKVM",
		Window:     1,
	},
	Security: Security{
		Download: DownloadSecurity{
			AllowPrivate:      false,
			AllowedHosts:      []string{},
			AllowedCIDRs:      []string{},
			NoProgressTimeout: DefaultDownloadNoProgressTimeoutSeconds,
			MaxImageBytes:     DefaultDownloadMaxImageBytes,
			ReservedFreeBytes: DefaultDownloadReservedFreeBytes,
		},
		Upload: UploadSecurity{
			MaxImageBytes:     DefaultUploadMaxImageBytes,
			ReservedFreeBytes: DefaultUploadReservedFreeBytes,
		},
		CSRF: CSRF{
			TrustedOrigins:     []string{},
			AllowMissingOrigin: false,
		},
		CORS: CORSSecurity{
			AllowedOrigins: []string{},
		},
		Redirect: RedirectSecurity{
			TrustedHosts: []string{},
		},
	},
	Stun: "stun.l.google.com:19302",
	Turn: Turn{
		TurnAddr: "",
		TurnUser: "",
		TurnCred: "",
	},
	HDMICapture: HDMICapture{
		Enabled: true,
	},
	Onboarding: Onboarding{
		CompletedVersion:       "",
		MobileCompletedVersion: "",
	},
	ServerFeatureSync: ServerFeatureSync{
		Enabled: true,
	},
	Authentication: "enable",
}

func checkDefaultValue() error {
	instance.FingerRobot = normalizeFingerRobotMode(instance.FingerRobot)

	if instance.JWT.SecretKey == "" {
		key, err := generateRandomSecretKey()
		if err != nil {
			return err
		}
		instance.JWT.SecretKey = key
		instance.JWT.RevokeTokensOnLogout = true
	}

	instance.JWT.RefreshTokenDuration = normalizeConfiguredDurationSeconds(
		instance.JWT.RefreshTokenDuration,
		DefaultJWTRefreshTokenDurationSeconds,
	)

	if instance.TwoFactorAuth.SecretFile == "" {
		instance.TwoFactorAuth.SecretFile = "/etc/kvm/totp.secret"
	}

	if instance.TwoFactorAuth.Issuer == "" {
		instance.TwoFactorAuth.Issuer = "NanoKVM"
	}

	if instance.TwoFactorAuth.Window < 0 || instance.TwoFactorAuth.Window > 2 {
		instance.TwoFactorAuth.Window = 1
	}

	if instance.Stun == "" {
		instance.Stun = "stun.l.google.com:19302"
	}

	if instance.Authentication == "" {
		instance.Authentication = "enable"
	}

	instance.Security.Download.NoProgressTimeout = normalizeConfiguredDurationSeconds(
		instance.Security.Download.NoProgressTimeout,
		DefaultDownloadNoProgressTimeoutSeconds,
	)
	instance.Security.Download.MaxImageBytes = normalizeDownloadStorageBytes(
		instance.Security.Download.MaxImageBytes,
		DefaultDownloadMaxImageBytes,
	)
	instance.Security.Download.ReservedFreeBytes = normalizeDownloadStorageBytes(
		instance.Security.Download.ReservedFreeBytes,
		DefaultDownloadReservedFreeBytes,
	)
	instance.Security.Upload.MaxImageBytes = normalizeDownloadStorageBytes(
		instance.Security.Upload.MaxImageBytes,
		DefaultUploadMaxImageBytes,
	)
	instance.Security.Upload.ReservedFreeBytes = normalizeDownloadStorageBytes(
		instance.Security.Upload.ReservedFreeBytes,
		DefaultUploadReservedFreeBytes,
	)

	if instance.Onboarding.CompletedVersion != "" && !IsValidOnboardingVersion(instance.Onboarding.CompletedVersion) {
		log.Printf("invalid onboarding completedVersion %q; resetting to empty", instance.Onboarding.CompletedVersion)
		instance.Onboarding.CompletedVersion = ""
	}

	if instance.Onboarding.MobileCompletedVersion != "" && !IsValidOnboardingVersion(instance.Onboarding.MobileCompletedVersion) {
		log.Printf("invalid onboarding mobileCompletedVersion %q; resetting to empty", instance.Onboarding.MobileCompletedVersion)
		instance.Onboarding.MobileCompletedVersion = ""
	}

	instance.Hardware = getHardware()
	return nil
}
