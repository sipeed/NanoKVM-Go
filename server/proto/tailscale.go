package proto

type TailscaleState string

type TailscaleMode string

type TailscaleManagedState string

type TailscaleRegion string

const (
	TailscaleInstalling TailscaleState = "installing"
	TailscaleNotInstall TailscaleState = "notInstall"
	TailscaleNotRunning TailscaleState = "notRunning"
	TailscaleNotLogin   TailscaleState = "notLogin"
	TailscaleStopped    TailscaleState = "stopped"
	TailscaleRunning    TailscaleState = "running"
)

const (
	TailscaleModeOfficial TailscaleMode = "official"
	TailscaleModeSipeed   TailscaleMode = "sipeed"
)

const (
	TailscaleRegionCN     TailscaleRegion = "cn"
	TailscaleRegionGlobal TailscaleRegion = "global"
)

const (
	TailscaleManagedDisabled          TailscaleManagedState = "disabled"
	TailscaleManagedEnrolling         TailscaleManagedState = "enrolling"
	TailscaleManagedEnabledConnecting TailscaleManagedState = "enabledConnecting"
	TailscaleManagedEnabledConnected  TailscaleManagedState = "enabledConnected"
	TailscaleManagedDegraded          TailscaleManagedState = "degraded"
	TailscaleManagedInvalidIdentity   TailscaleManagedState = "invalidIdentity"
)

type GetTailscaleStatusRsp struct {
	State                    TailscaleState        `json:"state"`
	Name                     string                `json:"name"`
	IP                       string                `json:"ip"`
	Account                  string                `json:"account"`
	RoutingAvailable         bool                  `json:"routingAvailable"`
	AdvertiseExitNode        bool                  `json:"advertiseExitNode"`
	AdvertiseSubnetRoute     bool                  `json:"advertiseSubnetRoute"`
	LANSubnet                string                `json:"lanSubnet"`
	AvailableModes           []TailscaleMode       `json:"availableModes"`
	ActiveMode               *TailscaleMode        `json:"activeMode"`
	ManagedState             TailscaleManagedState `json:"managedState"`
	ManagedError             string                `json:"managedError"`
	RetryAfterSeconds        int                   `json:"retryAfterSeconds"`
	SipeedDisclaimerAccepted bool                  `json:"sipeedDisclaimerAccepted"`
}

type LoginTailscaleRsp struct {
	Url string `json:"url"`
}

type SetTailscaleRoutingReq struct {
	Enabled   *bool  `json:"enabled" form:"enabled"`
	Challenge string `json:"challenge,omitempty" form:"challenge"`
}
