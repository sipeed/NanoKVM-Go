package config

import "time"

const (
	DefaultJWTRefreshTokenDurationSeconds   uint64 = 2678400
	DefaultDownloadNoProgressTimeoutSeconds uint64 = 60

	// Keep configured second counts representable as both time.Duration and int
	// on every supported build architecture. This still permits lifetimes of more
	// than 68 years, so rejecting larger values does not constrain a practical
	// NanoKVM configuration.
	maxConfiguredDurationSeconds uint64 = 1<<31 - 1
)

func normalizeConfiguredDurationSeconds(value, fallback uint64) uint64 {
	if value == 0 || value > maxConfiguredDurationSeconds {
		return fallback
	}
	return value
}

func JWTRefreshTokenLifetime(value uint64) time.Duration {
	seconds := normalizeConfiguredDurationSeconds(value, DefaultJWTRefreshTokenDurationSeconds)
	// #nosec G115 -- normalizeConfiguredDurationSeconds bounds seconds to MaxInt32.
	return time.Duration(seconds) * time.Second
}

func JWTRefreshTokenMaxAge(value uint64) int {
	seconds := normalizeConfiguredDurationSeconds(value, DefaultJWTRefreshTokenDurationSeconds)
	// #nosec G115 -- normalizeConfiguredDurationSeconds bounds seconds to MaxInt32.
	return int(seconds)
}

func DownloadNoProgressTimeout(value uint64) time.Duration {
	seconds := normalizeConfiguredDurationSeconds(value, DefaultDownloadNoProgressTimeoutSeconds)
	// #nosec G115 -- normalizeConfiguredDurationSeconds bounds seconds to MaxInt32.
	return time.Duration(seconds) * time.Second
}
