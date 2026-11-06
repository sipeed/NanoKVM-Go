package netbird

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
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
	reload  func() error
}

type loginAttempt struct {
	cancel context.CancelFunc
	done   chan struct{}
	url    string
	code   string
}

type Service struct {
	paths             filePaths
	system            systemService
	operation         atomic.Int32
	client            *http.Client
	localEndpoint     func() (string, int)
	tunAvailable      func() bool
	requiresChallenge func(string) bool
	verifyChallenge   func(*gin.Context, string, string) error
	mu                sync.Mutex
	pending           *loginAttempt
	loginError        string
	loginTimeout      time.Duration
}

var defaultService = sync.OnceValue(NewService)

func DefaultService() *Service { return defaultService() }

func NewService() *Service {
	s := &Service{
		paths: defaultPaths, loginTimeout: 10 * time.Minute,
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
		requiresChallenge: authservice.NetBirdActionRequiresChallenge,
		verifyChallenge:   authservice.VerifySensitiveActionChallenge,
	}
	s.client = &http.Client{
		Transport: &http.Transport{Proxy: nil, MaxIdleConnsPerHost: 2, IdleConnTimeout: 30 * time.Second,
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "unix", s.paths.socket)
			}},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	return s
}

func (s *Service) mutate(c *gin.Context, action string, run func(context.Context) error) {
	var rsp proto.Response
	op := int32(1)
	if action == authservice.SensitiveActionNetBirdInstall {
		op = 2
	}
	if !s.operation.CompareAndSwap(0, op) {
		rsp.ErrRsp(c, -1, "busy")
		return
	}
	defer s.operation.Store(0)
	if s.requiresChallenge(action) && s.verifyChallenge(c, authservice.ReadSensitiveChallenge(c), action) != nil {
		rsp.ErrRsp(c, -3, "sensitive action verification required")
		return
	}
	// A stop/logout may disconnect the caller; finish an authorized operation.
	timeout := time.Minute
	if op == 2 {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), timeout)
	defer cancel()
	if err := run(ctx); err != nil {
		rsp.ErrRsp(c, -1, err.Error())
		return
	}
	rsp.OkRsp(c)
}

func (s *Service) Install(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionNetBirdInstall, func(ctx context.Context) error {
		if err := s.install(ctx); err != nil {
			log.WithError(err).Warn("NetBird installation failed")
			if err.Error() == "externalInstallation" || err.Error() == "unsupportedArchitecture" {
				return err
			}
			return errors.New("installFailed")
		}
		return nil
	})
}

func (s *Service) Uninstall(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionNetBirdUninstall, func(ctx context.Context) error {
		if err := s.cancelLogin(ctx); err != nil {
			return err
		}
		if err := s.uninstall(); err != nil {
			return errors.New("uninstallFailed")
		}
		return nil
	})
}

func (s *Service) ensureStarted(ctx context.Context) error {
	if !s.installed() {
		return errors.New("notInstalled")
	}
	if !s.tunAvailable() {
		return errors.New("tunUnavailable")
	}
	running, err := s.system.running()
	if err != nil {
		return errors.New("serviceFailed")
	}
	if !running {
		if err := s.system.start(); err != nil {
			return errors.New("startFailed")
		}
	}
	return s.prepareDaemon(ctx)
}

func (s *Service) prepareDaemon(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for {
		var status daemonStatus
		probe, stop := context.WithTimeout(ctx, 3*time.Second)
		err := s.request(probe, "Status", map[string]any{}, &status)
		stop()
		if err == nil {
			if status.DaemonVersion != NetBirdVersion {
				return errors.New("versionMismatch")
			}
			if err := s.request(ctx, "SetConfig", peerConfig(), nil); err != nil {
				return errors.New("configureFailed")
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.New("apiUnavailable")
		case <-time.After(200 * time.Millisecond):
		}
	}
}

func (s *Service) Start(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionNetBirdStart, func(ctx context.Context) error {
		s.mu.Lock()
		pending := s.pending != nil
		s.mu.Unlock()
		if pending {
			return errors.New("busy")
		}
		// The daemon starts its own reconnect loop. An additional Up while that
		// loop is active blocks waiting for authentication, even with async=true.
		return s.ensureStarted(ctx)
	})
}

func (s *Service) Stop(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionNetBirdStop, func(ctx context.Context) error {
		if err := s.cancelLogin(ctx); err != nil {
			return err
		}
		if err := s.system.stop(); err != nil {
			return errors.New("serviceFailed")
		}
		return nil
	})
}

func (s *Service) Restart(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionNetBirdRestart, func(ctx context.Context) error {
		if err := s.cancelLogin(ctx); err != nil {
			return err
		}
		if !s.installed() {
			return errors.New("notInstalled")
		}
		if err := s.system.stop(); err != nil {
			return errors.New("serviceFailed")
		}
		return s.ensureStarted(ctx)
	})
}

func (s *Service) Login(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionNetBirdLogin, func(ctx context.Context) error {
		if !s.installed() {
			return errors.New("notInstalled")
		}
		enabled, err := s.system.enabled()
		if err != nil {
			return errors.New("serviceFailed")
		}
		running, err := s.system.running()
		if err != nil {
			return errors.New("serviceFailed")
		}
		if !enabled || !running {
			return errors.New("startRequired")
		}
		s.mu.Lock()
		pending := s.pending != nil
		s.loginError = ""
		s.mu.Unlock()
		if pending {
			return nil
		}
		if err := s.prepareDaemon(ctx); err != nil {
			return err
		}
		var reply loginResponse
		if err := s.request(ctx, "Login", loginOptions(), &reply); err != nil {
			return errors.New("loginFailed")
		}
		if !reply.NeedsSSOLogin {
			if err := s.request(ctx, "Up", map[string]any{"async": true}, nil); err != nil {
				return errors.New("startFailed")
			}
			return nil
		}
		uri, err := authorizationURL(reply)
		if err != nil {
			return err
		}
		waitCtx, cancel := context.WithTimeout(context.Background(), s.loginTimeout)
		attempt := &loginAttempt{cancel: cancel, done: make(chan struct{}), url: uri, code: reply.UserCode}
		s.mu.Lock()
		s.pending = attempt
		s.mu.Unlock()
		go s.waitLogin(waitCtx, attempt)
		return nil
	})
}

func (s *Service) waitLogin(ctx context.Context, attempt *loginAttempt) {
	defer close(attempt.done)
	defer attempt.cancel()
	err := s.request(ctx, "WaitSSOLogin", map[string]string{"userCode": attempt.code}, nil)
	// Serialize completion with stop, logout and uninstall. A canceled/replaced
	// waiter must never call Up after another mutation has stopped the service.
	for {
		s.mu.Lock()
		if s.pending != attempt {
			s.mu.Unlock()
			return
		}
		if s.operation.CompareAndSwap(0, 1) {
			s.mu.Unlock()
			break
		}
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			s.mu.Lock()
			if s.pending == attempt {
				s.pending = nil
				s.loginError = "loginExpired"
			}
			s.mu.Unlock()
			return
		case <-time.After(20 * time.Millisecond):
		}
	}
	defer s.operation.Store(0)
	failure := ""
	if err != nil {
		failure = "loginFailed"
		var remote *rpcError
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.As(err, &remote) && remote.Code == 4 {
			failure = "loginExpired"
		}
	} else {
		upCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		if s.request(upCtx, "Up", map[string]any{"async": true}, nil) != nil {
			failure = "startFailed"
		}
		cancel()
	}
	s.mu.Lock()
	if s.pending == attempt {
		s.pending = nil
		s.loginError = failure
	}
	s.mu.Unlock()
}

func (s *Service) cancelLogin(ctx context.Context) error {
	s.mu.Lock()
	attempt := s.pending
	s.pending = nil
	s.loginError = ""
	s.mu.Unlock()
	if attempt != nil {
		attempt.cancel()
		select {
		case <-attempt.done:
		case <-ctx.Done():
			return errors.New("cancelFailed")
		}
	}
	return nil
}

func (s *Service) CancelLogin(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionNetBirdLoginCancel, s.cancelLogin)
}

func (s *Service) Logout(c *gin.Context) {
	s.mutate(c, authservice.SensitiveActionNetBirdLogout, func(ctx context.Context) error {
		if err := s.cancelLogin(ctx); err != nil {
			return err
		}
		if err := s.ensureStarted(ctx); err != nil {
			return err
		}
		// Upstream Logout deregisters this peer. Preserve its state on failure so
		// a cloud/network error is visible and can be retried.
		if err := s.request(ctx, "Logout", map[string]any{}, nil); err != nil {
			return errors.New("logoutFailed")
		}
		if err := s.system.stop(); err != nil {
			return errors.New("serviceFailed")
		}
		return nil
	})
}

func (s *Service) GetStatus(c *gin.Context) {
	var rsp proto.Response
	c.Header("Cache-Control", "no-store")
	rsp.OkRspWithData(c, s.status(c.Request.Context()))
}

func (s *Service) status(ctx context.Context) proto.NetBirdStatusRsp {
	op := s.operation.Load()
	out := proto.NetBirdStatusRsp{State: "notInstalled", Busy: op != 0}
	if op == 2 {
		out.State = "installing"
		return out
	}
	if !s.installed() {
		return out
	}
	out.Version = NetBirdVersion
	var err error
	out.Enabled, err = s.system.enabled()
	if err != nil {
		out.State = "error"
		out.Error = "serviceFailed"
		return out
	}
	out.Running, err = s.system.running()
	if err != nil {
		out.State = "error"
		out.Error = "serviceFailed"
		return out
	}
	out.State = "stopped"
	if !out.Running {
		if out.Enabled {
			out.State = "error"
			out.Error = "startFailed"
		}
		return out
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var reply daemonStatus
	if s.request(ctx, "Status", map[string]any{"getFullPeerStatus": true, "shouldRunProbes": false}, &reply) != nil {
		out.State = "error"
		out.Error = "apiUnavailable"
		return out
	}
	if reply.DaemonVersion != NetBirdVersion {
		out.State = "error"
		out.Error = "versionMismatch"
		return out
	}
	out.Name = reply.FullStatus.LocalPeerState.FQDN
	switch reply.Status {
	case "NeedsLogin", "LoginFailed", "SessionExpired":
		out.State = "notLogin"
	case "Connecting":
		out.State = "connecting"
	case "Connected":
		out.IP = peerAddress(reply.FullStatus.LocalPeerState.IP)
		if reply.FullStatus.ManagementState.Connected && out.IP != "" {
			out.State = "connected"
		} else {
			out.State = "offline"
		}
	default:
		out.State = "offline"
	}
	if reply.Status == "SessionExpired" {
		out.Error = "loginExpired"
	}
	s.mu.Lock()
	if s.pending != nil {
		out.State = "authorizing"
		out.AuthURL = s.pending.url
		out.UserCode = s.pending.code
	}
	if s.loginError != "" {
		out.Error = s.loginError
	}
	s.mu.Unlock()
	if out.Enabled && out.State == "connected" {
		scheme, port := s.localEndpoint()
		out.AccessURL = accessURL(scheme, port, out.IP)
	}
	return out
}
