package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"NanoKVM-Server/config"
	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"
	"NanoKVM-Server/utils"

	"github.com/gin-gonic/gin"
)

const metricsAddress = "127.0.0.1:17401"

type systemService struct {
	running func() (bool, error)
	enabled func() (bool, error)
	start   func() error
	stop    func() error
	restart func() error
	reload  func() error
}

type Service struct {
	paths             filePaths
	system            systemService
	operation         atomic.Int32 // 0: idle, 1: mutation, 2: installation
	localEndpoint     func() (string, int)
	client            *http.Client
	requiresChallenge func(string) bool
	verifyChallenge   func(*gin.Context, string, string) error
}

var defaultService = sync.OnceValue(NewService)

func DefaultService() *Service { return defaultService() }

func NewService() *Service {
	return &Service{
		paths: defaultPaths,
		system: systemService{
			running: func() (bool, error) { return utils.IsServiceRunning(serviceName) },
			enabled: func() (bool, error) {
				state, err := utils.UnitFileState(serviceName)
				return state == "enabled" || state == "enabled-runtime", err
			},
			start:   func() error { return utils.StartService(serviceName, true) },
			stop:    func() error { return utils.StopService(serviceName, true) },
			restart: func() error { return utils.RestartService(serviceName) },
			reload:  utils.DaemonReload,
		},
		localEndpoint: func() (string, int) {
			cfg := config.GetInstance()
			if cfg.Proto == "http" {
				return "http", cfg.Port.Http
			}
			return "https", cfg.Port.Https
		},
		client: &http.Client{
			Timeout:       3 * time.Second,
			Transport:     &http.Transport{Proxy: nil},
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		requiresChallenge: authservice.CloudflareActionRequiresChallenge,
		verifyChallenge:   authservice.VerifySensitiveActionChallenge,
	}
}

func (s *Service) mutate(c *gin.Context, action, challenge string, run func(context.Context) error) {
	var rsp proto.Response
	operation := int32(1)
	if action == authservice.SensitiveActionCloudflareInstall {
		operation = 2
	}
	if !s.operation.CompareAndSwap(0, operation) {
		rsp.ErrRsp(c, -1, "busy")
		return
	}
	defer s.operation.Store(0)
	if s.requiresChallenge(action) {
		if err := s.verifyChallenge(c, challenge, action); err != nil {
			rsp.ErrRsp(c, -3, "sensitive action verification required")
			return
		}
	}
	// Restarting a tunnel can disconnect its caller. Complete authorized work
	// with a bounded lifetime even after the browser connection has closed.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 4*time.Minute)
	defer cancel()
	if err := run(ctx); err != nil {
		rsp.ErrRsp(c, -1, err.Error()) // Only stable error codes reach this boundary.
		return
	}
	rsp.OkRsp(c)
}

func (s *Service) Install(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionCloudflareInstall, authservice.ReadSensitiveChallenge(c), func(ctx context.Context) error {
		if err := s.install(ctx); err != nil {
			return errors.New("installFailed")
		}
		return nil
	})
}

func (s *Service) Uninstall(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionCloudflareUninstall, authservice.ReadSensitiveChallenge(c), func(context.Context) error {
		if err := s.uninstall(); err != nil {
			return errors.New("uninstallFailed")
		}
		return nil
	})
}

func (s *Service) GetConfig(c *gin.Context) {
	var rsp proto.Response
	c.Header("Cache-Control", "no-store")
	cfg, err := readConfig(s.paths)
	if err != nil {
		rsp.ErrRsp(c, -1, "configInvalid")
		return
	}
	rsp.OkRspWithData(c, proto.CloudflareConfigRsp{Mode: cfg.Mode, AccessURL: cfg.AccessURL, TokenSet: cfg.Token != ""})
}

// Clearing credentials uses the same protection policy as replacing them.
// It also works after uninstall, when only the saved configuration remains.
func (s *Service) ClearToken(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionCloudflareConfigure, authservice.ReadSensitiveChallenge(c), func(context.Context) error {
		if _, err := os.Lstat(s.paths.unit); err == nil {
			if s.system.stop() != nil {
				return errors.New("stopFailed")
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return errors.New("serviceFailed")
		}
		if err := os.Remove(s.paths.token); err != nil && !errors.Is(err, os.ErrNotExist) {
			return errors.New("saveFailed")
		}
		return nil
	})
}

func (s *Service) SetConfig(c *gin.Context) {
	var req proto.SetCloudflareConfigReq
	var rsp proto.Response
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10)
	if c.ShouldBindJSON(&req) != nil {
		rsp.ErrRsp(c, -1, "invalidConfig")
		return
	}
	if err := validateForm(&req); err != nil {
		rsp.ErrRsp(c, -1, err.Error())
		return
	}
	s.mutate(c, authservice.SensitiveActionCloudflareConfigure, req.Challenge, func(context.Context) error {
		if !s.installed() {
			return errors.New("notInstalled")
		}
		previous, err := readConfig(s.paths)
		if err != nil {
			return errors.New("configInvalid")
		}
		next := &clientConfig{Mode: previous.Mode, AccessURL: req.AccessURL, Token: previous.Token}
		if req.Token != nil {
			next.Token = *req.Token
		}
		if next.Token == "" {
			return errors.New("invalidToken")
		}
		return s.saveConfig(next, previous)
	})
}

func (s *Service) saveConfig(next, previous *clientConfig) error {
	enabled, err := s.system.enabled()
	if err != nil {
		return errors.New("serviceFailed")
	}
	running, err := s.system.running()
	if err != nil {
		return errors.New("serviceFailed")
	}
	paths := []string{s.paths.config, s.paths.token}
	old := make([][]byte, len(paths))
	for i, path := range paths {
		old[i], err = readPrivateFile(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return errors.New("saveFailed")
		}
	}
	dir := filepath.Dir(s.paths.config)
	if os.MkdirAll(dir, 0700) != nil || os.Chmod(dir, 0700) != nil {
		return errors.New("saveFailed")
	}
	restore := func() error {
		var errs []error
		for i, path := range paths {
			if old[i] == nil {
				err = os.Remove(path)
				if errors.Is(err, os.ErrNotExist) {
					err = nil
				}
			} else {
				err = replaceFile(path, old[i], 0600)
			}
			errs = append(errs, err)
		}
		return errors.Join(errs...)
	}
	metadata, _ := json.Marshal(next) // Token is deliberately excluded.
	for i, data := range [][]byte{metadata, []byte(next.Token)} {
		if replaceFile(paths[i], data, 0600) != nil {
			if restore() != nil {
				return errors.New("rollbackFailed")
			}
			return errors.New("saveFailed")
		}
	}
	// Editing the convenience URL does not need to interrupt active sessions.
	if next.Mode == modeNamed && (enabled || running) && next.Token != previous.Token {
		if s.system.restart() != nil {
			if restore() != nil || s.system.restart() != nil {
				return errors.New("rollbackFailed")
			}
			return errors.New("restartFailed")
		}
	}
	return nil
}

func (s *Service) start(context.Context) error {
	if !s.installed() {
		return errors.New("notInstalled")
	}
	if scheme, _ := s.localEndpoint(); scheme != "https" {
		return errors.New("httpsRequired")
	}
	cfg, err := readConfig(s.paths)
	if err != nil {
		return errors.New("configInvalid")
	}
	if cfg.Mode == modeNamed && cfg.Token == "" {
		return errors.New("invalidToken")
	}
	// Refresh the unit before starting, including the current local HTTPS port.
	if replaceFile(s.paths.unit, []byte(s.unitContents(cfg.Mode)), 0644) != nil || s.system.reload() != nil {
		return errors.New("serviceFailed")
	}
	if s.system.start() != nil {
		return errors.New("startFailed")
	}
	return nil
}

func (s *Service) Start(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionCloudflareStart, authservice.ReadSensitiveChallenge(c), s.start)
}

func (s *Service) Restart(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionCloudflareRestart, authservice.ReadSensitiveChallenge(c), s.start)
}

func (s *Service) Stop(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionCloudflareStop, authservice.ReadSensitiveChallenge(c), func(context.Context) error {
		if !s.installed() {
			return errors.New("notInstalled")
		}
		if s.system.stop() != nil {
			return errors.New("stopFailed")
		}
		return nil
	})
}

func (s *Service) GetStatus(c *gin.Context) {
	var rsp proto.Response
	c.Header("Cache-Control", "no-store")
	rsp.OkRspWithData(c, s.status(c.Request.Context()))
}

func (s *Service) status(ctx context.Context) proto.CloudflareStatusRsp {
	scheme, port := s.localEndpoint()
	op := s.operation.Load()
	status := proto.CloudflareStatusRsp{
		Mode: modeNamed, State: "notInstalled", Busy: op != 0,
		OriginURL: fmt.Sprintf("%s://127.0.0.1:%d", scheme, port), HTTPSAvailable: scheme == "https",
	}
	if op == 2 {
		status.State = "installing"
		return status
	}
	if !s.installed() {
		return status
	}
	version, _ := readPrivateFile(s.paths.version)
	status.Version = strings.TrimSpace(string(version))
	status.State = "error"
	var err error
	status.Enabled, err = s.system.enabled()
	if err != nil {
		status.Error = "serviceFailed"
		return status
	}
	status.Running, err = s.system.running()
	if err != nil {
		status.Error = "serviceFailed"
		return status
	}
	cfg, err := readConfig(s.paths)
	if err != nil {
		status.Error = "configInvalid"
		return status
	}
	status.Mode = cfg.Mode
	if cfg.Mode == modeNamed {
		status.AccessURL = cfg.AccessURL
	}
	if cfg.Mode == modeNamed && cfg.Token == "" {
		status.State = "notConfigured"
		return status
	}
	if !status.Running {
		if status.Enabled {
			status.Error = "connectionFailed"
			if cfg.Mode == modeQuick {
				status.Error = "quickConnectionFailed"
			}
		} else {
			status.State = "stopped"
		}
		return status
	}
	if !status.HTTPSAvailable {
		status.Error = "httpsRequired"
		return status
	}
	status.State = "connecting"
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+metricsAddress+"/ready", nil)
	response, err := s.client.Do(request)
	if err != nil {
		return status
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusServiceUnavailable {
		return status
	}
	var ready struct {
		ReadyConnections uint `json:"readyConnections"`
	}
	if response.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&ready) != nil {
		status.State, status.Error = "error", "serviceFailed"
	} else if ready.ReadyConnections > 0 {
		status.State = "connected"
	}
	if cfg.Mode == modeQuick && status.State == "connected" {
		status.AccessURL = s.quickTunnelURL(ctx)
		if status.AccessURL == "" {
			status.State = "connecting"
		}
	}
	return status
}
