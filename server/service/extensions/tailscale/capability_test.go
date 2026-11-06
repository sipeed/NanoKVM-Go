package tailscale

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/application"
)

func useTestTailscaleRegion(t *testing.T, region proto.TailscaleRegion) *tailscaleRegionStore {
	t.Helper()
	store := newTailscaleRegionStore(filepath.Join(t.TempDir(), "region"))
	if err := os.WriteFile(store.path, []byte(string(region)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return store
}

// useTestServerFeatureFlags replaces the manifest-read seam so tests never read
// /etc/kvm/server-feature.json, and restores the production reader on cleanup.
func useTestServerFeatureFlags(t *testing.T, enabled bool) {
	t.Helper()
	previous := loadServerFeatureFlags
	loadServerFeatureFlags = func() application.ServerFeatureFlags {
		return application.ServerFeatureFlags{TailscaleCnEnabled: enabled}
	}
	t.Cleanup(func() { loadServerFeatureFlags = previous })
}

func TestAvailableModesFollowManifestInEveryRegion(t *testing.T) {
	previousRegion := defaultTailscaleRegionStore
	t.Cleanup(func() { defaultTailscaleRegionStore = previousRegion })

	cases := []struct {
		name     string
		region   proto.TailscaleRegion
		manifest bool
		wantMode []proto.TailscaleMode
		want     bool
	}{
		{
			name:     "cn with manifest on offers both modes",
			region:   proto.TailscaleRegionCN,
			manifest: true,
			wantMode: []proto.TailscaleMode{proto.TailscaleModeOfficial, proto.TailscaleModeSipeed},
			want:     true,
		},
		{
			name:     "cn with manifest off offers official only",
			region:   proto.TailscaleRegionCN,
			manifest: false,
			wantMode: []proto.TailscaleMode{proto.TailscaleModeOfficial},
			want:     false,
		},
		{
			name:     "global with manifest on offers both modes",
			region:   proto.TailscaleRegionGlobal,
			manifest: true,
			wantMode: []proto.TailscaleMode{proto.TailscaleModeOfficial, proto.TailscaleModeSipeed},
			want:     true,
		},
		{
			name:     "global with manifest off offers official only",
			region:   proto.TailscaleRegionGlobal,
			manifest: false,
			wantMode: []proto.TailscaleMode{proto.TailscaleModeOfficial},
			want:     false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defaultTailscaleRegionStore = useTestTailscaleRegion(t, tc.region)
			useTestServerFeatureFlags(t, tc.manifest)

			if got := AvailableModes(); !reflect.DeepEqual(got, tc.wantMode) {
				t.Fatalf("AvailableModes() = %v, want %v", got, tc.wantMode)
			}
			if got := SipeedModeAvailable(); got != tc.want {
				t.Fatalf("SipeedModeAvailable() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAvailableModesForRegionComposition(t *testing.T) {
	if got, want := availableModesForRegion(proto.TailscaleRegionCN, true),
		[]proto.TailscaleMode{proto.TailscaleModeOfficial, proto.TailscaleModeSipeed}; !reflect.DeepEqual(got, want) {
		t.Fatalf("cn+on = %v, want %v", got, want)
	}
	if got, want := availableModesForRegion(proto.TailscaleRegionCN, false),
		[]proto.TailscaleMode{proto.TailscaleModeOfficial}; !reflect.DeepEqual(got, want) {
		t.Fatalf("cn+off = %v, want %v", got, want)
	}
	if !sipeedModeAllowed(proto.TailscaleRegionGlobal, true) {
		t.Fatal("global region blocked Sipeed despite an enabled manifest")
	}
	if sipeedModeAllowed(proto.TailscaleRegionGlobal, false) {
		t.Fatal("global region with manifest off exposed Sipeed")
	}
}
