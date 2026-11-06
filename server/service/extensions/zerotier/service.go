package zerotier

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"NanoKVM-Server/config"
	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"
	"NanoKVM-Server/utils"

	"github.com/gin-gonic/gin"
)

type systemService struct {
	running func() (bool, error)
	enabled func() (bool, error)
	start   func() error
	stop    func() error
	reload  func() error
}

type Service struct {
	paths             filePaths
	system            systemService
	operation         atomic.Int32
	apiURL            string
	client            *http.Client
	localEndpoint     func() (string, int)
	tunAvailable      func() bool
	requiresChallenge func(string) bool
	verifyChallenge   func(*gin.Context, string, string) error
}

var defaultService = sync.OnceValue(NewService)

func DefaultService() *Service { return defaultService() }

func NewService() *Service {
	return &Service{
		paths:  defaultPaths,
		apiURL: "http://127.0.0.1:9993",
		client: &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{Proxy: nil},
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		system: systemService{
			running: func() (bool, error) { return utils.IsServiceRunning(serviceName) },
			enabled: func() (bool, error) {
				state, err := utils.UnitFileState(serviceName)
				return state == "enabled" || state == "enabled-runtime", err
			},
			start:  func() error { return utils.StartService(serviceName, true) },
			stop:   func() error { return utils.StopService(serviceName, true) },
			reload: utils.DaemonReload,
		},
		localEndpoint: func() (string, int) {
			cfg := config.GetInstance()
			if cfg.Proto == "http" {
				return "http", cfg.Port.Http
			}
			return "https", cfg.Port.Https
		},
		tunAvailable: func() bool {
			info, err := os.Stat("/dev/net/tun")
			return err == nil && info.Mode()&os.ModeCharDevice != 0
		},
		requiresChallenge: authservice.ZeroTierActionRequiresChallenge,
		verifyChallenge:   authservice.VerifySensitiveActionChallenge,
	}
}

func (s *Service) mutate(c *gin.Context, action, challenge string, run func(context.Context) error) {
	var rsp proto.Response
	op := int32(1)
	if action == authservice.SensitiveActionZeroTierInstall {
		op = 2
	}
	if !s.operation.CompareAndSwap(0, op) {
		rsp.ErrRsp(c, -1, "busy")
		return
	}
	defer s.operation.Store(0)
	if s.requiresChallenge(action) {
		if s.verifyChallenge(c, challenge, action) != nil {
			rsp.ErrRsp(c, -3, "sensitive action verification required")
			return
		}
	}
	// Stop/leave can disconnect the caller; finish authorized work with a deadline.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 4*time.Minute)
	defer cancel()
	if err := run(ctx); err != nil {
		rsp.ErrRsp(c, -1, err.Error())
		return
	}
	rsp.OkRsp(c)
}

func (s *Service) Install(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionZeroTierInstall, authservice.ReadSensitiveChallenge(c), func(ctx context.Context) error {
		if err := s.install(ctx); err != nil {
			return errors.New("installFailed")
		}
		return nil
	})
}

func (s *Service) Uninstall(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionZeroTierUninstall, authservice.ReadSensitiveChallenge(c), func(context.Context) error {
		if err := s.uninstall(); err != nil {
			return errors.New("uninstallFailed")
		}
		return nil
	})
}

func (s *Service) start(context.Context) error {
	if !s.installed() {
		return errors.New("notInstalled")
	}
	if !s.tunAvailable() {
		return errors.New("tunUnavailable")
	}
	if s.system.start() != nil {
		return errors.New("startFailed")
	}
	return nil
}

func (s *Service) Start(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionZeroTierStart, authservice.ReadSensitiveChallenge(c), s.start)
}

func (s *Service) Restart(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionZeroTierRestart, authservice.ReadSensitiveChallenge(c), s.start)
}

func (s *Service) Stop(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionZeroTierStop, authservice.ReadSensitiveChallenge(c), func(context.Context) error {
		if !s.installed() {
			return errors.New("notInstalled")
		}
		if s.system.stop() != nil {
			return errors.New("stopFailed")
		}
		return nil
	})
}

func (s *Service) Join(c *gin.Context)  { s.changeNetwork(c, true) }
func (s *Service) Leave(c *gin.Context) { s.changeNetwork(c, false) }

func (s *Service) changeNetwork(c *gin.Context, join bool) {
	var req proto.ZeroTierNetworkReq
	var rsp proto.Response
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	if c.ShouldBindJSON(&req) != nil {
		rsp.ErrRsp(c, -1, "invalidNetworkID")
		return
	}
	id, err := normalizeNetworkID(req.NetworkID)
	if err != nil {
		rsp.ErrRsp(c, -1, "invalidNetworkID")
		return
	}
	action := authservice.SensitiveActionZeroTierLeave
	if join {
		action = authservice.SensitiveActionZeroTierJoin
	}
	s.mutate(c, action, req.Challenge, func(ctx context.Context) error {
		running := false
		if _, err := os.Lstat(s.paths.unit); err == nil {
			running, err = s.system.running()
			if err != nil {
				return errors.New("serviceFailed")
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return errors.New("serviceFailed")
		}
		if !running {
			if join {
				return errors.New("startRequired")
			}
			// Membership can also be cleared while stopped or uninstalled.
			for _, suffix := range []string{".conf", ".local.conf"} {
				if err := os.Remove(filepath.Join(s.paths.state, "networks.d", id+suffix)); err != nil && !errors.Is(err, os.ErrNotExist) {
					return errors.New("leaveFailed")
				}
			}
			return nil
		}
		if join {
			return s.request(ctx, http.MethodPost, "/network/"+id, map[string]bool{
				"allowManaged": true, "allowGlobal": false, "allowDefault": false, "allowDNS": false,
			}, nil)
		}
		return s.request(ctx, http.MethodDelete, "/network/"+id, nil, nil)
	})
}

func (s *Service) GetStatus(c *gin.Context) {
	var rsp proto.Response
	c.Header("Cache-Control", "no-store")
	rsp.OkRspWithData(c, s.status(c.Request.Context()))
}

func (s *Service) status(ctx context.Context) proto.ZeroTierStatusRsp {
	op := s.operation.Load()
	out := proto.ZeroTierStatusRsp{State: "notInstalled", Busy: op != 0, Networks: []proto.ZeroTierNetwork{}}
	var err error
	out.NodeID, out.Networks, err = s.savedState()
	if err != nil {
		out.Error = "configInvalid"
	}
	if op == 2 {
		out.State = "installing"
		return out
	}
	if !s.installed() {
		return out
	}
	out.Version = ZeroTierVersion
	out.State = "error"
	out.Enabled, err = s.system.enabled()
	if err != nil {
		out.Error = "serviceFailed"
		return out
	}
	out.Running, err = s.system.running()
	if err != nil {
		out.Error = "serviceFailed"
		return out
	}
	if !out.Running {
		if out.Enabled {
			out.Error = "startFailed"
		} else {
			out.State = "stopped"
		}
		return out
	}
	var node nodeStatus
	if err := s.request(ctx, http.MethodGet, "/status", nil, &node); err != nil {
		out.Error = err.Error()
		return out
	}
	if node.Version != ZeroTierVersion || !nodeIDPattern.MatchString(node.Address) {
		out.Error = "versionMismatch"
		return out
	}
	out.NodeID, out.Online = node.Address, node.Online
	var networks []networkStatus
	if err := s.request(ctx, http.MethodGet, "/network", nil, &networks); err != nil {
		out.Error = err.Error()
		return out
	}
	out.Error, out.Networks = "", []proto.ZeroTierNetwork{}
	scheme, port := s.localEndpoint()
	connected, pending, failed := false, false, false
	for _, n := range networks {
		if id, err := normalizeNetworkID(n.ID); err != nil || id != n.ID {
			continue
		}
		view := networkView(n, node.Online, scheme, port)
		out.Networks = append(out.Networks, view)
		switch view.Status {
		case "OK":
			connected = true
		case "REQUESTING_CONFIGURATION", "ACCESS_DENIED", "WAITING_ADDRESS":
			pending = true
		case "OFFLINE":
		default:
			failed = true
		}
	}
	sort.Slice(out.Networks, func(i, j int) bool { return out.Networks[i].ID < out.Networks[j].ID })
	switch {
	case !node.Online:
		out.State = "offline"
	case len(out.Networks) == 0:
		out.State = "noNetworks"
	case connected:
		out.State = "connected"
	case pending:
		out.State = "connecting"
	case failed:
		out.State = "error"
	default:
		out.State = "connecting"
	}
	return out
}
