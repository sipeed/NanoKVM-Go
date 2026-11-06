package vm

import (
	"testing"

	"NanoKVM-Server/common"
)

func TestLowPowerEnabledUsesIdleTimeout(t *testing.T) {
	tests := []struct {
		name          string
		idleTimeoutMS int
		want          bool
	}{
		{
			name:          "disabled when idle timeout is zero",
			idleTimeoutMS: 0,
			want:          false,
		},
		{
			name:          "enabled when idle timeout is positive",
			idleTimeoutMS: 3000,
			want:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &common.KvmCommConfig{
				Device: common.KvmCommDeviceConfig{
					IdleTimeoutMS: tt.idleTimeoutMS,
				},
			}

			if got := lowPowerEnabled(config); got != tt.want {
				t.Fatalf("expected %t, got %t", tt.want, got)
			}
		})
	}
}

func TestLowPowerEnabledNilConfig(t *testing.T) {
	if lowPowerEnabled(nil) {
		t.Fatalf("expected nil config to be disabled")
	}
}
