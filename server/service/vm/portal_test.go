package vm

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

type portalTestResponse struct {
	Code int                       `json:"code"`
	Data *proto.GetPortalConfigRsp `json:"data"`
}

func TestMissingOrMalformedPortalConfigUsesAutomaticDefault(t *testing.T) {
	setupPortalTest(t)

	for _, test := range []struct {
		name string
		read func(string) ([]byte, error)
	}{
		{name: "missing", read: func(string) ([]byte, error) { return nil, os.ErrNotExist }},
		{name: "malformed", read: func(string) ([]byte, error) { return []byte(`{`), nil }},
		{name: "invalid", read: func(string) ([]byte, error) { return []byte(`{"version":1,"mode":"broken"}`), nil }},
	} {
		t.Run(test.name, func(t *testing.T) {
			portalReadFile = test.read
			config, configured, err := readPortalConfig()
			if err != nil {
				t.Fatalf("read portal config: %v", err)
			}
			if configured || config.Mode != "auto" || config.Version != portalConfigVersion {
				t.Fatalf("fallback config = %+v configured=%v", config, configured)
			}
			if config.ManualRegions == nil || config.Resolutions == nil {
				t.Fatalf("fallback collections must be arrays: %+v", config)
			}
		})
	}
}

func TestValidPortalConfigRoundTrip(t *testing.T) {
	setupPortalTest(t)
	input := validPortalTestConfig()
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal portal config: %v", err)
	}
	portalReadFile = func(path string) ([]byte, error) {
		if path != PortalConfigFile {
			t.Fatalf("portal read path = %q", path)
		}
		return data, nil
	}

	config, configured, err := readPortalConfig()
	if err != nil || !configured {
		t.Fatalf("read portal config configured=%v err=%v", configured, err)
	}
	if config.SelectedRegionID != "desktop-1" || len(config.ManualRegions) != 1 {
		t.Fatalf("round-trip config = %+v", config)
	}
}

func TestPortalConfigValidationRejectsInvalidSelectionsAndBounds(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*proto.PortalConfig)
	}{
		{name: "mode", mutate: func(config *proto.PortalConfig) { config.Mode = "broken" }},
		{name: "region id", mutate: func(config *proto.PortalConfig) { config.ManualRegions[0].ID = "bad id" }},
		{name: "duplicate region", mutate: func(config *proto.PortalConfig) {
			config.ManualRegions = append(config.ManualRegions, config.ManualRegions[0])
		}},
		{name: "outside source", mutate: func(config *proto.PortalConfig) { config.ManualRegions[0].ContentRect.Left = 1000 }},
		{name: "duplicate resolution", mutate: func(config *proto.PortalConfig) {
			config.Resolutions = append(config.Resolutions, config.Resolutions[0])
		}},
		{name: "missing region", mutate: func(config *proto.PortalConfig) { config.SelectedRegionID = "missing" }},
		{name: "missing resolution", mutate: func(config *proto.PortalConfig) { config.SelectedRegionID = ""; config.SelectedResolution = "800x600" }},
		{name: "two selections", mutate: func(config *proto.PortalConfig) { config.SelectedResolution = "1280x720" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := validPortalTestConfig()
			test.mutate(&config)
			if err := validatePortalConfig(&config); err == nil {
				t.Fatalf("invalid config accepted: %+v", config)
			}
		})
	}
}

func TestSetPortalConfigNormalizesAndPersistsCollections(t *testing.T) {
	setupPortalTest(t)
	var persisted proto.PortalConfig
	portalWriteFile = func(path string, data []byte, mode os.FileMode) error {
		if path != PortalConfigFile || mode != 0o644 {
			t.Fatalf("portal write path=%q mode=%#o", path, mode)
		}
		return json.Unmarshal(data, &persisted)
	}

	rsp := performSetPortalBody(t, []byte(`{"version":1,"mode":"auto"}`))
	if rsp.Code != 0 {
		t.Fatalf("set portal response code = %d", rsp.Code)
	}
	if persisted.ManualRegions == nil || persisted.Resolutions == nil {
		t.Fatalf("persisted collections must be arrays: %+v", persisted)
	}
}

func TestGetPortalConfigReportsUnconfiguredDefault(t *testing.T) {
	setupPortalTest(t)
	portalReadFile = func(string) ([]byte, error) { return nil, os.ErrNotExist }

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/vm/portal", nil)
	NewService().GetPortalConfig(ctx)

	var rsp portalTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("unmarshal portal response: %v body=%s", err, recorder.Body.String())
	}
	if rsp.Code != 0 || rsp.Data == nil || rsp.Data.Configured || rsp.Data.Mode != "auto" {
		t.Fatalf("portal GET response = %+v", rsp)
	}
}

func validPortalTestConfig() proto.PortalConfig {
	return proto.PortalConfig{
		Version: portalConfigVersion,
		Mode:    "manual",
		ManualRegions: []proto.PortalManualRegion{{
			ID:           "desktop-1",
			SourceWidth:  1920,
			SourceHeight: 1080,
			ContentRect:  proto.PortalContentRect{Left: 240, Top: 0, Width: 1440, Height: 1080},
		}},
		SelectedRegionID: "desktop-1",
		Resolutions:      []proto.PortalResolution{{Width: 1280, Height: 720}},
	}
}

func setupPortalTest(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	originalReadFile := portalReadFile
	originalWriteFile := portalWriteFile
	t.Cleanup(func() {
		portalReadFile = originalReadFile
		portalWriteFile = originalWriteFile
	})
}

func performSetPortalBody(t *testing.T, body []byte) portalTestResponse {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/vm/portal", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	NewService().SetPortalConfig(ctx)

	var rsp portalTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("unmarshal portal response: %v body=%s", err, recorder.Body.String())
	}
	return rsp
}
