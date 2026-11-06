package ai

import (
	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

var (
	mcpActionRequiresChallenge             = authservice.MCPActionRequiresChallenge
	verifyMCPChallenge                     = authservice.VerifyMCPChallenge
	screenTimelapseEnableRequiresChallenge = authservice.ScreenTimelapseEnableRequiresChallenge
	verifyScreenTimelapseEnableChallenge   = authservice.VerifySensitiveActionChallenge
	verifyScreenTimelapsePlaybackChallenge = authservice.VerifyScreenTimelapsePlaybackChallenge
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) GetConfig(c *gin.Context) {
	var rsp proto.Response

	mcpEnabled, err := getMCPEnabled()
	if err != nil {
		log.Errorf("failed to read MCP service state: %v", err)
		rsp.ErrRsp(c, -1, "get AI config failed")
		return
	}

	cfg, err := loadConfig()
	if err != nil {
		log.Errorf("failed to load AI config: %v", err)
		rsp.ErrRsp(c, -1, "get AI config failed")
		return
	}

	rsp.OkRspWithData(c, newAIConfigResponse(mcpEnabled, cfg))
}

func (s *Service) SetConfig(c *gin.Context) {
	var req proto.SetAIConfigReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	action := authservice.SensitiveActionMCPDisable
	if *req.MCPEnabled {
		action = authservice.SensitiveActionMCPEnable
	}
	if !requireMCPChallenge(c, &rsp, action, req.Challenge) {
		return
	}

	cfg, err := updateConfig(func(cfg Config) (Config, error) {
		if *req.MCPEnabled {
			return ensureAPIKey(cfg)
		}
		return cfg, nil
	})
	if err != nil {
		log.Errorf("failed to save AI config: %v", err)
		rsp.ErrRsp(c, -2, "operation failed")
		return
	}
	snapshot, err := DefaultServiceStateManager().Set(aiServiceMCP, *req.MCPEnabled, "web")
	if err != nil {
		log.Errorf("failed to set MCP service state: %v", err)
		rsp.ErrRsp(c, -2, "operation failed")
		return
	}
	mcpState := snapshot.Services[aiServiceMCP]

	rsp.OkRspWithData(c, newAIConfigResponse(mcpState.Enabled, cfg))
}

func (s *Service) GetMCPAPIKey(c *gin.Context) {
	s.getMCPAPIKey(c, authservice.SensitiveActionMCPShowKey)
}

func (s *Service) CopyMCPAPIKey(c *gin.Context) {
	s.getMCPAPIKey(c, authservice.SensitiveActionMCPCopyKey)
}

func (s *Service) getMCPAPIKey(c *gin.Context, action string) {
	var req proto.MCPAPIKeyReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if !requireMCPChallenge(c, &rsp, action, req.Challenge) {
		return
	}

	cfg, err := loadConfig()
	if err != nil {
		log.Errorf("failed to load MCP API key: %v", err)
		rsp.ErrRsp(c, -1, "get MCP API key failed")
		return
	}

	rsp.OkRspWithData(c, &proto.GetMCPAPIKeyRsp{MCPAPIKey: cfg.MCPAPIKey})
}

func (s *Service) RegenerateMCPAPIKey(c *gin.Context) {
	var req proto.MCPAPIKeyReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if !requireMCPChallenge(c, &rsp, authservice.SensitiveActionMCPRegenerateKey, req.Challenge) {
		return
	}

	cfg, err := updateConfig(regenerateAPIKey)
	if err != nil {
		log.Errorf("failed to regenerate MCP API key: %v", err)
		rsp.ErrRsp(c, -1, "operation failed")
		return
	}
	mcpEnabled, err := getMCPEnabled()
	if err != nil {
		log.Errorf("failed to read MCP service state: %v", err)
		rsp.ErrRsp(c, -1, "operation failed")
		return
	}

	rsp.OkRspWithData(c, newAIConfigResponse(mcpEnabled, cfg))
}

// newAIConfigResponse deliberately returns only a non-sensitive API-key
// fingerprint. The complete credential remains available solely through the
// protected show/copy endpoints.
func newAIConfigResponse(mcpEnabled bool, cfg Config) *proto.GetAIConfigRsp {
	rsp := &proto.GetAIConfigRsp{MCPEnabled: mcpEnabled}
	if mcpEnabled {
		rsp.MCPAPIKeyMasked = maskMCPAPIKey(cfg.MCPAPIKey)
	}
	return rsp
}

func maskMCPAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return key[:1] + "..." + key[len(key)-1:]
	}
	if len(key) <= 16 {
		return key[:4] + "..." + key[len(key)-4:]
	}
	return key[:12] + "..." + key[len(key)-6:]
}

func requireMCPChallenge(c *gin.Context, rsp *proto.Response, action, challenge string) bool {
	if !mcpActionRequiresChallenge(action) {
		return true
	}
	if err := verifyMCPChallenge(c, challenge, action); err != nil {
		log.Errorf("verify MCP challenge for %s failed: %v", action, err)
		rsp.ErrRsp(c, -3, "sensitive action verification required")
		return false
	}
	return true
}

func screenTimelapsePlaybackAuthorized(c *gin.Context) bool {
	if err := verifyScreenTimelapsePlaybackChallenge(c, authservice.ReadSensitiveChallenge(c)); err != nil {
		log.Errorf("verify screen timelapse playback challenge failed: %v", err)
		return false
	}
	return true
}

func requireScreenTimelapsePlaybackAuthorization(c *gin.Context, rsp *proto.Response) bool {
	if screenTimelapsePlaybackAuthorized(c) {
		return true
	}
	rsp.ErrRsp(c, -3, "sensitive action verification required")
	return false
}

// AuthorizeScreenTimelapsePlayback creates or verifies the short-lived Screen
// Timelapse playback authorization before the browser opens recording data.
func (s *Service) AuthorizeScreenTimelapsePlayback(c *gin.Context) {
	var rsp proto.Response
	if !requireScreenTimelapsePlaybackAuthorization(c, &rsp) {
		return
	}
	rsp.OkRsp(c)
}

func getMCPEnabled() (bool, error) {
	entry, err := DefaultServiceStateManager().Get(aiServiceMCP)
	if err != nil {
		return false, err
	}
	return entry.Enabled, nil
}

func getScreenTimelapseEnabled() (bool, error) {
	entry, err := DefaultServiceStateManager().Get(aiServiceScreenTimelapse)
	if err != nil {
		return false, err
	}
	return entry.Enabled, nil
}

func (s *Service) GetScreenTimelapseState(c *gin.Context) {
	var rsp proto.Response

	enabled, err := getScreenTimelapseEnabled()
	if err != nil {
		log.Errorf("failed to read screen timelapse service state: %v", err)
		rsp.ErrRsp(c, -1, "get screen timelapse state failed")
		return
	}

	rsp.OkRspWithData(c, proto.GetScreenTimelapseStateRsp{Enabled: enabled})
}

func (s *Service) SetScreenTimelapseState(c *gin.Context) {
	var req proto.SetScreenTimelapseStateReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if *req.Enabled && screenTimelapseEnableRequiresChallenge() {
		if err := verifyScreenTimelapseEnableChallenge(c, req.Challenge, authservice.SensitiveActionScreenTimelapseEnable); err != nil {
			log.Errorf("verify screen timelapse enable challenge failed: %v", err)
			rsp.ErrRsp(c, -3, "sensitive action verification required")
			return
		}
	}

	snapshot, err := DefaultServiceStateManager().Set(aiServiceScreenTimelapse, *req.Enabled, "web")
	if err != nil {
		log.Errorf("failed to set screen timelapse service state: %v", err)
		rsp.ErrRsp(c, -2, "operation failed")
		return
	}
	screenTimelapseState := snapshot.Services[aiServiceScreenTimelapse]

	rsp.OkRspWithData(c, proto.GetScreenTimelapseStateRsp{Enabled: screenTimelapseState.Enabled})
}
