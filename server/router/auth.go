package router

import (
	"github.com/gin-gonic/gin"

	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/auth"
)

func authRouter(r *gin.Engine) {
	service := auth.NewService()

	r.POST("/api/auth/login", service.Login) // login
	r.GET("/api/auth/login/2fa-direct", service.GetDirectTwoFactorLoginStatus)

	api := r.Group("/api").Use(middleware.CheckToken())

	api.GET("/auth/password", service.IsPasswordUpdated) // is password updated
	api.GET("/auth/account", service.GetAccount)         // get account
	api.POST("/auth/password", service.ChangePassword)   // change password
	api.POST("/auth/logout", service.Logout)             // logout
	api.GET("/auth/2fa/status", service.TwoFactorStatus)
	api.POST("/auth/2fa/setup", service.TwoFactorSetup)
	api.POST("/auth/2fa/setup/cancel", service.TwoFactorSetupCancel)
	api.POST("/auth/2fa/enable", service.TwoFactorEnable)
	api.POST("/auth/2fa/disable", service.TwoFactorDisable)
	api.POST("/auth/2fa/verify-sensitive-action", service.VerifySensitiveAction)
	api.POST("/auth/2fa/recovery/enable", service.TwoFactorRecoveryEnable)
	api.POST("/auth/2fa/recovery/regenerate", service.TwoFactorRecoveryRegenerate)
	api.GET("/auth/2fa/protection", service.GetTwoFactorProtection)
	api.POST("/auth/2fa/protection/authorize", service.AuthorizeTwoFactorProtection)
	api.POST("/auth/2fa/protection", service.SetTwoFactorProtection)
}
