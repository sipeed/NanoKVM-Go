package stream

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"NanoKVM-Server/common"

	"github.com/gin-gonic/gin"
)

type streamSettingResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func TestEncoderSettingsPersistAfterRuntimeSuccess(t *testing.T) {
	screen := setupEncoderSettingsTest(t)
	screen.RateControlMode = common.RATE_CONTROL_VBR
	screen.BitRate = 8000
	screen.GOP = 50
	screen.FPS = 0

	var calls []string
	setKvmVisionRateControl = func(mode uint8) int {
		calls = append(calls, fmt.Sprintf("runtime:rate-control:%d", mode))
		return 0
	}
	setKvmVisionGOP = func(gop uint16) int {
		calls = append(calls, fmt.Sprintf("runtime:gop:%d", gop))
		return 0
	}
	setKvmVisionBitRate = func(bitRate uint16) int {
		calls = append(calls, fmt.Sprintf("runtime:bitrate:%d", bitRate))
		return 0
	}
	setKvmVisionFPS = func(fps uint8) int {
		calls = append(calls, fmt.Sprintf("runtime:fps:%d", fps))
		return 0
	}
	saveKvmCommEncoderRateControl = func(mode string) error {
		calls = append(calls, "persist:rate-control:"+mode)
		return nil
	}
	saveKvmCommEncoderBitRate = func(bitRate int) error {
		calls = append(calls, fmt.Sprintf("persist:bitrate:%d", bitRate))
		return nil
	}
	saveKvmCommEncoderGOP = func(gop int) error {
		calls = append(calls, fmt.Sprintf("persist:gop:%d", gop))
		return nil
	}
	saveKvmCommEncoderFPS = func(fps int) error {
		calls = append(calls, fmt.Sprintf("persist:fps:%d", fps))
		return nil
	}

	service := NewService()
	requests := []struct {
		name    string
		handler gin.HandlerFunc
		body    string
	}{
		{name: "rate control", handler: service.SetRateControlMode, body: `{"mode":"CBR"}`},
		{name: "bitrate", handler: service.SetQuality, body: `{"quality":12000}`},
		{name: "GOP", handler: service.SetGop, body: `{"gop":240}`},
		{name: "FPS", handler: service.SetFps, body: `{"fps":30}`},
	}
	for _, request := range requests {
		t.Run(request.name, func(t *testing.T) {
			rsp := postStreamSetting(t, request.handler, request.body)
			if rsp.Code != 0 {
				t.Fatalf("unexpected response: %#v", rsp)
			}
		})
	}

	wantCalls := []string{
		"runtime:rate-control:0",
		"persist:rate-control:cbr",
		"runtime:bitrate:12000",
		"persist:bitrate:12000",
		"runtime:gop:240",
		"persist:gop:240",
		"runtime:fps:30",
		"persist:fps:30",
	}
	if strings.Join(calls, "\n") != strings.Join(wantCalls, "\n") {
		t.Fatalf("unexpected call order:\n%s\nwant:\n%s", strings.Join(calls, "\n"), strings.Join(wantCalls, "\n"))
	}
	if screen.RateControlMode != common.RATE_CONTROL_CBR || screen.BitRate != 12000 || screen.GOP != 240 || screen.FPS != 30 {
		t.Fatalf("unexpected screen state: rate=%d bitrate=%d GOP=%d FPS=%d",
			screen.RateControlMode, screen.BitRate, screen.GOP, screen.FPS)
	}
}

func TestEncoderRuntimeFailureDoesNotPersist(t *testing.T) {
	screen := setupEncoderSettingsTest(t)
	screen.GOP = 50

	persistCalls := 0
	setKvmVisionGOP = func(uint16) int { return -1 }
	saveKvmCommEncoderGOP = func(int) error {
		persistCalls++
		return nil
	}

	rsp := postStreamSetting(t, NewService().SetGop, `{"gop":240}`)
	if rsp.Code != -3 {
		t.Fatalf("expected runtime failure, got %#v", rsp)
	}
	if persistCalls != 0 {
		t.Fatalf("expected no persistence after runtime failure, got %d calls", persistCalls)
	}
	if screen.GOP != 50 {
		t.Fatalf("expected GOP cache to remain 50, got %d", screen.GOP)
	}
}

func TestEncoderPersistenceFailureKeepsAppliedRuntimeState(t *testing.T) {
	screen := setupEncoderSettingsTest(t)
	screen.RateControlMode = common.RATE_CONTROL_VBR

	var runtimeModes []uint8
	setKvmVisionRateControl = func(mode uint8) int {
		runtimeModes = append(runtimeModes, mode)
		return 0
	}
	saveKvmCommEncoderRateControl = func(string) error {
		return errors.New("save failed")
	}

	rsp := postStreamSetting(t, NewService().SetRateControlMode, `{"mode":"cbr"}`)
	if rsp.Code != -4 {
		t.Fatalf("expected persistence failure, got %#v", rsp)
	}
	wantModes := []uint8{common.RATE_CONTROL_CBR}
	if len(runtimeModes) != len(wantModes) || runtimeModes[0] != wantModes[0] {
		t.Fatalf("unexpected runtime modes: got %v want %v", runtimeModes, wantModes)
	}
	if screen.RateControlMode != common.RATE_CONTROL_CBR {
		t.Fatalf("expected rate-control cache to reflect applied CBR, got %d", screen.RateControlMode)
	}
}

func TestBitRateRuntimeFailureDoesNotPersist(t *testing.T) {
	screen := setupEncoderSettingsTest(t)
	screen.BitRate = 8000

	persistCalls := 0
	setKvmVisionBitRate = func(uint16) int { return -1 }
	saveKvmCommEncoderBitRate = func(int) error {
		persistCalls++
		return nil
	}

	rsp := postStreamSetting(t, NewService().SetQuality, `{"quality":12000}`)
	if rsp.Code != -3 {
		t.Fatalf("expected runtime failure, got %#v", rsp)
	}
	if persistCalls != 0 {
		t.Fatalf("expected no persistence after runtime failure, got %d calls", persistCalls)
	}
	if screen.BitRate != 8000 {
		t.Fatalf("expected bitrate cache to remain 8000, got %d", screen.BitRate)
	}
}

func TestBitRatePersistenceFailureKeepsAppliedRuntimeState(t *testing.T) {
	screen := setupEncoderSettingsTest(t)
	screen.BitRate = 8000
	setKvmVisionBitRate = func(uint16) int { return 0 }
	saveKvmCommEncoderBitRate = func(int) error {
		return errors.New("save failed")
	}

	rsp := postStreamSetting(t, NewService().SetQuality, `{"quality":12000}`)
	if rsp.Code != -4 {
		t.Fatalf("expected persistence failure, got %#v", rsp)
	}
	if screen.BitRate != 12000 {
		t.Fatalf("expected bitrate cache to reflect applied 12000, got %d", screen.BitRate)
	}
}

func setupEncoderSettingsTest(t *testing.T) *common.Screen {
	t.Helper()

	oldSetRateControl := setKvmVisionRateControl
	oldSetGOP := setKvmVisionGOP
	oldSetBitRate := setKvmVisionBitRate
	oldSetFPS := setKvmVisionFPS
	oldSaveRateControl := saveKvmCommEncoderRateControl
	oldSaveBitRate := saveKvmCommEncoderBitRate
	oldSaveGOP := saveKvmCommEncoderGOP
	oldSaveFPS := saveKvmCommEncoderFPS

	setKvmVisionRateControl = func(uint8) int { return 0 }
	setKvmVisionGOP = func(uint16) int { return 0 }
	setKvmVisionBitRate = func(uint16) int { return 0 }
	setKvmVisionFPS = func(uint8) int { return 0 }
	saveKvmCommEncoderRateControl = func(string) error { return nil }
	saveKvmCommEncoderBitRate = func(int) error { return nil }
	saveKvmCommEncoderGOP = func(int) error { return nil }
	saveKvmCommEncoderFPS = func(int) error { return nil }

	screen := common.GetScreen()
	oldRateControl := screen.RateControlMode
	oldBitRate := screen.BitRate
	oldGOP := screen.GOP
	oldFPS := screen.FPS

	t.Cleanup(func() {
		setKvmVisionRateControl = oldSetRateControl
		setKvmVisionGOP = oldSetGOP
		setKvmVisionBitRate = oldSetBitRate
		setKvmVisionFPS = oldSetFPS
		saveKvmCommEncoderRateControl = oldSaveRateControl
		saveKvmCommEncoderBitRate = oldSaveBitRate
		saveKvmCommEncoderGOP = oldSaveGOP
		saveKvmCommEncoderFPS = oldSaveFPS
		screen.RateControlMode = oldRateControl
		screen.BitRate = oldBitRate
		screen.GOP = oldGOP
		screen.FPS = oldFPS
	})

	return screen
}

func postStreamSetting(t *testing.T, handler gin.HandlerFunc, body string) streamSettingResponse {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	handler(ctx)

	var rsp streamSettingResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("decode response %q: %v", recorder.Body.String(), err)
	}
	return rsp
}
