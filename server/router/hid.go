package router

import (
	"github.com/gin-gonic/gin"

	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/hid"
	"NanoKVM-Server/service/inputrouter"
)

func hidRouter(r *gin.Engine) {
	service := hid.NewService()
	inputRouterService := inputrouter.NewService()
	api := r.Group("/api").Use(middleware.CheckToken())

	api.POST("/hid/reset", service.Reset) // reset hid
	api.POST("/hid/paste", service.Paste) // paste

	api.GET("/hid/keyboard-layout", service.GetKeyboardLayout)
	api.POST("/hid/keyboard-layout", service.SetKeyboardLayout)

	api.GET("/hid/shortcuts", service.GetShortcuts)     // get shortcuts
	api.POST("/hid/shortcut", service.AddShortcut)      // add shortcut
	api.DELETE("/hid/shortcut", service.DeleteShortcut) // delete shortcut

	api.GET("/hid/shortcut/leader-key", service.GetLeaderKey)  // set shortcut leader key
	api.POST("/hid/shortcut/leader-key", service.SetLeaderKey) // set shortcut leader key

	api.GET("/hid/mode", service.GetHidMode)  // get hid mode
	api.POST("/hid/mode", service.SetHidMode) // set hid mode
	api.GET("/hid/pointer-mode", service.GetPointerMode)
	api.POST("/hid/pointer-mode", service.SetPointerMode)

	api.GET("/hid/input-router/status", inputRouterService.GetStatus)
	api.GET("/hid/input-router/trace-stream", inputRouterService.StreamTraces)
}
