package router

import (
	"github.com/gin-gonic/gin"

	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/network"
)

func networkRouter(r *gin.Engine) {
	service := network.NewService()

	r.POST("/api/network/wifi", service.ConnectWifiNoAuth)    // connect Wi-Fi without auth (only available in ap mode)
	r.POST("/api/network/wifi/verify", service.VerifyApLogin) // verify ap login
	r.GET("/api/network/wifi/scan", service.ScanWifiWithAPKeyOrToken)

	api := r.Group("/api").Use(middleware.CheckToken())

	api.POST("/network/wol", service.WakeOnLAN)           // wake on lan
	api.GET("/network/wol/mac", service.GetMac)           // get mac list
	api.DELETE("/network/wol/mac", service.DeleteMac)     // delete mac
	api.POST("/network/wol/mac/name", service.SetMacName) // set mac name

	api.GET("/network/static-ip", service.GetStaticIP)  // get static IP
	api.POST("/network/static-ip", service.SetStaticIP) // set static IP

	api.GET("/network/wifi", service.GetWifi)                         // get connected Wi-Fi
	api.POST("/network/wifi/connect", service.ConnectWifi)            // connect Wi-Fi
	api.POST("/network/wifi/disconnect", service.DisconnectWifi)      // disconnect Wi-Fi
	api.POST("/network/wifi/preset", service.PresetWifi)              // pre-store a target Wi-Fi
	api.GET("/network/wifi/preset", service.GetWifiPreset)            // get pre-stored Wi-Fi
	api.POST("/network/wifi/preset/remove", service.RemoveWifiPreset) // remove pre-stored Wi-Fi
}
