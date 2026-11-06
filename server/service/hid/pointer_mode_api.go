package hid

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

const pointerModeRequestMaxBytes = 1024

// SetPointerMode changes the device-wide HID2 profile through the transaction
// core. The operation deliberately uses a context without the request's
// cancellation: USB re-enumeration can tear down the requesting USB network
// connection, but the gadget transition must still finish (or roll back) so a
// later GET can report the authoritative result.
func (service *Service) SetPointerMode(c *gin.Context) {
	var response proto.Response
	req, err := decodeSetPointerModeRequest(c)
	if err != nil {
		response.ErrRsp(c, pointerModeAPIInvalidArgumentsCode, "invalid pointer mode request")
		return
	}

	requested, ok := parsePointerMode(req.Mode)
	if !ok || requested != PointerMode(req.Mode) {
		response.ErrRsp(c, pointerModeAPIInvalidArgumentsCode, "invalid pointer mode")
		return
	}

	manager := service.pointerModeManager
	if manager == nil {
		manager = getPointerModeManager()
	}
	if status := manager.status(); status.Switching {
		response.ErrRspWithData(c, pointerModeAPIInProgressCode, "pointer mode switch already in progress", pointerModeStatusResponse(status))
		return
	}
	if !service.pointerModeAPIMu.TryLock() {
		status := manager.status()
		status.Switching = true
		response.ErrRspWithData(c, pointerModeAPIInProgressCode, "pointer mode switch already in progress", pointerModeStatusResponse(status))
		return
	}
	defer service.pointerModeAPIMu.Unlock()

	requestContext := context.Background()
	if c.Request != nil {
		requestContext = context.WithoutCancel(c.Request.Context())
	}
	status, err := service.SwitchPointerMode(requestContext, requested)
	if err != nil {
		code, message := pointerModeAPIError(err, status)
		response.ErrRspWithData(c, code, message, pointerModeStatusResponse(status))
		return
	}

	response.OkRspWithData(c, pointerModeStatusResponse(status))
}

func decodeSetPointerModeRequest(c *gin.Context) (proto.SetPointerModeReq, error) {
	if c == nil || c.Request == nil || c.Request.Body == nil {
		return proto.SetPointerModeReq{}, errors.New("request body is missing")
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, pointerModeRequestMaxBytes+1))
	if err != nil {
		return proto.SetPointerModeReq{}, err
	}
	if len(body) == 0 || len(body) > pointerModeRequestMaxBytes {
		return proto.SetPointerModeReq{}, errors.New("request body size is invalid")
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var request proto.SetPointerModeReq
	if err := decoder.Decode(&request); err != nil {
		return proto.SetPointerModeReq{}, err
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return proto.SetPointerModeReq{}, errors.New("request contains multiple JSON values")
		}
		return proto.SetPointerModeReq{}, err
	}
	if strings.TrimSpace(request.Mode) == "" {
		return proto.SetPointerModeReq{}, errors.New("pointer mode is empty")
	}
	return request, nil
}

func pointerModeStatusResponse(status PointerModeStatus) *proto.GetPointerModeRsp {
	return &proto.GetPointerModeRsp{
		Mode:      string(status.Mode),
		Switching: status.Switching,
		Degraded:  status.Degraded,
		Error:     status.Error,
	}
}

const (
	pointerModeAPIInvalidArgumentsCode = -1
	pointerModeAPIInProgressCode       = -2
	pointerModeAPILockCode             = -3
	pointerModeAPIQuiesceCode          = -4
	pointerModeAPIScriptCode           = -5
	pointerModeAPIValidationCode       = -6
	pointerModeAPIResumeCode           = -7
	pointerModeAPIDegradedCode         = -8
	pointerModeAPIFailureCode          = -9
)

func pointerModeAPIError(err error, status PointerModeStatus) (int, string) {
	if errors.Is(err, ErrPointerModeSwitchInvalidMode) {
		return pointerModeAPIInvalidArgumentsCode, "invalid pointer mode"
	}
	if status.Degraded || errors.Is(err, ErrPointerModeSwitchDegraded) || errors.Is(err, ErrPointerModeSwitchRollback) {
		return pointerModeAPIDegradedCode, "pointer mode switch failed; device state is degraded"
	}
	if errors.Is(err, ErrPointerModeSwitchInProgress) {
		return pointerModeAPIInProgressCode, "pointer mode switch already in progress"
	}

	var switchError *PointerModeSwitchError
	if errors.As(err, &switchError) {
		switch switchError.Stage {
		case PointerModeSwitchStageLock:
			return pointerModeAPILockCode, "pointer mode switch lock failed"
		case PointerModeSwitchStageQuiesce, PointerModeSwitchStageHIDClose:
			return pointerModeAPIQuiesceCode, fmt.Sprintf("pointer mode switch failed during %s; previous mode was restored", switchError.Stage)
		case PointerModeSwitchStageScript:
			return pointerModeAPIScriptCode, "pointer mode switch failed during USB rebuild; previous mode was restored"
		case PointerModeSwitchStageValidation, PointerModeSwitchStageHIDOpen:
			return pointerModeAPIValidationCode, fmt.Sprintf("pointer mode switch failed during %s; previous mode was restored", switchError.Stage)
		case PointerModeSwitchStageResume:
			return pointerModeAPIResumeCode, "pointer mode switch completed but input could not be resumed"
		case PointerModeSwitchStageSnapshot, PointerModeSwitchStagePersistence:
			return pointerModeAPIFailureCode, fmt.Sprintf("pointer mode switch failed during %s", switchError.Stage)
		}
	}
	return pointerModeAPIFailureCode, "pointer mode switch failed"
}
