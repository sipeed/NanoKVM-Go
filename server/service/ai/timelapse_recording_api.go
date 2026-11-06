package ai

import (
	"errors"
	"time"

	"NanoKVM-Server/common"
	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

var (
	getScreenTimelapseRecordingConfig = common.GetKvmCommConfig
	setScreenTimelapseRecordingQP     = common.SetKvmCommTimelapseQP
)

func (s *Service) GetScreenTimelapseRecording(c *gin.Context) {
	var rsp proto.Response

	config, err := getScreenTimelapseRecordingConfig()
	if err != nil {
		log.Errorf("failed to load screen timelapse recording settings: %v", err)
		rsp.ErrRsp(c, -1, "failed to get screen timelapse recording settings")
		return
	}

	rsp.OkRspWithData(c, proto.GetScreenTimelapseRecordingRsp{
		QP: config.Timelapse.QP,
	})
}

func (s *Service) SetScreenTimelapseRecording(c *gin.Context) {
	var req proto.SetScreenTimelapseRecordingReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	qp := *req.QP
	changed, restarted, err := setScreenTimelapseRecordingQP(
		qp,
		0*time.Second,
	)
	if err != nil {
		log.Errorf("failed to set screen timelapse QP to %d: %v", qp, err)
		code, message := screenTimelapseRecordingResponseFromKvmCommError(err)
		rsp.ErrRsp(c, code, message)
		return
	}

	rsp.OkRspWithData(c, proto.SetScreenTimelapseRecordingRsp{
		QP:        qp,
		Changed:   changed,
		Restarted: restarted,
	})
}

func screenTimelapseRecordingResponseFromKvmCommError(err error) (int, string) {
	switch {
	case errors.Is(err, common.ErrKvmCommInvalidTimelapseSetting):
		return -2, "invalid screen timelapse QP"
	case errors.Is(err, common.ErrKvmCommLoadConfigFailed):
		return -3, "failed to load screen timelapse recording settings"
	case errors.Is(err, common.ErrKvmCommRestartFailed),
		errors.Is(err, common.ErrKvmCommRestartAndRestoreFailed),
		errors.Is(err, common.ErrKvmCommRestartAfterRestoreFailed):
		return -5, "failed to restart screen timelapse recording"
	case errors.Is(err, common.ErrKvmCommUpdateConfigFailed),
		errors.Is(err, common.ErrKvmCommSaveConfigFailed):
		return -4, "failed to save screen timelapse recording settings"
	default:
		return -4, "failed to save screen timelapse recording settings"
	}
}
