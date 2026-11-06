import { http } from 'msw';

import { validateConfig } from '@/pages/desktop/menu/settings/remote-network/components/cloudflare/model.ts';

import { fail, ok, readJsonObject, readString } from '../response.ts';
import type { MockState } from '../state.ts';

export function createCloudflareHandlers(origin: string, state: MockState) {
  function authorize(action: string, challenge: unknown) {
    if (!state.auth.twoFactorEnabled || !state.auth.twoFactorProtection.cloudflare) return true;
    const pending = state.auth.cloudflareChallenge;
    if (!pending || pending.action !== action || pending.token !== challenge) return false;
    state.auth.cloudflareChallenge = null;
    return true;
  }

  return [
    http.get(`${origin}/api/extensions/cloudflare/config`, () =>
      ok({
        mode: state.cloudflare.mode,
        accessUrl: state.cloudflare.accessUrl,
        tokenSet: Boolean(state.cloudflare.token)
      })
    ),
    http.get(`${origin}/api/extensions/cloudflare/status`, () => {
      const cf = state.cloudflare;
      return ok({
        mode: cf.mode,
        state: !cf.installed
          ? 'notInstalled'
          : cf.mode === 'named' && !cf.token
            ? 'notConfigured'
            : !cf.enabled
              ? 'stopped'
              : cf.error
                ? 'error'
                : 'connected',
        busy: false,
        enabled: cf.enabled,
        running: cf.enabled,
        accessUrl:
          cf.mode === 'quick' ? (cf.enabled && !cf.error ? cf.quickUrl : '') : cf.accessUrl,
        originUrl: 'https://127.0.0.1:443',
        httpsAvailable: true,
        version: cf.installed ? '2026.10.0' : '',
        error: cf.enabled ? cf.error : ''
      });
    }),
    http.post(`${origin}/api/extensions/cloudflare/mode`, async ({ request }) => {
      const body = await readJsonObject(request);
      if (body.mode !== 'named' && body.mode !== 'quick') return fail(-1, 'invalidConfig');
      if (!authorize('cloudflare_configure', body.challenge))
        return fail(-3, 'sensitive action verification required');
      if (state.cloudflare.mode !== body.mode) {
        state.cloudflare.enabled = false;
        state.cloudflare.quickUrl = '';
        state.cloudflare.mode = body.mode;
      }
      return ok(null);
    }),
    http.post(`${origin}/api/extensions/cloudflare/config`, async ({ request }) => {
      const body = await readJsonObject(request);
      const config = {
        accessUrl: (readString(body, 'accessUrl') ?? '').trim(),
        token: readString(body, 'token') ?? undefined
      };
      const invalid = validateConfig(config, Boolean(state.cloudflare.token));
      if (invalid) return fail(-1, invalid);
      if (!authorize('cloudflare_configure', body.challenge))
        return fail(-3, 'sensitive action verification required');
      if (!state.cloudflare.installed) return fail(-1, 'notInstalled');
      state.cloudflare.accessUrl = config.accessUrl;
      if (config.token !== undefined) state.cloudflare.token = config.token.trim();
      return ok(null);
    }),
    http.post(`${origin}/api/extensions/cloudflare/clear-token`, async ({ request }) => {
      const body = await readJsonObject(request);
      if (!authorize('cloudflare_configure', body.challenge))
        return fail(-3, 'sensitive action verification required');
      state.cloudflare.enabled = false;
      state.cloudflare.token = '';
      return ok(null);
    }),
    ...(['install', 'uninstall', 'start', 'stop', 'restart'] as const).map((action) =>
      http.post(`${origin}/api/extensions/cloudflare/${action}`, async ({ request }) => {
        const body = await readJsonObject(request);
        if (!authorize(`cloudflare_${action}`, body.challenge))
          return fail(-3, 'sensitive action verification required');
        if (action === 'install') state.cloudflare.installed = true;
        if (action === 'uninstall') {
          state.cloudflare.installed = false;
          state.cloudflare.enabled = false;
        }
        if (action === 'start' || action === 'restart') {
          if (!state.cloudflare.installed) return fail(-1, 'notInstalled');
          if (state.cloudflare.mode === 'named' && !state.cloudflare.token)
            return fail(-1, 'invalidToken');
          if (state.cloudflare.mode === 'quick')
            state.cloudflare.quickUrl = `https://mock-quick-${++state.cloudflare.quickRuns}.trycloudflare.com`;
          state.cloudflare.enabled = true;
        }
        if (action === 'stop') state.cloudflare.enabled = false;
        return ok(null);
      })
    )
  ];
}
