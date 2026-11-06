package ai

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	screenTimelapseRetentionRequestMaxBytes   = 16 << 10
	screenTimelapseRetentionChallengeMaxBytes = 512
)

var (
	screenTimelapseRetentionAPILoadConfig    = loadScreenTimelapseRetentionConfig
	screenTimelapseRetentionAPIUpdateCleanup = updateScreenTimelapseRetentionCleanup
	screenTimelapseRetentionAPIInventory     = inventoryScreenTimelapseRetention
	screenTimelapseRetentionAPIPlan          = planScreenTimelapseRetention
	screenTimelapseRetentionAPIRoot          = timelapseRoot
	screenTimelapseRetentionAPINow           = time.Now
	screenTimelapseRetentionAPIStatus        = GetScreenTimelapseRetentionStatus
	screenTimelapseRetentionAPITrigger       = TriggerScreenTimelapseRetentionCleanup
	screenTimelapseActionRequiresChallenge   = authservice.ScreenTimelapseActionRequiresChallenge
	verifyScreenTimelapseChallenge           = authservice.VerifySensitiveActionChallenge
)

// GetScreenTimelapseRetention returns the persisted policy plus the worker's
// last snapshot. It deliberately does not re-scan the recording directory.
func (s *Service) GetScreenTimelapseRetention(c *gin.Context) {
	var rsp proto.Response

	config, err := screenTimelapseRetentionAPILoadConfig()
	if err != nil {
		log.WithError(err).Error("failed to load screen timelapse retention config")
		rsp.ErrRsp(c, -2, "get screen timelapse retention failed")
		return
	}

	rsp.OkRspWithData(c, proto.GetScreenTimelapseRetentionRsp{
		Config: screenTimelapseRetentionPublicConfig(config),
		Status: screenTimelapseRetentionPublicStatus(screenTimelapseRetentionAPIStatus()),
	})
}

// SetScreenTimelapseRetention persists a policy with compare-and-swap revision
// semantics. The worker receives only a wake-up notification; this request
// never performs the potentially long recursive cleanup itself.
func (s *Service) SetScreenTimelapseRetention(c *gin.Context) {
	var req proto.SetScreenTimelapseRetentionReq
	var rsp proto.Response

	if err := decodeScreenTimelapseRetentionJSON(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if req.Revision == nil || !validScreenTimelapseRetentionChallenge(req.Challenge) {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	cleanup, err := screenTimelapseRetentionCleanupFromInput(req.Cleanup)
	if err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	current, err := screenTimelapseRetentionAPILoadConfig()
	if err != nil {
		log.WithError(err).Error("failed to load screen timelapse retention config for update")
		rsp.ErrRsp(c, -2, "operation failed")
		return
	}
	if current.Revision != *req.Revision {
		writeScreenTimelapseRetentionRevisionConflict(c, &rsp, current)
		return
	}
	if !requireScreenTimelapseChallenge(c, &rsp, authservice.SensitiveActionSetScreenTimelapseRetention, req.Challenge) {
		return
	}

	updated, changed, conflict, cleanupTriggered, err := screenTimelapseRetentionAPIUpdateCleanup(*req.Revision, cleanup)
	if err != nil {
		log.WithError(err).Error("failed to save screen timelapse retention config")
		rsp.ErrRsp(c, -2, "operation failed")
		return
	}
	if conflict {
		writeScreenTimelapseRetentionRevisionConflict(c, &rsp, updated)
		return
	}

	rsp.OkRspWithData(c, proto.SetScreenTimelapseRetentionRsp{
		Config:           screenTimelapseRetentionPublicConfig(updated),
		Changed:          changed,
		CleanupTriggered: cleanupTriggered,
	})
}

// CleanupScreenTimelapseRetention either previews a candidate policy (dry run)
// or asks the existing worker to run the persisted policy asynchronously.
func (s *Service) CleanupScreenTimelapseRetention(c *gin.Context) {
	var req proto.ScreenTimelapseRetentionCleanupReq
	var rsp proto.Response

	if err := decodeScreenTimelapseRetentionJSON(c, &req); err != nil || req.DryRun == nil || !validScreenTimelapseRetentionChallenge(req.Challenge) {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	if *req.DryRun {
		var config ScreenTimelapseRetentionConfig
		if req.Config != nil {
			cleanup, err := screenTimelapseRetentionCleanupFromInput(req.Config)
			if err != nil {
				rsp.ErrRsp(c, -1, "invalid arguments")
				return
			}
			config = ScreenTimelapseRetentionConfig{
				Version: screenTimelapseRetentionConfigVersion,
				Cleanup: cleanup,
			}
		} else {
			var err error
			config, err = screenTimelapseRetentionAPILoadConfig()
			if err != nil {
				log.WithError(err).Error("failed to load screen timelapse retention config for preview")
				rsp.ErrRsp(c, -2, "operation failed")
				return
			}
		}

		now := screenTimelapseRetentionAPINow()
		inventory, err := screenTimelapseRetentionAPIInventory(screenTimelapseRetentionAPIRoot(), now)
		if err != nil {
			log.WithError(err).Error("failed to inventory screen timelapse retention preview")
			rsp.ErrRsp(c, -2, "operation failed")
			return
		}
		plan, err := screenTimelapseRetentionAPIPlan(inventory, now, config)
		if err != nil {
			log.WithError(err).Error("failed to plan screen timelapse retention preview")
			rsp.ErrRsp(c, -2, "operation failed")
			return
		}

		rsp.OkRspWithData(c, screenTimelapseRetentionPreviewResponse(plan))
		return
	}

	if req.Config != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if _, err := screenTimelapseRetentionAPILoadConfig(); err != nil {
		log.WithError(err).Error("failed to load persisted screen timelapse retention config")
		rsp.ErrRsp(c, -2, "operation failed")
		return
	}
	if !requireScreenTimelapseChallenge(c, &rsp, authservice.SensitiveActionTriggerScreenTimelapseCleanup, req.Challenge) {
		return
	}
	if !screenTimelapseRetentionAPITrigger() {
		rsp.ErrRsp(c, -5, "cleanup worker unavailable")
		return
	}

	rsp.OkRspWithData(c, proto.ScreenTimelapseRetentionTriggerRsp{Accepted: true})
}

func decodeScreenTimelapseRetentionJSON(c *gin.Context, destination interface{}) error {
	if c == nil || c.Request == nil || c.Request.Body == nil {
		return errors.New("missing request body")
	}

	body := http.MaxBytesReader(c.Writer, c.Request.Body, screenTimelapseRetentionRequestMaxBytes)
	defer body.Close()
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}

	var trailing interface{}
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func validScreenTimelapseRetentionChallenge(challenge string) bool {
	return len(strings.TrimSpace(challenge)) <= screenTimelapseRetentionChallengeMaxBytes
}

func screenTimelapseRetentionCleanupFromInput(input *proto.ScreenTimelapseRetentionCleanupConfigInput) (ScreenTimelapseRetentionCleanupConfig, error) {
	if input == nil || input.Enabled == nil || input.RetentionDays == nil || input.MaxStorageMiB == nil {
		return ScreenTimelapseRetentionCleanupConfig{}, errScreenTimelapseRetentionInvalidConfig
	}

	cleanup := ScreenTimelapseRetentionCleanupConfig{
		Enabled:       *input.Enabled,
		RetentionDays: *input.RetentionDays,
		MaxStorageMiB: *input.MaxStorageMiB,
	}
	if err := validateScreenTimelapseRetentionConfig(ScreenTimelapseRetentionConfig{
		Version: screenTimelapseRetentionConfigVersion,
		Cleanup: cleanup,
	}); err != nil {
		return ScreenTimelapseRetentionCleanupConfig{}, err
	}
	return cleanup, nil
}

func requireScreenTimelapseChallenge(c *gin.Context, rsp *proto.Response, action, challenge string) bool {
	if !screenTimelapseActionRequiresChallenge(action) {
		return true
	}
	if err := verifyScreenTimelapseChallenge(c, challenge, action); err != nil {
		log.WithError(err).Errorf("verify screen timelapse challenge for %s failed", action)
		rsp.ErrRsp(c, -4, "sensitive action verification required")
		return false
	}
	return true
}

func writeScreenTimelapseRetentionRevisionConflict(c *gin.Context, rsp *proto.Response, current ScreenTimelapseRetentionConfig) {
	rsp.ErrRspWithData(c, -3, "revision conflict", proto.SetScreenTimelapseRetentionRsp{
		Config: screenTimelapseRetentionPublicConfig(current),
	})
}

func screenTimelapseRetentionPublicConfig(config ScreenTimelapseRetentionConfig) proto.ScreenTimelapseRetentionConfig {
	return proto.ScreenTimelapseRetentionConfig{
		Revision: config.Revision,
		Cleanup: proto.ScreenTimelapseRetentionCleanupConfig{
			Enabled:       config.Cleanup.Enabled,
			RetentionDays: config.Cleanup.RetentionDays,
			MaxStorageMiB: config.Cleanup.MaxStorageMiB,
		},
	}
}

func screenTimelapseRetentionPublicStatus(status ScreenTimelapseRetentionStatus) proto.ScreenTimelapseRetentionStatus {
	return proto.ScreenTimelapseRetentionStatus{
		Running:          status.Running,
		ScannedAt:        screenTimelapseRetentionTimestamp(status.ScannedAt),
		UsageBytes:       status.UsageBytes,
		ManagedDays:      status.ManagedDays,
		OverLimit:        status.OverLimit,
		BlockedReason:    status.BlockedReason,
		LastCleanupAt:    screenTimelapseRetentionTimestamp(status.LastCleanupAt),
		LastTrigger:      status.LastTrigger,
		LastDeletedDays:  status.LastDeletedDays,
		LastDeletedBytes: status.LastDeletedBytes,
		LastError:        status.LastError,
	}
}

func screenTimelapseRetentionTimestamp(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func screenTimelapseRetentionPreviewResponse(plan ScreenTimelapseRetentionPlan) proto.ScreenTimelapseRetentionPreviewRsp {
	candidates := make([]proto.ScreenTimelapseRetentionCleanupCandidate, 0, len(plan.Candidates))
	var wouldDeleteBytes int64
	for _, candidate := range plan.Candidates {
		candidates = append(candidates, proto.ScreenTimelapseRetentionCleanupCandidate{
			Date:    candidate.Date,
			Bytes:   candidate.Bytes,
			Reasons: append([]string(nil), candidate.Reasons...),
		})
		wouldDeleteBytes += candidate.Bytes
	}

	skipped := make([]proto.ScreenTimelapseRetentionCleanupSkipped, 0, len(plan.Skipped))
	for _, day := range plan.Skipped {
		skipped = append(skipped, proto.ScreenTimelapseRetentionCleanupSkipped{
			Date: day.Date, Bytes: day.Bytes, Reason: day.Reason,
		})
	}

	warnings := make([]proto.ScreenTimelapseRetentionWarning, 0, len(plan.Warnings))
	for _, warning := range plan.Warnings {
		warnings = append(warnings, proto.ScreenTimelapseRetentionWarning{
			Path: warning.Path, Reason: warning.Reason, Error: warning.Error,
		})
	}

	return proto.ScreenTimelapseRetentionPreviewRsp{
		DryRun:                true,
		BeforeBytes:           plan.BeforeBytes,
		ProjectedAfterBytes:   plan.ProjectedAfterBytes,
		OverLimitAfterCleanup: plan.OverLimit,
		WouldDeleteDays:       len(candidates),
		WouldDeleteBytes:      wouldDeleteBytes,
		Candidates:            candidates,
		Skipped:               skipped,
		Warnings:              warnings,
	}
}
