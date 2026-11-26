import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { after, before, test } from 'node:test';
import { createServer } from 'vite';

const webRoot = path.resolve(import.meta.dirname, '..');
let vite;
let model;
let warning;

before(async () => {
  vite = await createServer({
    root: webRoot,
    appType: 'custom',
    logLevel: 'error',
    server: { middlewareMode: true, hmr: false, ws: false }
  });
  [model, warning] = await Promise.all([
    vite.ssrLoadModule('/src/pages/desktop/menu/settings/ai/screen-timelapse-retention-model.ts'),
    vite.ssrLoadModule('/src/lib/screen-timelapse-retention-warning.ts')
  ]);
});

after(async () => {
  await vite.close();
});

test('retention conversion preserves whole MiB values through the GiB form', () => {
  assert.equal(model.screenTimelapseRetentionMiBToGiB(1536), 1.5);
  assert.equal(model.screenTimelapseRetentionGiBToMiB(1.5), 1536);
  assert.equal(model.screenTimelapseRetentionGiBToMiB(0.0004), 0);
  assert.equal(model.screenTimelapseRetentionGiBToMiB(-1), 0);
});

test('retention form derives enabled from its two limit fields', () => {
  assert.deepEqual(model.screenTimelapseRetentionFormValuesToCleanup({}), {
    enabled: false,
    retentionDays: 0,
    maxStorageMiB: 0
  });
  assert.deepEqual(model.screenTimelapseRetentionFormValuesToCleanup({ retentionDays: 30 }), {
    enabled: true,
    retentionDays: 30,
    maxStorageMiB: 0
  });
  assert.deepEqual(model.screenTimelapseRetentionFormValuesToCleanup({ maxStorageGiB: 1.5 }), {
    enabled: true,
    retentionDays: 0,
    maxStorageMiB: 1536
  });
  assert.deepEqual(
    model.screenTimelapseRetentionFormValuesToCleanup({ retentionDays: 0, maxStorageGiB: 0 }),
    {
      enabled: false,
      retentionDays: 0,
      maxStorageMiB: 0
    }
  );
});

test('disabled retention config preserves its suggested limits', () => {
  assert.deepEqual(
    model.screenTimelapseRetentionCleanupToFormValues({
      enabled: false,
      retentionDays: 30,
      maxStorageMiB: 4096
    }),
    { retentionDays: 30, maxStorageGiB: 4 }
  );
  assert.deepEqual(
    model.screenTimelapseRetentionCleanupToFormValues({
      enabled: true,
      retentionDays: 30,
      maxStorageMiB: 1536
    }),
    { retentionDays: 30, maxStorageGiB: 1.5 }
  );
});

test('retention stricter check catches newly enabled and reduced limits', () => {
  const disabled = { enabled: false, retentionDays: 0, maxStorageMiB: 0 };
  const thirtyDays = { enabled: true, retentionDays: 30, maxStorageMiB: 0 };

  assert.equal(model.isScreenTimelapseRetentionStricter(disabled, thirtyDays), true);
  assert.equal(
    model.isScreenTimelapseRetentionStricter(thirtyDays, {
      enabled: true,
      retentionDays: 60,
      maxStorageMiB: 0
    }),
    false
  );
  assert.equal(
    model.isScreenTimelapseRetentionStricter(thirtyDays, {
      enabled: true,
      retentionDays: 15,
      maxStorageMiB: 0
    }),
    true
  );
  assert.equal(
    model.isScreenTimelapseRetentionStricter(thirtyDays, {
      enabled: true,
      retentionDays: 30,
      maxStorageMiB: 2048
    }),
    true
  );
});

test('retention preview display model keeps candidate, protected, and warning state', () => {
  const display = model.screenTimelapseRetentionPreviewToDisplayModel({
    dryRun: true,
    beforeBytes: 4096,
    projectedAfterBytes: 1024,
    overLimitAfterCleanup: true,
    wouldDeleteDays: 1,
    wouldDeleteBytes: 3072,
    candidates: [{ date: '2026-07-01', bytes: 3072, reasons: ['expired'] }],
    skipped: [{ date: '2026-07-20', bytes: 1024, reason: 'today' }],
    warnings: [{ path: '/recordings/bad', reason: 'unreadable' }]
  });

  assert.equal(display.isEmpty, false);
  assert.equal(display.hasSkipped, true);
  assert.equal(display.hasWarnings, true);
  assert.equal(display.remainsOverLimit, true);
  assert.equal(display.wouldDeleteDays, 1);
  assert.equal(display.wouldDeleteBytes, 3072);
  assert.deepEqual(
    display.candidates.map((candidate) => candidate.date),
    ['2026-07-01']
  );
});

test('storage warning starts at 90 percent of an enabled capacity limit', () => {
  const miB = 1024 * 1024;
  const retention = {
    config: {
      revision: 1,
      cleanup: { enabled: true, retentionDays: 0, maxStorageMiB: 1024 }
    },
    status: { usageBytes: 921 * miB + 614 * 1024, overLimit: false }
  };

  assert.equal(warning.getScreenTimelapseStorageWarning(retention), null);
  const thresholdBytes = Math.ceil(1024 * miB * 0.9);
  assert.deepEqual(
    warning.getScreenTimelapseStorageWarning({
      ...retention,
      status: { usageBytes: thresholdBytes, overLimit: false }
    }),
    { usedBytes: thresholdBytes, limitBytes: 1024 * miB }
  );
  assert.deepEqual(
    warning.getScreenTimelapseStorageWarning({
      ...retention,
      status: { usageBytes: 1, overLimit: true }
    }),
    { usedBytes: 1, limitBytes: 1024 * miB }
  );
});

test('storage warning stays quiet without an enabled capacity limit', () => {
  const status = { usageBytes: 2 * 1024 * 1024 * 1024, overLimit: true };
  assert.equal(
    warning.getScreenTimelapseStorageWarning({
      config: { revision: 1, cleanup: { enabled: false, retentionDays: 0, maxStorageMiB: 1024 } },
      status
    }),
    null
  );
  assert.equal(
    warning.getScreenTimelapseStorageWarning({
      config: { revision: 1, cleanup: { enabled: true, retentionDays: 30, maxStorageMiB: 0 } },
      status
    }),
    null
  );
});

test('desktop notification polls retention status and shows a deduplicated top-right warning', () => {
  const source = fs.readFileSync(path.join(webRoot, 'src/pages/desktop/notification.tsx'), 'utf8');

  assert.match(source, /getScreenTimelapseRetention\(\)/);
  assert.match(source, /screenTimelapseRetentionStatusPollIntervalMs = 60 \* 1000/);
  assert.match(source, /screen_timelapse_storage_warning/);
  assert.match(source, /api\.warning\(\{/);
  assert.match(source, /placement: 'topRight'/);
  assert.match(source, /duration: null/);
  assert.match(source, /key: screenTimelapseStorageNotificationKey/);
  assert.doesNotMatch(source, /key: screenTimelapseStorageNotificationKey[\s\S]*?btn:/);
  assert.match(source, /api\.destroy\(screenTimelapseStorageNotificationKey\)/);
});
