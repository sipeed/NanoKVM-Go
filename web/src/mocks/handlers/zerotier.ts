import { http } from 'msw';

import { normalizeNetworkID } from '@/pages/desktop/menu/settings/remote-network/components/zerotier/model.ts';

import { fail, ok, readJsonObject, readString } from '../response.ts';
import type { MockState } from '../state.ts';

export function createZeroTierHandlers(origin: string, state: MockState) {
  function authorize(action: string, challenge: unknown) {
    if (!state.auth.twoFactorEnabled || !state.auth.twoFactorProtection.zerotier) return true;
    const pending = state.auth.zerotierChallenge;
    if (!pending || pending.action !== action || pending.token !== challenge) return false;
    state.auth.zerotierChallenge = null;
    return true;
  }
  return [
    http.get(`${origin}/api/extensions/zerotier/status`, () => {
      const zt = state.zerotier;
      const running = zt.installed && zt.enabled;
      const online = running && zt.online;
      const networks = Object.entries(zt.networks).map(([id, network]) => {
        const connected = online && network.status === 'OK' && network.addresses.length > 0;
        return {
          id,
          name: network.name,
          status: !running
            ? 'STOPPED'
            : !online
              ? 'OFFLINE'
              : network.status === 'OK' && !network.addresses.length
                ? 'WAITING_ADDRESS'
                : network.status,
          addresses: connected ? network.addresses : [],
          accessUrls: connected
            ? network.addresses.map(
                (address) => `https://${address.includes(':') ? `[${address}]` : address}`
              )
            : []
        };
      });
      return ok({
        state: !zt.installed
          ? 'notInstalled'
          : !running
            ? 'stopped'
            : !online
              ? 'offline'
              : !networks.length
                ? 'noNetworks'
                : networks.some((n) => n.status === 'OK')
                  ? 'connected'
                  : networks.some((n) =>
                        ['ACCESS_DENIED', 'REQUESTING_CONFIGURATION', 'WAITING_ADDRESS'].includes(
                          n.status
                        )
                      )
                    ? 'connecting'
                    : 'error',
        busy: false,
        enabled: zt.enabled,
        running,
        online,
        nodeId: zt.nodeId,
        version: zt.installed ? '1.14.1' : '',
        networks,
        error: ''
      });
    }),
    ...(['install', 'uninstall', 'start', 'stop', 'restart', 'join', 'leave'] as const).map(
      (action) =>
        http.post(`${origin}/api/extensions/zerotier/${action}`, async ({ request }) => {
          const body = await readJsonObject(request);
          const id = normalizeNetworkID(readString(body, 'networkId') ?? '');
          if ((action === 'join' || action === 'leave') && !id) return fail(-1, 'invalidNetworkID');
          if (!authorize(`zerotier_${action}`, body.challenge))
            return fail(-3, 'sensitive action verification required');
          const zt = state.zerotier;
          switch (action) {
            case 'install':
              zt.installed = true;
              break;
            case 'uninstall':
              zt.installed = false;
              zt.enabled = false;
              break;
            case 'start':
            case 'restart':
              if (!zt.installed) return fail(-1, 'notInstalled');
              zt.enabled = true;
              zt.nodeId ||= '1a2b3c4d5e';
              break;
            case 'stop':
              if (!zt.installed) return fail(-1, 'notInstalled');
              zt.enabled = false;
              break;
            case 'join':
              if (!zt.installed || !zt.enabled) return fail(-1, 'startRequired');
              zt.networks[id!] ??= { name: '', status: 'ACCESS_DENIED', addresses: [] };
              break;
            case 'leave':
              delete zt.networks[id!];
              break;
          }
          return ok(null);
        })
    )
  ];
}
