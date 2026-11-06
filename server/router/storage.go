package router

import (
	"github.com/gin-gonic/gin"

	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/storage"
)

func storageRouter(r *gin.Engine) {
	service := storage.NewService()
	api := r.Group("/api").Use(middleware.CheckToken())

	api.GET("/storage/media", service.GetMedia)
	api.GET("/storage/media/state", service.GetMediaState)
	api.GET("/storage/media/status", service.GetMediaStatus)
	api.GET("/storage/media/transfer", service.GetMediaTransferStatus)
	api.POST("/storage/media/mount", service.MountMedia)
	api.POST("/storage/media/eject", service.EjectMedia)
	api.POST("/storage/media/delete", service.DeleteMedia)
	api.GET("/storage/media/download", service.DownloadMedia)
	api.GET("/storage/media/files", service.ListMediaFiles)
	api.GET("/storage/media/files/download", service.DownloadMediaFile)
	api.POST("/storage/media/files/upload", service.UploadMediaFile)
	api.POST("/storage/media/files/upload/cancel", service.CancelMediaFileUpload)
	api.POST("/storage/media/files/mkdir", service.MkdirMediaFile)
	api.POST("/storage/media/files/delete", service.DeleteMediaFile)

	api.GET("/storage/image", service.GetImages)                        // get image list
	api.GET("/storage/image/mounted", service.GetMountedImage)          // get mounted image
	api.POST("/storage/image/upload", service.UploadImage)              // upload image
	api.POST("/storage/image/upload/cancel", service.CancelImageUpload) // cancel image upload
	api.POST("/storage/image/mount", service.MountImage)                // mount image
	api.POST("/storage/image/delete", service.DeleteImage)              // delete image

	api.POST("/storage/download/image", service.DownloadImage)              // download image
	api.POST("/storage/download/image/pause", service.PauseImageDownload)   // pause image download
	api.POST("/storage/download/image/resume", service.ResumeImageDownload) // resume image download
	api.POST("/storage/download/image/cancel", service.CancelImageDownload) // cancel image download
	api.GET("/storage/download/image/status", service.StatusImage)          // download image
	api.GET("/storage/download/image/enabled", service.ImageEnabled)        // download image
}
