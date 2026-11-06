package router

import (
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/ai"

	"github.com/gin-gonic/gin"
)

func aiRouter(r *gin.Engine) {
	service := ai.NewService()

	api := r.Group("/api").Use(middleware.CheckToken())
	api.GET("/ai/services", service.GetServices)
	api.GET("/ai/services/events", service.StreamServiceStateEvents)
	api.GET("/ai/config", service.GetConfig)
	api.POST("/ai/config", service.SetConfig)
	api.POST("/ai/mcp/key", service.GetMCPAPIKey)
	api.POST("/ai/mcp/key/copy", service.CopyMCPAPIKey)
	api.POST("/ai/mcp/key/regenerate", service.RegenerateMCPAPIKey)
	api.GET("/ai/memory-fabric", service.GetMemoryFabric)
	api.POST("/ai/memory-fabric", service.SetMemoryFabric)
	api.GET("/ai/memory-fabric/config", service.GetMemoryFabricConfig)
	api.PUT("/ai/memory-fabric/config", service.SetMemoryFabricConfig)
	api.GET("/ai/screen-timelapse/state", service.GetScreenTimelapseState)
	api.POST("/ai/screen-timelapse/state", service.SetScreenTimelapseState)
	api.GET("/ai/screen-timelapse/recording", service.GetScreenTimelapseRecording)
	api.PUT("/ai/screen-timelapse/recording", service.SetScreenTimelapseRecording)
	api.POST("/ai/screen-timelapse/access", service.AuthorizeScreenTimelapsePlayback)
	api.GET("/ai/screen-timelapse/retention", service.GetScreenTimelapseRetention)
	api.PUT("/ai/screen-timelapse/retention", service.SetScreenTimelapseRetention)
	api.POST("/ai/screen-timelapse/cleanup", service.CleanupScreenTimelapseRetention)
	api.GET("/ai/screen-timelapse/days", service.GetScreenTimelapseDays)
	api.DELETE("/ai/screen-timelapse/day", service.DeleteScreenTimelapseDay)
	api.GET("/ai/screen-timelapse/frames", service.GetScreenTimelapseFrames)
	api.GET("/ai/screen-timelapse/video", service.GetScreenTimelapseVideo)
	api.GET("/ai/screen-timelapse/search", service.SearchScreenTimelapse)

	mcpHandler := ai.NewMCPHandler()
	r.Any("/api/mcp", ai.MCPAPIKeyMiddleware(), gin.WrapH(mcpHandler))
}
