package proto

type IP struct {
	Name    string `json:"name"`
	Addr    string `json:"addr"`
	Version string `json:"version"`
	Type    string `json:"type"`
}

type GetInfoRsp struct {
	IPs          []IP   `json:"ips"`
	Mdns         string `json:"mdns"`
	Image        string `json:"image"`
	Application  string `json:"application"`
	DeviceKey    string `json:"deviceKey"`
	DeviceNumber string `json:"pn"`
	Arch         string `json:"arch"`
}

type GetHardwareRsp struct {
	Version string `json:"version"`
}

type FingerRobotDebugTestReq struct {
	ReleaseUS             int    `json:"releaseUs"`
	PressUS               int    `json:"pressUs"`
	PressSpeedUSPerSecond int    `json:"pressSpeedUsPerSecond"`
	Count                 *int64 `json:"count"`
	PressMS               int    `json:"pressMs"`
	Challenge             string `json:"challenge,omitempty"`
}

type FingerRobotDebugRangeRsp struct {
	Min     int `json:"min"`
	Max     int `json:"max"`
	Default int `json:"default"`
}

type FingerRobotDebugCountRsp struct {
	Min            int64 `json:"min"`
	Default        int64 `json:"default"`
	UnlimitedValue int64 `json:"unlimitedValue"`
}

type GetFingerRobotDebugRsp struct {
	Enabled               bool                      `json:"enabled"`
	Available             bool                      `json:"available"`
	ZeroUS                int                       `json:"zeroUs,omitempty"`
	ReleaseUS             *FingerRobotDebugRangeRsp `json:"releaseUs,omitempty"`
	PressUS               *FingerRobotDebugRangeRsp `json:"pressUs,omitempty"`
	PressSpeedUSPerSecond *FingerRobotDebugRangeRsp `json:"pressSpeedUsPerSecond,omitempty"`
	Count                 *FingerRobotDebugCountRsp `json:"count,omitempty"`
	PressMS               *FingerRobotDebugRangeRsp `json:"pressMs,omitempty"`
	ReleaseIntervalMS     int                       `json:"releaseIntervalMs,omitempty"`
	LeaseTimeoutMS        int                       `json:"leaseTimeoutMs,omitempty"`
	Running               bool                      `json:"running"`
	Current               int64                     `json:"current"`
	Total                 int64                     `json:"total"`
	State                 string                    `json:"state"`
	Result                string                    `json:"result"`
	Error                 string                    `json:"error"`
}

type FingerRobotStrengthByArmInput struct {
	Long  *int `json:"long"`
	Short *int `json:"short"`
}

type FingerRobotStrengthByArmRsp struct {
	Long  int `json:"long"`
	Short int `json:"short"`
}

type FingerRobotConfigRsp struct {
	Version       int                         `json:"version"`
	Revision      int64                       `json:"revision"`
	ActiveArm     string                      `json:"activeArm"`
	StrengthByArm FingerRobotStrengthByArmRsp `json:"strengthByArm"`
	PressMS       int                         `json:"pressMs"`
	Speed         string                      `json:"speed"`
}

type FingerRobotProfileRsp struct {
	ID              string `json:"id"`
	DisplayName     string `json:"displayName"`
	DefaultStrength int    `json:"defaultStrength"`
}

type FingerRobotRangeRsp struct {
	Min     int `json:"min"`
	Max     int `json:"max"`
	Default int `json:"default"`
}

type FingerRobotStatusRsp struct {
	Running bool   `json:"running"`
	State   string `json:"state"`
	Result  string `json:"result"`
	Error   string `json:"error"`
}

type GetFingerRobotRsp struct {
	Enabled      bool                    `json:"enabled"`
	Available    bool                    `json:"available"`
	Config       *FingerRobotConfigRsp   `json:"config,omitempty"`
	Profiles     []FingerRobotProfileRsp `json:"profiles"`
	Strength     FingerRobotRangeRsp     `json:"strength"`
	PressMS      FingerRobotRangeRsp     `json:"pressMs"`
	SpeedOptions []string                `json:"speedOptions"`
	DefaultSpeed string                  `json:"defaultSpeed"`
	Status       FingerRobotStatusRsp    `json:"status"`
}

type SetFingerRobotReq struct {
	Version       *int                           `json:"version"`
	Revision      *int64                         `json:"revision"`
	ActiveArm     *string                        `json:"activeArm"`
	StrengthByArm *FingerRobotStrengthByArmInput `json:"strengthByArm"`
	PressMS       *int                           `json:"pressMs"`
	Speed         *string                        `json:"speed"`
	Challenge     string                         `json:"challenge,omitempty"`
}

type SetFingerRobotRsp struct {
	Config  FingerRobotConfigRsp `json:"config"`
	Changed bool                 `json:"changed"`
}

type FingerRobotActionReq struct {
	Challenge string `json:"challenge,omitempty"`
}

type GetScriptsRsp struct {
	Files []string `json:"files"`
}

type UploadScriptRsp struct {
	File string `json:"file"`
}

type RunScriptReq struct {
	Name      string `json:"name" form:"name" validate:"required"`
	Type      string `json:"type" form:"type" validate:"required,oneof=foreground background"` // foreground | background
	Challenge string `json:"challenge,omitempty" form:"challenge"`
}

type RunScriptRsp struct {
	Log string `json:"log"`
}

type DeleteScriptReq struct {
	Name      string `validate:"required"`
	Challenge string `json:"challenge,omitempty" form:"challenge"`
}

type GetVirtualDeviceRsp struct {
	IsWebcamEnabled  bool   `json:"isWebcamEnabled"`
	IsNetworkEnabled bool   `json:"isNetworkEnabled"`
	IsDiskEnabled    bool   `json:"isDiskEnabled"`
	DiskMinSizeMiB   int64  `json:"diskMinSizeMiB"`
	DiskMaxSizeMiB   int64  `json:"diskMaxSizeMiB"`
	IsMicEnabled     bool   `json:"isMicEnabled"`
	IsSpeakerEnabled bool   `json:"isSpeakerEnabled"`
	AudioMode        string `json:"audioMode"`
	IsSdCardExist    bool   `json:"isSdCardExist"`
}

type UpdateVirtualDeviceReq struct {
	Device      string `json:"device" form:"device" validate:"required"`                                               // disk | network | audio | webcam
	Mode        string `json:"mode" form:"mode" validate:"required_if=Device audio,omitempty,oneof=on off mic duplex"` // audio: off, mic, duplex; webcam: on, off
	DiskSizeMiB int64  `json:"diskSizeMiB" form:"diskSizeMiB" validate:"omitempty,min=5120"`                           // required when initializing disk storage
}

type RefreshVirtualDeviceReq struct {
	Device string `json:"device" validate:"required"`
}

type SetMemoryLimitReq struct {
	Enabled bool  `validate:"omitempty"`
	Limit   int64 `validate:"omitempty"`
}

type GetMemoryLimitRsp struct {
	Enabled bool  `json:"enabled"`
	Limit   int64 `json:"limit"`
}

type GetSSHStateRsp struct {
	Enabled bool `json:"enabled"`
}

type GetMouseJigglerRsp struct {
	Enabled bool   `json:"enabled"`
	Mode    string `json:"mode"`
}

type SetMouseJigglerReq struct {
	Enabled bool   `validate:"omitempty"`
	Mode    string `validate:"omitempty"`
}

type GetMdnsStateRsp struct {
	Enabled bool `json:"enabled"`
}

type GetLowPowerRsp struct {
	Enabled bool `json:"enabled"`
}

type SetLowPowerReq struct {
	Enable bool `form:"enable" validate:"omitempty"`
}

type GetSwapRsp struct {
	Enabled bool `json:"enabled"`
}

type SetSwapReq struct {
	Enable *bool `json:"enable" form:"enable" validate:"required"`
}

type GetUsbIdRsp struct {
	Enabled      bool   `json:"enabled"`
	Vid          string `json:"vid"`
	Pid          string `json:"pid"`
	BcdDevice    string `json:"bcdDevice"`
	Serialnumber string `json:"serialnumber"`
	Manufacturer string `json:"manufacturer"`
	Product      string `json:"product"`
}

type SetUsbIdReq struct {
	Enabled      bool   `json:"enabled" form:"enabled"`
	Vid          string `json:"vid" form:"vid" validate:"omitempty"`
	Pid          string `json:"pid" form:"pid" validate:"omitempty"`
	BcdDevice    string `json:"bcdDevice" form:"bcdDevice" validate:"omitempty"`
	Serialnumber string `json:"serialnumber" form:"serialnumber" validate:"omitempty"`
	Manufacturer string `json:"manufacturer" form:"manufacturer" validate:"omitempty"`
	Product      string `json:"product" form:"product" validate:"omitempty"`
}

type GetUsbNetworkRsp struct {
	Mode       string `json:"mode"`
	IPv4       string `json:"ipv4"`
	Forwarding bool   `json:"forwarding"`
}

type SetUsbNetworkReq struct {
	Mode       string `json:"mode" form:"mode" validate:"required"`
	IPv4       string `json:"ipv4" form:"ipv4" validate:"omitempty"`
	Forwarding bool   `json:"forwarding" form:"forwarding"`
}

type GetBootStringRsp struct {
	Enabled bool   `json:"enabled"`
	Value   string `json:"value"`
}

type SetBootStringReq struct {
	Enabled bool   `json:"enabled" form:"enabled"`
	Value   string `json:"value" form:"value" validate:"omitempty"`
}

type SetHostnameReq struct {
	Hostname string `validate:"required"`
}

type GetHostnameRsp struct {
	Hostname string `json:"hostname"`
}

type SetWebTitleReq struct {
	Title string `validate:"omitempty"`
}

type GetWebTitleRsp struct {
	Title string `json:"title"`
}

type GetEdidRsp struct {
	Edid   string `json:"edid"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type SwitchEdidReq struct {
	Edid string `form:"edid" validate:"required"`
}

type GetEdidMonitorNameRsp struct {
	MonitorName string `json:"monitorName"`
}

type SetEdidMonitorNameReq struct {
	MonitorName string `json:"monitorName" form:"monitorName"`
}

type GetCustomEdidListRsp struct {
	EdidList []string `json:"edidList"`
}

type DeleteEdidReq struct {
	Edid string `form:"edid" validate:"required"`
}

type UploadEdidRsp struct {
	File string `json:"file"`
}

type DialBackgroundImage struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"modTime"`
	URL     string `json:"url"`
}

type GetDialBackgroundImagesRsp struct {
	Files   []DialBackgroundImage `json:"files"`
	Current string                `json:"current"`
}

type SwitchDialBackgroundReq struct {
	ImageName string `json:"imageName" form:"imageName" validate:"required"`
}

type UploadDialBackgroundRsp struct {
	File string `json:"file"`
}

type GetHdmiCaptureRsp struct {
	Enabled bool `json:"enabled"`
}

type SetHdmiCaptureReq struct {
	Enabled bool `json:"enabled" form:"enabled" validate:"omitempty"`
}

type GetOnboardingRsp struct {
	CompletedVersion       string `json:"completedVersion"`
	MobileCompletedVersion string `json:"mobileCompletedVersion"`
}

type SetOnboardingReq struct {
	CompletedVersion       string `json:"completedVersion" form:"completedVersion" validate:"omitempty,max=32"`
	MobileCompletedVersion string `json:"mobileCompletedVersion" form:"mobileCompletedVersion" validate:"omitempty,max=32"`
}

type SetTimeZoneReq struct {
	Timezone string `validate:"required"`
}

type SetTimeReq struct {
	Datetime string `json:"datetime"`
}

type SetTimeRsp struct {
	Timestamp int64 `json:"timestamp"`
}

type GetTimeZoneRsp struct {
	Timezone string `json:"timezone"`
}

type GetTimeStatusRsp struct {
	IsSynchronized bool  `json:"isSynchronized"`
	LastSyncTime   int64 `json:"lastSyncTime"`
}

type MenuBarConfig struct {
	DisabledItems []string `json:"disabledItems"`
}

type SetMenuBarConfigReq struct {
	DisabledItems []string `form:"disabledItems" json:"disabledItems" validate:"omitempty"`
}

type GetMenuBarConfigRsp struct {
	DisabledItems []string `json:"disabledItems"`
}

type PortalContentRect struct {
	Left   int `json:"left"`
	Top    int `json:"top"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type PortalManualRegion struct {
	ID           string            `json:"id"`
	SourceWidth  int               `json:"sourceWidth"`
	SourceHeight int               `json:"sourceHeight"`
	ContentRect  PortalContentRect `json:"contentRect"`
}

type PortalResolution struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// PortalAutoArea is the last stable auto-detected content area reported by a
// browser. It is optional state inside the version 1 config so older builds
// keep reading the same file.
type PortalAutoArea struct {
	SourceWidth  int               `json:"sourceWidth"`
	SourceHeight int               `json:"sourceHeight"`
	ContentRect  PortalContentRect `json:"contentRect"`
	UpdatedAt    int64             `json:"updatedAt"`
}

type PortalConfig struct {
	Version            int                  `json:"version"`
	Mode               string               `json:"mode"`
	ManualRegions      []PortalManualRegion `json:"manualRegions"`
	SelectedRegionID   string               `json:"selectedRegionId"`
	Resolutions        []PortalResolution   `json:"resolutions"`
	SelectedResolution string               `json:"selectedResolution"`
	AutoArea           *PortalAutoArea      `json:"autoArea,omitempty"`
}

type SetPortalDetectedAreaReq struct {
	SourceWidth  int                `json:"sourceWidth"`
	SourceHeight int                `json:"sourceHeight"`
	ContentRect  *PortalContentRect `json:"contentRect"`
}

type GetPortalConfigRsp struct {
	Configured bool `json:"configured"`
	PortalConfig
}

type InputProfilePoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type InputProfileCalibrationPoint struct {
	Source InputProfilePoint `json:"source"`
	Target InputProfilePoint `json:"target"`
}

type InputProfileCalibration struct {
	Points    map[string]InputProfileCalibrationPoint `json:"points"`
	UpdatedAt int64                                   `json:"updatedAt"`
}

type InputProfile struct {
	ID           string                             `json:"id"`
	Name         string                             `json:"name"`
	Origin       string                             `json:"origin"`
	Platform     string                             `json:"platform,omitempty"`
	Calibrations map[string]InputProfileCalibration `json:"calibrations"`
	CreatedAt    int64                              `json:"createdAt"`
	UpdatedAt    int64                              `json:"updatedAt"`
}

type InputProfileConfig struct {
	Version           int            `json:"version"`
	SelectedProfileID string         `json:"selectedProfileId"`
	Profiles          []InputProfile `json:"profiles"`
}

type GetInputProfileConfigRsp struct {
	Configured bool `json:"configured"`
	InputProfileConfig
}
