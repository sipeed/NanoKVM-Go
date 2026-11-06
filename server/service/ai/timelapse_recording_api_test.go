package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"NanoKVM-Server/common"
	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

type screenTimelapseRecordingAPIEnvelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func withScreenTimelapseRecordingAPITest(t *testing.T) {
	t.Helper()
	oldGet := getScreenTimelapseRecordingConfig
	oldSet := setScreenTimelapseRecordingQP
	t.Cleanup(func() {
		getScreenTimelapseRecordingConfig = oldGet
		setScreenTimelapseRecordingQP = oldSet
	})
}

func newScreenTimelapseRecordingAPIContext(method, target, body string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(method, target, bytes.NewBufferString(body))
	context.Request.Header.Set("Content-Type", "application/json")
	return context, recorder
}

func decodeScreenTimelapseRecordingAPIEnvelope(t *testing.T, recorder *httptest.ResponseRecorder) screenTimelapseRecordingAPIEnvelope {
	t.Helper()
	var envelope screenTimelapseRecordingAPIEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v body=%s", err, recorder.Body.String())
	}
	return envelope
}

func TestGetScreenTimelapseRecordingReturnsPersistedQP(t *testing.T) {
	withScreenTimelapseRecordingAPITest(t)
	getScreenTimelapseRecordingConfig = func() (*common.KvmCommConfig, error) {
		return &common.KvmCommConfig{
			Timelapse: common.KvmCommTimelapseConfig{QP: 40},
		}, nil
	}

	context, recorder := newScreenTimelapseRecordingAPIContext(http.MethodGet, "/api/ai/screen-timelapse/recording", "")
	new(Service).GetScreenTimelapseRecording(context)
	envelope := decodeScreenTimelapseRecordingAPIEnvelope(t, recorder)
	if envelope.Code != 0 {
		t.Fatalf("code=%d msg=%q", envelope.Code, envelope.Msg)
	}
	var data proto.GetScreenTimelapseRecordingRsp
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if data.QP != 40 {
		t.Fatalf("QP=%d, want 40", data.QP)
	}
}

func TestSetScreenTimelapseRecordingReturnsApplyResult(t *testing.T) {
	withScreenTimelapseRecordingAPITest(t)
	var gotQP int
	setScreenTimelapseRecordingQP = func(qp int, delay time.Duration) (bool, bool, error) {
		gotQP = qp
		if delay != 0 {
			t.Fatalf("restart delay=%s, want 0", delay)
		}
		return true, true, nil
	}

	context, recorder := newScreenTimelapseRecordingAPIContext(
		http.MethodPut,
		"/api/ai/screen-timelapse/recording",
		`{"qp":36}`,
	)
	new(Service).SetScreenTimelapseRecording(context)
	envelope := decodeScreenTimelapseRecordingAPIEnvelope(t, recorder)
	if envelope.Code != 0 || gotQP != 36 {
		t.Fatalf("code=%d msg=%q gotQP=%d", envelope.Code, envelope.Msg, gotQP)
	}
	var data proto.SetScreenTimelapseRecordingRsp
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		t.Fatalf("decode data: %v", err)
	}
	if data.QP != 36 || !data.Changed || !data.Restarted {
		t.Fatalf("unexpected response data: %#v", data)
	}
}

func TestSetScreenTimelapseRecordingRejectsInvalidRequest(t *testing.T) {
	withScreenTimelapseRecordingAPITest(t)
	context, recorder := newScreenTimelapseRecordingAPIContext(
		http.MethodPut,
		"/api/ai/screen-timelapse/recording",
		`{}`,
	)
	new(Service).SetScreenTimelapseRecording(context)
	envelope := decodeScreenTimelapseRecordingAPIEnvelope(t, recorder)
	if envelope.Code != -1 {
		t.Fatalf("code=%d, want -1", envelope.Code)
	}
}

func TestSetScreenTimelapseRecordingMapsKvmCommErrors(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		code int
	}{
		{name: "range", err: common.ErrKvmCommInvalidTimelapseSetting, code: -2},
		{name: "load", err: common.ErrKvmCommLoadConfigFailed, code: -3},
		{name: "save", err: common.ErrKvmCommSaveConfigFailed, code: -4},
		{name: "restart", err: common.ErrKvmCommRestartFailed, code: -5},
	} {
		t.Run(test.name, func(t *testing.T) {
			withScreenTimelapseRecordingAPITest(t)
			setScreenTimelapseRecordingQP = func(int, time.Duration) (bool, bool, error) {
				return false, false, fmt.Errorf("wrapped: %w", test.err)
			}

			context, recorder := newScreenTimelapseRecordingAPIContext(
				http.MethodPut,
				"/api/ai/screen-timelapse/recording",
				`{"qp":42}`,
			)
			new(Service).SetScreenTimelapseRecording(context)
			envelope := decodeScreenTimelapseRecordingAPIEnvelope(t, recorder)
			if envelope.Code != test.code {
				t.Fatalf("code=%d, want %d msg=%q", envelope.Code, test.code, envelope.Msg)
			}
		})
	}
}
