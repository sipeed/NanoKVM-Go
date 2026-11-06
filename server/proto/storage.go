package proto

type GetImagesRsp struct {
	Files []string `json:"files"`
}

type MountImageReq struct {
	File     string `form:"file" validate:"omitempty"`
	Cdrom    bool   `form:"cdrom" validate:"omitempty"`
	ReadOnly bool   `form:"readOnly" validate:"omitempty"`
}

type DownloadImageReq struct {
	SHA256 string `json:"sha256" form:"sha256"`
	File   string `json:"file" form:"file" validate:"required,url"`
}

type GetMountedImageRsp struct {
	Mounted  bool   `json:"mounted"`
	File     string `json:"file"`
	Cdrom    bool   `json:"cdrom"`
	ReadOnly bool   `json:"readOnly"`
}

type DeleteImageReq struct {
	File      string `json:"file" validate:"required"`
	Challenge string `json:"challenge"`
}

type ImageEnabledRsp struct {
	Enabled bool `json:"enabled"`
}

type StatusImageRsp struct {
	Status           string `json:"status"`
	File             string `json:"file"`
	Percentage       string `json:"percentage"`
	DownloadedBytes  int64  `json:"downloadedBytes"`
	TotalBytes       int64  `json:"totalBytes"`
	BytesPerSecond   int64  `json:"bytesPerSecond"`
	RemainingSeconds int64  `json:"remainingSeconds"`
}
