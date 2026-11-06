package apps

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestInstallStreamIncludesLogsAndFinalResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest("POST", "/api/apps/install/stream", nil)

	streamInstall(ginContext, func(_ context.Context, logWriter io.Writer) (interface{}, error) {
		_, _ = fmt.Fprintln(logWriter, "script output")
		return map[string]string{"directory": "demo"}, nil
	})

	body := recorder.Body.String()
	if !strings.Contains(body, `"type":"log","text":"script output\n"`) ||
		!strings.Contains(body, `"type":"result","data":{"directory":"demo"}`) {
		t.Fatalf("unexpected stream: %s", body)
	}
}

func TestInstallStreamReportsFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest("POST", "/api/apps/install/stream", nil)

	streamInstall(ginContext, func(_ context.Context, _ io.Writer) (interface{}, error) {
		return nil, errors.New("install failed")
	})

	if body := recorder.Body.String(); !strings.Contains(body, `"type":"result","code":-1,"msg":"install failed"`) {
		t.Fatalf("unexpected stream: %s", body)
	}
}
