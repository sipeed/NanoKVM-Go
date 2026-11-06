package stream

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"NanoKVM-Server/common"

	"github.com/gin-gonic/gin"
)

type getEncoderResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		RateControl string `json:"rate_control"`
		BitRate     int    `json:"bitrate"`
		GOP         int    `json:"gop"`
		FPS         int    `json:"fps"`
	} `json:"data"`
}

func TestGetEncoderReturnsPersistedSettings(t *testing.T) {
	setupGetEncoderTest(t, func() (*common.KvmCommConfig, error) {
		return &common.KvmCommConfig{
			Encoder: common.KvmCommEncoderConfig{
				RateControl: common.KvmCommEncoderRateControlCBR,
				BitRate:     12000,
				GOP:         240,
				FPS:         30,
			},
		}, nil
	})

	rsp := getEncoder(t)
	if rsp.Code != 0 || rsp.Msg != "success" {
		t.Fatalf("unexpected response: %#v", rsp)
	}
	if rsp.Data.RateControl != "cbr" || rsp.Data.BitRate != 12000 || rsp.Data.GOP != 240 || rsp.Data.FPS != 30 {
		t.Fatalf("unexpected encoder settings: %#v", rsp.Data)
	}
}

func TestGetEncoderReturnsErrorWhenConfigCannotBeLoaded(t *testing.T) {
	setupGetEncoderTest(t, func() (*common.KvmCommConfig, error) {
		return nil, errors.New("load failed")
	})

	rsp := getEncoder(t)
	if rsp.Code != -1 || rsp.Msg != "failed to get encoder settings" {
		t.Fatalf("unexpected response: %#v", rsp)
	}
}

func TestGetEncoderWaitsForEncoderSettingsUpdate(t *testing.T) {
	readStarted := make(chan struct{})
	setupGetEncoderTest(t, func() (*common.KvmCommConfig, error) {
		close(readStarted)
		return &common.KvmCommConfig{}, nil
	})

	encoderSettingsMu.Lock()
	locked := true
	t.Cleanup(func() {
		if locked {
			encoderSettingsMu.Unlock()
		}
	})

	done := make(chan struct{})
	handlerStarted := make(chan struct{})
	go func() {
		gin.SetMode(gin.TestMode)
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/api/stream/encoder", nil)
		close(handlerStarted)
		NewService().GetEncoder(ctx)
		close(done)
	}()
	<-handlerStarted

	select {
	case <-readStarted:
		t.Fatal("encoder config was read before the settings update lock was released")
	case <-time.After(100 * time.Millisecond):
	}

	encoderSettingsMu.Unlock()
	locked = false

	select {
	case <-readStarted:
	case <-time.After(time.Second):
		t.Fatal("encoder config was not read after the settings update lock was released")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("GetEncoder did not finish after the settings update lock was released")
	}
}

func setupGetEncoderTest(t *testing.T, get func() (*common.KvmCommConfig, error)) {
	t.Helper()

	oldGet := getKvmCommConfig
	getKvmCommConfig = get
	t.Cleanup(func() {
		getKvmCommConfig = oldGet
	})
}

func getEncoder(t *testing.T) getEncoderResponse {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/stream/encoder", nil)
	NewService().GetEncoder(ctx)

	var rsp getEncoderResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("decode response %q: %v", recorder.Body.String(), err)
	}
	return rsp
}
