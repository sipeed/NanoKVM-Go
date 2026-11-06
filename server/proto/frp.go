package proto

type FRPConfigRsp struct {
	ServerAddr string `json:"serverAddr"`
	ServerPort int    `json:"serverPort"`
	RemotePort int    `json:"remotePort"`
	TokenSet   bool   `json:"tokenSet"`
}

type SetFRPConfigReq struct {
	ServerAddr string `json:"serverAddr"`
	ServerPort int    `json:"serverPort"`
	RemotePort int    `json:"remotePort"`
	// Omission preserves the saved token; an explicit empty string clears it.
	Token     *string `json:"token"`
	Challenge string  `json:"challenge"`
}

type FRPStatusRsp struct {
	State     string `json:"state"`
	Busy      bool   `json:"busy"`
	Enabled   bool   `json:"enabled"`
	AccessURL string `json:"accessUrl"`
	Error     string `json:"error"`
}
