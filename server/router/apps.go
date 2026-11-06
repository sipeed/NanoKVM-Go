package router

import (
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/apps"

	"github.com/gin-gonic/gin"
)

func appsRouter(r *gin.Engine) {
	service := apps.NewService()
	api := r.Group("/api/apps").Use(middleware.CheckToken())
	api.GET("", service.List)
	api.POST("/inspect", service.Inspect)
	api.POST("/install", service.Install)
	api.POST("/install/stream", service.InstallStream)
	api.POST("/install/remote", service.InstallRemote)
	api.POST("/install/remote/stream", service.InstallRemoteStream)
	api.GET("/:id/download", service.Download)
	api.PUT("/:id/env", service.UpdateEnv)
	api.DELETE("/:id", service.Remove)
	api.GET("/servers", service.GetServers)
	api.PUT("/servers", service.PutServers)
	api.GET("/servers/:id/catalog", service.Catalog)
}
