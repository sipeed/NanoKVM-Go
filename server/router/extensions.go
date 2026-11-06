package router

import (
	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/extensions/cloudflare"
	"NanoKVM-Server/service/extensions/frp"
	"NanoKVM-Server/service/extensions/netbird"
	"NanoKVM-Server/service/extensions/tailscale"
	"NanoKVM-Server/service/extensions/zerotier"

	"github.com/gin-gonic/gin"
)

var newExtensionsTailscaleService = tailscale.DefaultService

func extensionsRouter(r *gin.Engine) {
	api := r.Group("/api/extensions").Use(middleware.CheckToken())
	frpService := frp.DefaultService()
	api.GET("/frp/status", frpService.GetStatus)
	api.GET("/frp/config", frpService.GetConfig)
	api.POST("/frp/config", frpService.SetConfig)
	api.POST("/frp/clear-token", frpService.ClearToken)
	api.POST("/frp/install", frpService.Install)
	api.POST("/frp/uninstall", frpService.Uninstall)
	api.POST("/frp/start", frpService.Start)
	api.POST("/frp/stop", frpService.Stop)
	api.POST("/frp/restart", frpService.Restart)

	cf := cloudflare.DefaultService()
	api.GET("/cloudflare/status", cf.GetStatus)
	api.GET("/cloudflare/config", cf.GetConfig)
	api.POST("/cloudflare/config", cf.SetConfig)
	api.POST("/cloudflare/mode", cf.SetMode)
	api.POST("/cloudflare/clear-token", cf.ClearToken)
	api.POST("/cloudflare/install", cf.Install)
	api.POST("/cloudflare/uninstall", cf.Uninstall)
	api.POST("/cloudflare/start", cf.Start)
	api.POST("/cloudflare/stop", cf.Stop)
	api.POST("/cloudflare/restart", cf.Restart)

	nb := netbird.DefaultService()
	api.GET("/netbird/status", nb.GetStatus)
	api.POST("/netbird/install", nb.Install)
	api.POST("/netbird/uninstall", nb.Uninstall)
	api.POST("/netbird/start", nb.Start)
	api.POST("/netbird/stop", nb.Stop)
	api.POST("/netbird/restart", nb.Restart)
	api.POST("/netbird/login", nb.Login)
	api.POST("/netbird/login/cancel", nb.CancelLogin)
	api.POST("/netbird/logout", nb.Logout)

	zt := zerotier.DefaultService()
	api.GET("/zerotier/status", zt.GetStatus)
	api.POST("/zerotier/install", zt.Install)
	api.POST("/zerotier/uninstall", zt.Uninstall)
	api.POST("/zerotier/start", zt.Start)
	api.POST("/zerotier/stop", zt.Stop)
	api.POST("/zerotier/restart", zt.Restart)
	api.POST("/zerotier/join", zt.Join)
	api.POST("/zerotier/leave", zt.Leave)

	ts := newExtensionsTailscaleService()

	api.POST("/tailscale/install", ts.Install)     // install tailscale
	api.POST("/tailscale/uninstall", ts.Uninstall) // uninstall tailscale
	api.GET("/tailscale/status", ts.GetStatus)     // get tailscale status
	api.POST("/tailscale/up", ts.Up)               // run tailscale up
	api.POST("/tailscale/down", ts.Down)           // run tailscale down
	api.POST("/tailscale/login", ts.Login)         // tailscale login
	api.POST("/tailscale/logout", ts.Logout)       // tailscale logout
	api.POST("/tailscale/start", ts.Start)         // tailscale start
	api.POST("/tailscale/stop", ts.Stop)           // tailscale stop
	api.POST("/tailscale/restart", ts.Restart)     // tailscale restart
	api.POST("/tailscale/exit-node", ts.SetExitNode)
	api.POST("/tailscale/subnet-route", ts.SetSubnetRoute)
	api.POST("/tailscale/sipeed/enable", ts.EnableSipeed)
	api.POST("/tailscale/sipeed/logout", ts.SipeedLogout)
	api.POST("/tailscale/sipeed/disclaimer/revoke", ts.SipeedRevokeDisclaimer)
	api.POST("/tailscale/sipeed/claim-code", ts.SipeedClaimCode)
	api.GET("/tailscale/sipeed/pending", ts.SipeedPending)
	api.POST("/tailscale/sipeed/pending/:id/approve", ts.SipeedApprove)
	api.POST("/tailscale/sipeed/pending/:id/reject", ts.SipeedReject)
	api.POST("/tailscale/sipeed/authorize", ts.SipeedAuthorize)
	api.GET("/tailscale/sipeed/devices", ts.SipeedDevices)
	api.POST("/tailscale/sipeed/devices/:id/name", ts.SipeedRenameDevice)
	api.DELETE("/tailscale/sipeed/devices/:id", ts.SipeedRemoveDevice)
}
