package hid

import (
	"context"
	"os"
	"os/exec"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	ModeNormal  = "normal"
	ModeHidOnly = "hid-only"
	HidOnlyFlag = "/dev/shm/tmp/hid_only"
	USBScript   = "/opt/scripts/usbdev.sh"
)

var hidModeCmdMap = map[string]string{
	ModeNormal:  "restart",
	ModeHidOnly: "hid-only",
}

func (s *Service) GetHidMode(c *gin.Context) {
	var rsp proto.Response

	mode := getHidMode()

	rsp.OkRspWithData(c, &proto.GetHidModeRsp{
		Mode: mode,
	})
	log.Debugf("get hid mode: %s", mode)
}

func (s *Service) SetHidMode(c *gin.Context) {
	var req proto.SetHidModeReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	arg, ok := hidModeCmdMap[req.Mode]
	if !ok {
		rsp.ErrRsp(c, -2, "invalid arguments")
		return
	}

	LockUsbGadgetMutation()
	defer UnlockUsbGadgetMutation()
	if mode := getHidMode(); req.Mode == mode {
		rsp.OkRsp(c)
		return
	}

	h := GetHid()
	if err := h.ResetInputRouting("HID mode change"); err != nil {
		log.Warnf("failed to release routed input before HID mode change: %s", err)
	}
	h.Lock()
	h.CloseNoLock()
	defer func() {
		h.OpenNoLock()
		h.Unlock()
	}()

	if err := exec.Command("bash", USBScript, arg).Run(); err != nil {
		rsp.ErrRsp(c, -3, "failed to set hid mode")
		log.Errorf("Failed to execute script: %v: %s", USBScript, err)
		return
	}

	time.Sleep(3 * time.Second)

	rsp.OkRsp(c)
	log.Debugf("set hid mode: %s", req.Mode)
}

func (s *Service) Reset(c *gin.Context) {
	var rsp proto.Response

	LockUsbGadgetMutation()
	defer UnlockUsbGadgetMutation()

	pointerModeTransaction := s.pointerModeTransaction()
	if pointerModeTransaction.manager.status().Degraded {
		status, err := pointerModeTransaction.recoverDegradedUSBLocked(context.Background())
		if err != nil {
			rsp.ErrRspWithData(c, -3, "failed to recover degraded HID state", pointerModeStatusResponse(status))
			log.Errorf("failed to recover degraded HID state: %s", err)
			return
		}

		rsp.OkRsp(c)
		log.Infof("recovered degraded HID state: pointer mode %s", status.Mode)
		return
	}

	mode := getHidMode()
	arg, ok := hidModeCmdMap[mode]
	if !ok {
		rsp.ErrRsp(c, -1, "invalid hid mode")
		log.Errorf("invalid hid mode: %s", mode)
		return
	}

	h := GetHid()
	if err := h.ResetInputRouting("HID reset"); err != nil {
		log.Warnf("failed to release routed input before HID reset: %s", err)
	}
	h.Lock()
	h.CloseNoLock()
	defer func() {
		h.OpenNoLock()
		h.Unlock()
	}()

	if err := exec.Command("bash", USBScript, arg).Run(); err != nil {
		rsp.ErrRsp(c, -2, "failed to reset")
		log.Errorf("Failed to execute script: %v: %s", USBScript, err)
		return
	}

	time.Sleep(3 * time.Second)

	rsp.OkRsp(c)
	log.Debugf("reset hid success")
}

func getHidMode() string {
	_, err := os.Stat(HidOnlyFlag)
	if err != nil {
		return ModeNormal
	}
	return ModeHidOnly
}
