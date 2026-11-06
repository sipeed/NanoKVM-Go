package vm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"NanoKVM-Server/config"
	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const fingerRobotDebugRequestMaxBytes = 4096

var (
	fingerRobotDebugModeEnabled = config.IsFingerRobotDebug
	fingerRobotDebugAuthEnabled = func() bool {
		return config.GetInstance().Authentication != "disable"
	}
	fingerRobotDebugAvailable  = IsFingerRobotAvailable
	fingerRobotDebugLimits     = GetFingerRobotLimits
	fingerRobotDebugStatus     = GetFingerRobotStatus
	fingerRobotDebugTouchLease = TouchFingerRobotDebugLease
	fingerRobotDebugStart      = StartFingerRobotDebug
	fingerRobotDebugStop       = StopFingerRobotDebug
	fingerRobotDebugReset      = ResetFingerRobot
)

func (s *Service) GetFingerRobotDebug(c *gin.Context) {
	var rsp proto.Response

	if !fingerRobotDebugModeEnabled() || !fingerRobotDebugAuthEnabled() {
		rsp.OkRspWithData(c, proto.GetFingerRobotDebugRsp{
			Enabled:   false,
			Available: false,
		})
		return
	}

	limits := fingerRobotDebugLimits()
	fingerRobotDebugTouchLease()
	status := fingerRobotDebugStatus()
	if !status.debug {
		status = FingerRobotStatus{
			State:  FingerRobotStateIdle,
			Result: FingerRobotResultIdle,
		}
	}
	rsp.OkRspWithData(c, proto.GetFingerRobotDebugRsp{
		Enabled:               true,
		Available:             fingerRobotDebugAvailable(),
		ZeroUS:                limits.ZeroUS,
		ReleaseUS:             fingerRobotDebugRange(limits.ReleaseMinUS, limits.ReleaseMaxUS, limits.ReleaseDefaultUS),
		PressUS:               fingerRobotDebugRange(limits.PressMinUS, limits.PressMaxUS, limits.PressDefaultUS),
		PressSpeedUSPerSecond: fingerRobotDebugRange(limits.PressSpeedMinUSPerSecond, limits.PressSpeedMaxUSPerSecond, limits.PressSpeedDefaultUSPerSecond),
		Count: &proto.FingerRobotDebugCountRsp{
			Min:            limits.CountMin,
			Default:        limits.CountDefault,
			UnlimitedValue: limits.CountUnlimitedValue,
		},
		PressMS:           fingerRobotDebugRange(limits.PressMinMS, limits.PressMaxMS, limits.PressDefaultMS),
		ReleaseIntervalMS: limits.ReleaseIntervalMS,
		LeaseTimeoutMS:    limits.LeaseTimeoutMS,
		Running:           status.Running,
		Current:           status.Current,
		Total:             status.Total,
		State:             string(status.State),
		Result:            string(status.Result),
		Error:             status.Error,
	})
}

func (s *Service) TestFingerRobotDebug(c *gin.Context) {
	var rsp proto.Response
	if !requireFingerRobotDebug(c, &rsp) {
		return
	}
	if !fingerRobotDebugAvailable() {
		rsp.ErrRsp(c, -3, "finger robot is unavailable")
		return
	}

	var req proto.FingerRobotDebugTestReq
	if err := decodeFingerRobotDebugJSON(c, &req); err != nil {
		rsp.ErrRsp(c, -1, fmt.Sprintf("invalid arguments: %s", err))
		return
	}
	if req.Count == nil {
		rsp.ErrRsp(c, -1, "invalid arguments: count is required")
		return
	}

	controllerReq := FingerRobotDebugRequest{
		ReleaseUS:             req.ReleaseUS,
		PressUS:               req.PressUS,
		PressSpeedUSPerSecond: req.PressSpeedUSPerSecond,
		Count:                 *req.Count,
		PressMS:               req.PressMS,
	}
	if err := validateFingerRobotDebugHTTPReq(controllerReq, fingerRobotDebugLimits()); err != nil {
		rsp.ErrRsp(c, -1, fmt.Sprintf("invalid arguments: %s", err))
		return
	}

	if targetPowerActionRequiresChallenge() {
		if err := verifyTargetPowerActionChallenge(c, req.Challenge, authservice.SensitiveActionTargetPower); err != nil {
			log.Errorf("verify finger robot debug challenge failed: %s", err)
			rsp.ErrRsp(c, -3, "sensitive action verification required")
			return
		}
	}

	if err := fingerRobotDebugStart(controllerReq); err != nil {
		switch {
		case errors.Is(err, ErrFingerRobotBusy):
			rsp.ErrRsp(c, -2, "finger robot operation is busy")
		case errors.Is(err, ErrFingerRobotInvalid):
			rsp.ErrRsp(c, -1, fmt.Sprintf("invalid arguments: %s", err))
		default:
			rsp.ErrRsp(c, -3, fmt.Sprintf("operation failed: %s", err))
		}
		return
	}

	rsp.OkRsp(c)
}

func (s *Service) StopFingerRobotDebug(c *gin.Context) {
	var rsp proto.Response
	if !requireFingerRobotDebug(c, &rsp) {
		return
	}

	if err := fingerRobotDebugStop(); err != nil {
		if errors.Is(err, ErrFingerRobotBusy) {
			rsp.ErrRsp(c, -2, "finger robot operation is busy")
		} else {
			rsp.ErrRsp(c, -3, fmt.Sprintf("operation failed: %s", err))
		}
		return
	}

	rsp.OkRsp(c)
}

func (s *Service) ResetFingerRobotDebug(c *gin.Context) {
	var rsp proto.Response
	if !requireFingerRobotDebug(c, &rsp) {
		return
	}

	if err := fingerRobotDebugReset(); err != nil {
		rsp.ErrRsp(c, -3, fmt.Sprintf("operation failed: %s", err))
		return
	}

	rsp.OkRsp(c)
}

func requireFingerRobotDebug(c *gin.Context, rsp *proto.Response) bool {
	if !fingerRobotDebugModeEnabled() {
		rsp.ErrRsp(c, -3, "finger robot debug mode is disabled")
		return false
	}
	if !fingerRobotDebugAuthEnabled() {
		rsp.ErrRsp(c, -3, "finger robot debug requires authentication")
		return false
	}
	return true
}

func fingerRobotDebugRange(minimum, maximum, defaultValue int) *proto.FingerRobotDebugRangeRsp {
	return &proto.FingerRobotDebugRangeRsp{
		Min:     minimum,
		Max:     maximum,
		Default: defaultValue,
	}
}

func validateFingerRobotDebugHTTPReq(req FingerRobotDebugRequest, limits FingerRobotLimits) error {
	if req.ReleaseUS < limits.ReleaseMinUS || req.ReleaseUS > limits.ReleaseMaxUS {
		return fmt.Errorf("releaseUs must be between %d and %d", limits.ReleaseMinUS, limits.ReleaseMaxUS)
	}
	if req.PressUS < limits.PressMinUS || req.PressUS > limits.PressMaxUS {
		return fmt.Errorf("pressUs must be between %d and %d", limits.PressMinUS, limits.PressMaxUS)
	}
	if req.PressUS >= req.ReleaseUS {
		return errors.New("pressUs must be less than releaseUs")
	}
	if req.PressSpeedUSPerSecond < limits.PressSpeedMinUSPerSecond || req.PressSpeedUSPerSecond > limits.PressSpeedMaxUSPerSecond {
		return fmt.Errorf("pressSpeedUsPerSecond must be between %d and %d", limits.PressSpeedMinUSPerSecond, limits.PressSpeedMaxUSPerSecond)
	}
	if req.Count != limits.CountUnlimitedValue && req.Count < limits.CountMin {
		return fmt.Errorf("count must be %d for unlimited testing or at least %d", limits.CountUnlimitedValue, limits.CountMin)
	}
	if req.PressMS < limits.PressMinMS || req.PressMS > limits.PressMaxMS {
		return fmt.Errorf("pressMs must be between %d and %d", limits.PressMinMS, limits.PressMaxMS)
	}

	return nil
}

func decodeFingerRobotDebugJSON(c *gin.Context, destination interface{}) error {
	if c == nil || c.Request == nil || c.Request.Body == nil {
		return errors.New("missing request body")
	}

	body := http.MaxBytesReader(c.Writer, c.Request.Body, fingerRobotDebugRequestMaxBytes)
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
		return err
	}
	return nil
}
