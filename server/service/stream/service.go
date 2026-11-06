package stream

import (
	"errors"
	"strings"
	"sync"
	"time"

	"NanoKVM-Server/common"
	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type Service struct{}

const kvmCommVideoModeRestartDelay = 5 * time.Second

var (
	encoderSettingsMu sync.Mutex

	setKvmVisionRateControl = func(mode uint8) int {
		return common.GetKvmVision().SetRateControl(mode)
	}
	setKvmVisionGOP = func(gop uint16) int {
		return common.GetKvmVision().SetGop(gop)
	}
	setKvmVisionBitRate = func(bitRate uint16) int {
		return common.GetKvmVision().SetBitRate(bitRate)
	}
	setKvmVisionFPS = func(fps uint8) int {
		return common.GetKvmVision().SetFps(fps)
	}

	saveKvmCommEncoderRateControl = common.SetKvmCommEncoderRateControl
	saveKvmCommEncoderBitRate     = common.SetKvmCommEncoderBitRate
	saveKvmCommEncoderGOP         = common.SetKvmCommEncoderGOP
	saveKvmCommEncoderFPS         = common.SetKvmCommEncoderFPS
)

func NewService() *Service {
	return &Service{}
}

func (s *Service) SetRateControlMode(c *gin.Context) {
	var req proto.SetRateControlModeReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	var (
		mode     uint8
		modeName string
	)

	switch strings.ToLower(req.Mode) {
	case "cbr":
		mode = common.RATE_CONTROL_CBR
		modeName = common.KvmCommEncoderRateControlCBR
	case "vbr":
		mode = common.RATE_CONTROL_VBR
		modeName = common.KvmCommEncoderRateControlVBR
	default:
		rsp.ErrRsp(c, -2, "invalid arguments")
		return
	}

	encoderSettingsMu.Lock()
	defer encoderSettingsMu.Unlock()

	screen := common.GetScreen()

	if result := setKvmVisionRateControl(mode); result != 0 {
		log.Errorf("failed to set rate control mode %s: %d", modeName, result)
		rsp.ErrRsp(c, -3, "set rate control failed")
		return
	}
	screen.RateControlMode = mode

	if err := saveKvmCommEncoderRateControl(modeName); err != nil {
		log.Errorf("failed to save rate control mode %s: %v", modeName, err)
		rsp.ErrRsp(c, -4, "failed to save rate control mode")
		return
	}

	rsp.OkRsp(c)
	log.Debugf("update rate control mode to %s", modeName)
}

func (s *Service) SetMode(c *gin.Context) {
	var req proto.SetModeReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	mode := strings.ToLower(req.Mode)
	streamType, ok := common.StreamTypeMap[mode]
	if !ok {
		rsp.ErrRsp(c, -2, "invalid arguments")
		return
	}

	kvmCommVideoMode, ok := kvmCommVideoModeFromStreamMode(mode)
	if !ok {
		rsp.ErrRsp(c, -2, "invalid arguments")
		return
	}

	if err := syncKvmCommVideoMode(kvmCommVideoMode); err != nil {
		log.Errorf("failed to sync kvmcomm video mode: %v", err)
		code, message := streamModeResponseFromKvmCommError(err)
		rsp.ErrRsp(c, code, message)
		return
	}

	common.GetScreen().StreamType = streamType
	SetActiveVideoMode(uint8(streamType))

	rsp.OkRsp(c)
	log.Debugf("update video mode to %s", req.Mode)
}

func (s *Service) SetQuality(c *gin.Context) {
	var req proto.SetQualityReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	screen := common.GetScreen()
	if req.Quality < 1000 || req.Quality > 20000 {
		rsp.ErrRsp(c, -2, "invalid arguments")
		return
	}

	encoderSettingsMu.Lock()
	defer encoderSettingsMu.Unlock()

	bitRate := uint16(req.Quality)
	if result := setKvmVisionBitRate(bitRate); result != 0 {
		log.Errorf("failed to set bit-rate %d: %d", req.Quality, result)
		rsp.ErrRsp(c, -3, "failed to set bit-rate")
		return
	}
	screen.BitRate = bitRate

	if err := saveKvmCommEncoderBitRate(req.Quality); err != nil {
		log.Errorf("failed to save bit-rate %d: %v", req.Quality, err)
		rsp.ErrRsp(c, -4, "failed to save bit-rate")
		return
	}

	rsp.OkRsp(c)
	log.Debugf("update bit-rate to %d", req.Quality)
}

func (s *Service) SetGop(c *gin.Context) {
	var req proto.SetGopReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	if req.Gop < 1 || req.Gop > 300 {
		rsp.ErrRsp(c, -2, "invalid arguments")
		return
	}

	gop := uint16(req.Gop)

	encoderSettingsMu.Lock()
	defer encoderSettingsMu.Unlock()

	screen := common.GetScreen()
	result := setKvmVisionGOP(gop)
	if result != 0 {
		log.Errorf("failed to set GOP: %d", result)
		rsp.ErrRsp(c, -3, "failed to set GOP")
		return
	}
	screen.GOP = gop

	if err := saveKvmCommEncoderGOP(req.Gop); err != nil {
		log.Errorf("failed to save GOP %d: %v", req.Gop, err)
		rsp.ErrRsp(c, -4, "failed to save GOP")
		return
	}

	rsp.OkRsp(c)
	log.Debugf("update GOP to %d", req.Gop)
}

func (s *Service) SetFps(c *gin.Context) {
	var req proto.SetFpsReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	if req.Fps < 0 || req.Fps > 120 {
		rsp.ErrRsp(c, -2, "invalid arguments")
		return
	}

	fps := uint8(req.Fps)

	encoderSettingsMu.Lock()
	defer encoderSettingsMu.Unlock()

	screen := common.GetScreen()
	result := setKvmVisionFPS(fps)
	if result != 0 {
		log.Errorf("failed to set FPS: %d", result)
		rsp.ErrRsp(c, -3, "set FPS failed")
		return
	}
	screen.FPS = fps

	if err := saveKvmCommEncoderFPS(req.Fps); err != nil {
		log.Errorf("failed to save FPS %d: %v", req.Fps, err)
		rsp.ErrRsp(c, -4, "failed to save FPS")
		return
	}
	rsp.OkRsp(c)
	log.Debugf("update FPS to %d", req.Fps)
}

func kvmCommVideoModeFromStreamMode(mode string) (string, bool) {
	switch mode {
	case "h264-webrtc", "h264-direct":
		return common.KvmCommVideoModeH264, true
	case "h265-webrtc", "h265-direct":
		return common.KvmCommVideoModeH265, true
	default:
		return "", false
	}
}

func syncKvmCommVideoMode(mode string) error {
	config, err := common.GetKvmCommConfig()
	if err != nil {
		return err
	}

	if strings.ToLower(config.Device.Mode) == mode {
		return nil
	}

	return common.SetKvmCommVideoMode(mode, true, kvmCommVideoModeRestartDelay)
}

func streamModeResponseFromKvmCommError(err error) (int, string) {
	switch {
	case errors.Is(err, common.ErrKvmCommLoadConfigFailed):
		return -3, "failed to load kvmcomm config"
	case errors.Is(err, common.ErrKvmCommInvalidVideoMode):
		return -4, "invalid kvmcomm video mode"
	case errors.Is(err, common.ErrKvmCommUpdateConfigFailed):
		return -5, "failed to update kvmcomm config"
	case errors.Is(err, common.ErrKvmCommSaveConfigFailed):
		return -6, "failed to save kvmcomm config"
	case errors.Is(err, common.ErrKvmCommRestartAndRestoreFailed):
		return -9, "failed to restart kvm_vin process and restore config"
	case errors.Is(err, common.ErrKvmCommRestartAfterRestoreFailed):
		return -10, "failed to restart kvm_vin process after restoring config"
	case errors.Is(err, common.ErrKvmCommRestartFailed):
		return -7, "failed to restart kvm_vin process"
	default:
		return -8, "failed to update kvmcomm video mode"
	}
}
