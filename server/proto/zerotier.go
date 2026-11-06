package proto

type ZeroTierNetworkReq struct {
	NetworkID string `json:"networkId"`
	Challenge string `json:"challenge"`
}

type ZeroTierNetwork struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Status     string   `json:"status"`
	Addresses  []string `json:"addresses"`
	AccessURLs []string `json:"accessUrls"`
}

type ZeroTierStatusRsp struct {
	State    string            `json:"state"`
	Busy     bool              `json:"busy"`
	Enabled  bool              `json:"enabled"`
	Running  bool              `json:"running"`
	Online   bool              `json:"online"`
	NodeID   string            `json:"nodeId"`
	Version  string            `json:"version"`
	Networks []ZeroTierNetwork `json:"networks"`
	Error    string            `json:"error"`
}
