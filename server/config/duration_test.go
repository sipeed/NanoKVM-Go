package config

import (
	"testing"
	"time"
)

func TestJWTRefreshTokenDurationBounds(t *testing.T) {
	tests := []struct {
		name     string
		value    uint64
		lifetime time.Duration
		maxAge   int
	}{
		{name: "configured", value: 3600, lifetime: time.Hour, maxAge: 3600},
		{name: "zero uses default", value: 0, lifetime: time.Duration(DefaultJWTRefreshTokenDurationSeconds) * time.Second, maxAge: int(DefaultJWTRefreshTokenDurationSeconds)},
		{name: "overflow uses default", value: maxConfiguredDurationSeconds + 1, lifetime: time.Duration(DefaultJWTRefreshTokenDurationSeconds) * time.Second, maxAge: int(DefaultJWTRefreshTokenDurationSeconds)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := JWTRefreshTokenLifetime(test.value); got != test.lifetime {
				t.Fatalf("JWTRefreshTokenLifetime(%d) = %s, want %s", test.value, got, test.lifetime)
			}
			if got := JWTRefreshTokenMaxAge(test.value); got != test.maxAge {
				t.Fatalf("JWTRefreshTokenMaxAge(%d) = %d, want %d", test.value, got, test.maxAge)
			}
		})
	}
}

func TestDownloadNoProgressTimeoutBounds(t *testing.T) {
	if got := DownloadNoProgressTimeout(15); got != 15*time.Second {
		t.Fatalf("DownloadNoProgressTimeout(15) = %s, want 15s", got)
	}
	wantDefault := time.Duration(DefaultDownloadNoProgressTimeoutSeconds) * time.Second
	for _, value := range []uint64{0, maxConfiguredDurationSeconds + 1} {
		if got := DownloadNoProgressTimeout(value); got != wantDefault {
			t.Fatalf("DownloadNoProgressTimeout(%d) = %s, want %s", value, got, wantDefault)
		}
	}
}
