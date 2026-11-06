package router

import (
	"testing"

	"NanoKVM-Server/service/extensions/tailscale"

	"github.com/gin-gonic/gin"
)

func TestTailscaleRoutesExcludeApplicationOwnedRegion(t *testing.T) {
	previousConstructor := newExtensionsTailscaleService
	t.Cleanup(func() {
		newExtensionsTailscaleService = previousConstructor
	})
	newExtensionsTailscaleService = func() *tailscale.Service { return &tailscale.Service{} }
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	extensionsRouter(engine)
	want := map[string]bool{
		"POST /api/extensions/tailscale/sipeed/enable":              false,
		"POST /api/extensions/tailscale/sipeed/logout":              false,
		"POST /api/extensions/tailscale/sipeed/claim-code":          false,
		"GET /api/extensions/tailscale/sipeed/pending":              false,
		"POST /api/extensions/tailscale/sipeed/pending/:id/approve": false,
		"POST /api/extensions/tailscale/sipeed/pending/:id/reject":  false,
		"POST /api/extensions/tailscale/sipeed/authorize":           false,
		"GET /api/extensions/tailscale/sipeed/devices":              false,
		"POST /api/extensions/tailscale/sipeed/devices/:id/name":    false,
		"DELETE /api/extensions/tailscale/sipeed/devices/:id":       false,
	}
	for _, route := range engine.Routes() {
		key := route.Method + " " + route.Path
		if route.Path == "/api/extensions/tailscale/region" {
			t.Fatalf("removed runtime region route is still registered: %s", key)
		}
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Fatalf("route %s was not registered", route)
		}
	}
}
