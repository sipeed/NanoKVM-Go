import { http } from '@/lib/http.ts';

export type CloudflareMode = 'named' | 'quick';
export type CloudflareConfig = { mode: CloudflareMode; accessUrl: string; tokenSet: boolean };
export type CloudflareConfigInput = { accessUrl: string; token?: string };
export type CloudflareAction =
  | 'install'
  | 'uninstall'
  | 'start'
  | 'stop'
  | 'restart'
  | 'clear-token';
export type CloudflareStatus = {
  mode: CloudflareMode;
  state:
    | 'notInstalled'
    | 'installing'
    | 'notConfigured'
    | 'stopped'
    | 'connecting'
    | 'connected'
    | 'error';
  busy: boolean;
  enabled: boolean;
  running: boolean;
  httpsAvailable: boolean;
  originUrl: string;
  accessUrl: string;
  version: string;
  error: string;
};

export function getStatus(signal?: AbortSignal) {
  return http.request({ method: 'get', url: '/api/extensions/cloudflare/status', signal });
}

export function getConfig(signal?: AbortSignal) {
  return http.request({ method: 'get', url: '/api/extensions/cloudflare/config', signal });
}

export function setConfig(config: CloudflareConfigInput, challenge?: string) {
  return http.post(
    '/api/extensions/cloudflare/config',
    { ...config, challenge },
    { timeout: 120_000 }
  );
}

export function setMode(mode: CloudflareMode, challenge?: string) {
  return http.post('/api/extensions/cloudflare/mode', { mode, challenge }, { timeout: 120_000 });
}

export function runAction(action: CloudflareAction, challenge?: string) {
  return http.post(`/api/extensions/cloudflare/${action}`, { challenge }, { timeout: 5 * 60_000 });
}
