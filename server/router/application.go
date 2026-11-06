package router

import (
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/application"
	"NanoKVM-Server/service/extensions/tailscale"

	"github.com/gin-gonic/gin"
)

func applicationRouter(r *gin.Engine) {
	service := application.NewService(tailscale.DefaultService())
	api := r.Group("/api").Use(middleware.CheckToken())

	api.GET("/application/version", service.GetVersion)            // get application version
	api.POST("/application/update", service.Update)                // update application
	api.POST("/application/update/offline", service.UpdateOffline) // upload an offline update
	api.GET("/application/update/status", service.GetUpdateStatus) // get authoritative update status
	api.POST("/application/update/status/acknowledge", service.AcknowledgeUpdateStatus)
	api.GET("/application/update/server", service.GetUpdateServer)
	api.POST("/application/update/server", service.SetUpdateServer)
	api.GET("/application/signature-verification", service.GetSignatureVerification)
	api.POST("/application/signature-verification", service.SetSignatureVerification)
	api.GET("/application/region", service.GetRegion)
	api.POST("/application/region", service.SetRegion)
	api.POST("/application/features/sync", service.SyncServerFeatures)

	api.GET("/application/preview", service.GetPreview)  // get preview updates state
	api.POST("/application/preview", service.SetPreview) // set preview updates state
}
