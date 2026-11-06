package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func useTempConfigurationFile(t *testing.T) string {
	t.Helper()

	originalPath := configurationFilePath
	path := filepath.Join(t.TempDir(), "server.yaml")
	configurationFilePath = path
	viper.Reset()
	t.Cleanup(func() {
		configurationFilePath = originalPath
		viper.Reset()
	})

	return path
}

func requireConfigurationFileMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat configuration file: %v", err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("configuration file mode = %04o, want %04o", got, want)
	}
}

func TestCreateConfigurationFileUses0600(t *testing.T) {
	path := useTempConfigurationFile(t)

	create()

	requireConfigurationFileMode(t, path, 0o600)
}

func TestSaveConfigurationFileUses0600(t *testing.T) {
	path := useTempConfigurationFile(t)
	if err := os.WriteFile(path, []byte("legacy"), 0o644); err != nil {
		t.Fatalf("write legacy configuration: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod legacy configuration: %v", err)
	}

	if err := save(*defaultConfig); err != nil {
		t.Fatalf("save configuration: %v", err)
	}

	requireConfigurationFileMode(t, path, 0o600)
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".server.yaml.*"))
	if err != nil {
		t.Fatalf("glob temporary files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("atomic configuration write left temporary files: %#v", matches)
	}
}

func TestReadConfigurationFileRepairsOverlyPermissiveMode(t *testing.T) {
	path := useTempConfigurationFile(t)
	content := []byte("port:\n  http: 80\n  https: 443\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write legacy configuration: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod legacy configuration: %v", err)
	}

	if err := readByFile(); err != nil {
		t.Fatalf("read configuration: %v", err)
	}

	requireConfigurationFileMode(t, path, 0o600)
}

func TestReadConfigurationFilePreservesMissingFileClassification(t *testing.T) {
	useTempConfigurationFile(t)

	err := readByFile()
	var notFoundErr viper.ConfigFileNotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Fatalf("read configuration error = %v, want Viper missing-file classification", err)
	}
}

func TestReadConfigurationFileClassifiesPermissionRepairFailure(t *testing.T) {
	path := useTempConfigurationFile(t)
	content := []byte("port:\n  http: 80\n  https: 443\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write legacy configuration: %v", err)
	}

	originalChmod := chmodConfigurationFile
	chmodConfigurationFile = func(*os.File, os.FileMode) error {
		return os.ErrPermission
	}
	t.Cleanup(func() {
		chmodConfigurationFile = originalChmod
	})

	err := readByFile()
	var permissionErr *configurationFilePermissionError
	if !errors.As(err, &permissionErr) {
		t.Fatalf("read configuration error = %v, want permission classification", err)
	}
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("read configuration error = %v, want wrapped permission error", err)
	}
	var notFoundErr viper.ConfigFileNotFoundError
	if errors.As(err, &notFoundErr) {
		t.Fatalf("permission failure was misclassified as missing configuration: %v", err)
	}
}

func TestReadConfigurationFileRejectsSymlinkWithoutChangingTarget(t *testing.T) {
	path := useTempConfigurationFile(t)
	target := filepath.Join(t.TempDir(), "outside.yaml")
	content := []byte("port:\n  http: 80\n  https: 443\n")
	if err := os.WriteFile(target, content, 0o644); err != nil {
		t.Fatalf("write symlink target: %v", err)
	}
	if err := os.Chmod(target, 0o644); err != nil {
		t.Fatalf("chmod symlink target: %v", err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatalf("create configuration symlink: %v", err)
	}

	err := readByFile()
	var permissionErr *configurationFilePermissionError
	if !errors.As(err, &permissionErr) {
		t.Fatalf("read configuration error = %v, want permission classification", err)
	}
	requireConfigurationFileMode(t, target, 0o644)
}
