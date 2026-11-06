package router

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestApplicationRouterRegistersUpdateStatusAcknowledgeAsMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	applicationRouter(router)

	found := false
	for _, route := range router.Routes() {
		if route.Path != "/api/application/update/status/acknowledge" {
			continue
		}
		if route.Method != http.MethodPost {
			t.Fatalf("acknowledge route method = %s, want POST", route.Method)
		}
		found = true
	}
	if !found {
		t.Fatal("authenticated application router did not register update status acknowledge route")
	}
}

func TestApplicationRouterRegistersRegionRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	applicationRouter(router)

	methods := map[string]bool{}
	for _, route := range router.Routes() {
		if route.Path != "/api/application/region" {
			continue
		}
		methods[route.Method] = true
	}
	if !methods[http.MethodGet] || !methods[http.MethodPost] {
		t.Fatalf("region routes = %v, want GET and POST", methods)
	}
}
