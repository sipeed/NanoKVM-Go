package frp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"NanoKVM-Server/config"
	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"
	"NanoKVM-Server/utils"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type systemService struct {
	running func() (bool, error)
	enabled func() (bool, error)
	start   func() error
	stop    func() error
	restart func() error
	reload  func() error
}

type Service struct {
	paths  filePaths
	system systemService
	// 0 = idle, 1 = a mutation, 2 = installing. Status reads never wait for a download.
	operation         atomic.Int32
	localEndpoint     func() (string, int)
	client            *http.Client
	requiresChallenge func(string) bool
	verifyChallenge   func(*gin.Context, string, string) error
}

var defaultService = sync.OnceValue(NewService)

func DefaultService() *Service { return defaultService() }

func NewService() *Service {
	s := &Service{
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
			// Paging on the memory-limited device can delay even a local status
			// response. Allow it to complete before reporting a pending connection.
			Timeout: 10 * time.Second,
			// Never send local admin credentials through an environment proxy or redirect.
			Transport:     &http.Transport{Proxy: nil},
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		requiresChallenge: authservice.FRPActionRequiresChallenge,
		verifyChallenge:   authservice.VerifySensitiveActionChallenge,
	}
	return s
}

func (s *Service) mutate(c *gin.Context, action string, challenge string, run func(context.Context) error) {
	var rsp proto.Response
	operation := int32(1)
	if action == authservice.SensitiveActionFRPInstall {
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
	// A tunnel restart can close the very request that initiated it. Finish the
	// authorized transaction even if that browser connection goes away.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 4*time.Minute)
	defer cancel()
	if err := run(ctx); err != nil {
		// Errors returned by these operations are stable codes, never config text.
		rsp.ErrRsp(c, -1, err.Error())
		return
	}
	rsp.OkRsp(c)
}

func (s *Service) Install(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionFRPInstall, authservice.ReadSensitiveChallenge(c), func(ctx context.Context) error {
		if err := s.install(ctx); err != nil {
			log.Warnf("frp install failed: %v", err)
			return errors.New("installFailed")
		}
		return nil
	})
}

func (s *Service) Uninstall(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionFRPUninstall, authservice.ReadSensitiveChallenge(c), func(context.Context) error {
		if err := s.uninstall(); err != nil {
			return errors.New("uninstallFailed")
		}
		return nil
	})
}

func (s *Service) GetConfig(c *gin.Context) {
	var rsp proto.Response
	c.Header("Cache-Control", "no-store")
	cfg, err := readConfig(s.paths.config)
	if errors.Is(err, os.ErrNotExist) {
		rsp.OkRspWithData(c, proto.FRPConfigRsp{ServerPort: 7000, RemotePort: 8443})
	} else if err != nil {
		rsp.ErrRsp(c, -1, "configInvalid")
	} else {
		rsp.OkRspWithData(c, cfg.response())
	}
}

// ClearToken is a credential configuration operation. Stop and disable first,
// so a cleared credential cannot remain in use by an already running client.
func (s *Service) ClearToken(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionFRPConfigure, authservice.ReadSensitiveChallenge(c), func(context.Context) error {
		cfg, err := readConfig(s.paths.config)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return errors.New("configInvalid")
		}
		if _, err := os.Lstat(s.paths.unit); err == nil {
			if s.system.stop() != nil {
				return errors.New("stopFailed")
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return errors.New("serviceFailed")
		}
		if cfg == nil || cfg.Auth.Token == "" {
			return nil
		}
		cfg.Auth.Token = ""
		if writeConfig(s.paths.config, cfg) != nil {
			return errors.New("saveFailed")
		}
		return nil
	})
}

func (s *Service) SetConfig(c *gin.Context) {
	var req proto.SetFRPConfigReq
	var rsp proto.Response
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10)
	if err := c.ShouldBindJSON(&req); err != nil {
		rsp.ErrRsp(c, -1, "invalidConfig")
		return
	}
	if err := validateForm(&req); err != nil {
		rsp.ErrRsp(c, -1, err.Error())
		return
	}
	s.mutate(c, authservice.SensitiveActionFRPConfigure, req.Challenge, func(ctx context.Context) error {
		if !s.installed() {
			return errors.New("notInstalled")
		}
		old, err := readConfig(s.paths.config)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return errors.New("configInvalid")
		}
		_, port := s.localEndpoint()
		next, err := makeConfig(req, old, port)
		if err != nil {
			return errors.New("saveFailed")
		}
		return s.saveConfig(next)
	})
}

func (s *Service) saveConfig(cfg *clientConfig) error {
	enabled, err := s.system.enabled()
	if err != nil {
		return errors.New("serviceFailed")
	}
	running, err := s.system.running()
	if err != nil {
		return errors.New("serviceFailed")
	}
	old, err := os.ReadFile(s.paths.config)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.New("saveFailed")
	}
	if err := os.MkdirAll(filepath.Dir(s.paths.config), 0700); err != nil {
		return errors.New("saveFailed")
	}
	if err := os.Chmod(filepath.Dir(s.paths.config), 0700); err != nil {
		return errors.New("saveFailed")
	}
	if err := writeConfig(s.paths.config, cfg); err != nil {
		return errors.New("saveFailed")
	}
	if enabled || running {
		if err := s.system.restart(); err != nil {
			if len(old) == 0 {
				err = os.Remove(s.paths.config)
			} else {
				err = replaceFile(s.paths.config, old, 0600)
			}
			if err != nil {
				return errors.New("rollbackFailed")
			}
			if len(old) > 0 {
				if err := s.system.restart(); err != nil {
					return errors.New("rollbackFailed")
				}
			}
			return errors.New("restartFailed")
		}
	}
	return nil
}

func (s *Service) start(_ context.Context) error {
	if !s.installed() {
		return errors.New("notInstalled")
	}
	cfg, err := readConfig(s.paths.config)
	if err != nil {
		return errors.New("configInvalid")
	}
	_, port := s.localEndpoint()
	cfg.Proxies[0].LocalPort = port
	if err := writeConfig(s.paths.config, cfg); err != nil {
		return errors.New("configInvalid")
	}
	if err := s.system.start(); err != nil {
		return errors.New("startFailed")
	}
	return nil
}

func (s *Service) Start(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionFRPStart, authservice.ReadSensitiveChallenge(c), s.start)
}

func (s *Service) Restart(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionFRPRestart, authservice.ReadSensitiveChallenge(c), s.start)
}

func (s *Service) Stop(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionFRPStop, authservice.ReadSensitiveChallenge(c), func(context.Context) error {
		if !s.installed() {
			return errors.New("notInstalled")
		}
		if err := s.system.stop(); err != nil {
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

func (s *Service) status(ctx context.Context) proto.FRPStatusRsp {
	op := s.operation.Load()
	status := proto.FRPStatusRsp{State: "notInstalled", Busy: op != 0}
	if op == 2 {
		status.State = "installing"
		return status
	}
	if !s.installed() {
		return status
	}
	cfg, err := readConfig(s.paths.config)
	if errors.Is(err, os.ErrNotExist) {
		status.State = "notConfigured"
		return status
	}
	status.State = "error"
	if err != nil {
		status.Error = "configInvalid"
		return status
	}
	scheme, _ := s.localEndpoint()
	status.AccessURL = cfg.accessURL(scheme)
	status.Enabled, err = s.system.enabled()
	if err != nil {
		status.Error = "serviceFailed"
		return status
	}
	running, err := s.system.running()
	if err != nil {
		status.Error = "serviceFailed"
		return status
	}
	if !running {
		if status.Enabled {
			status.Error = "connectionFailed"
		} else {
			status.State = "stopped"
		}
		return status
	}
	status.State = "connecting"
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:"+strconv.Itoa(adminPort)+"/api/status", nil)
	request.SetBasicAuth(cfg.WebServer.User, cfg.WebServer.Password)
	response, err := s.client.Do(request)
	if err != nil {
		return status
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		status.State = "error"
		status.Error = "serviceFailed"
		return status
	}
	var proxies map[string][]struct {
		Name   string `json:"name"`
		Status string `json:"status"`
		Err    string `json:"err"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&proxies); err != nil {
		status.State = "error"
		status.Error = "serviceFailed"
		return status
	}
	for _, proxy := range proxies["tcp"] {
		if proxy.Name != cfg.Proxies[0].Name {
			continue
		}
		if proxy.Err != "" || proxy.Status == "start error" || proxy.Status == "check failed" || proxy.Status == "closed" {
			status.State = "error"
			status.Error = "proxyFailed"
		} else if proxy.Status == "running" {
			status.State = "connected"
		}
		break
	}
	return status
}
