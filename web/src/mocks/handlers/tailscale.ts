import { http } from 'msw';
import type { DefaultBodyType, HttpResponse } from 'msw';

import { fail, ok, readBoolean, readJsonObject, readString } from '../response.ts';
import type { MockState } from '../state.ts';

function sipeedModeEnabled(state: MockState): boolean {
  return state.application.tailscaleCnEnabled;
}

/**
 * Mirrors the server's feature gate: when the server-feature manifest
 * closes the managed mode, every Sipeed endpoint
 * answers the same -1 envelope as production before any state or cloud call.
 */
function featureGate(state: MockState): HttpResponse<DefaultBodyType> | null {
  if (!sipeedModeEnabled(state)) {
    return fail(-1, 'managed tailscale mode is unavailable', status(state));
  }
  return null;
}

function status(state: MockState) {
  const sipeed = sipeedModeEnabled(state);
  return {
    state: state.tailscale.state,
    name: state.tailscale.hostname,
    ip: state.tailscale.ip,
    account: state.tailscale.login ? 'mock@example.invalid' : '',
    routingAvailable: state.tailscale.routingAvailable,
    advertiseExitNode: state.tailscale.advertiseExitNode,
    advertiseSubnetRoute: state.tailscale.advertiseSubnetRoute,
    lanSubnet: state.tailscale.lanSubnet,
    availableModes: sipeed ? ['official', 'sipeed'] : ['official'],
    // The server never reflects a registered managed identity while the
    // manifest closes the mode, so the mock must not either.
    activeMode:
      !sipeed && state.tailscale.activeMode === 'sipeed' ? null : state.tailscale.activeMode,
    managedState: sipeed ? state.tailscale.managedState : 'disabled',
    managedError: sipeed ? state.tailscale.managedError : '',
    retryAfterSeconds: sipeed ? state.tailscale.retryAfterSeconds : 0,
    sipeedDisclaimerAccepted: state.tailscale.sipeedDisclaimerAccepted
  };
}

function advanceManagedEnable(state: MockState) {
  if (
    state.tailscale.managedState !== 'enrolling' &&
    state.tailscale.managedState !== 'enabledConnecting'
  ) {
    return;
  }
  if (state.tailscale.managedPollsRemaining > 0) {
    state.tailscale.managedPollsRemaining -= 1;
    state.tailscale.managedState = 'enabledConnecting';
    return;
  }
  state.tailscale.state = 'running';
  state.tailscale.login = true;
  state.tailscale.activeMode = 'sipeed';
  state.tailscale.managedState = 'enabledConnected';
  state.tailscale.managedError = '';
  if (!state.tailscale.sipeedDevices.some((device) => device.isSelf)) {
    state.tailscale.sipeedDevices.unshift({
      nodeId: 'mock-self-node',
      name: state.tailscale.hostname,
      ips: [state.tailscale.ip],
      online: true,
      lastSeen: new Date().toISOString(),
      isSelf: true
    });
  }
}

function verifyLoginChallenge(state: MockState, challenge: string | null): boolean {
  if (!state.auth.twoFactorEnabled || !state.auth.twoFactorProtection.tailscale) return true;
  if (!state.auth.tailscaleLoginChallenge || challenge !== state.auth.tailscaleLoginChallenge) {
    return false;
  }
  state.auth.tailscaleLoginChallenge = null;
  return true;
}

function verifyManagementChallenge(state: MockState, challenge: string | null, login: boolean) {
  if (!state.auth.twoFactorEnabled || !state.auth.twoFactorProtection.tailscale) return true;
  if (login) return verifyLoginChallenge(state, challenge);
  return typeof challenge === 'string' && challenge.trim() !== '';
}

function managementGuard(state: MockState): HttpResponse<DefaultBodyType> | null {
  if (!state.tailscale.sipeedIdentityValid) {
    state.tailscale.activeMode = null;
    state.tailscale.managedState = 'disabled';
    state.tailscale.managedError = '';
    state.tailscale.login = false;
    state.tailscale.state = 'notLogin';
    return fail(-2, 'managed tailscale identity expired', status(state));
  }
  if (Date.now() < state.tailscale.sipeedManagementRateLimitUntil) {
    const retryAfter = Math.max(
      1,
      Math.ceil((state.tailscale.sipeedManagementRateLimitUntil - Date.now()) / 1000)
    );
    state.tailscale.retryAfterSeconds = retryAfter;
    return fail(-2, 'managed tailscale operation is rate limited', status(state));
  }
  state.tailscale.retryAfterSeconds = 0;
  return null;
}

function waitForMockPending(request: Request): Promise<void> {
  if (request.signal.aborted) return Promise.resolve();

  return new Promise((resolve) => {
    let timer: ReturnType<typeof setTimeout>;
    const finish = () => {
      clearTimeout(timer);
      request.signal.removeEventListener('abort', finish);
      resolve();
    };
    timer = setTimeout(finish, 250);
    request.signal.addEventListener('abort', finish, { once: true });
  });
}

function clearManagedIdentity(state: MockState) {
  state.tailscale.activeMode = null;
  state.tailscale.managedState = 'disabled';
  state.tailscale.managedError = '';
  state.tailscale.retryAfterSeconds = 0;
  state.tailscale.managedNetworkRegistered = false;
  state.tailscale.sipeedClaimCode = null;
  state.tailscale.sipeedPending = [];
  state.tailscale.sipeedDevices = [];
  state.tailscale.login = false;
  state.tailscale.state = 'notLogin';
}

function verifyRoutingChallenge(
  state: MockState,
  action: 'exit-node' | 'subnet-route',
  challenge: string | null
): boolean {
  if (!state.auth.twoFactorEnabled || !state.auth.twoFactorProtection.tailscale) return true;

  const key =
    action === 'exit-node' ? 'tailscaleExitNodeChallenge' : 'tailscaleSubnetRouteChallenge';
  if (!state.auth[key] || challenge !== state.auth[key]) return false;

  state.auth[key] = null;
  return true;
}

export function createTailscaleHandlers(origin: string, state: MockState) {
  let nextClaimCode = 501530;
  let startupPollsRemaining = 0;
  return [
    http.get(`${origin}/api/extensions/tailscale/status`, () => {
      if (startupPollsRemaining > 0 && --startupPollsRemaining === 0) {
        state.tailscale.state = state.tailscale.login ? 'running' : 'notLogin';
        if (state.tailscale.login && state.tailscale.activeMode === null) {
          state.tailscale.activeMode = 'official';
        }
      }
      advanceManagedEnable(state);
      return ok(status(state));
    }),
    http.post(`${origin}/api/extensions/tailscale/install`, () => {
      state.tailscale.state = 'notLogin';
      return ok(status(state));
    }),
    http.post(`${origin}/api/extensions/tailscale/uninstall`, () => {
      startupPollsRemaining = 0;
      state.tailscale.state = 'notInstall';
      state.tailscale.login = false;
      state.tailscale.activeMode = null;
      state.tailscale.managedState = 'disabled';
      state.tailscale.managedError = '';
      state.tailscale.retryAfterSeconds = 0;
      state.tailscale.managedNetworkRegistered = false;
      state.tailscale.sipeedIdentityValid = true;
      state.tailscale.sipeedClaimCode = null;
      state.tailscale.sipeedPending = [];
      state.tailscale.sipeedDevices = [];
      return ok(status(state));
    }),
    http.post(`${origin}/api/extensions/tailscale/start`, () => {
      if (state.tailscale.state === 'notRunning') {
        startupPollsRemaining = 2;
      }
      return ok(status(state));
    }),
    http.post(`${origin}/api/extensions/tailscale/restart`, () => ok(status(state))),
    http.post(`${origin}/api/extensions/tailscale/stop`, () => {
      startupPollsRemaining = 0;
      if (state.tailscale.activeMode === 'sipeed') clearManagedIdentity(state);
      state.tailscale.activeMode = null;
      state.tailscale.state = 'notRunning';
      return ok(status(state));
    }),
    http.post(`${origin}/api/extensions/tailscale/up`, () => {
      state.tailscale.state = 'running';
      return ok(status(state));
    }),
    http.post(`${origin}/api/extensions/tailscale/down`, () => {
      state.tailscale.state = 'stopped';
      return ok(status(state));
    }),
    http.post(`${origin}/api/extensions/tailscale/login`, () => {
      state.tailscale.login = true;
      state.tailscale.state = 'running';
      state.tailscale.activeMode = 'official';
      return ok({ url: '' });
    }),
    http.post(`${origin}/api/extensions/tailscale/logout`, () => {
      state.tailscale.login = false;
      state.tailscale.state = 'notLogin';
      state.tailscale.activeMode = null;
      return ok(status(state));
    }),
    http.post(`${origin}/api/extensions/tailscale/sipeed/enable`, async ({ request }) => {
      const gate = featureGate(state);
      if (gate) return gate;
      const body = await readJsonObject(request);
      if (!verifyLoginChallenge(state, readString(body, 'challenge'))) {
        return fail(-3, 'sensitive action verification required', status(state));
      }
      if (
        state.tailscale.managedState === 'enrolling' ||
        state.tailscale.managedState === 'enabledConnecting'
      ) {
        return fail(-1, 'tailscale operation already in progress', status(state));
      }
      if (!state.tailscale.sipeedDisclaimerAccepted) {
        if (readBoolean(body, 'acceptDisclaimer') !== true) {
          return fail(-1, 'sipeed disclaimer acceptance is required', status(state));
        }
        state.tailscale.sipeedDisclaimerAccepted = true;
      }
      if (state.tailscale.retryAfterSeconds > 0) {
        state.tailscale.activeMode = state.tailscale.activeMode ?? 'sipeed';
        state.tailscale.managedState = 'degraded';
        state.tailscale.managedError = 'rateLimited';
        return fail(-1, 'managed tailscale enable is rate limited', status(state));
      }
      // Starting Sipeed logs out the old account instead of saving its profile.
      if (state.tailscale.activeMode !== 'sipeed') state.tailscale.login = false;
      startupPollsRemaining = 0;
      if (state.tailscale.managedNetworkRegistered) {
        state.tailscale.managedJoinKeyRequests += 1;
      } else {
        state.tailscale.managedNetworkRegistered = true;
        state.tailscale.managedEnrollRequests += 1;
      }
      state.tailscale.activeMode = 'sipeed';
      state.tailscale.sipeedIdentityValid = true;
      state.tailscale.managedState = 'enrolling';
      state.tailscale.managedError = '';
      state.tailscale.retryAfterSeconds = 0;
      state.tailscale.managedPollsRemaining = 1;
      return ok(status(state));
    }),
    http.post(`${origin}/api/extensions/tailscale/sipeed/logout`, async ({ request }) => {
      const gate = featureGate(state);
      if (gate) return gate;
      const guard = managementGuard(state);
      if (guard) return guard;
      const body = await readJsonObject(request);
      if (!verifyManagementChallenge(state, readString(body, 'challenge'), false)) {
        return fail(-3, 'sensitive action verification required', status(state));
      }
      const cloudCleanupFailed = state.tailscale.sipeedCloudCleanupFailed;
      clearManagedIdentity(state);
      return ok({ cloudCleanupFailed });
    }),
    http.post(
      `${origin}/api/extensions/tailscale/sipeed/disclaimer/revoke`,
      async ({ request }) => {
        const gate = featureGate(state);
        if (gate) return gate;
        const body = await readJsonObject(request);
        if (!verifyManagementChallenge(state, readString(body, 'challenge'), false)) {
          return fail(-3, 'sensitive action verification required', status(state));
        }
        if (['enrolling', 'enabledConnecting'].includes(state.tailscale.managedState)) {
          return fail(-1, 'tailscale operation already in progress', status(state));
        }
        state.tailscale.sipeedDisclaimerAccepted = false;
        const cloudCleanupFailed = state.tailscale.sipeedCloudCleanupFailed;
        clearManagedIdentity(state);
        state.tailscale.state = 'notRunning';
        return ok({ cloudCleanupFailed });
      }
    ),
    http.post(`${origin}/api/extensions/tailscale/sipeed/claim-code`, async ({ request }) => {
      const gate = featureGate(state);
      if (gate) return gate;
      const guard = managementGuard(state);
      if (guard) return guard;
      const body = await readJsonObject(request);
      if (!verifyManagementChallenge(state, readString(body, 'challenge'), true)) {
        return fail(-3, 'sensitive action verification required', status(state));
      }
      // Keep the shared code unless the browser explicitly requests renewal.
      const cached = state.tailscale.sipeedClaimCode;
      if (
        !readBoolean(body, 'refresh') &&
        cached &&
        Date.parse(cached.expiresAt) - Date.now() > 20_000
      ) {
        return ok(cached);
      }
      const expiresAt = new Date(Date.now() + 10 * 60 * 1000).toISOString();
      state.tailscale.sipeedClaimCode = {
        code: String(nextClaimCode++).padStart(6, '0'),
        expiresAt,
        singleUse: true
      };
      return ok(state.tailscale.sipeedClaimCode);
    }),
    http.get(`${origin}/api/extensions/tailscale/sipeed/pending`, async ({ request }) => {
      const gate = featureGate(state);
      if (gate) return gate;
      const guard = managementGuard(state);
      if (guard) return guard;
      const claim = state.tailscale.sipeedClaimCode;
      if (state.tailscale.sipeedPending.length === 0) {
        await waitForMockPending(request);
      }
      if (state.tailscale.sipeedPending.length > 0 && state.tailscale.sipeedClaimCode === claim) {
        state.tailscale.sipeedClaimCode = null;
      }
      return ok({ pending: state.tailscale.sipeedPending });
    }),
    http.post(
      `${origin}/api/extensions/tailscale/sipeed/pending/:id/approve`,
      async ({ request, params }) => {
        const gate = featureGate(state);
        if (gate) return gate;
        const guard = managementGuard(state);
        if (guard) return guard;
        const body = await readJsonObject(request);
        if (!verifyManagementChallenge(state, readString(body, 'challenge'), true)) {
          return fail(-3, 'sensitive action verification required', status(state));
        }
        const pendingId = String(params.id ?? '');
        const index = state.tailscale.sipeedPending.findIndex(
          (item) => item.pendingId === pendingId
        );
        if (index < 0) return fail(-1, 'pending request not found', {});
        const pending = state.tailscale.sipeedPending.splice(index, 1)[0];
        const name =
          readString(body, 'name')?.trim() ||
          pending.clientUa ||
          `device-${state.tailscale.sipeedNextDeviceId}`;
        const device = {
          nodeId: `mock-node-${state.tailscale.sipeedNextDeviceId++}`,
          name,
          ips: [`100.64.0.${40 + state.tailscale.sipeedNextDeviceId}`],
          online: true,
          lastSeen: new Date().toISOString(),
          isSelf: false
        };
        state.tailscale.sipeedDevices.push(device);
        state.tailscale.sipeedClaimCode = null;
        return ok({ nodeId: device.nodeId, name: device.name, ips: device.ips });
      }
    ),
    http.post(
      `${origin}/api/extensions/tailscale/sipeed/pending/:id/reject`,
      async ({ request, params }) => {
        const gate = featureGate(state);
        if (gate) return gate;
        const guard = managementGuard(state);
        if (guard) return guard;
        const body = await readJsonObject(request);
        if (!verifyManagementChallenge(state, readString(body, 'challenge'), true)) {
          return fail(-3, 'sensitive action verification required', status(state));
        }
        const pendingId = String(params.id ?? '');
        const index = state.tailscale.sipeedPending.findIndex(
          (item) => item.pendingId === pendingId
        );
        if (index < 0) return fail(-1, 'pending request not found', {});
        state.tailscale.sipeedPending.splice(index, 1);
        state.tailscale.sipeedClaimCode = null;
        return ok({ pendingId, rejected: true });
      }
    ),
    http.post(`${origin}/api/extensions/tailscale/sipeed/authorize`, async ({ request }) => {
      const gate = featureGate(state);
      if (gate) return gate;
      const guard = managementGuard(state);
      if (guard) return guard;
      const body = await readJsonObject(request);
      if (!verifyManagementChallenge(state, readString(body, 'challenge'), true)) {
        return fail(-3, 'sensitive action verification required', status(state));
      }
      const code = (readString(body, 'code') ?? '').toUpperCase().replace(/-/g, '');
      if (!/^[23456789ABCDEFGHJKMNPQRSTVWXYZ]{8}$/.test(code)) {
        return fail(-1, 'invalid authorization code', {});
      }
      const name =
        readString(body, 'name')?.trim() || `device-${state.tailscale.sipeedNextDeviceId}`;
      const device = {
        nodeId: `mock-node-${state.tailscale.sipeedNextDeviceId++}`,
        name,
        ips: [`100.64.0.${40 + state.tailscale.sipeedNextDeviceId}`],
        online: true,
        lastSeen: new Date().toISOString(),
        isSelf: false
      };
      state.tailscale.sipeedDevices.push(device);
      state.tailscale.sipeedClaimCode = null;
      return ok({ nodeId: device.nodeId, name: device.name, ips: device.ips });
    }),
    http.get(`${origin}/api/extensions/tailscale/sipeed/devices`, () => {
      const gate = featureGate(state);
      if (gate) return gate;
      const guard = managementGuard(state);
      if (guard) return guard;
      return ok({ networkId: 'mock-network', devices: state.tailscale.sipeedDevices });
    }),
    http.post(
      `${origin}/api/extensions/tailscale/sipeed/devices/:id/name`,
      async ({ request, params }) => {
        const gate = featureGate(state);
        if (gate) return gate;
        const guard = managementGuard(state);
        if (guard) return guard;
        const body = await readJsonObject(request);
        if (!verifyManagementChallenge(state, readString(body, 'challenge'), true)) {
          return fail(-3, 'sensitive action verification required', status(state));
        }
        const nodeId = String(params.id ?? '');
        const name = readString(body, 'name')?.trim() ?? '';
        const device = state.tailscale.sipeedDevices.find((item) => item.nodeId === nodeId);
        if (!device || !name) return fail(-1, 'device not found', {});
        device.name = name;
        return ok({ nodeId, name });
      }
    ),
    http.delete(
      `${origin}/api/extensions/tailscale/sipeed/devices/:id`,
      async ({ request, params }) => {
        const gate = featureGate(state);
        if (gate) return gate;
        const guard = managementGuard(state);
        if (guard) return guard;
        const body = await readJsonObject(request);
        if (!verifyManagementChallenge(state, readString(body, 'challenge'), true)) {
          return fail(-3, 'sensitive action verification required', status(state));
        }
        const nodeId = String(params.id ?? '');
        const device = state.tailscale.sipeedDevices.find((item) => item.nodeId === nodeId);
        if (!device) return fail(-1, 'device not found', {});
        if (device.isSelf) return fail(-1, 'the current device can only be removed by logout', {});
        state.tailscale.sipeedDevices = state.tailscale.sipeedDevices.filter(
          (item) => item.nodeId !== nodeId
        );
        return ok({ nodeId, removed: true });
      }
    ),
    http.post(`${origin}/api/extensions/tailscale/exit-node`, async ({ request }) => {
      const body = await readJsonObject(request);
      const enabled = readBoolean(body, 'enabled');
      if (enabled === null) return fail(-1, 'invalid parameters', status(state));
      if (!verifyRoutingChallenge(state, 'exit-node', readString(body, 'challenge'))) {
        return fail(-3, 'sensitive action verification required', status(state));
      }

      state.tailscale.advertiseExitNode = enabled;
      return ok(status(state));
    }),
    http.post(`${origin}/api/extensions/tailscale/subnet-route`, async ({ request }) => {
      const body = await readJsonObject(request);
      const enabled = readBoolean(body, 'enabled');
      if (enabled === null) return fail(-1, 'invalid parameters', status(state));
      if (!verifyRoutingChallenge(state, 'subnet-route', readString(body, 'challenge'))) {
        return fail(-3, 'sensitive action verification required', status(state));
      }
      if (enabled && (!state.tailscale.routingAvailable || !state.tailscale.lanSubnet)) {
        return fail(-1, 'subnet route unavailable', status(state));
      }

      state.tailscale.advertiseSubnetRoute = enabled;
      return ok(status(state));
    })
  ];
}
