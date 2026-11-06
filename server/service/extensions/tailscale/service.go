package tailscale

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	sipeed           *sipeedManager
	disclaimerStore  sipeedDisclaimerStore
	regionStore      *tailscaleRegionStore
	newActionClient  func() tailscaleActionClient
	uninstallAction  func() error
	waitAfterRestart func()
}

type tailscaleActionClient interface {
	Start() error
	Restart() error
	Stop() error
	Up() error
	Down() error
	Logout() error
}

// tailscaleLoginClient is the LocalClient subset used by Login.
type tailscaleLoginClient interface {
	Status() (*TsStatus, error)
	Start() error
	Login() (string, error)
}

var (
	tailscaleActionRequiresChallenge = authservice.TailscaleActionRequiresChallenge
	verifyTailscaleChallenge         = authservice.VerifySensitiveActionChallenge
	newTailscaleLoginClient          = func(ctx context.Context) tailscaleLoginClient {
		return NewLocalClientWithContext(ctx)
	}
	defaultServiceOnce sync.Once
	defaultService     *Service
)

const tailscaleLoginStatusRetryDelay = 3 * time.Second

var StateMap = map[string]proto.TailscaleState{
	"NoState":    proto.TailscaleNotRunning,
	"Starting":   proto.TailscaleNotRunning,
	"NeedsLogin": proto.TailscaleNotLogin,
	"Running":    proto.TailscaleRunning,
	"Stopped":    proto.TailscaleStopped,
}

func effectiveTailscaleState(status *TsStatus, prefs *TsPrefs) (proto.TailscaleState, bool) {
	if prefs != nil && prefs.LoggedOut {
		return proto.TailscaleNotLogin, true
	}
	if status == nil {
		return "", false
	}
	state, ok := StateMap[status.BackendState]
	return state, ok
}

func NewService() *Service {
	return &Service{
		sipeed:          newSipeedManager(defaultSipeedManagerDeps()),
		disclaimerStore: configSipeedDisclaimerStore{},
		regionStore:     defaultTailscaleRegionStore,
		newActionClient: func() tailscaleActionClient {
			return NewLocalClient()
		},
		uninstallAction: uninstall,
		waitAfterRestart: func() {
			time.Sleep(10 * time.Second)
		},
	}
}

// DefaultService is shared by the authenticated HTTP router and the root-local
// UI socket so region changes and Tailscale actions observe the same managed
// identity state.
func DefaultService() *Service {
	defaultServiceOnce.Do(func() {
		defaultService = NewService()
	})
	return defaultService
}

func (s *Service) actionClient() tailscaleActionClient {
	if s != nil && s.newActionClient != nil {
		return s.newActionClient()
	}
	return NewLocalClient()
}

func (s *Service) runUninstall() error {
	if s != nil && s.uninstallAction != nil {
		return s.uninstallAction()
	}
	return uninstall()
}

func (s *Service) waitForRestart() {
	if s != nil && s.waitAfterRestart != nil {
		s.waitAfterRestart()
		return
	}
	time.Sleep(10 * time.Second)
}

func (s *Service) cleanupSipeedBeforeLifecycle(ctx context.Context) error {
	if s == nil || s.sipeed == nil || !s.sipeed.hasManagedRecord() {
		return nil
	}
	_, err := s.sipeed.logout(ctx)
	return err
}

func beginTailscaleMutation(c *gin.Context, rsp *proto.Response, installing bool) (func(), bool) {
	release, ok := tailscaleMutations.tryBegin(installing)
	if !ok {
		rsp.ErrRsp(c, -1, "tailscale operation already in progress")
		return nil, false
	}
	return release, true
}

// requireTailscaleChallenge enforces advanced 2FA protection for listed ops.
// Returns false when the handler should stop (error already written).
func requireTailscaleChallenge(c *gin.Context, rsp *proto.Response, action string) bool {
	return requireTailscaleChallengeValue(c, rsp, action, authservice.ReadSensitiveChallenge(c))
}

func requireTailscaleChallengeValue(
	c *gin.Context,
	rsp *proto.Response,
	action string,
	challenge string,
) bool {
	if !tailscaleActionRequiresChallenge(action) {
		return true
	}
	if err := verifyTailscaleChallenge(c, challenge, action); err != nil {
		log.Errorf("verify tailscale challenge for %s failed: %s", action, err)
		rsp.ErrRsp(c, -3, "sensitive action verification required")
		return false
	}
	return true
}

func (s *Service) Install(c *gin.Context) {
	var rsp proto.Response

	if !requireTailscaleChallenge(c, &rsp, authservice.SensitiveActionTailscaleInstall) {
		return
	}
	release, ok := beginTailscaleMutation(c, &rsp, true)
	if !ok {
		return
	}
	defer release()

	if !isInstalled() {
		if err := install(); err != nil {
			log.Errorf("tailscale install failed: %s", err)
			rsp.ErrRsp(c, -1, "install failed")
			return
		}
	}

	rsp.OkRsp(c)
	log.Debugf("install tailscale successfully")
}

func (s *Service) Uninstall(c *gin.Context) {
	var rsp proto.Response

	if !requireTailscaleChallenge(c, &rsp, authservice.SensitiveActionTailscaleUninstall) {
		return
	}
	release, ok := beginTailscaleMutation(c, &rsp, false)
	if !ok {
		return
	}
	defer release()

	if err := s.cleanupSipeedBeforeLifecycle(c.Request.Context()); err != nil {
		log.Errorf("failed to clear Sipeed identity before tailscale uninstall: %s", err)
		rsp.ErrRsp(c, -1, "uninstall failed")
		return
	}

	if err := s.runUninstall(); err != nil {
		rsp.ErrRsp(c, -1, "uninstall failed")
		return
	}

	rsp.OkRsp(c)
	log.Debugf("uninstall tailscale successfully")
}

func (s *Service) Start(c *gin.Context) {
	var rsp proto.Response

	if !requireTailscaleChallenge(c, &rsp, authservice.SensitiveActionTailscaleStart) {
		return
	}
	release, ok := beginTailscaleMutation(c, &rsp, false)
	if !ok {
		return
	}
	defer release()

	err := s.actionClient().Start()
	if err != nil {
		rsp.ErrRsp(c, -1, "start failed")
		log.Errorf("failed to run tailscale start: %s", err)
		return
	}

	rsp.OkRsp(c)
	log.Debugf("tailscale start successfully")
}

func (s *Service) Restart(c *gin.Context) {
	var rsp proto.Response

	if !requireTailscaleChallenge(c, &rsp, authservice.SensitiveActionTailscaleRestart) {
		return
	}
	release, ok := beginTailscaleMutation(c, &rsp, false)
	if !ok {
		return
	}
	defer release()

	err := s.actionClient().Restart()
	if err != nil {
		rsp.ErrRsp(c, -1, "restart failed")
		log.Errorf("failed to run tailscale restart: %s", err)
		return
	}

	s.waitForRestart()

	rsp.OkRsp(c)
	log.Debugf("tailscale restart successfully")
}

// deinitializeManaged is the transport-independent core of Stop: under the
// global mutation lock it clears any Sipeed managed identity (logout, which
// also best-effort destroys the cloud network) and then stops tailscaled and
// disables its autostart. The ctx scopes the logout phase; a caller that must
// survive its own client disconnect (the root-local gate driver) passes a
// detached context instead of the request context.
func (s *Service) deinitializeManaged(ctx context.Context) error {
	release, ok := tailscaleMutations.tryBegin(false)
	if !ok {
		return errTailscaleMutationBusy
	}
	defer release()

	if err := s.cleanupSipeedBeforeLifecycle(ctx); err != nil {
		return fmt.Errorf("clear sipeed identity before tailscale stop: %w", err)
	}
	if err := s.actionClient().Stop(); err != nil {
		return fmt.Errorf("run tailscale stop: %w", err)
	}
	return nil
}

func (s *Service) Stop(c *gin.Context) {
	var rsp proto.Response

	if !requireTailscaleChallenge(c, &rsp, authservice.SensitiveActionTailscaleStop) {
		return
	}

	if err := s.deinitializeManaged(c.Request.Context()); err != nil {
		if errors.Is(err, errTailscaleMutationBusy) {
			rsp.ErrRsp(c, -1, "tailscale operation already in progress")
			return
		}
		log.Errorf("failed to stop tailscale: %s", err)
		rsp.ErrRsp(c, -1, "stop failed")
		return
	}

	rsp.OkRsp(c)
	log.Debugf("tailscale stop successfully")
}

func (s *Service) Up(c *gin.Context) {
	var rsp proto.Response

	if !requireTailscaleChallenge(c, &rsp, authservice.SensitiveActionTailscaleUp) {
		return
	}
	release, ok := beginTailscaleMutation(c, &rsp, false)
	if !ok {
		return
	}
	defer release()

	err := s.actionClient().Up()
	if err != nil {
		rsp.ErrRsp(c, -1, "tailscale up failed")
		log.Errorf("failed to run tailscale up: %s", err)
		return
	}

	time.Sleep(1 * time.Second)

	rsp.OkRsp(c)
	log.Debugf("run tailscale up successfully")
}

func (s *Service) Down(c *gin.Context) {
	var rsp proto.Response

	if !requireTailscaleChallenge(c, &rsp, authservice.SensitiveActionTailscaleDown) {
		return
	}
	release, ok := beginTailscaleMutation(c, &rsp, false)
	if !ok {
		return
	}
	defer release()

	err := s.actionClient().Down()
	if err != nil {
		rsp.ErrRsp(c, -1, "tailscale down failed")
		log.Errorf("failed to run tailscale down: %s", err)
		return
	}

	rsp.OkRsp(c)
	log.Debugf("run tailscale down successfully")
}

func (s *Service) Login(c *gin.Context) {
	var rsp proto.Response

	if !requireTailscaleChallenge(c, &rsp, authservice.SensitiveActionTailscaleLogin) {
		return
	}
	release, ok := beginTailscaleMutation(c, &rsp, false)
	if !ok {
		return
	}
	defer release()

	// Bind LocalAPI calls and retry waits to request cancellation.
	ctx := c.Request.Context()
	if s.sipeed != nil && s.sipeed.snapshot().ManagedState == proto.TailscaleManagedInvalidIdentity {
		if err := s.sipeed.clearIdentity(); err != nil {
			log.Errorf("failed to clear invalid Sipeed identity: %s", err)
			rsp.ErrRsp(c, -2, "clear managed identity failed")
			return
		}
	}
	client := newTailscaleLoginClient(ctx)
	status, err := client.Status()
	if err != nil {
		// Avoid starting tailscaled for an already-canceled request.
		if ctx.Err() != nil {
			log.Errorf("failed to get tailscale status: %s", err)
			rsp.ErrRsp(c, -1, "unknown status")
			return
		}

		_ = client.Start()
		if err := waitForTailscaleLoginRetry(ctx); err != nil {
			log.Errorf("failed waiting for tailscale status retry: %s", err)
			rsp.ErrRsp(c, -1, "unknown status")
			return
		}
		status, err = client.Status()
	}

	if err != nil {
		log.Errorf("failed to get tailscale status: %s", err)
		rsp.ErrRsp(c, -1, "unknown status")
		return
	}

	if status.BackendState == "Running" {
		rsp.OkRspWithData(c, &proto.LoginTailscaleRsp{})
		return
	}

	// get login url
	url, err := client.Login()
	if err != nil {
		log.Errorf("failed to run tailscale login: %s", err)
		rsp.ErrRsp(c, -2, "login failed")
		return
	}

	rsp.OkRspWithData(c, &proto.LoginTailscaleRsp{
		Url: url,
	})

	log.Debugf("generated tailscale login url")
}

func waitForTailscaleLoginRetry(ctx context.Context) error {
	timer := time.NewTimer(tailscaleLoginStatusRetryDelay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (s *Service) Logout(c *gin.Context) {
	var rsp proto.Response

	if !requireTailscaleChallenge(c, &rsp, authservice.SensitiveActionTailscaleLogout) {
		return
	}
	release, ok := beginTailscaleMutation(c, &rsp, false)
	if !ok {
		return
	}
	defer release()

	err := s.actionClient().Logout()
	if err != nil {
		rsp.ErrRsp(c, -1, "logout failed")
		log.Errorf("failed to run tailscale logout: %s", err)
		return
	}

	rsp.OkRsp(c)
	log.Debugf("tailscale logout successfully")
}

func (s *Service) GetStatus(c *gin.Context) {
	var rsp proto.Response
	data, err := s.getStatus(c.Request.Context())
	if err != nil {
		log.Errorf("get tailscale status failed: %s", err)
		rsp.ErrRsp(c, -1, "unknown state")
		return
	}
	rsp.OkRspWithData(c, data)
	log.Debugf("get tailscale status successfully")
}

func (s *Service) getStatus(ctx context.Context) (*proto.GetTailscaleStatusRsp, error) {
	region := s.currentRegion()
	manifestEnabled := s.manifestEnabled()
	data := &proto.GetTailscaleStatusRsp{
		AvailableModes:           availableModesForRegion(region, manifestEnabled),
		ManagedState:             proto.TailscaleManagedDisabled,
		SipeedDisclaimerAccepted: s.sipeedDisclaimerAccepted(),
	}
	sipeedAvailable := sipeedModeAllowed(region, manifestEnabled)
	if sipeedAvailable && s.sipeed != nil {
		s.sipeed.appendStatus(data)
	}

	if tailscaleMutations.installing() {
		data.State = proto.TailscaleInstalling
		s.setManagedActiveMode(data)
		return data, nil
	}

	if !isInstalled() {
		data.State = proto.TailscaleNotInstall
		s.setManagedActiveMode(data)
		degradeConnectedManagedStatus(data, "clientMissing")
		return data, nil
	}

	client := NewLocalClientWithContext(ctx)
	status, err := client.Status()
	if err != nil {
		log.Debugf("failed to get tailscale status: %s", err)
		data.State = proto.TailscaleNotRunning
		s.setManagedActiveMode(data)
		degradeConnectedManagedStatus(data, "localAPIUnavailable")
		return data, nil
	}

	prefs, prefsErr := client.Prefs()
	if prefsErr != nil {
		log.Debugf("failed to get tailscale preferences: %s", prefsErr)
	}
	state, ok := effectiveTailscaleState(status, prefs)
	if !ok {
		return nil, errors.New("unknown tailscale state")
	}
	if state == proto.TailscaleNotLogin {
		data.State = state
		s.setManagedActiveMode(data)
		degradeConnectedManagedStatus(data, "connectionLost")
		return data, nil
	}

	ipv4 := ""
	for _, tailscaleIp := range status.Self.TailscaleIPs {
		ip := net.ParseIP(tailscaleIp)
		if ip != nil && ip.To4() != nil {
			ipv4 = ip.String()
		}
	}

	data.State = state
	data.IP = ipv4
	data.Name = status.Self.HostName

	profile, profileErr := client.CurrentProfile()
	managedRecord := sipeedAvailable && s.sipeed != nil && s.sipeed.hasManagedRecord()
	isSipeedProfile := s.isCurrentSipeedProfile(
		region,
		profile,
		profileErr,
		managedRecord,
	)
	if isSipeedProfile {
		mode := proto.TailscaleModeSipeed
		data.ActiveMode = &mode
		if data.ManagedState == proto.TailscaleManagedDisabled {
			data.ManagedState = proto.TailscaleManagedInvalidIdentity
			data.ManagedError = "identityMissing"
		}
		if data.ManagedState == proto.TailscaleManagedEnabledConnected {
			switch {
			case profileErr != nil:
				degradeConnectedManagedStatus(data, "profileReadFailed")
			case profile.ControlURL != sipeedControlURL:
				degradeConnectedManagedStatus(data, "profileChanged")
			case state != proto.TailscaleRunning || ipv4 == "":
				degradeConnectedManagedStatus(data, "connectionLost")
			}
		}
	} else {
		if normalizeNonSipeedProfileStatus(data, managedRecord, profile, profileErr) {
			data.Account = status.CurrentTailnet.Name
			view, routingErr := readRoutingView(client)
			if routingErr != nil {
				log.Warnf("failed to get tailscale routing preferences: %s", routingErr)
			} else {
				data.RoutingAvailable = view.available
				data.AdvertiseExitNode = view.exitNode
				data.AdvertiseSubnetRoute = view.subnet
				data.LANSubnet = view.lanSubnet
			}
		}
	}
	return data, nil
}

// normalizeNonSipeedProfileStatus keeps a failed or rate-limited Sipeed
// operation visible alongside a readable official profile. A clean official
// profile remains authoritative for ActiveMode; a readable empty profile does
// not hide the managed retry page.
func normalizeNonSipeedProfileStatus(
	data *proto.GetTailscaleStatusRsp,
	managedRecord bool,
	profile *TsProfile,
	profileErr error,
) bool {
	if data == nil {
		return false
	}
	// An explicitly requested mode switch can still be reading the old
	// official profile while enrollment runs. Keep the selected managed mode
	// visible until that transition either completes or rolls back.
	if managedRecord && (data.ManagedState == proto.TailscaleManagedEnrolling ||
		data.ManagedState == proto.TailscaleManagedEnabledConnecting) {
		mode := proto.TailscaleModeSipeed
		data.ActiveMode = &mode
		return false
	}
	officialProfileActive := isOfficialTailscaleProfileActive(profile, profileErr, data.State)
	if !managedRecord && profileErr == nil {
		data.ManagedState = proto.TailscaleManagedDisabled
		data.ManagedError = ""
		data.RetryAfterSeconds = 0
	}
	if managedRecord && data.ManagedState == proto.TailscaleManagedEnabledConnected {
		switch {
		case profileErr != nil:
			degradeConnectedManagedStatus(data, "profileReadFailed")
		case profile == nil || (profile.ID == "" && profile.NodeID == "" && profile.ControlURL == ""):
			degradeConnectedManagedStatus(data, "connectionLost")
		default:
			degradeConnectedManagedStatus(data, "profileChanged")
		}
	}
	if officialProfileActive {
		mode := proto.TailscaleModeOfficial
		data.ActiveMode = &mode
	} else if managedRecord {
		mode := proto.TailscaleModeSipeed
		data.ActiveMode = &mode
	}
	return officialProfileActive
}

func isOfficialTailscaleProfileActive(
	profile *TsProfile,
	profileErr error,
	state proto.TailscaleState,
) bool {
	return profileErr == nil && state != proto.TailscaleNotLogin && profile != nil &&
		(profile.NodeID != "" || state == proto.TailscaleRunning || state == proto.TailscaleStopped)
}

func (s *Service) isCurrentSipeedProfile(
	region proto.TailscaleRegion,
	profile *TsProfile,
	profileErr error,
	managedActive bool,
) bool {
	if !sipeedModeAllowed(region, s.manifestEnabled()) {
		return false
	}
	if profileErr == nil {
		return profile != nil && profile.ControlURL == sipeedControlURL
	}
	return managedActive
}

func degradeConnectedManagedStatus(data *proto.GetTailscaleStatusRsp, code string) {
	if data == nil || data.ManagedState != proto.TailscaleManagedEnabledConnected {
		return
	}
	data.ManagedState = proto.TailscaleManagedDegraded
	data.ManagedError = code
}

func (s *Service) setManagedActiveMode(data *proto.GetTailscaleStatusRsp) {
	if !s.sipeedEnabled() || s.sipeed == nil || !s.sipeed.hasManagedRecord() {
		return
	}
	mode := proto.TailscaleModeSipeed
	data.ActiveMode = &mode
}
