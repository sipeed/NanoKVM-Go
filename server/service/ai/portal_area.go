package ai

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/vm"
)

// contentArea is the effective content rectangle of the remote frame in
// normalized 0..1 source coordinates. mode "none" means the whole frame.
//
// ponytail: the auto area is reported by a browser, never detected here. A host
// letterbox change made while no web UI is open keeps the last persisted rect
// until the aspect guard rejects it. Upgrade: detect on raw VIN frames.
type contentArea struct {
	left      float64
	top       float64
	width     float64
	height    float64
	source    string
	updatedAt int64
}

const (
	areaSourceAuto   = "auto"
	areaSourceManual = "manual"
	areaSourceNone   = "none"

	// Matches web/src/lib/screen-crop.ts: a region recorded for a different
	// frame aspect cannot be rescaled safely.
	portalAspectTolerance = 0.01

	// Measured on the AX620E IVPS/JPEG path through the snapshot socket: odd
	// crop origins and crops smaller than 32px are rejected with
	// "-15 IVPS config failed", and odd sizes grow by one pixel.
	minimumCropPixels = 32

	// kvm_vin rejects a crop origin outside the frame with this code. It means
	// the cached frame size went stale after a resolution change.
	snapshotRetROIOutOfBounds = -12
)

var (
	portalCaptureSize captureSizeCache
	loadPortalConfig  = vm.LoadPortalConfig
)

// captureSizeCache remembers the snapshot frame size. The snapshot protocol
// takes pixel crops but never reports the frame size before the capture, so the
// size is learned once from a full-frame capture.
//
// ponytail: learned size instead of a queried one. Upgrade: extend the snapshot
// protocol with normalized crop parameters.
type captureSizeCache struct {
	mu     sync.Mutex
	width  int
	height int
}

func (cache *captureSizeCache) get() (int, int, bool) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.width, cache.height, cache.width > 0 && cache.height > 0
}

func (cache *captureSizeCache) set(width int, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	cache.mu.Lock()
	cache.width = width
	cache.height = height
	cache.mu.Unlock()
}

func (cache *captureSizeCache) invalidate() {
	cache.mu.Lock()
	cache.width = 0
	cache.height = 0
	cache.mu.Unlock()
}

// ensureCaptureSize returns the current snapshot frame size, learning it with a
// throwaway full-frame capture when it is still unknown.
func (cache *captureSizeCache) ensureCaptureSize(ctx context.Context) (int, int, bool) {
	if width, height, ok := cache.get(); ok {
		return width, height, true
	}

	timeoutMS := defaultSnapshotTimeout
	header, _, err := captureSnapshot(ctx, SnapshotRequest{
		Quality:   40,
		TimeoutMS: &timeoutMS,
	})
	if err != nil || !header.OK {
		log.Warnf("portal area: cannot learn capture size: %v (ret %d)", err, header.RetCode)
		return 0, 0, false
	}

	cache.set(header.Width, header.Height)
	return header.Width, header.Height, true
}

// currentContentArea resolves the persisted portal configuration into the crop
// that user control already applies. When the configuration cannot be read the
// whole frame is used, which is the pre-existing MCP behavior.
func currentContentArea(ctx context.Context) (contentArea, int, int, bool) {
	full := contentArea{width: 1, height: 1, source: areaSourceNone}

	config, _, err := loadPortalConfig()
	if err != nil || config == nil {
		if err != nil {
			log.Warnf("portal area: cannot read portal config: %v", err)
		}
		return full, 0, 0, false
	}
	if !portalConfigNeedsFrameSize(config) {
		return full, 0, 0, false
	}

	width, height, ok := portalCaptureSize.ensureCaptureSize(ctx)
	if !ok {
		return full, 0, 0, false
	}

	area := resolveContentArea(config, width, height)
	if area.source == areaSourceNone {
		return full, width, height, false
	}
	return area, width, height, true
}

// portalConfigNeedsFrameSize reports whether an active crop has to be rescaled
// against the current frame size at all.
func portalConfigNeedsFrameSize(config *proto.PortalConfig) bool {
	switch config.Mode {
	case "manual":
		if config.SelectedRegionID != "" {
			return true
		}
		return config.SelectedResolution != ""
	case "auto":
		return config.AutoArea != nil
	default:
		return false
	}
}

// resolveContentArea mirrors the browser crop rules for the given frame size.
func resolveContentArea(config *proto.PortalConfig, sourceWidth int, sourceHeight int) contentArea {
	none := contentArea{width: 1, height: 1, source: areaSourceNone}
	if config == nil || sourceWidth <= 0 || sourceHeight <= 0 {
		return none
	}

	switch config.Mode {
	case "manual":
		if config.SelectedRegionID != "" {
			for _, region := range config.ManualRegions {
				if region.ID != config.SelectedRegionID {
					continue
				}
				if !sameFrameAspect(region.SourceWidth, region.SourceHeight, sourceWidth, sourceHeight) {
					return none
				}
				return areaFromRect(region.SourceWidth, region.SourceHeight, region.ContentRect, areaSourceManual, 0)
			}
			return none
		}
		if config.SelectedResolution != "" {
			for _, resolution := range config.Resolutions {
				if fmt.Sprintf("%dx%d", resolution.Width, resolution.Height) != config.SelectedResolution {
					continue
				}
				return resolutionArea(resolution, sourceWidth, sourceHeight)
			}
		}
		return none
	case "auto":
		if config.AutoArea == nil {
			return none
		}
		if !sameFrameAspect(config.AutoArea.SourceWidth, config.AutoArea.SourceHeight, sourceWidth, sourceHeight) {
			return none
		}
		return areaFromRect(
			config.AutoArea.SourceWidth,
			config.AutoArea.SourceHeight,
			config.AutoArea.ContentRect,
			areaSourceAuto,
			config.AutoArea.UpdatedAt,
		)
	default:
		return none
	}
}

func areaFromRect(
	sourceWidth int,
	sourceHeight int,
	rect proto.PortalContentRect,
	source string,
	updatedAt int64,
) contentArea {
	if sourceWidth <= 0 || sourceHeight <= 0 {
		return contentArea{width: 1, height: 1, source: areaSourceNone}
	}

	left := float64(rect.Left) / float64(sourceWidth)
	top := float64(rect.Top) / float64(sourceHeight)
	width := float64(rect.Width) / float64(sourceWidth)
	height := float64(rect.Height) / float64(sourceHeight)

	return contentArea{
		left:      clamp01(left),
		top:       clamp01(top),
		width:     clamp01(width),
		height:    clamp01(height),
		source:    source,
		updatedAt: updatedAt,
	}
}

// resolutionArea mirrors getScreenCropFromResolution: the largest centered
// rectangle of the requested aspect inside the source frame.
func resolutionArea(resolution proto.PortalResolution, sourceWidth int, sourceHeight int) contentArea {
	none := contentArea{width: 1, height: 1, source: areaSourceNone}
	if resolution.Width <= 0 || resolution.Height <= 0 || sourceWidth <= 0 || sourceHeight <= 0 {
		return none
	}

	targetRatio := float64(resolution.Width) / float64(resolution.Height)
	sourceRatio := float64(sourceWidth) / float64(sourceHeight)

	width := 1.0
	height := 1.0
	if targetRatio < sourceRatio {
		width = targetRatio / sourceRatio
	} else {
		height = sourceRatio / targetRatio
	}

	return contentArea{
		left:   clamp01((1 - width) / 2),
		top:    clamp01((1 - height) / 2),
		width:  clamp01(width),
		height: clamp01(height),
		source: areaSourceManual,
	}
}

func sameFrameAspect(recordedWidth int, recordedHeight int, sourceWidth int, sourceHeight int) bool {
	if recordedWidth <= 0 || recordedHeight <= 0 || sourceWidth <= 0 || sourceHeight <= 0 {
		return false
	}

	recordedRatio := float64(recordedWidth) / float64(recordedHeight)
	sourceRatio := float64(sourceWidth) / float64(sourceHeight)

	return math.Abs(recordedRatio-sourceRatio) <= portalAspectTolerance
}

// cropped reports whether the area removes anything from the frame.
func (area contentArea) cropped() bool {
	return area.source != areaSourceNone &&
		(area.left > 0 || area.top > 0 || area.width < 1 || area.height < 1)
}

// toSource maps a normalized content coordinate into the normalized source frame.
func (area contentArea) toSource(x float64, y float64) (float64, float64) {
	return clamp01(area.left + clamp01(x)*area.width),
		clamp01(area.top + clamp01(y)*area.height)
}

// pixelRect converts the area into a snapshot crop the encoder accepts.
func (area contentArea) pixelRect(sourceWidth int, sourceHeight int) (int, int, int, int) {
	if sourceWidth < minimumCropPixels || sourceHeight < minimumCropPixels {
		return 0, 0, 0, 0
	}

	x := int(math.Round(clamp01(area.left) * float64(sourceWidth)))
	y := int(math.Round(clamp01(area.top) * float64(sourceHeight)))
	w := int(math.Round(clamp01(area.width) * float64(sourceWidth)))
	h := int(math.Round(clamp01(area.height) * float64(sourceHeight)))

	x = clampInt(x, 0, sourceWidth-minimumCropPixels)
	y = clampInt(y, 0, sourceHeight-minimumCropPixels)
	w = clampInt(w, minimumCropPixels, sourceWidth-x)
	h = clampInt(h, minimumCropPixels, sourceHeight-y)

	return evenRect(x, y, w, h)
}

func clamp01(value float64) float64 {
	if math.IsNaN(value) {
		return 0
	}
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

// resolveAbsolutePoint maps a normalized MCP coordinate into the HID coordinate
// the browser would send for the same visible point. The default content space
// is the full content area, which is the same crop user control applies, and the
// device input profile calibration is applied on top. A screenshot sub-crop
// never changes it, so clicks stay correct when a caller crops a screenshot.
//
// Coordinates outside 0..1 are rejected instead of clamped: silently turning a
// pixel value into a frame corner would click somewhere the caller never
// intended.
func resolveAbsolutePoint(ctx context.Context, x float64, y float64, coordinateSpace string) (float64, float64, error) {
	if !(x >= 0 && x <= 1) || !(y >= 0 && y <= 1) {
		return 0, 0, fmt.Errorf(
			"absolute x/y are normalized 0.0..1.0 inside the screenshot content area, got x=%g y=%g; divide image pixels by the screenshot width and height",
			x, y,
		)
	}

	if strings.EqualFold(strings.TrimSpace(coordinateSpace), "source") {
		return x, y, nil
	}

	area, _, _, ok := currentContentArea(ctx)
	if !ok {
		area = contentArea{width: 1, height: 1, source: areaSourceNone}
	}
	calibratedX, calibratedY := calibratedAbsolutePoint(area, x, y)
	return calibratedX, calibratedY, nil
}

// snapshotPlan is the resolved snapshot request plus the region metadata that
// belongs in the tool result.
//
// contentRect is the full content area inside the source frame, which is the
// space absolute mouse coordinates live in. captureRect is the part of the
// source frame the returned image shows; it is smaller than contentRect only
// after a caller crop.
type snapshotPlan struct {
	request      SnapshotRequest
	area         contentArea
	contentRect  *ScreenshotContentRect
	captureRect  *ScreenshotContentRect
	sourceWidth  int
	sourceHeight int
	cropped      bool
}

func buildSnapshotPlan(ctx context.Context, input ScreenshotParams) snapshotPlan {
	plan := snapshotPlan{
		request: SnapshotRequest{Quality: input.Quality, TimeoutMS: input.TimeoutMS},
		area:    contentArea{width: 1, height: 1, source: areaSourceNone},
	}

	area, sourceWidth, sourceHeight, active := currentContentArea(ctx)
	plan.area = area
	plan.sourceWidth, plan.sourceHeight = sourceWidth, sourceHeight

	contentX, contentY, contentW, contentH := 0, 0, 0, 0
	if active {
		contentX, contentY, contentW, contentH = area.pixelRect(sourceWidth, sourceHeight)
		if contentW <= 0 || contentH <= 0 {
			// The frame is smaller than the encoder's minimum crop, so the
			// content area degenerates to the whole frame and mouse coordinates
			// must follow it instead of the configured area.
			plan.area = contentArea{width: 1, height: 1, source: areaSourceNone}
		}
	}
	if contentW <= 0 || contentH <= 0 {
		// The content area is the whole frame. A cached frame size is enough to
		// report it without spending an extra capture.
		if plan.sourceWidth <= 0 || plan.sourceHeight <= 0 {
			if cachedWidth, cachedHeight, ok := portalCaptureSize.get(); ok {
				plan.sourceWidth, plan.sourceHeight = cachedWidth, cachedHeight
			}
		}
		contentX, contentY, contentW, contentH = 0, 0, plan.sourceWidth, plan.sourceHeight
	}
	plan.contentRect = newScreenshotContentRect(contentX, contentY, contentW, contentH)

	if plan.area.source == areaSourceNone {
		// Without an active portal crop a caller crop keeps its legacy
		// frame-relative meaning and kvm_vin validates its bounds.
		if input.W > 0 && input.H > 0 {
			plan.request.X = input.X
			plan.request.Y = input.Y
			plan.request.W = input.W
			plan.request.H = input.H
			plan.captureRect = newScreenshotContentRect(input.X, input.Y, input.W, input.H)
			plan.cropped = true
		}
		return plan
	}

	x, y, w, h := applySubCrop(input, contentX, contentY, contentW, contentH)
	if w > 0 && h > 0 && !(x == 0 && y == 0 && w == plan.sourceWidth && h == plan.sourceHeight) {
		plan.request.X = x
		plan.request.Y = y
		plan.request.W = w
		plan.request.H = h
		plan.captureRect = newScreenshotContentRect(x, y, w, h)
		plan.cropped = true
	}

	return plan
}

func newScreenshotContentRect(left int, top int, width int, height int) *ScreenshotContentRect {
	if width <= 0 || height <= 0 {
		return nil
	}
	return &ScreenshotContentRect{Left: left, Top: top, Width: width, Height: height}
}

// applySubCrop interprets the legacy screenshot x/y/w/h inside the content
// rectangle, which is passed in as x/y/w/h and is always a valid encoder crop
// here. A partial request keeps the content rectangle untouched, so the
// coordinate space of absolute mouse input never changes.
func applySubCrop(input ScreenshotParams, x int, y int, w int, h int) (int, int, int, int) {
	if input.W <= 0 || input.H <= 0 {
		return x, y, w, h
	}

	offsetX := clampInt(input.X, 0, w-minimumCropPixels)
	offsetY := clampInt(input.Y, 0, h-minimumCropPixels)
	width := clampInt(input.W, minimumCropPixels, w-offsetX)
	height := clampInt(input.H, minimumCropPixels, h-offsetY)

	return evenRect(x+offsetX, y+offsetY, width, height)
}

// evenRect aligns a crop to what the encoder accepts: an even origin and an
// even size of at least minimumCropPixels. A request is never trimmed below
// what was asked for, because a slightly larger crop is better than a failed
// capture.
func evenRect(x int, y int, w int, h int) (int, int, int, int) {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	x -= x % 2
	y -= y % 2

	if w > 0 {
		w = clampInt(w-w%2, minimumCropPixels, w)
	}
	if h > 0 {
		h = clampInt(h-h%2, minimumCropPixels, h)
	}

	return x, y, w, h
}
