import { http } from '@/lib/http.ts';

export type SipeedClaimCode = {
  code: string;
  expiresAt: string;
  singleUse: boolean;
};

export type SipeedPendingRequest = {
  pendingId: string;
  sas: string[];
  createdAt: string;
  clientIp?: string;
  clientUa?: string;
};

export type SipeedNode = {
  nodeId: string;
  name: string;
  ips: string[];
};

export type SipeedDevice = SipeedNode & {
  online: boolean;
  lastSeen?: string;
  isSelf: boolean;
};

export type SipeedDevices = {
  networkId?: string;
  devices: SipeedDevice[];
};

export type SipeedLogoutResponse = {
  cloudCleanupFailed: boolean;
};

/** Post with challenge body only when a challenge is present (preserve legacy no-body calls). */
function postWithOptionalChallenge(url: string, challenge?: string, timeoutMs?: number) {
  const config = timeoutMs ? { timeout: timeoutMs } : undefined;
  if (challenge) {
    return http.post(url, { challenge }, config);
  }
  return http.post(url, undefined, config);
}

/** Post a boolean setting while preserving the sensitive-action challenge contract. */
function postSettingWithOptionalChallenge(url: string, enabled: boolean, challenge?: string) {
  return http.post(url, challenge ? { enabled, challenge } : { enabled });
}

// install tailscale (download can take several minutes on device links)
export function install(challenge?: string) {
  return postWithOptionalChallenge('/api/extensions/tailscale/install', challenge, 5 * 60 * 1000);
}

// uninstall tailscale
export function uninstall(challenge?: string) {
  return postWithOptionalChallenge('/api/extensions/tailscale/uninstall', challenge);
}

// get tailscale status
export function getStatus() {
  return http.get('/api/extensions/tailscale/status');
}

// enable or retry the Sipeed managed identity
export function enableSipeed(challenge?: string, acceptDisclaimer = false) {
  return http.post('/api/extensions/tailscale/sipeed/enable', {
    acceptDisclaimer,
    ...(challenge ? { challenge } : {})
  });
}

export function logoutSipeed(challenge?: string) {
  return postWithOptionalChallenge('/api/extensions/tailscale/sipeed/logout', challenge);
}

export function revokeSipeedDisclaimer(challenge?: string) {
  return postWithOptionalChallenge('/api/extensions/tailscale/sipeed/disclaimer/revoke', challenge);
}

export function getSipeedClaimCode(challenge?: string, refresh = false) {
  if (refresh) {
    return http.post('/api/extensions/tailscale/sipeed/claim-code', {
      refresh: true,
      ...(challenge ? { challenge } : {})
    });
  }
  return postWithOptionalChallenge('/api/extensions/tailscale/sipeed/claim-code', challenge);
}

export function getSipeedPending(signal?: AbortSignal) {
  return http.request({
    method: 'get',
    url: '/api/extensions/tailscale/sipeed/pending',
    params: { wait: 20 },
    timeout: 40 * 1000,
    signal
  });
}

export function approveSipeedPending(pendingId: string, name?: string, challenge?: string) {
  return postSipeedMutation(
    `/api/extensions/tailscale/sipeed/pending/${encodeURIComponent(pendingId)}/approve`,
    {
      name,
      challenge
    }
  );
}

export function rejectSipeedPending(pendingId: string, challenge?: string) {
  return postSipeedMutation(
    `/api/extensions/tailscale/sipeed/pending/${encodeURIComponent(pendingId)}/reject`,
    {
      challenge
    }
  );
}

export function authorizeSipeed(code: string, name?: string, challenge?: string) {
  return postSipeedMutation('/api/extensions/tailscale/sipeed/authorize', {
    code,
    name,
    challenge
  });
}

export function getSipeedDevices() {
  return http.get('/api/extensions/tailscale/sipeed/devices');
}

export function renameSipeedDevice(nodeId: string, name: string, challenge?: string) {
  return postSipeedMutation(
    `/api/extensions/tailscale/sipeed/devices/${encodeURIComponent(nodeId)}/name`,
    {
      name,
      challenge
    }
  );
}

export function removeSipeedDevice(nodeId: string, challenge?: string) {
  return http.delete(
    `/api/extensions/tailscale/sipeed/devices/${encodeURIComponent(nodeId)}`,
    challenge ? { challenge } : undefined
  );
}

function postSipeedMutation(url: string, payload: Record<string, string | undefined>) {
  const data = Object.fromEntries(Object.entries(payload).filter(([, value]) => value));
  return http.post(url, data);
}

// start tailscale
export function start(challenge?: string) {
  return postWithOptionalChallenge('/api/extensions/tailscale/start', challenge);
}

// restart tailscale
export function restart(challenge?: string) {
  return postWithOptionalChallenge('/api/extensions/tailscale/restart', challenge);
}

// stop tailscale
export function stop(challenge?: string) {
  return postWithOptionalChallenge('/api/extensions/tailscale/stop', challenge);
}

// run tailscale up
export function up(challenge?: string) {
  return postWithOptionalChallenge('/api/extensions/tailscale/up', challenge);
}

// run tailscale down
export function down(challenge?: string) {
  return postWithOptionalChallenge('/api/extensions/tailscale/down', challenge);
}

// login tailscale
export function login(challenge?: string) {
  return postWithOptionalChallenge('/api/extensions/tailscale/login', challenge);
}

// logout tailscale
export function logout(challenge?: string) {
  return postWithOptionalChallenge('/api/extensions/tailscale/logout', challenge);
}

// advertise this device as an exit node
export function setExitNode(enabled: boolean, challenge?: string) {
  return postSettingWithOptionalChallenge(
    '/api/extensions/tailscale/exit-node',
    enabled,
    challenge
  );
}

// advertise this device's directly connected LAN subnet
export function setSubnetRoute(enabled: boolean, challenge?: string) {
  return postSettingWithOptionalChallenge(
    '/api/extensions/tailscale/subnet-route',
    enabled,
    challenge
  );
}
