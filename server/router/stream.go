package router

import (
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/media"
	"NanoKVM-Server/service/stream"
	"NanoKVM-Server/service/stream/direct"
	h265direct "NanoKVM-Server/service/stream/h265/direct"
	h265webrtc "NanoKVM-Server/service/stream/h265/webrtc"
	"NanoKVM-Server/service/stream/webrtc"

	"github.com/gin-gonic/gin"
)

func streamRouter(r *gin.Engine) {
	service := stream.NewService()

	api := r.Group("/api").Use(middleware.CheckToken())

	api.POST("/stream/rate-control", service.SetRateControlMode) // set rate control mode
	api.POST("/stream/mode", service.SetMode)                    // set stream mode
	api.POST("/stream/quality", service.SetQuality)              // set stream bit-rate
	api.POST("/stream/gop", service.SetGop)                      // set stream gop
	api.POST("/stream/fps", service.SetFps)                      // set stream fps
	api.GET("/stream/encoder", service.GetEncoder)               // get encoder settings
	api.GET("/stream/webcam", stream.ConnectWebcam)

	api.GET("/stream/h264/direct", direct.Connect)     // h264 stream (direct)
	api.GET("/stream/h264/webrtc", webrtc.Connect)     // h264 stream (webrtc)
	api.GET("/stream/h265/direct", h265direct.Connect) // h265 stream (direct)
	api.GET("/stream/h265/webrtc", h265webrtc.Connect) // h265 stream (webrtc)

	// This endpoint is intentionally outside the normal JWT-protected group:
	// it accepts only an opaque, short-lived media-session capability minted by
	// MCP or the authenticated control API. The WebRTC media track remains the
	// data plane; the token is never a replacement for the web login token.
	r.GET("/api/media/webrtc", requireMediaSession(), webrtc.Connect)
	r.GET("/api/media/audio/webrtc", requireMediaSession(), webrtc.ConnectMediaAudio)
}

func requireMediaSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("media_session")
		session, ok := media.DefaultManager().Get(token)
		if !ok {
			c.JSON(401, gin.H{"error": "invalid or expired media session"})
			c.Abort()
			return
		}
		c.Set("mediaSession", session)
		c.Next()
	}
}
