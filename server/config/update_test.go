package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestNormalizeUpdateServer(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{"", ""},
		{" \t\n", ""},
		{" https://updates.example.com/releases/// ", "https://updates.example.com/releases"},
		{"http://192.168.1.20:8080/kvm", "http://192.168.1.20:8080/kvm"},
		{"http://[::1]:8080/", "http://[::1]:8080"},
		{"https://updates.example.com/releases%20test", "https://updates.example.com/releases%20test"},
	} {
		t.Run(test.input, func(t *testing.T) {
			got, err := NormalizeUpdateServer(test.input)
			if err != nil || got != test.want {
				t.Fatalf("NormalizeUpdateServer(%q) = %q, %v; want %q", test.input, got, err, test.want)
			}
		})
	}
	for _, input := range []string{
		"updates.example.com/releases", "/releases", "//updates.example.com", "file:///tmp/updates",
		"ftp://updates.example.com", "https:///releases", "https://", "https://:8080",
		"https://user:password@updates.example.com", "https://updates.example.com?token=secret",
		"https://updates.example.com?", "https://updates.example.com#", "https://updates.example.com/#fragment",
		"https://updates.example.com:invalid", "https://updates.example.com:0", "https://updates.example.com:65536",
		"https://updates.example.com:", "https://updates.example.com\\releases",
		"https://updates.example.com/\nreleases", "https://updates.example.com/%invalid",
		"https://updates.example.com/" + strings.Repeat("a", 2048),
	} {
		t.Run(input, func(t *testing.T) {
			if _, err := NormalizeUpdateServer(input); err == nil {
				t.Fatalf("accepted invalid update server %q", input)
			}
		})
	}
}

func prepareUpdateConfiguration(t *testing.T) string {
	t.Helper()
	path := useTempConfigurationFile(t)
	if err := save(*defaultConfig); err != nil {
		t.Fatal(err)
	}
	GetInstance()
	original := instance
	instance = *defaultConfig
	instance.JWT.SecretKey = "preserved-secret"
	t.Cleanup(func() { instance = original })
	return path
}

func TestUpdateServerPersistsAndReloads(t *testing.T) {
	path := prepareUpdateConfiguration(t)
	if GetUpdateServer() != "" {
		t.Fatal("existing devices must default to the official server")
	}
	for _, server := range []string{"http://192.168.1.20:8080/updates", ""} {
		if err := SetUpdateServer(" " + server + " "); err != nil {
			t.Fatal(err)
		}
		reader := viper.New()
		reader.SetConfigFile(path)
		if err := reader.ReadInConfig(); err != nil {
			t.Fatal(err)
		}
		var reloaded Config
		if err := reader.Unmarshal(&reloaded); err != nil {
			t.Fatal(err)
		}
		if reloaded.Update.Server != server || GetUpdateServer() != server {
			t.Fatalf("update server was not persisted and published: %+v", reloaded.Update)
		}
		if reloaded.JWT.SecretKey != "preserved-secret" || !reloaded.HDMICapture.Enabled {
			t.Fatal("saving update server changed unrelated settings")
		}
		requireConfigurationFileMode(t, path, 0o600)
	}
}

func TestUpdateServerFailurePreservesConfiguration(t *testing.T) {
	path := prepareUpdateConfiguration(t)
	const previous = "https://updates.example.com/releases"
	if err := SetUpdateServer(previous); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := SetUpdateServer("file:///tmp/releases"); err == nil {
		t.Fatal("expected URL validation failure")
	}
	configurationFilePath = filepath.Join(path, "invalid.yaml")
	if err := SetUpdateServer(""); err == nil {
		t.Fatal("expected persistence failure")
	}
	if GetUpdateServer() != previous {
		t.Fatal("failed write changed active update server")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(before) {
		t.Fatal("failed write changed persisted configuration")
	}
}
