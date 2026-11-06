package inputrouter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

func TestGetStatusExposesRegistryAndRuntime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := New(&fakeOutput{}, NewTraceHub(8))
	service := &Service{router: router}
	context, recorder := testContext(http.MethodGet, "/api/hid/input-router/status")

	service.GetStatus(context)

	var response struct {
		Code int    `json:"code"`
		Data Status `json:"data"`
	}
	decodeResponse(t, recorder, &response)
	if response.Code != 0 || response.Data.ProtocolVersion != ProtocolVersion || response.Data.MaxContacts != MaxContacts {
		t.Fatalf("status response = %+v", response)
	}
	if len(response.Data.Runtime.Routes) != 5 {
		t.Fatalf("status routes = %+v", response.Data.Runtime.Routes)
	}
}

func testContext(method, target string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(method, target, nil)
	return context, recorder
}

func decodeResponse(t *testing.T, recorder *httptest.ResponseRecorder, target interface{}) {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), target); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	var envelope proto.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil || envelope.Msg != "success" {
		t.Fatalf("response envelope = %+v, error %v", envelope, err)
	}
}
