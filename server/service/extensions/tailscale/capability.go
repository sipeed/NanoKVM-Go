package tailscale

import (
	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/application"
)

// loadServerFeatureFlags is the manifest-read seam. Production points at the
// real /etc/kvm/server-feature.json reader; tests override it and restore it
// with t.Cleanup so they never read device state.
var loadServerFeatureFlags = application.LoadServerFeatureFlags

func AvailableModes() []proto.TailscaleMode {
	return availableModesForRegion(
		defaultTailscaleRegionStore.load(),
		loadServerFeatureFlags().TailscaleCnEnabled,
	)
}

// The legacy region marker no longer restricts managed networking. The
// server-feature manifest remains authoritative in every region.
func sipeedModeAllowed(_ proto.TailscaleRegion, manifestEnabled bool) bool {
	return manifestEnabled
}

func availableModesForRegion(region proto.TailscaleRegion, manifestEnabled bool) []proto.TailscaleMode {
	modes := []proto.TailscaleMode{proto.TailscaleModeOfficial}
	if sipeedModeAllowed(region, manifestEnabled) {
		modes = append(modes, proto.TailscaleModeSipeed)
	}
	return modes
}

func SipeedModeAvailable() bool {
	return sipeedModeAllowed(
		defaultTailscaleRegionStore.load(),
		loadServerFeatureFlags().TailscaleCnEnabled,
	)
}
