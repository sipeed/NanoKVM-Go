package tailscale

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNewInstallStagingCreatesPrivateFiles(t *testing.T) {
	parent := t.TempDir()
	staging, err := newInstallStaging(parent, "tailscale_latest_arm64.tgz")
	if err != nil {
		t.Fatalf("newInstallStaging() error = %v", err)
	}

	if filepath.Dir(staging.dir) != parent {
		t.Fatalf("staging directory = %q, want child of %q", staging.dir, parent)
	}
	dirInfo, err := os.Lstat(staging.dir)
	if err != nil {
		t.Fatalf("stat staging directory: %v", err)
	}
	if !dirInfo.IsDir() || dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("staging directory mode = %v, want drwx------", dirInfo.Mode())
	}

	archiveInfo, err := os.Lstat(staging.archive)
	if err != nil {
		t.Fatalf("stat staging archive: %v", err)
	}
	if !archiveInfo.Mode().IsRegular() || archiveInfo.Mode().Perm() != 0o600 {
		t.Fatalf("staging archive mode = %v, want -rw-------", archiveInfo.Mode())
	}

	staging.cleanup()
	staging.cleanup()
	if _, err := os.Lstat(staging.dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("staging directory remains after cleanup: %v", err)
	}
}

func TestNewInstallStagingRejectsNestedPackageName(t *testing.T) {
	if staging, err := newInstallStaging(t.TempDir(), "../tailscale.tgz"); err == nil || staging != nil {
		t.Fatalf("newInstallStaging() = %#v, %v; want invalid package name", staging, err)
	}
}

func TestValidateInstallPayloadRequiresRegularFiles(t *testing.T) {
	extracted := t.TempDir()
	if err := os.MkdirAll(filepath.Join(extracted, "systemd"), 0o755); err != nil {
		t.Fatalf("create systemd directory: %v", err)
	}
	for path, content := range map[string]string{
		filepath.Join(extracted, "tailscale"):           "client",
		filepath.Join(extracted, "tailscaled"):          "daemon",
		filepath.Join(extracted, "systemd", Tailscaled): "unit",
	} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	payload, err := validateInstallPayload(extracted)
	if err != nil {
		t.Fatalf("validateInstallPayload() error = %v", err)
	}
	if payload.tailscale != filepath.Join(extracted, "tailscale") ||
		payload.tailscaled != filepath.Join(extracted, "tailscaled") ||
		payload.service != filepath.Join(extracted, "systemd", Tailscaled) {
		t.Fatalf("unexpected payload paths: %+v", payload)
	}

	if err := os.Remove(payload.service); err != nil {
		t.Fatalf("remove service file: %v", err)
	}
	if err := os.Mkdir(payload.service, 0o700); err != nil {
		t.Fatalf("replace service with directory: %v", err)
	}
	if _, err := validateInstallPayload(extracted); err == nil {
		t.Fatal("validateInstallPayload accepted a non-regular service file")
	}
}
