package proto

type ApplicationRegion string

const (
	ApplicationRegionCN     ApplicationRegion = "cn"
	ApplicationRegionGlobal ApplicationRegion = "global"
)

type GetVersionRsp struct {
	Current string `json:"current"`
	Latest  string `json:"latest"`
}

type GetPreviewRsp struct {
	Enabled bool `json:"enabled"`
}

type GetApplicationRegionRsp struct {
	Region             ApplicationRegion `json:"region"`
	CloudCleanupFailed bool              `json:"cloudCleanupFailed,omitempty"`
}

type SetApplicationRegionReq struct {
	Region ApplicationRegion `json:"region"`
}

type SetPreviewReq struct {
	Enable bool `validate:"omitempty"`
}

type GetUpdateServerRsp struct {
	Server string `json:"server"`
}

type SetUpdateServerReq struct {
	Server *string `json:"server"`
}

type GetUpdateSignatureVerificationRsp struct {
	Enabled bool `json:"enabled"`
}

type SetUpdateSignatureVerificationReq struct {
	Enabled   *bool  `json:"enabled"`
	Challenge string `json:"challenge"`
}

type AcknowledgeApplicationUpdateStatusReq struct {
	OperationID string `json:"operationId" validate:"required"`
}
