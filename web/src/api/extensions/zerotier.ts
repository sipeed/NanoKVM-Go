import { http } from '@/lib/http.ts';

export type ZeroTierAction =
  | 'install'
  | 'uninstall'
  | 'start'
  | 'stop'
  | 'restart'
  | 'join'
  | 'leave';
export type ZeroTierNetwork = {
  id: string;
  name: string;
  status: string;
  addresses: string[];
  accessUrls: string[];
};
export type ZeroTierStatus = {
  state:
    | 'notInstalled'
    | 'installing'
    | 'stopped'
    | 'offline'
    | 'noNetworks'
    | 'connecting'
    | 'connected'
    | 'error';
  busy: boolean;
  enabled: boolean;
  running: boolean;
  online: boolean;
  nodeId: string;
  version: string;
  networks: ZeroTierNetwork[];
  error: string;
};

export function getStatus(signal?: AbortSignal) {
  return http.request({ method: 'get', url: '/api/extensions/zerotier/status', signal });
}

export function runAction(action: ZeroTierAction, challenge?: string, networkId?: string) {
  return http.post(
    `/api/extensions/zerotier/${action}`,
    { challenge, networkId },
    { timeout: 5 * 60_000 }
  );
}
