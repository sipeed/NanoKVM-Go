package router

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHidRouterRegistersPointerModeEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	hidRouter(router)

	found := map[string]bool{}
	for _, route := range router.Routes() {
		if route.Path != "/api/hid/pointer-mode" {
			continue
		}
		if route.Method != http.MethodGet && route.Method != http.MethodPost {
			t.Fatalf("pointer-mode route method = %s, want GET or POST", route.Method)
		}
		found[route.Method] = true
	}
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		if !found[method] {
			t.Fatalf("authenticated HID router did not register pointer-mode %s route", method)
		}
	}
}
