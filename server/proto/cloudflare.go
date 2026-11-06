package proto

type CloudflareConfigRsp struct {
	Mode      string `json:"mode"`
	AccessURL string `json:"accessUrl"`
	TokenSet  bool   `json:"tokenSet"`
}

type SetCloudflareModeReq struct {
	Mode      string `json:"mode"`
	Challenge string `json:"challenge"`
}

type SetCloudflareConfigReq struct {
	AccessURL string  `json:"accessUrl"`
	Token     *string `json:"token"`
	Challenge string  `json:"challenge"`
}

type CloudflareStatusRsp struct {
	Mode           string `json:"mode"`
	State          string `json:"state"`
	Busy           bool   `json:"busy"`
	Enabled        bool   `json:"enabled"`
	Running        bool   `json:"running"`
	AccessURL      string `json:"accessUrl"`
	OriginURL      string `json:"originUrl"`
	HTTPSAvailable bool   `json:"httpsAvailable"`
	Version        string `json:"version"`
	Error          string `json:"error"`
}
