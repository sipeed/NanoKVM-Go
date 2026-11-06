package tailscale

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

type fakeTailscaleActionClient struct {
	events         *[]string
	startCalls     int
	restartCalls   int
	stopCalls      int
	stopErr        error
	upCalls        int
	downCalls      int
	logoutCalls    int
	uninstallCalls int
}

func (fake *fakeTailscaleActionClient) record(event string) {
	if fake.events != nil {
		*fake.events = append(*fake.events, "action."+event)
	}
}

func (fake *fakeTailscaleActionClient) Start() error {
	fake.startCalls++
	fake.record("start")
	return nil
}

func (fake *fakeTailscaleActionClient) Restart() error {
	fake.restartCalls++
	fake.record("restart")
	return nil
}

func (fake *fakeTailscaleActionClient) Stop() error {
	fake.stopCalls++
	fake.record("stop")
	return fake.stopErr
}

func (fake *fakeTailscaleActionClient) Up() error {
	fake.upCalls++
	fake.record("up")
	return nil
}

func (fake *fakeTailscaleActionClient) Down() error {
	fake.downCalls++
	fake.record("down")
	return nil
}

func (fake *fakeTailscaleActionClient) Logout() error {
	fake.logoutCalls++
	fake.record("logout")
	return nil
}

func (fake *fakeTailscaleActionClient) Uninstall() error {
	fake.uninstallCalls++
	fake.record("uninstall")
	return nil
}

func newTailscaleLifecycleContext(target string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(recorder)
	ginContext.Request = httptest.NewRequest(http.MethodPost, target, nil)
	return ginContext, recorder
}

func decodeTailscaleLifecycleResponse(t *testing.T, recorder *httptest.ResponseRecorder) proto.Response {
	t.Helper()
	var response proto.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode lifecycle response: %v: %s", err, recorder.Body.String())
	}
	return response
}

func newManagedLifecycleService(
	t *testing.T,
	events *[]string,
) (*Service, *sipeedManager, sipeedPersistedState, *fakeSipeedLocalClient, *fakeSipeedPairingClient, *fakeTailscaleActionClient) {
	t.Helper()
	useTestServerFeatureFlags(t, true)
	local, pairing := successfulSipeedFakes(events)
	manager, state := newConnectedSipeedLogoutManager(t, local, pairing)
	local.currentProfiles = []*TsProfile{{
		ID: state.ProfileID, NodeID: state.NodeID, ControlURL: sipeedControlURL,
	}}
	local.profiles = []TsProfile{{ID: "empty-profile", ControlURL: "https://controlplane.tailscale.com"}}
	action := &fakeTailscaleActionClient{events: events}
	service := &Service{
		sipeed:      manager,
		regionStore: useTestTailscaleRegion(t, proto.TailscaleRegionCN),
		newActionClient: func() tailscaleActionClient {
			return action
		},
		uninstallAction:  action.Uninstall,
		waitAfterRestart: func() {},
	}
	return service, manager, state, local, pairing, action
}

func TestSipeedAndOfficialLogoutHandlersRemainSeparate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Sipeed logout destroys the managed network", func(t *testing.T) {
		isolatedMutationCoordinator(t)
		disableTailscaleChallengesForTest(t)
		var events []string
		service, manager, state, _, pairing, action := newManagedLifecycleService(t, &events)

		ginContext, recorder := newTailscaleLifecycleContext("/api/extensions/tailscale/sipeed/logout")
		service.SipeedLogout(ginContext)
		response := decodeTailscaleLifecycleResponse(t, recorder)
		if response.Code != 0 {
			t.Fatalf("Sipeed logout response = %+v", response)
		}
		if len(pairing.deleteNetworkIDs) != 1 || pairing.deleteNetworkIDs[0] != state.NetworkID ||
			len(pairing.deleteNodeIDs) != 0 {
			t.Fatalf("Sipeed logout cleanup = networks %v, nodes %v", pairing.deleteNetworkIDs, pairing.deleteNodeIDs)
		}
		if action.logoutCalls != 0 {
			t.Fatalf("Sipeed logout used the official logout action %d times", action.logoutCalls)
		}
		if got := manager.snapshot(); got.ManagedState != proto.TailscaleManagedDisabled {
			t.Fatalf("Sipeed logout kept local identity: %+v", got)
		}
	})

	t.Run("Sipeed logout keeps the compatible cloud failure field", func(t *testing.T) {
		isolatedMutationCoordinator(t)
		disableTailscaleChallengesForTest(t)
		service, manager, _, _, pairing, _ := newManagedLifecycleService(t, nil)
		pairing.deleteNetworkErr = &sipeedPairingAPIError{
			HTTPStatus: http.StatusBadGateway,
			Code:       "upstream_error",
		}

		ginContext, recorder := newTailscaleLifecycleContext("/api/extensions/tailscale/sipeed/logout")
		service.SipeedLogout(ginContext)
		var response struct {
			Code int             `json:"code"`
			Data sipeedLogoutRsp `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode Sipeed logout response: %v", err)
		}
		if response.Code != 0 || !response.Data.CloudCleanupFailed {
			t.Fatalf("Sipeed cloud failure response = %+v", response)
		}
		if got := manager.snapshot(); got.ManagedState != proto.TailscaleManagedDisabled || got.DeviceToken != "" {
			t.Fatalf("Sipeed cloud failure kept local identity: %+v", got)
		}
	})

	t.Run("official logout stays local", func(t *testing.T) {
		isolatedMutationCoordinator(t)
		disableTailscaleChallengesForTest(t)
		action := &fakeTailscaleActionClient{}
		service := &Service{
			regionStore: useTestTailscaleRegion(t, proto.TailscaleRegionGlobal),
			newActionClient: func() tailscaleActionClient {
				return action
			},
		}

		ginContext, recorder := newTailscaleLifecycleContext("/api/extensions/tailscale/logout")
		service.Logout(ginContext)
		response := decodeTailscaleLifecycleResponse(t, recorder)
		if response.Code != 0 || action.logoutCalls != 1 {
			t.Fatalf("official logout response=%+v calls=%d", response, action.logoutCalls)
		}
	})
}

func TestStopAndUninstallDestroyManagedNetworkFirst(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name        string
		target      string
		run         func(*Service, *gin.Context)
		actionName  string
		actionCalls func(*fakeTailscaleActionClient) int
	}{
		{
			name: "stop", target: "/api/extensions/tailscale/stop",
			run: (*Service).Stop, actionName: "action.stop",
			actionCalls: func(fake *fakeTailscaleActionClient) int { return fake.stopCalls },
		},
		{
			name: "uninstall", target: "/api/extensions/tailscale/uninstall",
			run: (*Service).Uninstall, actionName: "action.uninstall",
			actionCalls: func(fake *fakeTailscaleActionClient) int { return fake.uninstallCalls },
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			isolatedMutationCoordinator(t)
			disableTailscaleChallengesForTest(t)
			var events []string
			service, manager, state, _, pairing, action := newManagedLifecycleService(t, &events)

			ginContext, recorder := newTailscaleLifecycleContext(testCase.target)
			testCase.run(service, ginContext)
			response := decodeTailscaleLifecycleResponse(t, recorder)
			if response.Code != 0 || testCase.actionCalls(action) != 1 {
				t.Fatalf("%s response=%+v calls=%d", testCase.name, response, testCase.actionCalls(action))
			}
			if len(pairing.deleteNetworkIDs) != 1 || pairing.deleteNetworkIDs[0] != state.NetworkID ||
				len(pairing.deleteNodeIDs) != 0 {
				t.Fatalf("%s cleanup = networks %v, nodes %v", testCase.name, pairing.deleteNetworkIDs, pairing.deleteNodeIDs)
			}
			if len(events) == 0 || events[len(events)-1] != testCase.actionName {
				t.Fatalf("%s events = %v", testCase.name, events)
			}
			if got := manager.snapshot(); got.ManagedState != proto.TailscaleManagedDisabled {
				t.Fatalf("%s kept local identity: %+v", testCase.name, got)
			}
		})
	}
}

func TestStopAndUninstallContinueAfterCloudDeleteFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name        string
		target      string
		run         func(*Service, *gin.Context)
		actionCalls func(*fakeTailscaleActionClient) int
	}{
		{
			name: "stop", target: "/api/extensions/tailscale/stop", run: (*Service).Stop,
			actionCalls: func(fake *fakeTailscaleActionClient) int { return fake.stopCalls },
		},
		{
			name: "uninstall", target: "/api/extensions/tailscale/uninstall", run: (*Service).Uninstall,
			actionCalls: func(fake *fakeTailscaleActionClient) int { return fake.uninstallCalls },
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			isolatedMutationCoordinator(t)
			disableTailscaleChallengesForTest(t)
			service, manager, _, _, pairing, action := newManagedLifecycleService(t, nil)
			pairing.deleteNetworkErr = &sipeedPairingAPIError{
				HTTPStatus: http.StatusBadGateway,
				Code:       "upstream_error",
			}

			ginContext, recorder := newTailscaleLifecycleContext(testCase.target)
			testCase.run(service, ginContext)
			response := decodeTailscaleLifecycleResponse(t, recorder)
			if response.Code != 0 || testCase.actionCalls(action) != 1 {
				t.Fatalf("%s response=%+v calls=%d", testCase.name, response, testCase.actionCalls(action))
			}
			if got := manager.snapshot(); got.ManagedState != proto.TailscaleManagedDisabled || got.DeviceToken != "" {
				t.Fatalf("%s cloud failure kept local identity: %+v", testCase.name, got)
			}
		})
	}
}

func TestStopAndUninstallAbortAfterLocalLogoutFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name        string
		target      string
		run         func(*Service, *gin.Context)
		actionCalls func(*fakeTailscaleActionClient) int
	}{
		{
			name: "stop", target: "/api/extensions/tailscale/stop", run: (*Service).Stop,
			actionCalls: func(fake *fakeTailscaleActionClient) int { return fake.stopCalls },
		},
		{
			name: "uninstall", target: "/api/extensions/tailscale/uninstall", run: (*Service).Uninstall,
			actionCalls: func(fake *fakeTailscaleActionClient) int { return fake.uninstallCalls },
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			isolatedMutationCoordinator(t)
			disableTailscaleChallengesForTest(t)
			service, manager, state, local, pairing, action := newManagedLifecycleService(t, nil)
			local.logoutErr = errors.New("local logout failed")

			ginContext, recorder := newTailscaleLifecycleContext(testCase.target)
			testCase.run(service, ginContext)
			response := decodeTailscaleLifecycleResponse(t, recorder)
			if response.Code != -1 || testCase.actionCalls(action) != 0 {
				t.Fatalf("%s response=%+v calls=%d", testCase.name, response, testCase.actionCalls(action))
			}
			if len(pairing.deleteNetworkIDs) != 0 {
				t.Fatalf("%s reached cloud cleanup after local failure: %v", testCase.name, pairing.deleteNetworkIDs)
			}
			if got := manager.snapshot(); got.DeviceToken != state.DeviceToken ||
				got.ManagedState != proto.TailscaleManagedEnabledConnected {
				t.Fatalf("%s local failure discarded identity: %+v", testCase.name, got)
			}
		})
	}
}

func TestStopAbortsWhenLocalIdentityCannotBeCleared(t *testing.T) {
	gin.SetMode(gin.TestMode)
	isolatedMutationCoordinator(t)
	disableTailscaleChallengesForTest(t)
	service, manager, state, _, pairing, action := newManagedLifecycleService(t, nil)
	if err := os.Remove(manager.deps.store.path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("blocked-state", manager.deps.store.path); err != nil {
		t.Fatal(err)
	}

	ginContext, recorder := newTailscaleLifecycleContext("/api/extensions/tailscale/stop")
	service.Stop(ginContext)
	response := decodeTailscaleLifecycleResponse(t, recorder)
	if response.Code != -1 || action.stopCalls != 0 {
		t.Fatalf("stop response=%+v calls=%d", response, action.stopCalls)
	}
	if len(pairing.deleteNetworkIDs) != 1 || pairing.deleteNetworkIDs[0] != state.NetworkID {
		t.Fatalf("network cleanup calls = %v", pairing.deleteNetworkIDs)
	}
	if got := manager.snapshot(); got.DeviceToken != state.DeviceToken ||
		got.ManagedState != proto.TailscaleManagedEnabledConnected {
		t.Fatalf("failed local clear discarded identity: %+v", got)
	}
}

func TestOfficialStopAndUninstallRemainUnchanged(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name        string
		target      string
		run         func(*Service, *gin.Context)
		actionCalls func(*fakeTailscaleActionClient) int
	}{
		{
			name: "stop", target: "/api/extensions/tailscale/stop", run: (*Service).Stop,
			actionCalls: func(fake *fakeTailscaleActionClient) int { return fake.stopCalls },
		},
		{
			name: "uninstall", target: "/api/extensions/tailscale/uninstall", run: (*Service).Uninstall,
			actionCalls: func(fake *fakeTailscaleActionClient) int { return fake.uninstallCalls },
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			isolatedMutationCoordinator(t)
			disableTailscaleChallengesForTest(t)
			action := &fakeTailscaleActionClient{}
			service := &Service{
				regionStore: useTestTailscaleRegion(t, proto.TailscaleRegionGlobal),
				newActionClient: func() tailscaleActionClient {
					return action
				},
				uninstallAction: action.Uninstall,
			}

			ginContext, recorder := newTailscaleLifecycleContext(testCase.target)
			testCase.run(service, ginContext)
			response := decodeTailscaleLifecycleResponse(t, recorder)
			if response.Code != 0 || testCase.actionCalls(action) != 1 {
				t.Fatalf("%s response=%+v calls=%d", testCase.name, response, testCase.actionCalls(action))
			}
		})
	}
}

func TestRestartPreservesManagedIdentityAndNetwork(t *testing.T) {
	gin.SetMode(gin.TestMode)
	isolatedMutationCoordinator(t)
	disableTailscaleChallengesForTest(t)
	service, manager, state, local, pairing, action := newManagedLifecycleService(t, nil)

	ginContext, recorder := newTailscaleLifecycleContext("/api/extensions/tailscale/restart")
	service.Restart(ginContext)
	response := decodeTailscaleLifecycleResponse(t, recorder)
	if response.Code != 0 || action.restartCalls != 1 {
		t.Fatalf("restart response=%+v calls=%d", response, action.restartCalls)
	}
	if local.currentCalls != 0 || len(pairing.deleteNetworkIDs) != 0 || len(pairing.deleteNodeIDs) != 0 {
		t.Fatalf(
			"restart touched managed logout: profileCalls=%d networks=%v nodes=%v",
			local.currentCalls,
			pairing.deleteNetworkIDs,
			pairing.deleteNodeIDs,
		)
	}
	if got := manager.snapshot(); got.DeviceToken != state.DeviceToken || got.NetworkID != state.NetworkID ||
		got.ManagedState != state.ManagedState {
		t.Fatalf("restart changed managed identity: %+v", got)
	}
}

var _ tailscaleActionClient = (*fakeTailscaleActionClient)(nil)
