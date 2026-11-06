import { http } from 'msw';

import { validateConfig } from '@/pages/desktop/menu/settings/remote-network/components/frp/model.ts';

import { fail, ok, readJsonObject, readNumber, readString } from '../response.ts';
import type { MockState } from '../state.ts';

export function createFRPHandlers(origin: string, state: MockState) {
  function authorize(action: string, challenge: unknown) {
    if (!state.auth.twoFactorEnabled || !state.auth.twoFactorProtection.frp) return true;
    const pending = state.auth.frpChallenge;
    if (!pending || pending.action !== action || pending.token !== challenge) return false;
    state.auth.frpChallenge = null;
    return true;
  }

  return [
    http.get(`${origin}/api/extensions/frp/config`, () =>
      ok({
        serverAddr: state.frp.serverAddr,
        serverPort: state.frp.serverPort,
        remotePort: state.frp.remotePort,
        tokenSet: Boolean(state.frp.token)
      })
    ),
    http.get(`${origin}/api/extensions/frp/status`, () => {
      const frp = state.frp;
      return ok({
        state: !frp.installed
          ? 'notInstalled'
          : !frp.serverAddr
            ? 'notConfigured'
            : !frp.enabled
              ? 'stopped'
              : frp.error
                ? 'error'
                : 'connected',
        busy: false,
        enabled: frp.enabled,
        accessUrl: frp.serverAddr
          ? `https://${frp.serverAddr.includes(':') ? `[${frp.serverAddr}]` : frp.serverAddr}:${frp.remotePort}`
          : '',
        error: frp.enabled ? frp.error : ''
      });
    }),
    http.post(`${origin}/api/extensions/frp/config`, async ({ request }) => {
      const body = await readJsonObject(request);
      const config = {
        serverAddr: (readString(body, 'serverAddr') ?? '').trim(),
        serverPort: readNumber(body, 'serverPort') ?? 0,
        remotePort: readNumber(body, 'remotePort') ?? 0
      };
      const invalid = validateConfig(config);
      if (invalid) return fail(-1, invalid);
      if (!authorize('frp_configure', body.challenge))
        return fail(-3, 'sensitive action verification required');
      if (!state.frp.installed) return fail(-1, 'notInstalled');
      Object.assign(state.frp, config);
      const token = readString(body, 'token');
      if (token !== null) state.frp.token = token;
      return ok(null);
    }),
    http.post(`${origin}/api/extensions/frp/clear-token`, async ({ request }) => {
      const body = await readJsonObject(request);
      if (!authorize('frp_configure', body.challenge))
        return fail(-3, 'sensitive action verification required');
      state.frp.enabled = false;
      state.frp.token = '';
      return ok(null);
    }),
    ...(['install', 'uninstall', 'start', 'stop', 'restart'] as const).map((action) =>
      http.post(`${origin}/api/extensions/frp/${action}`, async ({ request }) => {
        const body = await readJsonObject(request);
        if (!authorize(`frp_${action}`, body.challenge))
          return fail(-3, 'sensitive action verification required');
        if (action === 'install') state.frp.installed = true;
        if (action === 'uninstall') {
          state.frp.installed = false;
          state.frp.enabled = false;
        }
        if (action === 'start' || action === 'restart') {
          if (!state.frp.installed) return fail(-1, 'notInstalled');
          if (!state.frp.serverAddr) return fail(-1, 'configInvalid');
          state.frp.enabled = true;
        }
        if (action === 'stop') state.frp.enabled = false;
        return ok(null);
      })
    )
  ];
}
