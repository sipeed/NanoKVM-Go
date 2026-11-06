package cloudflare

import (
	"errors"
	"os"
	"testing"

	authservice "NanoKVM-Server/service/auth"

	"github.com/gin-gonic/gin"
)

func TestClearTokenStopsBeforeDeletingAndKeepsAccessURL(t *testing.T) {
	s := testService(t)
	configure(t, s, "saved")
	metadata, err := os.ReadFile(s.paths.config)
	if err != nil {
		t.Fatal(err)
	}
	stops := 0
	s.system.stop = func() error {
		stops++
		if _, err := os.Stat(s.paths.token); err != nil {
			t.Fatal("token deleted before service stopped")
		}
		return nil
	}
	if rsp, _ := request(t, s.ClearToken, nil); rsp.Code != 0 {
		t.Fatal(rsp)
	}
	if stops != 1 {
		t.Fatal("service was not stopped")
	}
	if _, err := os.Stat(s.paths.token); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("token file still exists")
	}
	after, _ := os.ReadFile(s.paths.config)
	if string(after) != string(metadata) {
		t.Fatal("metadata was changed")
	}
	if cfg, err := readConfig(s.paths); err != nil || cfg.Token != "" || cfg.AccessURL != "https://kvm.example.com" {
		t.Fatal("incorrect cleared state", err)
	}
	s.system.stop = func() error { return nil }
	if rsp, _ := request(t, s.ClearToken, nil); rsp.Code != 0 {
		t.Fatal("clear must be idempotent", rsp)
	}
}

func TestClearTokenHonorsProtectionAndStopFailure(t *testing.T) {
	s := testService(t)
	configure(t, s, "saved")
	s.requiresChallenge = func(action string) bool {
		if action != authservice.SensitiveActionCloudflareConfigure {
			t.Fatal("wrong credential protection action")
		}
		return true
	}
	s.verifyChallenge = func(*gin.Context, string, string) error { return errors.New("denied") }
	stops := 0
	s.system.stop = func() error { stops++; return errors.New("failed") }
	if rsp, _ := request(t, s.ClearToken, nil); rsp.Code != -3 || stops != 0 {
		t.Fatal("unauthorized clear reached service")
	}
	s.requiresChallenge = func(string) bool { return false }
	if rsp, _ := request(t, s.ClearToken, nil); rsp.Code == 0 {
		t.Fatal("stop failure ignored")
	}
	if _, err := os.Stat(s.paths.token); err != nil {
		t.Fatal("stop failure removed token")
	}
	if err := os.Remove(s.paths.unit); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(s.paths.binary); err != nil {
		t.Fatal(err)
	}
	if rsp, _ := request(t, s.ClearToken, nil); rsp.Code != 0 {
		t.Fatal("cannot clear after uninstall", rsp)
	}
	if _, err := os.Stat(s.paths.token); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("saved token survived clear")
	}
}
