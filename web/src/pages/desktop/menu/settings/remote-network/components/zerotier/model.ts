import type { ZeroTierStatus } from '@/api/extensions/zerotier.ts';

import type { RemoteNetworkStatus } from '../../types.ts';

export function normalizeNetworkID(value: string): string | undefined {
  const id = value.trim().toLowerCase();
  return /^[0-9a-f]{16}$/.test(id) && id !== '0000000000000000' ? id : undefined;
}

export function toRemoteNetworkStatus(status: ZeroTierStatus): RemoteNetworkStatus {
  if (status.busy || status.state === 'installing' || status.state === 'connecting') return 'busy';
  if (status.state === 'connected') return 'on';
  if (status.state === 'error' || status.state === 'offline') return 'bad';
  return 'off';
}

export function networkMessageKey(status: string): string {
  const keys: Record<string, string> = {
    OK: 'connected',
    STOPPED: 'stopped',
    OFFLINE: 'offline',
    REQUESTING_CONFIGURATION: 'requesting',
    ACCESS_DENIED: 'denied',
    NOT_FOUND: 'notFound',
    PORT_ERROR: 'portError',
    CLIENT_TOO_OLD: 'clientTooOld',
    AUTHENTICATION_REQUIRED: 'authRequired',
    WAITING_ADDRESS: 'waitingAddress'
  };
  return `zerotier.${keys[status] ?? 'error'}`;
}

export function errorMessageKey(code: string): string {
  return `zerotier.${['invalidNetworkID', 'startRequired', 'tunUnavailable', 'apiUnavailable', 'versionMismatch', 'installFailed'].includes(code) ? code : 'failed'}`;
}
