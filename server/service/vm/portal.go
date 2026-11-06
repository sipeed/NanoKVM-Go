package vm

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

const (
	PortalConfigFile    = "/etc/kvm/portal.json"
	portalConfigVersion = 1
	portalConfigLimit   = 32
	portalDimensionMax  = 32768
)

var (
	portalConfigMutex sync.RWMutex
	portalReadFile    = os.ReadFile
	portalWriteFile   = writeMenuBarFileAtomically
	portalIDPattern   = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

	// portalAutoAreaWritten remembers the auto area that reached disk so repeated
	// browser reports do not wear the boot medium.
	portalAutoAreaWritten struct {
		sync.Mutex
		area *proto.PortalAutoArea
	}
)

func (s *Service) GetPortalConfig(c *gin.Context) {
	var rsp proto.Response

	config, configured, err := readPortalConfig()
	if err != nil {
		rsp.ErrRsp(c, -1, "get portal config failed")
		return
	}

	rsp.OkRspWithData(c, &proto.GetPortalConfigRsp{
		Configured:   configured,
		PortalConfig: *config,
	})
}

func (s *Service) SetPortalConfig(c *gin.Context) {
	var req proto.PortalConfig
	var rsp proto.Response

	if err := c.ShouldBindJSON(&req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if err := validatePortalConfig(&req); err != nil {
		rsp.ErrRsp(c, -1, "invalid portal config")
		return
	}
	normalizePortalConfig(&req)

	data, err := json.Marshal(&req)
	if err != nil {
		rsp.ErrRsp(c, -2, "encode portal config failed")
		return
	}

	portalConfigMutex.Lock()
	// The browser config payload never carries the detected auto area, so keep
	// the stored one instead of dropping it on every settings save.
	if req.AutoArea == nil {
		if stored, _, storedErr := readPortalConfigLocked(); storedErr == nil {
			req.AutoArea = stored.AutoArea
			if data, err = json.Marshal(&req); err != nil {
				portalConfigMutex.Unlock()
				rsp.ErrRsp(c, -2, "encode portal config failed")
				return
			}
		}
	}
	err = portalWriteFile(PortalConfigFile, data, 0o644)
	portalConfigMutex.Unlock()
	if err != nil {
		rsp.ErrRsp(c, -2, "write portal config failed")
		return
	}

	portalAutoAreaWritten.Lock()
	portalAutoAreaWritten.area = req.AutoArea
	portalAutoAreaWritten.Unlock()

	rsp.OkRsp(c)
}

func readPortalConfig() (*proto.PortalConfig, bool, error) {
	portalConfigMutex.RLock()
	defer portalConfigMutex.RUnlock()

	return readPortalConfigLocked()
}

// LoadPortalConfig returns the portal configuration without holding the file
// lock after the call. Other services (MCP) need the same persisted state the
// browser UI writes.
func LoadPortalConfig() (*proto.PortalConfig, bool, error) {
	return readPortalConfig()
}

func readPortalConfigLocked() (*proto.PortalConfig, bool, error) {
	data, err := portalReadFile(PortalConfigFile)
	if errors.Is(err, os.ErrNotExist) {
		return defaultPortalConfig(), false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var config proto.PortalConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return defaultPortalConfig(), false, nil
	}
	if err := validatePortalConfig(&config); err != nil {
		return defaultPortalConfig(), false, nil
	}
	normalizePortalConfig(&config)

	return &config, true, nil
}

func normalizePortalConfig(config *proto.PortalConfig) {
	if config.ManualRegions == nil {
		config.ManualRegions = []proto.PortalManualRegion{}
	}
	if config.Resolutions == nil {
		config.Resolutions = []proto.PortalResolution{}
	}
}
func defaultPortalConfig() *proto.PortalConfig {
	return &proto.PortalConfig{
		Version:       portalConfigVersion,
		Mode:          "auto",
		ManualRegions: []proto.PortalManualRegion{},
		Resolutions:   []proto.PortalResolution{},
	}
}

func validatePortalConfig(config *proto.PortalConfig) error {
	if config == nil || config.Version != portalConfigVersion {
		return errors.New("unsupported portal config version")
	}
	if config.Mode != "off" && config.Mode != "auto" && config.Mode != "manual" {
		return errors.New("invalid portal mode")
	}
	if len(config.ManualRegions) > portalConfigLimit || len(config.Resolutions) > portalConfigLimit {
		return errors.New("too many portal entries")
	}

	regionIDs := make(map[string]struct{}, len(config.ManualRegions))
	for _, region := range config.ManualRegions {
		if !portalIDPattern.MatchString(region.ID) {
			return errors.New("invalid portal region id")
		}
		if _, exists := regionIDs[region.ID]; exists {
			return errors.New("duplicate portal region id")
		}
		regionIDs[region.ID] = struct{}{}
		if err := validatePortalRegion(region); err != nil {
			return err
		}
	}

	resolutionKeys := make(map[string]struct{}, len(config.Resolutions))
	for _, resolution := range config.Resolutions {
		if !validPortalDimension(resolution.Width) || !validPortalDimension(resolution.Height) {
			return errors.New("invalid portal resolution")
		}
		key := fmt.Sprintf("%dx%d", resolution.Width, resolution.Height)
		if _, exists := resolutionKeys[key]; exists {
			return errors.New("duplicate portal resolution")
		}
		resolutionKeys[key] = struct{}{}
	}

	if config.SelectedRegionID != "" && config.SelectedResolution != "" {
		return errors.New("multiple portal selections")
	}
	if config.SelectedRegionID != "" {
		if _, exists := regionIDs[config.SelectedRegionID]; !exists {
			return errors.New("selected portal region not found")
		}
	}
	if config.SelectedResolution != "" {
		if _, exists := resolutionKeys[config.SelectedResolution]; !exists {
			return errors.New("selected portal resolution not found")
		}
	}

	if config.AutoArea != nil {
		if err := validatePortalRegion(proto.PortalManualRegion{
			ID:           "auto",
			SourceWidth:  config.AutoArea.SourceWidth,
			SourceHeight: config.AutoArea.SourceHeight,
			ContentRect:  config.AutoArea.ContentRect,
		}); err != nil {
			return fmt.Errorf("invalid portal auto area: %w", err)
		}
		if config.AutoArea.UpdatedAt < 0 {
			return errors.New("invalid portal auto area timestamp")
		}
	}

	return nil
}

func validatePortalRegion(region proto.PortalManualRegion) error {
	if !validPortalDimension(region.SourceWidth) || !validPortalDimension(region.SourceHeight) {
		return errors.New("invalid portal source dimensions")
	}
	rect := region.ContentRect
	if rect.Left < 0 || rect.Top < 0 || rect.Width <= 0 || rect.Height <= 0 {
		return errors.New("invalid portal content rectangle")
	}
	if rect.Left > region.SourceWidth-rect.Width || rect.Top > region.SourceHeight-rect.Height {
		return errors.New("portal content rectangle outside source")
	}
	return nil
}

func validPortalDimension(value int) bool {
	return value > 0 && value <= portalDimensionMax
}

// SetPortalDetectedArea stores the last stable auto-detected content area
// reported by a browser. The area is runtime state, not a user setting, so it
// lives beside the portal config instead of the selection fields.
func (s *Service) SetPortalDetectedArea(c *gin.Context) {
	var req proto.SetPortalDetectedAreaReq
	var rsp proto.Response

	if err := c.ShouldBindJSON(&req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	var next *proto.PortalAutoArea
	if req.ContentRect != nil {
		if !validPortalDimension(req.SourceWidth) || !validPortalDimension(req.SourceHeight) {
			rsp.ErrRsp(c, -1, "invalid detected area")
			return
		}
		// The browser reports a rounded downscaled estimate, so a rectangle can
		// overshoot the frame by a pixel or two. Clamp it instead of dropping the
		// whole report: a slightly trimmed area is still the correct crop.
		rect := clampPortalContentRect(*req.ContentRect, req.SourceWidth, req.SourceHeight)
		if rect.Width <= 0 || rect.Height <= 0 {
			rsp.ErrRsp(c, -1, "invalid detected area")
			return
		}
		next = &proto.PortalAutoArea{
			SourceWidth:  req.SourceWidth,
			SourceHeight: req.SourceHeight,
			ContentRect:  rect,
			UpdatedAt:    time.Now().UnixMilli(),
		}
	}

	if !portalAutoAreaChanged(next) {
		rsp.OkRsp(c)
		return
	}

	portalConfigMutex.Lock()
	config, _, err := readPortalConfigLocked()
	if err == nil {
		config.AutoArea = next
		var data []byte
		if data, err = json.Marshal(config); err == nil {
			err = portalWriteFile(PortalConfigFile, data, 0o644)
		}
	}
	portalConfigMutex.Unlock()

	if err != nil {
		rsp.ErrRsp(c, -2, "write portal detected area failed")
		return
	}

	portalAutoAreaWritten.Lock()
	portalAutoAreaWritten.area = next
	portalAutoAreaWritten.Unlock()

	rsp.OkRsp(c)
}

// clampPortalContentRect trims a reported rectangle into the source frame.
func clampPortalContentRect(rect proto.PortalContentRect, sourceWidth int, sourceHeight int) proto.PortalContentRect {
	return proto.PortalContentRect{
		Left:   min(max(rect.Left, 0), sourceWidth),
		Top:    min(max(rect.Top, 0), sourceHeight),
		Width:  min(max(rect.Width, 0), sourceWidth-min(max(rect.Left, 0), sourceWidth)),
		Height: min(max(rect.Height, 0), sourceHeight-min(max(rect.Top, 0), sourceHeight)),
	}
}

// portalAutoAreaChanged reports whether the reported area differs from the last
// one written to disk. The tolerance matches the browser-side tracker so a
// drifting border estimate does not rewrite the config file on every frame.
func portalAutoAreaChanged(next *proto.PortalAutoArea) bool {
	portalAutoAreaWritten.Lock()
	defer portalAutoAreaWritten.Unlock()

	last := portalAutoAreaWritten.area
	if last == nil {
		config, _, err := readPortalConfig()
		if err == nil {
			last = config.AutoArea
		}
		portalAutoAreaWritten.area = last
	}

	return !sameDetectedArea(last, next)
}

func sameDetectedArea(first *proto.PortalAutoArea, second *proto.PortalAutoArea) bool {
	if first == nil || second == nil {
		return first == nil && second == nil
	}
	if first.SourceWidth != second.SourceWidth || first.SourceHeight != second.SourceHeight {
		return false
	}

	tolerance := first.SourceWidth
	if first.SourceHeight > tolerance {
		tolerance = first.SourceHeight
	}
	tolerance = tolerance * 2 / 1000
	if tolerance < 2 {
		tolerance = 2
	}

	return absDiff(first.ContentRect.Left, second.ContentRect.Left) <= tolerance &&
		absDiff(first.ContentRect.Top, second.ContentRect.Top) <= tolerance &&
		absDiff(first.ContentRect.Width, second.ContentRect.Width) <= tolerance &&
		absDiff(first.ContentRect.Height, second.ContentRect.Height) <= tolerance
}

func absDiff(first int, second int) int {
	if first > second {
		return first - second
	}
	return second - first
}
