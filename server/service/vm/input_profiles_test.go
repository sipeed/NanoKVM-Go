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

type inputProfileTestResponse struct {
	Code int                             `json:"code"`
	Data *proto.GetInputProfileConfigRsp `json:"data"`
}

func TestMissingInputProfilesUseAutomaticDefault(t *testing.T) {
	setupInputProfileTest(t)
	inputProfileReadFile = func(string) ([]byte, error) { return nil, os.ErrNotExist }

	config, configured, err := readInputProfileConfig()
	if err != nil || configured || config.SelectedProfileID != autoInputProfileID {
		t.Fatalf("input profile fallback config=%+v configured=%v err=%v", config, configured, err)
	}
	if len(config.Profiles) != 2 || config.Profiles[0].ID != autoInputProfileID ||
		config.Profiles[1].ID != defaultInputProfileID {
		t.Fatalf("unexpected default input profile order: %+v", config.Profiles)
	}
}

func TestInputProfileConfigRoundTrip(t *testing.T) {
	setupInputProfileTest(t)
	config := validInputProfileTestConfig()
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	inputProfileReadFile = func(path string) ([]byte, error) {
		if path != InputProfileConfigFile {
			t.Fatalf("input profile read path = %q", path)
		}
		return data, nil
	}

	actual, configured, err := readInputProfileConfig()
	if err != nil || !configured || actual.SelectedProfileID != "android-game" {
		t.Fatalf("input profile round trip config=%+v configured=%v err=%v", actual, configured, err)
	}
}

func TestInputProfileConfigMigratesLegacyDigitizerMode(t *testing.T) {
	setupInputProfileTest(t)
	config := validInputProfileTestConfig()
	calibration := config.Profiles[2].Calibrations["touch-screen"]
	config.Profiles[2].Calibrations = map[string]proto.InputProfileCalibration{
		"absolute-android": calibration,
	}
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	inputProfileReadFile = func(string) ([]byte, error) { return data, nil }

	actual, configured, err := readInputProfileConfig()
	if err != nil || !configured {
		t.Fatalf("legacy input profile read config=%+v configured=%v err=%v", actual, configured, err)
	}
	calibrations := actual.Profiles[2].Calibrations
	if _, exists := calibrations[legacyInputProfileMode]; exists {
		t.Fatalf("legacy calibration key survived migration: %+v", calibrations)
	}
	if _, exists := calibrations[canonicalInputProfileMode]; !exists {
		t.Fatalf("canonical calibration key missing after migration: %+v", calibrations)
	}
}

func TestInputProfileValidationRejectsInvalidData(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*proto.InputProfileConfig)
	}{
		{name: "selected", mutate: func(config *proto.InputProfileConfig) { config.SelectedProfileID = "missing" }},
		{name: "duplicate", mutate: func(config *proto.InputProfileConfig) { config.Profiles = append(config.Profiles, config.Profiles[2]) }},
		{name: "mode", mutate: func(config *proto.InputProfileConfig) {
			config.Profiles[2].Calibrations["bad"] = config.Profiles[2].Calibrations["touch-screen"]
		}},
		{name: "coordinate", mutate: func(config *proto.InputProfileConfig) {
			calibration := config.Profiles[2].Calibrations["touch-screen"]
			point := calibration.Points["top-left"]
			point.Target.X = 2
			calibration.Points["top-left"] = point
			config.Profiles[2].Calibrations["touch-screen"] = calibration
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := validInputProfileTestConfig()
			test.mutate(&config)
			if validateInputProfileConfig(&config) == nil {
				t.Fatalf("invalid input profile config accepted: %+v", config)
			}
		})
	}
}

func TestSetInputProfileConfigPersistsJSON(t *testing.T) {
	setupInputProfileTest(t)
	var persisted proto.InputProfileConfig
	inputProfileWriteFile = func(path string, data []byte, mode os.FileMode) error {
		if path != InputProfileConfigFile || mode != 0o644 {
			t.Fatalf("input profile write path=%q mode=%#o", path, mode)
		}
		return json.Unmarshal(data, &persisted)
	}
	data, _ := json.Marshal(validInputProfileTestConfig())
	rsp := performSetInputProfileBody(t, data)
	if rsp.Code != 0 || persisted.SelectedProfileID != "android-game" {
		t.Fatalf("input profile set response=%+v persisted=%+v", rsp, persisted)
	}
}

func validInputProfileTestConfig() proto.InputProfileConfig {
	points := map[string]proto.InputProfileCalibrationPoint{}
	for _, corner := range inputProfileCorners {
		points[corner] = proto.InputProfileCalibrationPoint{
			Source: proto.InputProfilePoint{X: 0.1, Y: 0.1},
			Target: proto.InputProfilePoint{X: 0.2, Y: 0.2},
		}
	}
	return proto.InputProfileConfig{
		Version:           1,
		SelectedProfileID: "android-game",
		Profiles: []proto.InputProfile{
			{ID: autoInputProfileID, Name: "Automatic calibration", Origin: "builtin", Calibrations: map[string]proto.InputProfileCalibration{}},
			{ID: defaultInputProfileID, Name: "No calibration", Origin: "builtin", Calibrations: map[string]proto.InputProfileCalibration{}},
			{ID: "android-game", Name: "Android Game", Origin: "user", Platform: "android", CreatedAt: 1, UpdatedAt: 2, Calibrations: map[string]proto.InputProfileCalibration{
				"touch-screen": {Points: points, UpdatedAt: 2},
			}},
		},
	}
}

func setupInputProfileTest(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	readFile := inputProfileReadFile
	writeFile := inputProfileWriteFile
	t.Cleanup(func() {
		inputProfileReadFile = readFile
		inputProfileWriteFile = writeFile
	})
}

func performSetInputProfileBody(t *testing.T, body []byte) inputProfileTestResponse {
	t.Helper()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/vm/input-profiles", bytes.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	NewService().SetInputProfileConfig(ctx)
	var rsp inputProfileTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("unmarshal input profile response: %v body=%s", err, recorder.Body.String())
	}
	return rsp
}
