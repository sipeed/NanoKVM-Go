package router

import (
	"github.com/gin-gonic/gin"

	"NanoKVM-Server/middleware"
	"NanoKVM-Server/service/vm"
)

func vmRouter(r *gin.Engine) {
	service := vm.NewService()

	api := r.Group("/api").Use(middleware.CheckToken())

	api.GET("/vm/info", service.GetInfo)         // get device information
	api.GET("/vm/hardware", service.GetHardware) // get hardware version

	api.GET("/vm/finger-robot", service.GetFingerRobot)
	api.PUT("/vm/finger-robot", service.SetFingerRobot)
	api.POST("/vm/finger-robot/press", service.PressFingerRobot)
	api.POST("/vm/finger-robot/release", service.ReleaseFingerRobot)
	api.POST("/vm/finger-robot/replace-arm", service.ReplaceFingerRobotArm)
	api.GET("/vm/finger-robot/debug", service.GetFingerRobotDebug)
	api.POST("/vm/finger-robot/debug/test", service.TestFingerRobotDebug)
	api.POST("/vm/finger-robot/debug/stop", service.StopFingerRobotDebug)
	api.POST("/vm/finger-robot/debug/reset", service.ResetFingerRobotDebug)

	api.POST("/vm/terminal/auth", service.TerminalAuth) // authorize a one-time terminal grant
	api.GET("/vm/terminal", service.Terminal)           // web terminal

	api.GET("/vm/script", service.GetScripts)           // get script
	api.POST("/vm/script/upload", service.UploadScript) // upload script
	api.POST("/vm/script/run", service.RunScript)       // run script
	api.DELETE("/vm/script", service.DeleteScript)      // delete script

	api.GET("/vm/device/virtual", service.GetVirtualDevice)              // get virtual device
	api.POST("/vm/device/virtual", service.UpdateVirtualDevice)          // update virtual device
	api.POST("/vm/device/virtual/refresh", service.RefreshVirtualDevice) // refresh virtual device
	api.POST("/vm/device/virtual/disk/disable", service.DisableVirtualDisk)

	api.GET("/vm/hdmi/capture", service.GetHdmiCapture)  // get HDMI capture status
	api.POST("/vm/hdmi/capture", service.SetHdmiCapture) // set HDMI capture status

	api.GET("/vm/ssh", service.GetSSHState)         // get SSH state
	api.POST("/vm/ssh/enable", service.EnableSSH)   // enable SSH
	api.POST("/vm/ssh/disable", service.DisableSSH) // disable SSH

	api.GET("/vm/edid", service.GetEdid)     // get current EDID
	api.POST("/vm/edid", service.SwitchEdid) // switch EDID
	api.GET("/vm/edid/monitor-name", service.GetEdidMonitorName)
	api.POST("/vm/edid/monitor-name", service.SetEdidMonitorName)
	api.POST("/vm/edid/upload", service.UploadEdid)       // upload EDID
	api.GET("/vm/edid/custom", service.GetCustomEdidList) // get custom EDID list
	api.POST("/vm/edid/delete", service.DeleteEdid)       // delete custom EDID

	api.GET("/vm/dial-background", service.GetDialBackgroundImages)             // get dial background images
	api.GET("/vm/dial-background/image/:image", service.GetDialBackgroundImage) // get dial background image
	api.POST("/vm/dial-background", service.SwitchDialBackground)               // switch dial background image
	api.POST("/vm/dial-background/upload", service.UploadDialBackgroundImage)   // upload dial background image

	api.GET("/vm/mouse-jiggler", service.GetMouseJiggler)   // get mouse jiggler
	api.POST("/vm/mouse-jiggler/", service.SetMouseJiggler) // set mouse jiggler

	api.GET("/vm/hostname", service.GetHostname)  // Get Hostname
	api.POST("/vm/hostname", service.SetHostname) // Set Hostname

	api.GET("/vm/web-title", service.GetWebTitle)  // Get web title
	api.POST("/vm/web-title", service.SetWebTitle) // Set web title

	api.GET("/vm/mdns", service.GetMdnsState)         // get mDNS state
	api.POST("/vm/mdns/enable", service.EnableMdns)   // enable mDNS
	api.POST("/vm/mdns/disable", service.DisableMdns) // disable mDNS

	api.GET("/vm/low-power", service.GetLowPower)  // get low power status
	api.POST("/vm/low-power", service.SetLowPower) // set low power mode

	api.GET("/vm/swap", service.GetSwap)  // get swap status
	api.POST("/vm/swap", service.SetSwap) // set swap mode

	api.GET("/vm/usb-id", service.GetUsbId)            // get USB device identity
	api.POST("/vm/usb-id", service.SetUsbId)           // set USB device identity
	api.GET("/vm/usb-network", service.GetUsbNetwork)  // get USB network configuration
	api.POST("/vm/usb-network", service.SetUsbNetwork) // set USB network configuration

	api.GET("/vm/disk-inquiry", service.GetDiskInquiry)             // get mass storage inquiry string
	api.POST("/vm/disk-inquiry", service.SetDiskInquiry)            // set mass storage inquiry string
	api.GET("/vm/disk-inquiry-cdrom", service.GetDiskInquiryCdrom)  // get CD-ROM inquiry string
	api.POST("/vm/disk-inquiry-cdrom", service.SetDiskInquiryCdrom) // set CD-ROM inquiry string
	api.GET("/vm/uac2-function-name", service.GetUac2FunctionName)  // get UAC2 function name
	api.POST("/vm/uac2-function-name", service.SetUac2FunctionName) // set UAC2 function name

	api.POST("/vm/system/reboot", service.Reboot)                       // reboot system
	api.POST("/vm/system/reboot-to-flashing", service.RebootToFlashing) // reboot to flashing mode

	api.POST("/vm/timezone", service.SetTimeZone)     // set time zone
	api.GET("/vm/timezone", service.GetTimeZone)      // get current time zone
	api.GET("/vm/time", service.GetTime)              // get time in real time
	api.POST("/vm/time", service.SetTime)             // set system time manually
	api.GET("/vm/time/status", service.GetTimeStatus) // get time status
	api.POST("/vm/time/sync", service.SyncTime)       // synchronize time

	api.GET("/vm/menubar", service.GetMenuBarConfig)  // get menu bar config
	api.POST("/vm/menubar", service.SetMenuBarConfig) // set menu bar config

	api.GET("/vm/onboarding", service.GetOnboarding)  // get onboarding completion state
	api.POST("/vm/onboarding", service.SetOnboarding) // mark onboarding completed

	api.GET("/vm/portal", service.GetPortalConfig)
	api.POST("/vm/portal", service.SetPortalConfig)
	api.POST("/vm/portal/area", service.SetPortalDetectedArea) // report the auto-detected content area
	api.GET("/vm/input-profiles", service.GetInputProfileConfig)
	api.POST("/vm/input-profiles", service.SetInputProfileConfig)
}
