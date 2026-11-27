package ai

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"NanoKVM-Server/service/media"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultTypeDelayMS  = 30
	defaultHoldMS       = 50
	defaultClickDelayMS = 50
	maxTypeTextRunes    = 1024
)

type TypeTextParams struct {
	Text    string `json:"text" jsonschema:"text to type on the remote host"`
	DelayMS int    `json:"delayMs,omitempty" jsonschema:"delay between HID reports in milliseconds"`
}

type TypeTextResult struct {
	Typed        int    `json:"typed"`
	Skipped      string `json:"skipped,omitempty"`
	SkippedCount int    `json:"skippedCount"`
}

type PressKeysParams struct {
	Keys   []string `json:"keys" jsonschema:"KeyboardEvent.code key names"`
	HoldMS int      `json:"holdMs,omitempty" jsonschema:"hold time in milliseconds"`
}

type MoveMouseParams struct {
	Mode            string  `json:"mode" jsonschema:"absolute or relative"`
	X               float64 `json:"x,omitempty" jsonschema:"absolute normalized x inside the screenshot content area, 0.0 to 1.0 of the image width (never pixels)"`
	Y               float64 `json:"y,omitempty" jsonschema:"absolute normalized y inside the screenshot content area, 0.0 to 1.0 of the image height (never pixels)"`
	DeltaX          int     `json:"deltaX,omitempty" jsonschema:"relative x movement"`
	DeltaY          int     `json:"deltaY,omitempty" jsonschema:"relative y movement"`
	CoordinateSpace string  `json:"coordinateSpace,omitempty" jsonschema:"content (default, relative to the screenshot content area) or source (whole captured frame)"`
}

type ClickMouseParams struct {
	Button          string  `json:"button" jsonschema:"left, right, middle, back, or forward"`
	Clicks          int     `json:"clicks,omitempty" jsonschema:"number of clicks"`
	Mode            string  `json:"mode,omitempty" jsonschema:"absolute or relative"`
	X               float64 `json:"x,omitempty" jsonschema:"absolute normalized x inside the screenshot content area, 0.0 to 1.0 of the image width (never pixels)"`
	Y               float64 `json:"y,omitempty" jsonschema:"absolute normalized y inside the screenshot content area, 0.0 to 1.0 of the image height (never pixels)"`
	CoordinateSpace string  `json:"coordinateSpace,omitempty" jsonschema:"content (default, relative to the screenshot content area) or source (whole captured frame)"`
}

type ScrollMouseParams struct {
	DeltaY int `json:"deltaY" jsonschema:"vertical wheel movement"`
}

type ScreenshotParams struct {
	Quality   int  `json:"quality,omitempty"`
	TimeoutMS *int `json:"timeoutMs,omitempty"`
	X         int  `json:"x,omitempty" jsonschema:"crop origin x inside the content area; needs w and h"`
	Y         int  `json:"y,omitempty" jsonschema:"crop origin y inside the content area; needs w and h"`
	W         int  `json:"w,omitempty" jsonschema:"crop width inside the content area; 0 keeps the whole content area; a crop does not change mouse coordinates"`
	H         int  `json:"h,omitempty" jsonschema:"crop height inside the content area; 0 keeps the whole content area; a crop does not change mouse coordinates"`
}

// ScreenshotContentRect is a rectangle of the captured source frame in source
// pixels. It describes either the full content area or the returned image.
type ScreenshotContentRect struct {
	Left   int `json:"left"`
	Top    int `json:"top"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type ScreenshotResult struct {
	OK         bool   `json:"ok"`
	RetCode    int    `json:"retCode"`
	Message    string `json:"message"`
	SessionID  string `json:"sessionId,omitempty"`
	CaptureID  string `json:"captureId,omitempty"`
	CapturedAt string `json:"capturedAt,omitempty"`
	Source     string `json:"source,omitempty"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Size       int    `json:"size"`
	// SourceWidth/SourceHeight describe the whole captured frame. Width/Height
	// above stay the size of the returned image.
	//
	// Content is the full content area: the space absolute mouse coordinates live
	// in. Capture is the part of the source frame the returned image shows, which
	// is smaller than Content only after a caller crop.
	SourceWidth     int                    `json:"sourceWidth,omitempty"`
	SourceHeight    int                    `json:"sourceHeight,omitempty"`
	CoordinateSpace string                 `json:"coordinateSpace,omitempty"`
	AreaSource      string                 `json:"areaSource,omitempty"`
	AreaUpdatedAt   int64                  `json:"areaUpdatedAt,omitempty"`
	Content         *ScreenshotContentRect `json:"content,omitempty"`
	Capture         *ScreenshotContentRect `json:"capture,omitempty"`
}

type ToolOKResult struct {
	OK bool `json:"ok"`
}

type MediaSessionCreateParams struct {
	SpeakerReceive bool `json:"speakerReceive,omitempty" jsonschema:"receive the controlled host speaker/HDMI audio; NanoKVM server to client; add a recvonly Opus audio transceiver"`
	MicSend        bool `json:"micSend,omitempty" jsonschema:"send audio from the client to the controlled host UAC2 virtual microphone as a 48 kHz stereo Opus WebRTC track"`
	VideoReceive   bool `json:"videoReceive,omitempty" jsonschema:"receive the controlled host video track; uses the combined audio/video signaling endpoint"`
	TTLSeconds     int  `json:"ttlSeconds,omitempty" jsonschema:"authorization lifetime in seconds; default and maximum are 600; expiration blocks new signaling"`
}

type MediaSessionResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	SessionID       string             `json:"sessionId"`
	Token           string             `json:"token,omitempty"`
	SignalingPath   string             `json:"signalingPath"`
	ExpiresAt       string             `json:"expiresAt"`
	Capabilities    media.Capabilities `json:"capabilities"`
	Signaling       *SignalingEndpoint `json:"signaling,omitempty"`
	Audio           *AudioContract     `json:"audio,omitempty"`
	TokenUsage      *TokenUsage        `json:"tokenUsage,omitempty"`
}

type MediaSessionTokenParams struct {
	Token string `json:"token" jsonschema:"opaque media-session token returned by media_session_create; sensitive; never log it"`
}

var captureSnapshot = CaptureSnapshotContext

func NewMCPHandler() http.Handler {
	remote := NewRemote()
	limiter := newToolRateLimiter(20 * time.Millisecond)
	server := mcp.NewServer(&mcp.Implementation{Name: "nanokvm-remote-control", Version: "v1.0.0"}, &mcp.ServerOptions{
		Capabilities: &mcp.ServerCapabilities{Tools: &mcp.ToolCapabilities{ListChanged: true}},
	})
	controlNames := registerTools(server, remote, limiter)
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server { return server }, nil)
	return newMCPController(server, handler, newMemfabBridge(memfabToolSocketPath), getMemoryFabricEnabled, controlNames)
}

func registerTools(server *mcp.Server, remote *Remote, limiter *toolRateLimiter) map[string]struct{} {
	names := make(map[string]struct{})
	registerControlTool(names, server, &mcp.Tool{Name: "type_text", Description: "Type literal text using the target keyboard layout selected on the NanoKVM. Send punctuation and supported accents directly. Unsupported characters fail before typing."}, typeTextHandler(remote, limiter))
	registerControlTool(names, server, &mcp.Tool{Name: "press_keys", Description: "Press one keyboard shortcut on the remote host"}, pressKeysHandler(remote, limiter))
	registerControlTool(names, server, &mcp.Tool{Name: "move_mouse", Description: "Move the remote mouse pointer. Absolute x/y are normalized coordinates (0.0-1.0, never pixels) inside the screenshot content area, which is the de-letterboxed frame the screenshot tool returns."}, moveMouseHandler(remote, limiter))
	registerControlTool(names, server, &mcp.Tool{Name: "click_mouse", Description: "Click a remote mouse button. Absolute x/y are normalized coordinates (0.0-1.0, never pixels) in the same content area as move_mouse and screenshot. If the target is an iPhone, must right-click once before the first click."}, clickMouseHandler(remote, limiter))
	registerControlTool(names, server, &mcp.Tool{Name: "scroll_mouse", Description: "Scroll the remote mouse wheel"}, scrollMouseHandler(remote, limiter))
	registerControlTool(names, server, &mcp.Tool{Name: "screenshot", Description: "Capture a JPEG screenshot from the remote display. The image is cropped to the effective content area, so black borders are removed whenever portal cropping is enabled; absolute mouse coordinates stay relative to that content area, not to the returned image."}, screenshotHandler(limiter))
	registerControlTool(names, server, &mcp.Tool{Name: "media_session_help", Description: "Return the complete machine-readable NanoKVM WebRTC media protocol: control/data-plane boundary, signaling, track directions, exact Opus framing, lifecycle, pacing, interruption, recovery, and client examples. Call this before implementing a real-time media client."}, mediaSessionHelpHandler(limiter))
	registerControlTool(names, server, &mcp.Tool{Name: "media_session_create", Description: "Create a short-lived capability-scoped WebRTC authorization. The returned token is only for the signaling URL; real-time audio is carried by WebRTC RTP tracks, never by MCP."}, mediaSessionCreateHandler(limiter))
	registerControlTool(names, server, &mcp.Tool{Name: "media_session_status", Description: "Inspect an active media session without revealing its token. This cannot recover a lost token."}, mediaSessionStatusHandler(limiter))
	registerControlTool(names, server, &mcp.Tool{Name: "media_session_close", Description: "Revoke future signaling access. The client must also close its WebSocket, PeerConnection, senders, receivers, and local media queues."}, mediaSessionCloseHandler(limiter))
	return names
}

func registerControlTool[In, Out any](names map[string]struct{}, server *mcp.Server, tool *mcp.Tool, handler mcp.ToolHandlerFor[In, Out]) {
	if _, duplicate := names[tool.Name]; duplicate {
		panic(fmt.Sprintf("duplicate permanent control tool %q", tool.Name))
	}
	names[tool.Name] = struct{}{}
	mcp.AddTool(server, tool, handler)
}

func typeTextHandler(remote *Remote, limiter *toolRateLimiter) mcp.ToolHandlerFor[TypeTextParams, TypeTextResult] {
	return func(_ context.Context, _ *mcp.CallToolRequest, input TypeTextParams) (*mcp.CallToolResult, TypeTextResult, error) {
		if err := checkRate(limiter, "type_text"); err != nil {
			return nil, TypeTextResult{}, err
		}

		textRunes := []rune(input.Text)
		if len(textRunes) > maxTypeTextRunes {
			return nil, TypeTextResult{}, fmt.Errorf("text exceeds maximum length of %d runes", maxTypeTextRunes)
		}

		err := remote.TypeText(input.Text, time.Duration(clampDelayMS(input.DelayMS, defaultTypeDelayMS))*time.Millisecond)
		if err != nil {
			return nil, TypeTextResult{}, err
		}

		return nil, TypeTextResult{
			Typed: len(textRunes),
		}, nil
	}
}

func pressKeysHandler(remote *Remote, limiter *toolRateLimiter) mcp.ToolHandlerFor[PressKeysParams, ToolOKResult] {
	return func(_ context.Context, _ *mcp.CallToolRequest, input PressKeysParams) (*mcp.CallToolResult, ToolOKResult, error) {
		if err := checkRate(limiter, "press_keys"); err != nil {
			return nil, ToolOKResult{}, err
		}
		if len(input.Keys) == 0 {
			return nil, ToolOKResult{}, fmt.Errorf("keys must not be empty")
		}

		if err := remote.PressKeys(input.Keys, time.Duration(clampDelayMS(input.HoldMS, defaultHoldMS))*time.Millisecond); err != nil {
			return nil, ToolOKResult{}, err
		}
		return nil, ToolOKResult{OK: true}, nil
	}
}

func moveMouseHandler(remote *Remote, limiter *toolRateLimiter) mcp.ToolHandlerFor[MoveMouseParams, ToolOKResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input MoveMouseParams) (*mcp.CallToolResult, ToolOKResult, error) {
		if err := checkRate(limiter, "move_mouse"); err != nil {
			return nil, ToolOKResult{}, err
		}

		switch normalizedMode(input.Mode, "absolute") {
		case "absolute":
			x, y, err := resolveAbsolutePoint(ctx, input.X, input.Y, input.CoordinateSpace)
			if err != nil {
				return nil, ToolOKResult{}, err
			}
			if err := remote.MoveAbsolute(x, y); err != nil {
				return nil, ToolOKResult{}, err
			}
		case "relative":
			if err := remote.MoveRelative(input.DeltaX, input.DeltaY); err != nil {
				return nil, ToolOKResult{}, err
			}
		default:
			return nil, ToolOKResult{}, fmt.Errorf("unknown mouse mode: %s", input.Mode)
		}

		return nil, ToolOKResult{OK: true}, nil
	}
}

func clickMouseHandler(remote *Remote, limiter *toolRateLimiter) mcp.ToolHandlerFor[ClickMouseParams, ToolOKResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ClickMouseParams) (*mcp.CallToolResult, ToolOKResult, error) {
		if err := checkRate(limiter, "click_mouse"); err != nil {
			return nil, ToolOKResult{}, err
		}

		switch normalizedMode(input.Mode, "relative") {
		case "absolute":
			button := strings.ToLower(strings.TrimSpace(input.Button))
			if button == "" {
				button = "left"
			}
			if button != "left" && button != "right" {
				return nil, ToolOKResult{}, fmt.Errorf("absolute mouse clicks only support left and right buttons")
			}

			x, y, err := resolveAbsolutePoint(ctx, input.X, input.Y, input.CoordinateSpace)
			if err != nil {
				return nil, ToolOKResult{}, err
			}
			clicks := clampClickCount(input.Clicks)
			for i := 0; i < clicks; i++ {
				if err := remote.TouchAbsoluteButton(x, y, button, defaultClickDelayMS*time.Millisecond); err != nil {
					return nil, ToolOKResult{}, err
				}
				if i+1 < clicks {
					time.Sleep(defaultClickDelayMS * time.Millisecond)
				}
			}
			return nil, ToolOKResult{OK: true}, nil
		case "relative":
		default:
			return nil, ToolOKResult{}, fmt.Errorf("unknown mouse mode: %s", input.Mode)
		}

		button := strings.ToLower(strings.TrimSpace(input.Button))
		if button == "" {
			button = "left"
		}
		if err := remote.Click(button, clampClickCount(input.Clicks), defaultClickDelayMS*time.Millisecond); err != nil {
			return nil, ToolOKResult{}, err
		}

		return nil, ToolOKResult{OK: true}, nil
	}
}

func scrollMouseHandler(remote *Remote, limiter *toolRateLimiter) mcp.ToolHandlerFor[ScrollMouseParams, ToolOKResult] {
	return func(_ context.Context, _ *mcp.CallToolRequest, input ScrollMouseParams) (*mcp.CallToolResult, ToolOKResult, error) {
		if err := checkRate(limiter, "scroll_mouse"); err != nil {
			return nil, ToolOKResult{}, err
		}

		if err := remote.Scroll(input.DeltaY); err != nil {
			return nil, ToolOKResult{}, err
		}
		return nil, ToolOKResult{OK: true}, nil
	}
}

func screenshotHandler(limiter *toolRateLimiter) mcp.ToolHandlerFor[ScreenshotParams, ScreenshotResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ScreenshotParams) (*mcp.CallToolResult, ScreenshotResult, error) {
		if err := checkRate(limiter, "screenshot"); err != nil {
			return nil, ScreenshotResult{}, err
		}

		plan := buildSnapshotPlan(ctx, input)
		if plan.cropped {
			// A cropped capture reports only the returned image dimensions. Probe a
			// fresh full frame first so a resolution change cannot leave the cached
			// source size (and therefore the crop coordinates/metadata) stale.
			portalCaptureSize.invalidate()
			portalCaptureSize.ensureCaptureSize(ctx)
			plan = buildSnapshotPlan(ctx, input)
		}
		header, jpeg, err := captureSnapshot(ctx, plan.request)
		if err == nil && header.RetCode == snapshotRetROIOutOfBounds && plan.cropped {
			// The cached frame size is stale after a resolution change: learn it
			// again and repeat the crop once.
			portalCaptureSize.invalidate()
			plan = buildSnapshotPlan(ctx, input)
			header, jpeg, err = captureSnapshot(ctx, plan.request)
		}
		if err == nil && header.OK && !plan.cropped {
			portalCaptureSize.set(header.Width, header.Height)
		}

		output := ScreenshotResult{
			OK:              header.OK,
			RetCode:         header.RetCode,
			Message:         header.Message,
			SessionID:       header.SessionID,
			CaptureID:       header.CaptureID,
			CapturedAt:      header.CapturedAt,
			Source:          header.Source,
			Width:           header.Width,
			Height:          header.Height,
			Size:            header.Size,
			CoordinateSpace: "content",
			AreaSource:      plan.area.source,
			AreaUpdatedAt:   plan.area.updatedAt,
			SourceWidth:     plan.sourceWidth,
			SourceHeight:    plan.sourceHeight,
			Content:         plan.contentRect,
			Capture:         plan.captureRect,
		}
		if !plan.cropped && header.OK {
			// A full-frame capture reports the real frame size, which is the
			// content area as well as the returned image.
			output.SourceWidth = header.Width
			output.SourceHeight = header.Height
			output.Content = &ScreenshotContentRect{Width: header.Width, Height: header.Height}
			output.Capture = &ScreenshotContentRect{Width: header.Width, Height: header.Height}
		}

		if err != nil {
			output.OK = false
			output.Message = err.Error()
			return toolError(err.Error()), output, nil
		}

		if !header.OK || len(jpeg) == 0 {
			message := header.Message
			if message == "" {
				message = "screenshot capture failed"
			}
			return toolError(message), output, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.ImageContent{Data: jpeg, MIMEType: "image/jpeg"},
			},
		}, output, nil
	}
}

func mediaSessionHelpHandler(limiter *toolRateLimiter) mcp.ToolHandlerFor[struct{}, MediaSessionHelpResult] {
	return func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, MediaSessionHelpResult, error) {
		if err := checkRate(limiter, "media_session_help"); err != nil {
			return nil, MediaSessionHelpResult{}, err
		}
		return nil, newMediaSessionHelp(), nil
	}
}

func mediaSessionCreateHandler(limiter *toolRateLimiter) mcp.ToolHandlerFor[MediaSessionCreateParams, MediaSessionResult] {
	return func(_ context.Context, _ *mcp.CallToolRequest, input MediaSessionCreateParams) (*mcp.CallToolResult, MediaSessionResult, error) {
		if err := checkRate(limiter, "media_session_create"); err != nil {
			return nil, MediaSessionResult{}, err
		}
		ttl := time.Duration(input.TTLSeconds) * time.Second
		session, err := media.DefaultManager().Create("mcp", media.Capabilities{
			SpeakerReceive: input.SpeakerReceive,
			MicSend:        input.MicSend,
			VideoReceive:   input.VideoReceive,
		}, ttl)
		if err != nil {
			return toolError(err.Error()), MediaSessionResult{}, nil
		}
		signalingPath := "/api/media/audio/webrtc?media_session=" + session.Token
		if session.Capabilities.VideoReceive {
			signalingPath = "/api/media/webrtc?media_session=" + session.Token
		}
		return nil, MediaSessionResult{
			ProtocolVersion: mediaProtocolVersion,
			SessionID:       session.ID,
			Token:           session.Token,
			SignalingPath:   signalingPath,
			ExpiresAt:       session.ExpiresAt.Format(time.RFC3339),
			Capabilities:    session.Capabilities,
			Signaling:       newSignalingEndpoint(signalingPath),
			Audio:           ptrAudioContract(newAudioContract()),
			TokenUsage:      newTokenUsage(),
		}, nil
	}
}

func mediaSessionStatusHandler(limiter *toolRateLimiter) mcp.ToolHandlerFor[MediaSessionTokenParams, MediaSessionResult] {
	return func(_ context.Context, _ *mcp.CallToolRequest, input MediaSessionTokenParams) (*mcp.CallToolResult, MediaSessionResult, error) {
		if err := checkRate(limiter, "media_session_status"); err != nil {
			return nil, MediaSessionResult{}, err
		}
		session, ok := media.DefaultManager().Get(input.Token)
		if !ok {
			return toolError("media session not found or expired"), MediaSessionResult{}, nil
		}
		path := "/api/media/audio/webrtc"
		if session.Capabilities.VideoReceive {
			path = "/api/media/webrtc"
		}
		return nil, MediaSessionResult{ProtocolVersion: mediaProtocolVersion, SessionID: session.ID, SignalingPath: path, ExpiresAt: session.ExpiresAt.Format(time.RFC3339), Capabilities: session.Capabilities, Signaling: newSignalingEndpoint(path), Audio: ptrAudioContract(newAudioContract())}, nil
	}
}

func mediaSessionCloseHandler(limiter *toolRateLimiter) mcp.ToolHandlerFor[MediaSessionTokenParams, ToolOKResult] {
	return func(_ context.Context, _ *mcp.CallToolRequest, input MediaSessionTokenParams) (*mcp.CallToolResult, ToolOKResult, error) {
		if err := checkRate(limiter, "media_session_close"); err != nil {
			return nil, ToolOKResult{}, err
		}
		if !media.DefaultManager().Close(input.Token) {
			return toolError("media session not found or expired"), ToolOKResult{}, nil
		}
		return nil, ToolOKResult{OK: true}, nil
	}
}

func clampDelayMS(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return clampInt(value, 0, 1000)
}

func clampClickCount(value int) int {
	return clampInt(value, 1, 5)
}

func checkRate(limiter *toolRateLimiter, tool string) error {
	if limiter == nil {
		return nil
	}
	if !limiter.Allow(tool) {
		return fmt.Errorf("rate limit exceeded for %s", tool)
	}
	return nil
}

func toolError(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		IsError: true,
	}
}

func normalizedMode(mode string, fallback string) string {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	if normalized == "" {
		return fallback
	}
	return normalized
}
