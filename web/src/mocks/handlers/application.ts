import { http } from 'msw';

import type { ApplicationUpdateStatus } from '@/api/application.ts';

import type { MockHubs } from '../hubs.ts';
import { fail, ok, readBoolean, readJsonObject, readString } from '../response.ts';
import type { MockState } from '../state.ts';

export function createApplicationHandlers(origin: string, state: MockState, hubs: MockHubs) {
  return [
    http.get(`${origin}/api/application/version`, () =>
      ok({ current: state.application.current, latest: state.application.latest })
    ),
    http.get(`${origin}/api/application/update/server`, () =>
      ok({ server: state.application.updateServer })
    ),
    http.post(`${origin}/api/application/update/server`, async ({ request }) => {
      const server = readString(await readJsonObject(request), 'server');
      if (server === null) {
        return fail(-1, 'invalid arguments', { server: state.application.updateServer });
      }
      state.application.updateServer = server.trim().replace(/\/+$/, '');
      return ok({ server: state.application.updateServer });
    }),
    http.get(`${origin}/api/application/region`, () => ok({ region: state.application.region })),
    http.post(`${origin}/api/application/features/sync`, () => ok(null)),
    http.post(`${origin}/api/application/region`, async ({ request }) => {
      const region = readString(await readJsonObject(request), 'region');
      if (region !== 'cn' && region !== 'global') {
        return fail(-1, 'invalid parameters', {
          region: state.application.region,
          cloudCleanupFailed: false
        });
      }

      state.application.region = region;
      return ok({ region, cloudCleanupFailed: false });
    }),
    http.post(`${origin}/api/application/update`, () => {
      if (
        state.application.update.phase !== 'idle' &&
        state.application.update.phase !== 'succeeded' &&
        state.application.update.phase !== 'failed'
      ) {
        return fail(-1, 'update busy', state.application.update);
      }
      hubs.publishUpdate(state, 'online');
      return ok(state.application.update);
    }),
    http.post(`${origin}/api/application/update/offline`, async ({ request }) => {
      if (
        state.application.update.phase !== 'idle' &&
        state.application.update.phase !== 'succeeded' &&
        state.application.update.phase !== 'failed'
      ) {
        return fail(-1, 'update busy', state.application.update);
      }
      const data = await request.formData();
      const entries = [...data.entries()];
      const file = entries[0]?.[1];
      if (
        entries.length !== 1 ||
        entries[0][0] !== 'file' ||
        typeof file === 'string' ||
        typeof file?.name !== 'string'
      ) {
        return fail<ApplicationUpdateStatus | null>(-1, 'one update file is required', null);
      }
      hubs.publishUpdate(state, 'offline');
      return ok(state.application.update);
    }),
    http.get(`${origin}/api/application/update/status`, () => ok(state.application.update)),
    http.post(`${origin}/api/application/update/status/acknowledge`, async ({ request }) => {
      const operationId = readString(await readJsonObject(request), 'operationId');
      const current = state.application.update;
      if (!operationId) return fail(-1, 'invalid arguments', current);
      if (current.phase === 'idle') return ok(current);
      if (
        current.operationId !== operationId ||
        (current.phase !== 'succeeded' && current.phase !== 'failed')
      ) {
        return fail(-2, 'update status cannot be acknowledged', current);
      }
      state.application.update = {
        operationId: '',
        source: '',
        version: '',
        phase: 'idle',
        progress: 0,
        errorCode: '',
        startedAt: '',
        updatedAt: ''
      };
      return ok(state.application.update);
    }),
    http.get(`${origin}/api/application/signature-verification`, () =>
      ok({ enabled: state.application.signatureVerificationEnabled })
    ),
    http.post(`${origin}/api/application/signature-verification`, async ({ request }) => {
      const body = await readJsonObject(request);
      const enabled = readBoolean(body, 'enabled');
      const challenge = readString(body, 'challenge');
      if (enabled === null) {
        return fail(-1, 'invalid parameters', {
          enabled: state.application.signatureVerificationEnabled
        });
      }
      if (challenge === null || challenge !== state.auth.updateSignatureVerificationChallenge) {
        return fail(-4, 'sensitive action authorization required', {
          enabled: state.application.signatureVerificationEnabled
        });
      }

      state.auth.updateSignatureVerificationChallenge = null;
      state.application.signatureVerificationEnabled = enabled;
      return ok({ enabled });
    }),
    http.get(`${origin}/api/application/preview`, () =>
      ok({ enabled: state.application.previewEnabled })
    ),
    http.post(`${origin}/api/application/preview`, async ({ request }) => {
      const enabled = readBoolean(await readJsonObject(request), 'enable');
      if (enabled !== null) state.application.previewEnabled = enabled;
      return ok({ enabled: state.application.previewEnabled });
    })
  ];
}
