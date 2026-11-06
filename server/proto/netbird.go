package proto

type NetBirdStatusRsp struct {
	State     string `json:"state"`
	Busy      bool   `json:"busy"`
	Enabled   bool   `json:"enabled"`
	Running   bool   `json:"running"`
	Version   string `json:"version"`
	Name      string `json:"name"`
	IP        string `json:"ip"`
	AccessURL string `json:"accessUrl"`
	AuthURL   string `json:"authUrl"`
	UserCode  string `json:"userCode"`
	Error     string `json:"error"`
}
