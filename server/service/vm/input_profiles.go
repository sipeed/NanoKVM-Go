package vm

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"regexp"
	"strings"
	"sync"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

const (
	InputProfileConfigFile    = "/etc/kvm/input-profiles.json"
	inputProfileConfigVersion = 1
	inputProfileLimit         = 32
	inputProfileNameLimit     = 64
	defaultInputProfileID     = "default-no-calibration"
	autoInputProfileID        = "automatic-crop-calibration"
	legacyInputProfileMode    = "absolute-android"
	canonicalInputProfileMode = "digitizer-stylus"
)

var (
	inputProfileConfigMutex sync.RWMutex
	inputProfileReadFile    = os.ReadFile
	inputProfileWriteFile   = writeMenuBarFileAtomically
	inputProfileIDPattern   = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
	inputProfileCorners     = []string{"top-left", "top-right", "bottom-right", "bottom-left"}
	inputProfileModes       = map[string]struct{}{
		"absolute":                {},
		canonicalInputProfileMode: {},
		legacyInputProfileMode:    {}, // legacy persisted key
		"touch-screen":            {},
	}
	inputProfilePlatforms = map[string]struct{}{
		"": {}, "ios": {}, "android": {}, "windows": {}, "macos": {}, "linux": {},
	}
)

func (s *Service) GetInputProfileConfig(c *gin.Context) {
	var rsp proto.Response
	config, configured, err := readInputProfileConfig()
	if err != nil {
		rsp.ErrRsp(c, -1, "get input profile config failed")
		return
	}
	rsp.OkRspWithData(c, &proto.GetInputProfileConfigRsp{
		Configured:         configured,
		InputProfileConfig: *config,
	})
}

func (s *Service) SetInputProfileConfig(c *gin.Context) {
	var req proto.InputProfileConfig
	var rsp proto.Response
	if err := c.ShouldBindJSON(&req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if err := validateInputProfileConfig(&req); err != nil {
		rsp.ErrRsp(c, -1, "invalid input profile config")
		return
	}
	normalizeInputProfileConfig(&req)
	data, err := json.Marshal(&req)
	if err != nil {
		rsp.ErrRsp(c, -2, "encode input profile config failed")
		return
	}

	inputProfileConfigMutex.Lock()
	err = inputProfileWriteFile(InputProfileConfigFile, data, 0o644)
	inputProfileConfigMutex.Unlock()
	if err != nil {
		rsp.ErrRsp(c, -2, "write input profile config failed")
		return
	}
	rsp.OkRsp(c)
}

func readInputProfileConfig() (*proto.InputProfileConfig, bool, error) {
	inputProfileConfigMutex.RLock()
	defer inputProfileConfigMutex.RUnlock()

	return readInputProfileConfigLocked()
}

// LoadInputProfileConfig returns the device input profile store. Other services
// (MCP) apply the same calibration the browser applies to user input.
func LoadInputProfileConfig() (*proto.InputProfileConfig, bool, error) {
	return readInputProfileConfig()
}

func readInputProfileConfigLocked() (*proto.InputProfileConfig, bool, error) {
	data, err := inputProfileReadFile(InputProfileConfigFile)
	if errors.Is(err, os.ErrNotExist) {
		return defaultInputProfileConfig(), false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var config proto.InputProfileConfig
	if json.Unmarshal(data, &config) != nil || validateInputProfileConfig(&config) != nil {
		return defaultInputProfileConfig(), false, nil
	}
	normalizeInputProfileConfig(&config)
	return &config, true, nil
}

func defaultInputProfileConfig() *proto.InputProfileConfig {
	return &proto.InputProfileConfig{
		Version:           inputProfileConfigVersion,
		SelectedProfileID: autoInputProfileID,
		Profiles: []proto.InputProfile{
			{
				ID:           autoInputProfileID,
				Name:         "Automatic calibration",
				Origin:       "builtin",
				Calibrations: map[string]proto.InputProfileCalibration{},
			},
			{
				ID:           defaultInputProfileID,
				Name:         "No calibration",
				Origin:       "builtin",
				Calibrations: map[string]proto.InputProfileCalibration{},
			},
		},
	}
}

func normalizeInputProfileConfig(config *proto.InputProfileConfig) {
	if config.Profiles == nil {
		config.Profiles = []proto.InputProfile{}
	}
	for index := range config.Profiles {
		if config.Profiles[index].Calibrations == nil {
			config.Profiles[index].Calibrations = map[string]proto.InputProfileCalibration{}
		}
		if calibration, exists := config.Profiles[index].Calibrations[legacyInputProfileMode]; exists {
			if _, canonicalExists := config.Profiles[index].Calibrations[canonicalInputProfileMode]; !canonicalExists {
				config.Profiles[index].Calibrations[canonicalInputProfileMode] = calibration
			}
			delete(config.Profiles[index].Calibrations, legacyInputProfileMode)
		}
	}
}

func validateInputProfileConfig(config *proto.InputProfileConfig) error {
	if config == nil || config.Version != inputProfileConfigVersion {
		return errors.New("unsupported input profile config version")
	}
	if len(config.Profiles) == 0 || len(config.Profiles) > inputProfileLimit {
		return errors.New("invalid input profile count")
	}

	profileIDs := make(map[string]struct{}, len(config.Profiles))
	for _, profile := range config.Profiles {
		if err := validateInputProfile(profile); err != nil {
			return err
		}
		if _, exists := profileIDs[profile.ID]; exists {
			return errors.New("duplicate input profile id")
		}
		profileIDs[profile.ID] = struct{}{}
	}
	if _, exists := profileIDs[config.SelectedProfileID]; !exists {
		return errors.New("selected input profile not found")
	}
	return nil
}

func validateInputProfile(profile proto.InputProfile) error {
	if !inputProfileIDPattern.MatchString(profile.ID) {
		return errors.New("invalid input profile id")
	}
	name := strings.TrimSpace(profile.Name)
	if name == "" || len([]rune(name)) > inputProfileNameLimit {
		return errors.New("invalid input profile name")
	}
	if profile.Origin != "user" && profile.Origin != "builtin" {
		return errors.New("invalid input profile origin")
	}
	if _, exists := inputProfilePlatforms[profile.Platform]; !exists {
		return errors.New("invalid input profile platform")
	}
	if profile.CreatedAt < 0 || profile.UpdatedAt < 0 {
		return errors.New("invalid input profile timestamp")
	}
	for mode, calibration := range profile.Calibrations {
		if _, exists := inputProfileModes[mode]; !exists {
			return errors.New("invalid input profile mode")
		}
		if err := validateInputProfileCalibration(calibration); err != nil {
			return err
		}
	}
	return nil
}

func validateInputProfileCalibration(calibration proto.InputProfileCalibration) error {
	if calibration.UpdatedAt < 0 || len(calibration.Points) != len(inputProfileCorners) {
		return errors.New("invalid input profile calibration")
	}
	for _, corner := range inputProfileCorners {
		point, exists := calibration.Points[corner]
		if !exists || !validNormalizedPoint(point.Source) || !validNormalizedPoint(point.Target) {
			return errors.New("invalid input profile calibration point")
		}
	}
	return nil
}

func validNormalizedPoint(point proto.InputProfilePoint) bool {
	return !math.IsNaN(point.X) && !math.IsInf(point.X, 0) && point.X >= 0 && point.X <= 1 &&
		!math.IsNaN(point.Y) && !math.IsInf(point.Y, 0) && point.Y >= 0 && point.Y <= 1
}
