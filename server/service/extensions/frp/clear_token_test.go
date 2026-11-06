package frp

import (
	"errors"
	"os"
	"testing"

	authservice "NanoKVM-Server/service/auth"

	"github.com/gin-gonic/gin"
)

func TestClearTokenStopsBeforeClearingAndPreservesOtherSettings(t *testing.T) {
	s := testService(t)
	if rsp, _ := request(t, s.SetConfig, validForm()); rsp.Code != 0 {
		t.Fatal(rsp)
	}
	previous, err := readConfig(s.paths.config)
	if err != nil {
		t.Fatal(err)
	}
	stops := 0
	s.system.stop = func() error {
		stops++
		cfg, err := readConfig(s.paths.config)
		if err != nil || cfg.Auth.Token != previous.Auth.Token {
			t.Fatal("credential changed before stop completed", err)
		}
		return nil
	}
	if rsp, _ := request(t, s.ClearToken, nil); rsp.Code != 0 {
		t.Fatal(rsp)
	}
	next, err := readConfig(s.paths.config)
	if err != nil {
		t.Fatal(err)
	}
	if stops != 1 || next.Auth.Token != "" || next.ServerAddr != previous.ServerAddr || next.ServerPort != previous.ServerPort ||
		next.Proxies[0] != previous.Proxies[0] || next.WebServer != previous.WebServer {
		t.Fatal("clearing token changed unrelated settings")
	}
	s.system.stop = func() error { return nil }
	if rsp, _ := request(t, s.ClearToken, nil); rsp.Code != 0 {
		t.Fatal("clear must be idempotent", rsp)
	}
}

func TestClearTokenHonorsProtectionAndStopFailure(t *testing.T) {
	s := testService(t)
	if rsp, _ := request(t, s.SetConfig, validForm()); rsp.Code != 0 {
		t.Fatal(rsp)
	}
	s.requiresChallenge = func(action string) bool {
		if action != authservice.SensitiveActionFRPConfigure {
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
	if cfg, err := readConfig(s.paths.config); err != nil || cfg.Auth.Token == "" {
		t.Fatal("stop failure cleared credential", err)
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
	if cfg, err := readConfig(s.paths.config); err != nil || cfg.Auth.Token != "" {
		t.Fatal("saved token survived clear", err)
	}
}
