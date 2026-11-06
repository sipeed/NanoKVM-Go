package tailscale

import "testing"

func TestMapTailscaleArch(t *testing.T) {
	cases := map[string]string{
		"aarch64": "arm64",
		"arm64":   "arm64",
		"x86_64":  "amd64",
		"amd64":   "amd64",
		"armv7l":  "arm",
		"armv7":   "arm",
		"armhf":   "arm",
		"arm":     "arm",
		"riscv64": "riscv64",
		"":        "arm64",
	}
	for in, want := range cases {
		if got := mapTailscaleArch(in); got != want {
			t.Fatalf("mapTailscaleArch(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestGetTailscaleNameUsesPinnedVersion(t *testing.T) {
	// Ensure package name always matches the reviewed release rather than the
	// moving latest archive.
	name := GetTailscaleName()
	prefix := "tailscale_" + TailscaleVersion + "_"
	if len(name) < len(prefix)+len("arm.tgz") {
		t.Fatalf("unexpected package name: %s", name)
	}
	if name[:len(prefix)] != prefix {
		t.Fatalf("package name must start with %s, got %s", prefix, name)
	}
	if name[len(name)-4:] != ".tgz" {
		t.Fatalf("package name must end with .tgz, got %s", name)
	}
}
