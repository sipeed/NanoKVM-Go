import type { CloudflareConfigInput, CloudflareStatus } from '@/api/extensions/cloudflare.ts';

import type { RemoteNetworkStatus } from '../../types.ts';

export function toRemoteNetworkStatus(status: CloudflareStatus): RemoteNetworkStatus {
  if (status.busy || status.state === 'installing' || status.state === 'connecting') return 'busy';
  if (status.state === 'connected') return 'on';
  if (status.state === 'error') return 'bad';
  return 'off';
}

export function validateConfig(
  config: CloudflareConfigInput,
  tokenSet = false
): string | undefined {
  const address = config.accessUrl.trim();
  if (address) {
    try {
      const url = new URL(address);
      if (
        address.length > 2048 ||
        /[\s\\]/.test(address) ||
        url.protocol !== 'https:' ||
        url.username ||
        url.password ||
        url.search ||
        url.hash ||
        /[?#]/.test(address) ||
        url.pathname !== '/'
      )
        return 'invalidURL';
    } catch {
      return 'invalidURL';
    }
  }
  if (config.token === undefined) return tokenSet ? undefined : 'invalidToken';
  const value = config.token.trim();
  if (!value || value.length > 4096 || !/^[A-Za-z0-9+/]+={0,2}$/.test(value)) return 'invalidToken';
  try {
    const token = JSON.parse(atob(value));
    if (
      typeof token.a !== 'string' ||
      !token.a ||
      typeof token.s !== 'string' ||
      !atob(token.s) ||
      typeof token.t !== 'string' ||
      !/^[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}$/i.test(token.t) ||
      token.t === '00000000-0000-0000-0000-000000000000'
    )
      return 'invalidToken';
  } catch {
    return 'invalidToken';
  }
  return undefined;
}

export function errorMessageKey(code: string): string {
  switch (code) {
    case 'invalidToken':
    case 'invalidURL':
    case 'httpsRequired':
    case 'installFailed':
    case 'connectionFailed':
    case 'quickConnectionFailed':
    case 'rollbackFailed':
      return `cloudflare.${code}`;
    case 'configInvalid':
    case 'invalidConfig':
      return 'cloudflare.invalidConfig';
    default:
      return 'cloudflare.failed';
  }
}
