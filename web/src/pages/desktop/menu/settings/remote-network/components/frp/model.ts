import type { FRPConfigInput, FRPStatus } from '@/api/extensions/frp.ts';

import type { RemoteNetworkStatus } from '../../types.ts';

export function toRemoteNetworkStatus(status: FRPStatus): RemoteNetworkStatus {
  if (status.busy || status.state === 'installing' || status.state === 'connecting') return 'busy';
  if (status.state === 'connected') return 'on';
  if (status.state === 'error') return 'bad';
  return 'off';
}

export function validateConfig(config: FRPConfigInput): string | undefined {
  if (!config.serverAddr.trim()) return 'invalidServer';
  const ports = [config.serverPort, config.remotePort];
  if (
    ports.some((port) => !Number.isInteger(port) || port < 1 || port > 65535) ||
    ports[0] === ports[1]
  ) {
    return 'invalidPort';
  }
  return undefined;
}

export function errorMessageKey(code: string): string {
  switch (code) {
    case 'invalidServer':
      return 'frp.invalidServer';
    case 'invalidPort':
      return 'frp.invalidPort';
    case 'invalidToken':
      return 'frp.invalidToken';
    case 'configInvalid':
    case 'invalidConfig':
      return 'frp.invalidConfig';
    case 'installFailed':
      return 'frp.installFailed';
    case 'connectionFailed':
      return 'frp.connectionFailed';
    case 'proxyFailed':
      return 'frp.proxyFailed';
    case 'rollbackFailed':
      return 'frp.rollbackFailed';
    default:
      return 'frp.failed';
  }
}
