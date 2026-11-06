import { delay, http } from 'msw';

import { fail, ok, readJsonObject } from '../response.ts';
import type { MockState } from '../state.ts';

export function createNetBirdHandlers(origin: string, state: MockState) {
  return [
    http.get(`${origin}/api/extensions/netbird/status`, () => {
      const nb = state.netbird;
      if (
        nb.installed &&
        nb.enabled &&
        nb.state === 'authorizing' &&
        nb.authorizationPollsRemaining > 0
      ) {
        nb.authorizationPollsRemaining--;
        if (nb.authorizationPollsRemaining === 0) {
          nb.authorized = true;
          nb.state = 'connected';
        }
      }
      const running = nb.installed && nb.enabled;
      const connected = running && nb.state === 'connected';
      const authorizing = running && nb.state === 'authorizing';
      return ok({
        state:
          nb.busy && !nb.installed
            ? 'installing'
            : !nb.installed
              ? 'notInstalled'
              : !running
                ? 'stopped'
                : nb.state,
        busy: nb.busy,
        enabled: nb.enabled,
        running,
        version: nb.installed ? '0.80.0' : '',
        name: connected ? 'NanoKVM.netbird.cloud' : '',
        ip: connected ? '100.91.0.25' : '',
        accessUrl: connected ? 'https://100.91.0.25' : '',
        authUrl: authorizing ? 'https://app.netbird.io/' : '',
        userCode: authorizing ? 'MOCK-CODE' : '',
        error: nb.error
      });
    }),
    ...(
      [
        'install',
        'uninstall',
        'start',
        'stop',
        'restart',
        'login',
        'login/cancel',
        'logout'
      ] as const
    ).map((action) =>
      http.post(`${origin}/api/extensions/netbird/${action}`, async ({ request }) => {
        const body = await readJsonObject(request);
        const actionName = `netbird_${action.replace('/', '_')}`;
        if (state.auth.twoFactorEnabled && state.auth.twoFactorProtection.netbird) {
          const challenge = state.auth.netbirdChallenge;
          if (!challenge || challenge.action !== actionName || challenge.token !== body.challenge) {
            return fail(-3, 'sensitive action verification required');
          }
          state.auth.netbirdChallenge = null;
        }
        const nb = state.netbird;
        if (nb.busy) return fail(-1, 'busy');
        if (action !== 'install' && action !== 'uninstall' && !nb.installed)
          return fail(-1, 'notInstalled');
        if (action === 'login' && !nb.enabled) return fail(-1, 'startRequired');
        nb.error = '';
        switch (action) {
          case 'install':
            nb.busy = true;
            await delay(300);
            nb.installed = true;
            nb.busy = false;
            break;
          case 'uninstall':
            nb.installed = false;
            nb.enabled = false;
            nb.state = nb.authorized ? 'connected' : 'notLogin';
            break;
          case 'start':
            if (nb.state === 'authorizing') return fail(-1, 'busy');
            nb.enabled = true;
            nb.state = nb.authorized ? 'connected' : 'notLogin';
            break;
          case 'restart':
            nb.enabled = true;
            nb.state = nb.authorized ? 'connected' : 'notLogin';
            break;
          case 'stop':
            nb.enabled = false;
            nb.state = nb.authorized ? 'connected' : 'notLogin';
            break;
          case 'login':
            if (nb.state === 'authorizing') break;
            nb.state = nb.authorized ? 'connected' : 'authorizing';
            nb.authorizationPollsRemaining = 3;
            break;
          case 'login/cancel':
            if (nb.state === 'authorizing') nb.state = 'notLogin';
            nb.authorizationPollsRemaining = 0;
            break;
          case 'logout':
            nb.authorized = false;
            nb.enabled = false;
            nb.state = 'notLogin';
            break;
        }
        return ok(null);
      })
    )
  ];
}
