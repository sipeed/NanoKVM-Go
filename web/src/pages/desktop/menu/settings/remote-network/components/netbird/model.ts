import type { NetBirdStatus } from '@/api/extensions/netbird.ts';

import type { RemoteNetworkStatus } from '../../types.ts';

export function isLoginAvailable(status: NetBirdStatus | undefined): boolean {
  return Boolean(status?.enabled && status.running);
}

export function toRemoteNetworkStatus(status: NetBirdStatus): RemoteNetworkStatus {
  if (status.busy || ['installing', 'authorizing', 'connecting'].includes(status.state))
    return 'busy';
  if (status.state === 'connected') return 'on';
  if (status.state === 'error' || status.state === 'offline' || status.error) return 'bad';
  return 'off';
}

export function errorMessageKey(code: string): string {
  const codes = [
    'installFailed',
    'loginFailed',
    'loginExpired',
    'startRequired',
    'logoutFailed',
    'tunUnavailable',
    'apiUnavailable',
    'versionMismatch',
    'externalInstallation',
    'unsupportedArchitecture'
  ];
  return `netbird.${codes.includes(code) ? code : 'failed'}`;
}
