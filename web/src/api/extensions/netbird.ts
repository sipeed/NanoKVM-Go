import { http } from '@/lib/http.ts';

export type NetBirdAction =
  | 'install'
  | 'uninstall'
  | 'start'
  | 'stop'
  | 'restart'
  | 'login'
  | 'login_cancel'
  | 'logout';

export type NetBirdStatus = {
  state:
    | 'notInstalled'
    | 'installing'
    | 'stopped'
    | 'notLogin'
    | 'authorizing'
    | 'connecting'
    | 'connected'
    | 'offline'
    | 'error';
  busy: boolean;
  enabled: boolean;
  running: boolean;
  version: string;
  name: string;
  ip: string;
  accessUrl: string;
  authUrl: string;
  userCode: string;
  error: string;
};

export function getStatus(signal?: AbortSignal) {
  return http.request({ method: 'get', url: '/api/extensions/netbird/status', signal });
}

export function runAction(action: NetBirdAction, challenge?: string) {
  const path = action === 'login_cancel' ? 'login/cancel' : action;
  return http.post(`/api/extensions/netbird/${path}`, { challenge }, { timeout: 5 * 60_000 });
}
