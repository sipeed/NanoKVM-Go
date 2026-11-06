package proto

type GetAIConfigRsp struct {
	MCPEnabled      bool   `json:"mcpEnabled"`
	MCPAPIKeyMasked string `json:"mcpApiKeyMasked,omitempty"`
}

type GetMCPAPIKeyRsp struct {
	MCPAPIKey string `json:"mcpApiKey"`
}

type AIServiceCapability struct {
	ID           string `json:"id"`
	Supported    bool   `json:"supported"`
	MinMemoryMiB int    `json:"minMemoryMiB,omitempty"`
}

type GetAIServicesRsp struct {
	MemoryMiB  int                   `json:"memoryMiB"`
	MemoryTier string                `json:"memoryTier"`
	Services   []AIServiceCapability `json:"services"`
}

type SetAIConfigReq struct {
	MCPEnabled *bool  `json:"mcpEnabled" form:"mcpEnabled" validate:"required"`
	Challenge  string `json:"challenge" form:"challenge"`
}

type MCPAPIKeyReq struct {
	Challenge string `json:"challenge" form:"challenge"`
}

type GetMemoryFabricRsp struct {
	Enabled bool `json:"enabled"`
	Running bool `json:"running"`
}

type SetMemoryFabricReq struct {
	Enabled *bool `json:"enabled" form:"enabled" validate:"required"`
}

type GetScreenTimelapseStateRsp struct {
	Enabled bool `json:"enabled"`
}

type SetScreenTimelapseStateReq struct {
	Enabled   *bool  `json:"enabled" form:"enabled" validate:"required"`
	Challenge string `json:"challenge" form:"challenge"`
}

type GetScreenTimelapseRecordingRsp struct {
	QP int `json:"qp"`
}

type SetScreenTimelapseRecordingReq struct {
	QP *int `json:"qp" form:"qp" validate:"required"`
}

type SetScreenTimelapseRecordingRsp struct {
	QP        int  `json:"qp"`
	Changed   bool `json:"changed"`
	Restarted bool `json:"restarted"`
}

type GetScreenTimelapseDaysReq struct {
	Recent int `form:"recent"`
}

type ScreenTimelapseDay struct {
	Date       string `json:"date"`
	VideoURL   string `json:"videoUrl"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
	DurationMs int64  `json:"durationMs"`
	FrameCount int    `json:"frameCount"`
	Filesize   int64  `json:"filesize"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
	Deletable  bool   `json:"deletable"`
}

type GetScreenTimelapseDaysRsp struct {
	Days []ScreenTimelapseDay `json:"days"`
}

type GetScreenTimelapseDateReq struct {
	Date string `form:"date" validate:"required"`
}

type DeleteScreenTimelapseDayReq struct {
	Date string `form:"date" validate:"required"`
}

type DeleteScreenTimelapseDayRsp struct {
	Date         string `json:"date"`
	DeletedBytes int64  `json:"deletedBytes"`
}

type ScreenTimelapseFrame struct {
	PTS         int64   `json:"pts"`
	CurrentTime float64 `json:"currentTime"`
	CapturedAt  string  `json:"capturedAt"`
}

type GetScreenTimelapseFramesRsp struct {
	Date       string                 `json:"date"`
	DurationMs int64                  `json:"durationMs"`
	FrameCount int                    `json:"frameCount"`
	Frames     []ScreenTimelapseFrame `json:"frames"`
}

type SearchScreenTimelapseReq struct {
	Query string `form:"query" validate:"required"`
	Limit int    `form:"limit"`
}

type ScreenTimelapseSearchResult struct {
	CapturedAt        string  `json:"capturedAt"`
	Date              string  `json:"date"`
	PTS               int64   `json:"pts,omitempty"`
	CurrentTime       float64 `json:"currentTime,omitempty"`
	MatchedFrameTime  string  `json:"matchedFrameTime,omitempty"`
	Snippet           string  `json:"snippet,omitempty"`
	Available         bool    `json:"available"`
	UnavailableReason string  `json:"unavailableReason,omitempty"`
}

type SearchScreenTimelapseRsp struct {
	Query                   string                        `json:"query"`
	Count                   int                           `json:"count"`
	Results                 []ScreenTimelapseSearchResult `json:"results"`
	SkippedMissingFileCount int                           `json:"skippedMissingFileCount"`
}

// ScreenTimelapseRetentionCleanupConfig is the public cleanup policy. The
// persisted format also has an internal version, which is intentionally not
// exposed through this API contract.
type ScreenTimelapseRetentionCleanupConfig struct {
	Enabled       bool  `json:"enabled"`
	RetentionDays int   `json:"retentionDays"`
	MaxStorageMiB int64 `json:"maxStorageMiB"`
}

// ScreenTimelapseRetentionCleanupConfigInput uses pointers so the handler can
// distinguish a missing field from an explicit zero that disables a limit.
type ScreenTimelapseRetentionCleanupConfigInput struct {
	Enabled       *bool  `json:"enabled"`
	RetentionDays *int   `json:"retentionDays"`
	MaxStorageMiB *int64 `json:"maxStorageMiB"`
}

type ScreenTimelapseRetentionConfig struct {
	Revision int64                                 `json:"revision"`
	Cleanup  ScreenTimelapseRetentionCleanupConfig `json:"cleanup"`
}

type ScreenTimelapseRetentionStatus struct {
	Running          bool   `json:"running"`
	ScannedAt        string `json:"scannedAt"`
	UsageBytes       int64  `json:"usageBytes"`
	ManagedDays      int    `json:"managedDays"`
	OverLimit        bool   `json:"overLimit"`
	BlockedReason    string `json:"blockedReason"`
	LastCleanupAt    string `json:"lastCleanupAt"`
	LastTrigger      string `json:"lastTrigger"`
	LastDeletedDays  int    `json:"lastDeletedDays"`
	LastDeletedBytes int64  `json:"lastDeletedBytes"`
	LastError        string `json:"lastError"`
}

type GetScreenTimelapseRetentionRsp struct {
	Config ScreenTimelapseRetentionConfig `json:"config"`
	Status ScreenTimelapseRetentionStatus `json:"status"`
}

type SetScreenTimelapseRetentionReq struct {
	Revision  *int64                                      `json:"revision"`
	Cleanup   *ScreenTimelapseRetentionCleanupConfigInput `json:"cleanup"`
	Challenge string                                      `json:"challenge"`
}

type SetScreenTimelapseRetentionRsp struct {
	Config           ScreenTimelapseRetentionConfig `json:"config"`
	Changed          bool                           `json:"changed"`
	CleanupTriggered bool                           `json:"cleanupTriggered"`
}

type ScreenTimelapseRetentionCleanupReq struct {
	DryRun    *bool                                       `json:"dryRun"`
	Config    *ScreenTimelapseRetentionCleanupConfigInput `json:"config,omitempty"`
	Challenge string                                      `json:"challenge"`
}

type ScreenTimelapseRetentionCleanupCandidate struct {
	Date    string   `json:"date"`
	Bytes   int64    `json:"bytes"`
	Reasons []string `json:"reasons"`
}

type ScreenTimelapseRetentionCleanupSkipped struct {
	Date   string `json:"date"`
	Bytes  int64  `json:"bytes,omitempty"`
	Reason string `json:"reason"`
}

type ScreenTimelapseRetentionWarning struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
	Error  string `json:"error,omitempty"`
}

type ScreenTimelapseRetentionPreviewRsp struct {
	DryRun                bool                                       `json:"dryRun"`
	BeforeBytes           int64                                      `json:"beforeBytes"`
	ProjectedAfterBytes   int64                                      `json:"projectedAfterBytes"`
	OverLimitAfterCleanup bool                                       `json:"overLimitAfterCleanup"`
	WouldDeleteDays       int                                        `json:"wouldDeleteDays"`
	WouldDeleteBytes      int64                                      `json:"wouldDeleteBytes"`
	Candidates            []ScreenTimelapseRetentionCleanupCandidate `json:"candidates"`
	Skipped               []ScreenTimelapseRetentionCleanupSkipped   `json:"skipped"`
	Warnings              []ScreenTimelapseRetentionWarning          `json:"warnings"`
}

type ScreenTimelapseRetentionTriggerRsp struct {
	Accepted bool `json:"accepted"`
}
