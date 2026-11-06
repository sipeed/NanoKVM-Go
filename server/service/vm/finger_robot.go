package vm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const fingerRobotRequestMaxBytes = 4096

var (
	targetPowerActionRequiresChallenge = authservice.TargetPowerActionRequiresChallenge
	verifyTargetPowerActionChallenge   = authservice.VerifySensitiveActionChallenge

	fingerRobotNormalSettingEnabled = IsFingerRobotSettingEnabled
	fingerRobotNormalAvailable      = IsFingerRobotAvailable
	fingerRobotNormalStatus         = GetFingerRobotStatus
	fingerRobotNormalListProfiles   = ListFingerRobotArmProfiles
	fingerRobotNormalLoadConfig     = LoadFingerRobotConfig
	fingerRobotNormalUpdateConfig   = UpdateFingerRobotConfig
	fingerRobotNormalPress          = PressFingerRobot
	fingerRobotNormalRelease        = ReleaseFingerRobot
	fingerRobotNormalReplaceArm     = ResetFingerRobot
)

// GetFingerRobot returns only product-level settings and display metadata.
// Raw PWM profile values deliberately remain private to the controller.
// Product-level endpoints intentionally remain available in both normal and
// debug modes: Debug adds the engineering API instead of replacing this one.
func (s *Service) GetFingerRobot(c *gin.Context) {
	var rsp proto.Response

	data, err := getFingerRobotResponse()
	if err != nil {
		log.WithError(err).Error("failed to get finger robot configuration")
		rsp.ErrRspWithData(c, -5, "get finger robot configuration failed", data)
		return
	}

	rsp.OkRspWithData(c, data)
}

func (s *Service) SetFingerRobot(c *gin.Context) {
	var req proto.SetFingerRobotReq
	var rsp proto.Response

	if err := decodeFingerRobotJSON(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	update, expectedRevision, err := fingerRobotConfigUpdateFromRequest(req)
	if err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if err := ValidateFingerRobotArmProfiles(); err != nil {
		err = releaseAfterFingerRobotConfigFailure(err)
		log.WithError(err).Error("finger robot profiles failed validation")
		rsp.ErrRsp(c, -5, "operation failed")
		return
	}
	if err := validateFingerRobotConfigUpdate(update); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	// Saving product preferences does not depend on the menu-bar switch or PWM
	// availability and never moves the servo.
	current, err := fingerRobotNormalLoadConfig()
	if err != nil {
		err = releaseAfterFingerRobotConfigFailure(err)
		log.WithError(err).Error("failed to load finger robot configuration for update")
		rsp.ErrRsp(c, -5, "operation failed")
		return
	}
	if current.Revision != expectedRevision {
		writeFingerRobotRevisionConflict(c, &rsp, current)
		return
	}

	updated, changed, err := fingerRobotNormalUpdateConfig(expectedRevision, update)
	if err != nil {
		if errors.Is(err, ErrFingerRobotConfigRevisionConflict) {
			writeFingerRobotRevisionConflict(c, &rsp, updated)
			return
		}
		if errors.Is(err, ErrFingerRobotConfigInvalid) {
			err = releaseAfterFingerRobotConfigFailure(err)
		}
		log.WithError(err).Error("failed to save finger robot configuration")
		rsp.ErrRsp(c, -5, "operation failed")
		return
	}

	rsp.OkRspWithData(c, proto.SetFingerRobotRsp{
		Config:  fingerRobotPublicConfig(updated),
		Changed: changed,
	})
}

func (s *Service) PressFingerRobot(c *gin.Context) {
	var req proto.FingerRobotActionReq
	var rsp proto.Response

	if err := decodeFingerRobotJSON(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	pressRequest, err := snapshotConfiguredFingerRobotPress()
	if err != nil {
		log.WithError(err).Error("failed to load finger robot press configuration")
		rsp.ErrRsp(c, -5, "operation failed")
		return
	}
	if !requireFingerRobotChallenge(c, &rsp, req.Challenge) {
		return
	}

	if err := fingerRobotNormalPress(pressRequest); err != nil {
		switch {
		case errors.Is(err, ErrFingerRobotBusy):
			rsp.ErrRsp(c, -2, "finger robot operation is busy")
			return
		case errors.Is(err, ErrFingerRobotUnavailable):
			rsp.ErrRsp(c, -5, "finger robot is unavailable")
			return
		}
		log.WithError(err).Error("failed to press finger robot")
		rsp.ErrRsp(c, -5, "operation failed")
		return
	}

	rsp.OkRsp(c)
}

func (s *Service) ReleaseFingerRobot(c *gin.Context) {
	var req proto.FingerRobotActionReq
	var rsp proto.Response

	if err := decodeFingerRobotJSON(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if err := fingerRobotNormalRelease(); err != nil {
		log.WithError(err).Error("failed to release finger robot")
		rsp.ErrRsp(c, -5, "operation failed")
		return
	}

	rsp.OkRsp(c)
}

func (s *Service) ReplaceFingerRobotArm(c *gin.Context) {
	var req *proto.FingerRobotActionReq
	var rsp proto.Response

	if err := decodeFingerRobotJSON(c, &req); err != nil || req == nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if !fingerRobotNormalSettingEnabled() || !fingerRobotNormalAvailable() {
		rsp.ErrRsp(c, -5, "finger robot is unavailable")
		return
	}
	if !requireFingerRobotChallenge(c, &rsp, req.Challenge) {
		return
	}

	if err := fingerRobotNormalReplaceArm(); err != nil {
		log.WithError(err).Error("failed to position finger robot for arm replacement")
		rsp.ErrRsp(c, -5, "operation failed")
		return
	}

	rsp.OkRsp(c)
}

// snapshotConfiguredFingerRobotPress takes one complete persisted selection.
func snapshotConfiguredFingerRobotPress() (FingerRobotPressRequest, error) {
	config, err := fingerRobotNormalLoadConfig()
	if err != nil {
		return FingerRobotPressRequest{}, releaseAfterFingerRobotConfigFailure(err)
	}

	strength, err := config.StrengthByArm.StrengthForArm(config.ActiveArm)
	if err != nil {
		return FingerRobotPressRequest{}, releaseAfterFingerRobotConfigFailure(err)
	}

	return FingerRobotPressRequest{
		ProfileID: config.ActiveArm,
		Strength:  strength,
		PressMS:   config.PressMS,
		Speed:     config.Speed,
	}, nil
}

func getFingerRobotResponse() (proto.GetFingerRobotRsp, error) {
	limits := GetFingerRobotLimits()
	data := proto.GetFingerRobotRsp{
		Profiles:     []proto.FingerRobotProfileRsp{},
		Strength:     proto.FingerRobotRangeRsp{Min: 1, Max: FingerRobotStrengthLevelCount, Default: FingerRobotDefaultStrengthLevel},
		PressMS:      proto.FingerRobotRangeRsp{Min: limits.PressMinMS, Max: limits.PressMaxMS, Default: FingerRobotDefaultPressMS},
		SpeedOptions: []string{string(FingerRobotSpeedSlow), string(FingerRobotSpeedFast)},
		DefaultSpeed: string(FingerRobotSpeedSlow),
	}
	refreshFingerRobotRuntimeResponse(&data)

	profiles, err := fingerRobotNormalListProfiles()
	if err != nil {
		err = releaseAfterFingerRobotConfigFailure(err)
		refreshFingerRobotRuntimeResponse(&data)
		return data, err
	}
	data.Profiles = make([]proto.FingerRobotProfileRsp, 0, len(profiles))
	for _, profile := range profiles {
		data.Profiles = append(data.Profiles, proto.FingerRobotProfileRsp{
			ID:              profile.ID,
			DisplayName:     profile.DisplayName,
			DefaultStrength: profile.DefaultStrength,
		})
	}

	config, err := fingerRobotNormalLoadConfig()
	if err != nil {
		err = releaseAfterFingerRobotConfigFailure(err)
		refreshFingerRobotRuntimeResponse(&data)
		return data, err
	}
	publicConfig := fingerRobotPublicConfig(config)
	data.Config = &publicConfig
	return data, nil
}

func refreshFingerRobotRuntimeResponse(data *proto.GetFingerRobotRsp) {
	data.Enabled = fingerRobotNormalSettingEnabled()
	data.Available = fingerRobotNormalAvailable()
	data.Status = fingerRobotPublicStatus(fingerRobotNormalStatus())
}

func fingerRobotConfigUpdateFromRequest(req proto.SetFingerRobotReq) (FingerRobotConfigUpdate, int64, error) {
	if req.Version == nil || req.Revision == nil || req.ActiveArm == nil ||
		req.StrengthByArm == nil || req.StrengthByArm.Long == nil || req.StrengthByArm.Short == nil ||
		req.PressMS == nil || req.Speed == nil {
		return FingerRobotConfigUpdate{}, 0, ErrFingerRobotConfigInvalid
	}
	if *req.Version != FingerRobotConfigVersion || *req.Revision < 1 {
		return FingerRobotConfigUpdate{}, 0, ErrFingerRobotConfigInvalid
	}

	return FingerRobotConfigUpdate{
		ActiveArm: *req.ActiveArm,
		StrengthByArm: FingerRobotStrengthByArm{
			Long:  *req.StrengthByArm.Long,
			Short: *req.StrengthByArm.Short,
		},
		PressMS: *req.PressMS,
		Speed:   FingerRobotSpeed(*req.Speed),
	}, *req.Revision, nil
}

func fingerRobotPublicConfig(config FingerRobotConfig) proto.FingerRobotConfigRsp {
	return proto.FingerRobotConfigRsp{
		Version:   config.Version,
		Revision:  config.Revision,
		ActiveArm: config.ActiveArm,
		StrengthByArm: proto.FingerRobotStrengthByArmRsp{
			Long:  config.StrengthByArm.Long,
			Short: config.StrengthByArm.Short,
		},
		PressMS: config.PressMS,
		Speed:   string(config.Speed),
	}
}

func fingerRobotPublicStatus(status FingerRobotStatus) proto.FingerRobotStatusRsp {
	publicError := ""
	if status.Error != "" {
		publicError = "operation failed"
	}
	return proto.FingerRobotStatusRsp{
		Running: status.Running,
		State:   string(status.State),
		Result:  string(status.Result),
		Error:   publicError,
	}
}

func releaseAfterFingerRobotConfigFailure(configErr error) error {
	if configErr == nil {
		return nil
	}
	return errors.Join(configErr, fingerRobotNormalRelease())
}

func requireFingerRobotChallenge(c *gin.Context, rsp *proto.Response, challenge string) bool {
	if !targetPowerActionRequiresChallenge() {
		return true
	}
	if err := verifyTargetPowerActionChallenge(c, challenge, authservice.SensitiveActionTargetPower); err != nil {
		log.WithError(err).Error("verify finger robot challenge failed")
		rsp.ErrRsp(c, -3, "sensitive action verification required")
		return false
	}
	return true
}

func writeFingerRobotRevisionConflict(c *gin.Context, rsp *proto.Response, current FingerRobotConfig) {
	rsp.ErrRspWithData(c, -6, "revision conflict", proto.SetFingerRobotRsp{
		Config: fingerRobotPublicConfig(current),
	})
}

func decodeFingerRobotJSON(c *gin.Context, destination interface{}) error {
	if c == nil || c.Request == nil || c.Request.Body == nil {
		return errors.New("missing request body")
	}

	body := http.MaxBytesReader(c.Writer, c.Request.Body, fingerRobotRequestMaxBytes)
	defer body.Close()
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}

	var trailing interface{}
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return fmt.Errorf("trailing JSON: %w", err)
	}
	return nil
}
