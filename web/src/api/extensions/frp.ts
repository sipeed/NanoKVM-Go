import { http } from '@/lib/http.ts';

export type FRPConfig = {
  serverAddr: string;
  serverPort: number;
  remotePort: number;
  tokenSet: boolean;
};

export type FRPConfigInput = Omit<FRPConfig, 'tokenSet'> & { token?: string };
export type FRPAction = 'install' | 'uninstall' | 'start' | 'stop' | 'restart' | 'clear-token';

export type FRPStatus = {
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
  accessUrl: string;
  error: string;
};

export function getStatus(signal?: AbortSignal) {
  return http.request({ method: 'get', url: '/api/extensions/frp/status', signal });
}

export function getConfig(signal?: AbortSignal) {
  return http.request({ method: 'get', url: '/api/extensions/frp/config', signal });
}

export function setConfig(config: FRPConfigInput, challenge?: string) {
  return http.post('/api/extensions/frp/config', { ...config, challenge }, { timeout: 120_000 });
}

export function runAction(action: FRPAction, challenge?: string) {
  return http.post(`/api/extensions/frp/${action}`, { challenge }, { timeout: 5 * 60_000 });
}
