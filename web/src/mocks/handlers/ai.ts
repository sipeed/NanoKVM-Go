import { http, HttpResponse } from 'msw';

import type {
  ScreenTimelapseRetentionCandidate,
  ScreenTimelapseRetentionCleanupConfig,
  ScreenTimelapseRetentionConfig,
  ScreenTimelapseRetentionPreview
} from '@/api/ai.ts';

import type { MockHubs } from '../hubs.ts';
import type { MockEnvelope } from '../response.ts';
import {
  fail,
  isRecord,
  ok,
  readBoolean,
  readJsonObject,
  readNumber,
  readString
} from '../response.ts';
import type { MockScreenTimelapseRetentionDay, MockState } from '../state.ts';

const mockScreenTimelapseRetentionToday = '2026-07-20';
const mockScreenTimelapseRetentionMiBBytes = 1024 * 1024;
const mockScreenTimelapseRetentionRunDelayMs = 100;
const mockScreenTimelapseRetentionChallenge = 'mock-sensitive-action';
const mockScreenTimelapseRetentionMaxDays = 3650;
const mockScreenTimelapseRetentionMaxChallengeBytes = 512;

type MockScreenTimelapseRetentionPutResult = {
  config: ScreenTimelapseRetentionConfig;
  changed: boolean;
  cleanupTriggered: boolean;
};

type MockScreenTimelapseRetentionCleanupResult =
  | ScreenTimelapseRetentionPreview
  | { accepted: boolean };

type MockScreenTimelapseRecordingResult = {
  qp: number;
  changed: boolean;
  restarted: boolean;
};

function getServices() {
  return {
    memoryMiB: 2048,
    memoryTier: 'mock',
    services: [
      { id: 'memoryFabric', supported: true, minMemoryMiB: 512 },
      { id: 'screenTimelapse', supported: true, minMemoryMiB: 512 }
    ]
  };
}

function mcpAuthorizationRequired(state: MockState) {
  return (
    state.auth.twoFactorEnabled &&
    state.auth.twoFactorProtection.mcp &&
    state.auth.mcpAuthorizationUntil <= Date.now()
  );
}

function mcpAuthorizationError<T>(state: MockState, data: T) {
  if (!mcpAuthorizationRequired(state)) return null;
  return fail(-3, 'sensitive action verification required', data);
}

function maskMCPAPIKey(key: string) {
  if (!key) return '';
  if (key.length <= 8) return `${key.slice(0, 1)}...${key.slice(-1)}`;
  if (key.length <= 16) return `${key.slice(0, 4)}...${key.slice(-4)}`;
  return `${key.slice(0, 12)}...${key.slice(-6)}`;
}

function getMCPConfig(state: MockState) {
  if (!state.ai.mcpEnabled) return { mcpEnabled: false };
  return {
    mcpEnabled: true,
    mcpApiKeyMasked: maskMCPAPIKey(state.ai.mcpApiKey)
  };
}

function screenTimelapsePlaybackAuthorizationRequired(state: MockState) {
  return (
    state.auth.twoFactorEnabled &&
    state.auth.twoFactorProtection.screenTimelapse &&
    state.auth.screenTimelapseAuthorizationUntil <= Date.now()
  );
}

function screenTimelapsePlaybackAuthorizationError<T>(state: MockState, data: T) {
  if (!screenTimelapsePlaybackAuthorizationRequired(state)) return null;
  return fail(-3, 'sensitive action verification required', data);
}

function hasOnlyProperties(value: Record<string, unknown>, allowed: string[]): boolean {
  return Object.keys(value).every((key) => allowed.includes(key));
}

function hasProperty(value: Record<string, unknown>, key: string): boolean {
  return Object.prototype.hasOwnProperty.call(value, key);
}

function isNonNegativeSafeInteger(value: unknown): value is number {
  return typeof value === 'number' && Number.isSafeInteger(value) && value >= 0;
}

function readScreenTimelapseRetentionChallenge(body: Record<string, unknown>): string | null {
  const value = body.challenge;
  if (value === undefined || value === null) return '';
  if (
    typeof value !== 'string' ||
    value.trim().length > mockScreenTimelapseRetentionMaxChallengeBytes
  ) {
    return null;
  }
  return value;
}

function readScreenTimelapseRetentionCleanup(
  value: unknown
): ScreenTimelapseRetentionCleanupConfig | null {
  if (
    !isRecord(value) ||
    !hasOnlyProperties(value, ['enabled', 'retentionDays', 'maxStorageMiB']) ||
    typeof value.enabled !== 'boolean' ||
    !isNonNegativeSafeInteger(value.retentionDays) ||
    value.retentionDays > mockScreenTimelapseRetentionMaxDays ||
    !isNonNegativeSafeInteger(value.maxStorageMiB)
  ) {
    return null;
  }
  if (value.enabled && value.retentionDays === 0 && value.maxStorageMiB === 0) {
    return null;
  }
  return {
    enabled: value.enabled,
    retentionDays: value.retentionDays,
    maxStorageMiB: value.maxStorageMiB
  };
}

function sameScreenTimelapseRetentionCleanup(
  left: ScreenTimelapseRetentionCleanupConfig,
  right: ScreenTimelapseRetentionCleanupConfig
): boolean {
  return (
    left.enabled === right.enabled &&
    left.retentionDays === right.retentionDays &&
    left.maxStorageMiB === right.maxStorageMiB
  );
}

function screenTimelapseRetentionChallengeRequired(state: MockState): boolean {
  return state.auth.twoFactorEnabled && state.auth.twoFactorProtection.screenTimelapse;
}

function hasScreenTimelapseRetentionChallenge(state: MockState, challenge: string): boolean {
  return (
    !screenTimelapseRetentionChallengeRequired(state) ||
    challenge === mockScreenTimelapseRetentionChallenge
  );
}

function mockScreenTimelapseRetentionStorageBytes(maxStorageMiB: number): number {
  if (maxStorageMiB === 0) return 0;
  if (maxStorageMiB > Math.floor(Number.MAX_SAFE_INTEGER / mockScreenTimelapseRetentionMiBBytes)) {
    return Number.MAX_SAFE_INTEGER;
  }
  return maxStorageMiB * mockScreenTimelapseRetentionMiBBytes;
}

function mockScreenTimelapseRetentionCutoff(retentionDays: number): string {
  const cutoff = new Date(`${mockScreenTimelapseRetentionToday}T00:00:00.000Z`);
  cutoff.setUTCDate(cutoff.getUTCDate() - (retentionDays - 1));
  return cutoff.toISOString().slice(0, 10);
}

function planMockScreenTimelapseRetention(
  days: MockScreenTimelapseRetentionDay[],
  cleanup: ScreenTimelapseRetentionCleanupConfig
): ScreenTimelapseRetentionPreview {
  const sortedDays = [...days].sort((left, right) => left.date.localeCompare(right.date));
  const beforeBytes = sortedDays.reduce((total, day) => total + day.bytes, 0);
  const candidates = new Map<string, ScreenTimelapseRetentionCandidate>();
  let projectedAfterBytes = beforeBytes;

  const addCandidate = (day: MockScreenTimelapseRetentionDay, reason: string) => {
    const current = candidates.get(day.date);
    if (current) {
      if (!current.reasons.includes(reason)) current.reasons.push(reason);
      return;
    }
    candidates.set(day.date, { date: day.date, bytes: day.bytes, reasons: [reason] });
    projectedAfterBytes = Math.max(0, projectedAfterBytes - day.bytes);
  };

  if (cleanup.enabled) {
    if (cleanup.retentionDays > 0) {
      const cutoff = mockScreenTimelapseRetentionCutoff(cleanup.retentionDays);
      sortedDays.forEach((day) => {
        if (day.date !== mockScreenTimelapseRetentionToday && day.date < cutoff) {
          addCandidate(day, 'expired');
        }
      });
    }

    if (cleanup.maxStorageMiB > 0) {
      const quotaBytes = mockScreenTimelapseRetentionStorageBytes(cleanup.maxStorageMiB);
      for (const day of sortedDays) {
        if (projectedAfterBytes <= quotaBytes) break;
        if (day.date === mockScreenTimelapseRetentionToday) continue;
        addCandidate(day, 'quota');
      }
    }
  }

  const selected = [...candidates.values()];
  const wouldDeleteBytes = selected.reduce((total, candidate) => total + candidate.bytes, 0);
  const quotaBytes = mockScreenTimelapseRetentionStorageBytes(cleanup.maxStorageMiB);

  return {
    dryRun: true,
    beforeBytes,
    projectedAfterBytes,
    overLimitAfterCleanup:
      cleanup.enabled && cleanup.maxStorageMiB > 0 && projectedAfterBytes > quotaBytes,
    wouldDeleteDays: selected.length,
    wouldDeleteBytes,
    candidates: selected,
    skipped: cleanup.enabled
      ? sortedDays
          .filter((day) => day.date === mockScreenTimelapseRetentionToday)
          .map((day) => ({ date: day.date, bytes: day.bytes, reason: 'today' }))
      : [],
    warnings: []
  };
}

function screenTimelapseRetentionPutResult(
  state: MockState,
  changed: boolean,
  cleanupTriggered: boolean
): MockScreenTimelapseRetentionPutResult {
  return {
    config: state.ai.screenTimelapseRetention.config,
    changed,
    cleanupTriggered
  };
}

export function createAIHandlers(origin: string, state: MockState, hubs: MockHubs) {
  function triggerMockScreenTimelapseRetentionCleanup(trigger: 'config_saved' | 'manual') {
    const retention = state.ai.screenTimelapseRetention;
    if (retention.status.running) return;

    retention.status.running = true;
    retention.status.blockedReason = '';
    retention.status.lastError = '';
    retention.status.lastTrigger = trigger;

    setTimeout(() => {
      const plan = planMockScreenTimelapseRetention(retention.days, retention.config.cleanup);
      const deletedDates = new Set(plan.candidates.map((candidate) => candidate.date));
      retention.days = retention.days.filter((day) => !deletedDates.has(day.date));
      retention.status.running = false;
      retention.status.scannedAt = '2026-07-20T03:10:00Z';
      retention.status.usageBytes = plan.projectedAfterBytes;
      retention.status.managedDays = retention.days.length;
      retention.status.overLimit = plan.overLimitAfterCleanup;
      retention.status.blockedReason = '';
      retention.status.lastCleanupAt = '2026-07-20T03:10:00Z';
      retention.status.lastDeletedDays = plan.wouldDeleteDays;
      retention.status.lastDeletedBytes = plan.wouldDeleteBytes;
      retention.status.lastError = '';
    }, mockScreenTimelapseRetentionRunDelayMs);
  }

  return [
    http.get(`${origin}/api/ai/services`, () => ok(getServices())),
    http.get(`${origin}/api/ai/config`, () => ok(getMCPConfig(state))),
    http.post(`${origin}/api/ai/config`, async ({ request }) => {
      const authorizationError = mcpAuthorizationError(state, getMCPConfig(state));
      if (authorizationError) return authorizationError;
      const enabled = readBoolean(await readJsonObject(request), 'mcpEnabled');
      if (enabled !== null) {
        state.ai.mcpEnabled = enabled;
        hubs.publishAIService(state, 'mcp', enabled);
      }
      return ok(getMCPConfig(state));
    }),
    http.post(`${origin}/api/ai/mcp/key`, () => {
      const authorizationError = mcpAuthorizationError(state, { mcpApiKey: '' });
      if (authorizationError) return authorizationError;
      return ok({ mcpApiKey: state.ai.mcpApiKey });
    }),
    http.post(`${origin}/api/ai/mcp/key/copy`, () => {
      const authorizationError = mcpAuthorizationError(state, { mcpApiKey: '' });
      if (authorizationError) return authorizationError;
      return ok({ mcpApiKey: state.ai.mcpApiKey });
    }),
    http.post(`${origin}/api/ai/mcp/key/regenerate`, () => {
      const authorizationError = mcpAuthorizationError(state, getMCPConfig(state));
      if (authorizationError) return authorizationError;
      state.ai.mcpApiKey = 'nag_mcp_mock-development-key-rotated-not-a-secret-0';
      return ok(getMCPConfig(state));
    }),
    http.get(`${origin}/api/ai/memory-fabric`, () => ok(state.ai.memoryFabric)),
    http.post(`${origin}/api/ai/memory-fabric`, async ({ request }) => {
      const enabled = readBoolean(await readJsonObject(request), 'enabled');
      if (enabled !== null) {
        state.ai.memoryFabric.enabled = enabled;
        state.ai.memoryFabric.running = enabled;
        hubs.publishAIService(state, 'memoryFabric', enabled);
      }
      return ok(state.ai.memoryFabric);
    }),
    http.get(`${origin}/api/ai/memory-fabric/config`, () =>
      ok({ config: state.ai.memoryFabricConfig })
    ),
    http.put(`${origin}/api/ai/memory-fabric/config`, async ({ request }) => {
      const body = await readJsonObject(request);
      const models = isRecord(body.models) ? body.models : null;
      const defaultModel = models && isRecord(models.default) ? models.default : null;
      const current = state.ai.memoryFabricConfig.models.default;
      const model = defaultModel ? readString(defaultModel, 'model') : null;
      const baseUrl = defaultModel ? readString(defaultModel, 'baseUrl') : null;
      const maxTokens = defaultModel ? readNumber(defaultModel, 'maxTokens') : null;
      const timeoutSeconds = defaultModel ? readNumber(defaultModel, 'timeoutSeconds') : null;
      const apiKey = defaultModel ? readString(defaultModel, 'apiKey') : null;

      if (model) current.model = model;
      if (baseUrl) current.baseUrl = baseUrl;
      if (maxTokens !== null) current.maxTokens = maxTokens;
      if (timeoutSeconds !== null) current.timeoutSeconds = timeoutSeconds;
      if (apiKey !== null) current.apiKeyConfigured = apiKey !== '';
      state.ai.memoryFabricConfig.revision = `mock-revision-${state.ai.serviceRevision + 1}`;

      return ok({
        config: state.ai.memoryFabricConfig,
        changed: true,
        restarted: state.ai.memoryFabric.enabled
      });
    }),
    http.get(`${origin}/api/ai/screen-timelapse/state`, () =>
      ok({ enabled: state.ai.screenTimelapseEnabled })
    ),
    http.post<never, never, MockEnvelope<{ enabled: boolean } | null>>(
      `${origin}/api/ai/screen-timelapse/state`,
      async ({ request }) => {
        const body = await readJsonObject(request);
        const enabled = readBoolean(body, 'enabled');
        const challenge = readString(body, 'challenge');
        if (enabled === null) {
          return fail(-1, 'invalid arguments', null);
        }
        if (
          enabled &&
          state.auth.twoFactorEnabled &&
          state.auth.twoFactorProtection.screenTimelapse
        ) {
          if (
            !state.auth.screenTimelapseChallenge ||
            challenge !== state.auth.screenTimelapseChallenge
          ) {
            return fail(-3, 'sensitive action verification required', null);
          }
          state.auth.screenTimelapseChallenge = null;
        }

        state.ai.screenTimelapseEnabled = enabled;
        hubs.publishAIService(state, 'screenTimelapse', enabled);
        return ok({ enabled: state.ai.screenTimelapseEnabled });
      }
    ),
    http.get(`${origin}/api/ai/screen-timelapse/recording`, () =>
      ok(state.ai.screenTimelapseRecording)
    ),
    http.put<never, never, MockEnvelope<MockScreenTimelapseRecordingResult>>(
      `${origin}/api/ai/screen-timelapse/recording`,
      async ({ request }) => {
        const body = await readJsonObject(request);
        const qp = readNumber(body, 'qp');
        if (
          !hasOnlyProperties(body, ['qp']) ||
          qp === null ||
          !Number.isInteger(qp) ||
          qp < 36 ||
          qp > 51
        ) {
          return fail(-2, 'invalid screen timelapse QP');
        }

        const changed = state.ai.screenTimelapseRecording.qp !== qp;
        state.ai.screenTimelapseRecording = { qp };
        return ok({
          qp,
          changed,
          restarted: changed && state.ai.screenTimelapseEnabled
        });
      }
    ),
    http.post(`${origin}/api/ai/screen-timelapse/access`, async ({ request }) => {
      const authorizationError = screenTimelapsePlaybackAuthorizationError(state, {});
      if (!authorizationError) return ok({});

      const challenge = readString(await readJsonObject(request), 'challenge');
      if (
        !state.auth.screenTimelapsePlaybackChallenge ||
        challenge !== state.auth.screenTimelapsePlaybackChallenge
      ) {
        return authorizationError;
      }

      state.auth.screenTimelapsePlaybackChallenge = null;
      state.auth.screenTimelapseAuthorizationUntil = Date.now() + 5 * 60 * 1000;
      return ok({});
    }),
    http.get(`${origin}/api/ai/screen-timelapse/days`, () => {
      const authorizationError = screenTimelapsePlaybackAuthorizationError(state, {
        days: []
      });
      if (authorizationError) return authorizationError;
      return ok({ days: state.ai.screenTimelapseDays });
    }),
    http.get(`${origin}/api/ai/screen-timelapse/retention`, () =>
      ok({
        config: state.ai.screenTimelapseRetention.config,
        status: state.ai.screenTimelapseRetention.status
      })
    ),
    http.put(`${origin}/api/ai/screen-timelapse/retention`, async ({ request }) => {
      const body = await readJsonObject(request);
      const revision = body.revision;
      const cleanup = readScreenTimelapseRetentionCleanup(body.cleanup);
      const challenge = readScreenTimelapseRetentionChallenge(body);
      if (
        !hasOnlyProperties(body, ['revision', 'cleanup', 'challenge']) ||
        !isNonNegativeSafeInteger(revision) ||
        cleanup === null ||
        challenge === null
      ) {
        return fail<MockScreenTimelapseRetentionPutResult>(-1, 'invalid arguments');
      }

      const retention = state.ai.screenTimelapseRetention;
      if (retention.config.revision !== revision) {
        return fail<MockScreenTimelapseRetentionPutResult>(
          -3,
          'revision conflict',
          screenTimelapseRetentionPutResult(state, false, false)
        );
      }
      if (!hasScreenTimelapseRetentionChallenge(state, challenge)) {
        return fail<MockScreenTimelapseRetentionPutResult>(
          -4,
          'sensitive action verification required'
        );
      }
      if (sameScreenTimelapseRetentionCleanup(retention.config.cleanup, cleanup)) {
        return ok<MockScreenTimelapseRetentionPutResult>(
          screenTimelapseRetentionPutResult(state, false, false)
        );
      }

      retention.config = {
        revision: retention.config.revision + 1,
        cleanup
      };
      triggerMockScreenTimelapseRetentionCleanup('config_saved');
      return ok<MockScreenTimelapseRetentionPutResult>(
        screenTimelapseRetentionPutResult(state, true, true)
      );
    }),
    http.post(`${origin}/api/ai/screen-timelapse/cleanup`, async ({ request }) => {
      const body = await readJsonObject(request);
      const dryRun = body.dryRun;
      const challenge = readScreenTimelapseRetentionChallenge(body);
      if (
        !hasOnlyProperties(body, ['dryRun', 'config', 'challenge']) ||
        typeof dryRun !== 'boolean' ||
        challenge === null
      ) {
        return fail<MockScreenTimelapseRetentionCleanupResult>(-1, 'invalid arguments');
      }

      const retention = state.ai.screenTimelapseRetention;
      const candidateConfig = body.config;
      if (dryRun) {
        const cleanup =
          candidateConfig === undefined || candidateConfig === null
            ? retention.config.cleanup
            : readScreenTimelapseRetentionCleanup(candidateConfig);
        if (cleanup === null) {
          return fail<MockScreenTimelapseRetentionCleanupResult>(-1, 'invalid arguments');
        }
        return ok<MockScreenTimelapseRetentionCleanupResult>(
          planMockScreenTimelapseRetention(retention.days, cleanup)
        );
      }

      if (hasProperty(body, 'config') && candidateConfig !== null) {
        return fail<MockScreenTimelapseRetentionCleanupResult>(-1, 'invalid arguments');
      }
      if (!hasScreenTimelapseRetentionChallenge(state, challenge)) {
        return fail<MockScreenTimelapseRetentionCleanupResult>(
          -4,
          'sensitive action verification required'
        );
      }

      triggerMockScreenTimelapseRetentionCleanup('manual');
      return ok<MockScreenTimelapseRetentionCleanupResult>({ accepted: true });
    }),
    http.get(`${origin}/api/ai/screen-timelapse/frames`, ({ request }) => {
      const authorizationError = screenTimelapsePlaybackAuthorizationError(state, {
        date: '',
        durationMs: 0,
        frameCount: 0,
        frames: []
      });
      if (authorizationError) return authorizationError;
      const date = new URL(request.url).searchParams.get('date') || '2026-01-02';
      return ok({
        date,
        durationMs: 300_000,
        frameCount: 5,
        frames: [
          { pts: 0, currentTime: 0, capturedAt: '2026-01-02T03:00:00Z' },
          { pts: 60_000, currentTime: 60, capturedAt: '2026-01-02T03:01:00Z' }
        ]
      });
    }),
    http.get(`${origin}/api/ai/screen-timelapse/video`, () => {
      if (screenTimelapsePlaybackAuthorizationRequired(state)) {
        return new HttpResponse(null, { status: 403 });
      }
      return new HttpResponse('mock-screen-timelapse-video', {
        headers: { 'content-type': 'video/mp2t' }
      });
    }),
    http.delete(`${origin}/api/ai/screen-timelapse/day`, ({ request }) => {
      const authorizationError = screenTimelapsePlaybackAuthorizationError(state, {
        date: '',
        deletedBytes: 0
      });
      if (authorizationError) return authorizationError;
      const date = new URL(request.url).searchParams.get('date') || '';
      const dayIndex = state.ai.screenTimelapseDays.findIndex((day) => day.date === date);
      // Mirrors the backend: deleting a missing day succeeds with 0 bytes.
      const deletedBytes = dayIndex >= 0 ? state.ai.screenTimelapseDays[dayIndex].filesize : 0;
      if (dayIndex >= 0) state.ai.screenTimelapseDays.splice(dayIndex, 1);
      return ok({ date, deletedBytes });
    }),
    http.get(`${origin}/api/ai/screen-timelapse/search`, ({ request }) => {
      const query = new URL(request.url).searchParams.get('query') || '';
      const authorizationError = screenTimelapsePlaybackAuthorizationError(state, {
        query,
        count: 0,
        results: [],
        skippedMissingFileCount: 0
      });
      if (authorizationError) return authorizationError;
      return ok({
        query,
        count: query ? 2 : 0,
        results: query
          ? [
              {
                capturedAt: '2026-01-02T03:01:00Z',
                date: '2026-01-02',
                pts: 60_000,
                currentTime: 60,
                snippet: 'Mocked screen timelapse result',
                available: true
              },
              {
                capturedAt: '2025-12-28T04:15:00Z',
                date: '2025-12-28',
                snippet: 'Mocked OCR text remains searchable after cleanup',
                available: false,
                unavailableReason: 'missing_video'
              }
            ]
          : [],
        skippedMissingFileCount: 0
      });
    })
  ];
}
