import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { after, before, test } from 'node:test';
import { createServer } from 'vite';

const webRoot = path.resolve(import.meta.dirname, '..');
let vite;
let model;
let locale;
let settingSource;
let panelSource;

const debugResponse = {
  enabled: true,
  available: true,
  zeroUs: 1500,
  releaseUs: { min: 889, max: 2050, default: 2050 },
  pressUs: { min: 889, max: 2000, default: 899 },
  count: { min: 1, default: 50, unlimitedValue: 0 },
  pressMs: { min: 100, max: 30_000, default: 100 },
  pressSpeedUsPerSecond: { min: 200, max: 2000, default: 2000 },
  releaseIntervalMs: 500,
  leaseTimeoutMs: 3000,
  running: false,
  current: 0,
  total: 0,
  state: 'idle',
  result: 'idle',
  error: ''
};

before(async () => {
  vite = await createServer({
    root: webRoot,
    appType: 'custom',
    logLevel: 'error',
    server: { middlewareMode: true, hmr: false, ws: false }
  });
  [model, locale, settingSource, panelSource] = await Promise.all([
    vite.ssrLoadModule('/src/pages/desktop/menu/settings/device/finger-robot-debug-model.ts'),
    vite.ssrLoadModule('/src/pages/desktop/menu/settings/device/finger-robot-debug-text.ts'),
    readFile(path.join(webRoot, 'src/pages/desktop/menu/settings/device/finger-robot.tsx'), 'utf8'),
    readFile(
      path.join(webRoot, 'src/pages/desktop/menu/settings/device/finger-robot-debug.tsx'),
      'utf8'
    )
  ]);
});

after(async () => {
  await vite.close();
});

test('disabled, incomplete, and malformed responses hide debug controls', () => {
  assert.equal(model.parseFingerRobotDebugData(null), null);
  assert.equal(model.parseFingerRobotDebugData({ enabled: false, available: true }), null);
  assert.equal(model.parseFingerRobotDebugData({ enabled: true, available: false }), null);
  assert.equal(
    model.parseFingerRobotDebugData({ ...debugResponse, zeroUs: 1499 }),
    null,
    'the fixed zero point cannot be replaced by a response value'
  );
  assert.equal(
    model.parseFingerRobotDebugData({
      ...debugResponse,
      pressUs: { min: 889, max: 2000, default: 2100 }
    }),
    null
  );
  assert.equal(
    model.parseFingerRobotDebugData({
      ...debugResponse,
      pressSpeedUsPerSecond: { min: 200, max: 2000, default: 100 }
    }),
    null
  );
  assert.equal(
    model.parseFingerRobotDebugData({
      ...debugResponse,
      count: { min: 1, default: 1, unlimitedValue: -1 }
    }),
    null
  );
});

test('an enabled but unavailable controller remains visible and keeps polling', () => {
  const unavailable = model.parseFingerRobotDebugData({
    ...debugResponse,
    available: false,
    error: 'waiting for GPIO loopback'
  });
  assert.ok(unavailable);
  assert.equal(unavailable.available, false);
  assert.equal(unavailable.error, 'waiting for GPIO loopback');
  assert.equal(model.shouldPollFingerRobotDebugStatus(false, false), true);
  assert.equal(model.shouldPollFingerRobotDebugStatus(true, true), true);
  assert.equal(model.shouldPollFingerRobotDebugStatus(false, true), false);

  assert.match(panelSource, /!debugData\.available/);
  assert.match(panelSource, /message=\{text\.unavailable\}/);
  assert.doesNotMatch(panelSource, /description=\{text\.reconnecting\}/);
  assert.match(panelSource, /formDisabled \|\| !debugData\.available \|\| draftError !== null/);
});

test('the settings switch mounts and unmounts the debug panel', () => {
  assert.match(
    settingSource,
    /return !disabledItems\.includes\(powerMenuItem\);/,
    'power disabled means the debug panel is hidden'
  );
  assert.match(settingSource, /checked=\{debugVisible\}/);
  assert.match(
    settingSource,
    /\{debugVisible && <FingerRobotDebug \/>\}/,
    'conditional rendering must unmount the debug component when switched off'
  );
});

test('debug response exposes only the five editable values with required defaults', () => {
  const parsed = model.parseFingerRobotDebugData(debugResponse);
  assert.ok(parsed);
  assert.deepEqual(Array.from(model.fingerRobotDebugEditableFields), [
    'releaseUs',
    'pressUs',
    'count',
    'pressMs',
    'pressSpeedUsPerSecond'
  ]);
  assert.deepEqual(model.createFingerRobotDebugDraft(parsed), {
    releaseUs: 2050,
    pressUs: 899,
    count: 50,
    pressMs: 100,
    pressSpeedUsPerSecond: 2000,
    unlimited: false
  });
  assert.deepEqual(model.fingerRobotDebugDefaults, {
    releaseUs: 2050,
    pressUs: 899,
    count: 50,
    pressMs: 100,
    pressSpeedUsPerSecond: 2000,
    unlimited: false
  });
});

test('draft validation uses device ranges and the fixed decreasing direction', () => {
  const parsed = model.parseFingerRobotDebugData(debugResponse);
  assert.ok(parsed);

  const valid = {
    releaseUs: 2050,
    pressUs: 1900,
    count: 20,
    pressMs: 2525,
    pressSpeedUsPerSecond: 1000,
    unlimited: false
  };
  assert.equal(model.validateFingerRobotDebugDraft(valid, parsed), null);
  assert.deepEqual(model.snapshotFingerRobotDebugDraft(valid, parsed), {
    releaseUs: 2050,
    pressUs: 1900,
    count: 20,
    pressMs: 2525,
    pressSpeedUsPerSecond: 1000
  });
  assert.equal(
    model.validateFingerRobotDebugDraft(
      {
        releaseUs: 2050,
        pressUs: 2000,
        count: 1,
        pressMs: 100,
        pressSpeedUsPerSecond: 200,
        unlimited: false
      },
      parsed
    ),
    null
  );
  assert.equal(
    model.validateFingerRobotDebugDraft({ ...valid, releaseUs: 1800 }, parsed),
    'direction'
  );
  assert.equal(
    model.validateFingerRobotDebugDraft({ ...valid, pressUs: 1200, releaseUs: 1200 }, parsed),
    'direction'
  );
  assert.equal(model.validateFingerRobotDebugDraft({ ...valid, count: null }, parsed), 'count');
  assert.equal(model.validateFingerRobotDebugDraft({ ...valid, count: 501 }, parsed), null);
  assert.equal(
    model.validateFingerRobotDebugDraft({ ...valid, count: Number.MAX_SAFE_INTEGER + 1 }, parsed),
    'count'
  );
  assert.equal(
    model.validateFingerRobotDebugDraft({ ...valid, pressSpeedUsPerSecond: 199 }, parsed),
    'pressSpeedUsPerSecond'
  );
  assert.equal(
    model.validateFingerRobotDebugDraft({ ...valid, pressSpeedUsPerSecond: 2001 }, parsed),
    'pressSpeedUsPerSecond'
  );
  assert.equal(
    model.validateFingerRobotDebugDraft({ ...valid, count: 50_000, pressMs: 30_000 }, parsed),
    null
  );

  const unlimited = { ...valid, count: null, unlimited: true };
  assert.equal(model.validateFingerRobotDebugDraft(unlimited, parsed), null);
  assert.deepEqual(model.snapshotFingerRobotDebugDraft(unlimited, parsed), {
    releaseUs: 2050,
    pressUs: 1900,
    count: 0,
    pressMs: 2525,
    pressSpeedUsPerSecond: 1000
  });
});

test('engineering panel provides explicit English and Chinese copy', () => {
  const english = locale.getFingerRobotDebugText('en');
  const chinese = locale.getFingerRobotDebugText('zh');
  assert.equal(english.title, 'Engineering debug mode');
  assert.equal(english.pressSpeed, 'Press speed');
  assert.equal(english.pressTime, 'Press hold duration');
  assert.equal(chinese.title, '工程调试模式');
  assert.equal(chinese.pressSpeed, '按压速度');
  assert.equal(chinese.pressTime, '按下保持时间');
  assert.equal(locale.getFingerRobotDebugText('zh_tw').stop, '停止測試');
  assert.equal(locale.getFingerRobotDebugText('zh_tw').reset, '重設為 1500 us');
  assert.equal(locale.getFingerRobotDebugText('zh_tw').unlimited, '無限測試');
  assert.equal(locale.getFingerRobotDebugText('de').title, 'Technischer Debug-Modus');
  // 未支持的语言仍须回退英文
  assert.equal(locale.getFingerRobotDebugText('xx-YY').title, 'Engineering debug mode');
  assert.equal(locale.formatFingerRobotState('resetting', english), 'Resetting');
  assert.equal(locale.formatFingerRobotResult('stopped', english), 'Stopped');
  assert.equal(locale.formatFingerRobotResult('reset', english), 'Reset to 1500 us');
});

test('running state swaps the primary action from start to stop', () => {
  assert.equal(model.getFingerRobotDebugPrimaryAction(false), 'start');
  assert.equal(model.getFingerRobotDebugPrimaryAction(true), 'stop');
});

test('only an explicit sensitive-action response is returned to the challenge gate', () => {
  const verificationRequired = {
    code: -3,
    msg: 'sensitive action verification required'
  };
  assert.equal(
    model.getFingerRobotDebugChallengeResult(verificationRequired),
    verificationRequired
  );
  assert.deepEqual(
    model.getFingerRobotDebugChallengeResult({
      code: -4,
      msg: 'Sensitive Action Verification expired'
    }),
    { code: -4, msg: 'Sensitive Action Verification expired' }
  );
  assert.equal(
    model.getFingerRobotDebugChallengeResult({ code: -3, msg: 'finger robot is unavailable' }),
    undefined
  );
  assert.equal(
    model.getFingerRobotDebugChallengeResult({ code: -3, msg: 'operation failed: pwm error' }),
    undefined
  );
  assert.equal(model.getFingerRobotDebugChallengeResult({ code: 0, msg: 'ok' }), undefined);
});

test('factory reset is blocked by pending actions but remains available while running', () => {
  const idle = {
    running: false,
    isGateBusy: false,
    isStarting: false,
    isStopping: false,
    isResetting: false
  };

  assert.equal(model.isFingerRobotDebugResetDisabled(idle), false);
  assert.equal(model.isFingerRobotDebugResetDisabled({ ...idle, running: true }), false);
  assert.equal(model.isFingerRobotDebugResetDisabled({ ...idle, isGateBusy: true }), true);
  assert.equal(model.isFingerRobotDebugResetDisabled({ ...idle, isStarting: true }), true);
  assert.equal(model.isFingerRobotDebugResetDisabled({ ...idle, isStopping: true }), true);
  assert.equal(model.isFingerRobotDebugResetDisabled({ ...idle, isResetting: true }), true);
});

test('a delayed start callback cannot call the test API after unmount', async () => {
  let testApiCalls = 0;
  const callTestApi = async () => {
    testApiCalls += 1;
    return { code: 0 };
  };

  const unmountedResult = model.runFingerRobotDebugStartIfActive(false, false, callTestApi);
  assert.equal(unmountedResult, undefined);
  assert.equal(testApiCalls, 0);

  const duplicateResult = model.runFingerRobotDebugStartIfActive(true, true, callTestApi);
  assert.equal(duplicateResult, undefined);
  assert.equal(testApiCalls, 0);

  const activeResult = model.runFingerRobotDebugStartIfActive(true, false, callTestApi);
  assert.deepEqual(await activeResult, { code: 0 });
  assert.equal(testApiCalls, 1);
});

test('progress distinguishes idle zero from an unlimited sequence', () => {
  assert.equal(model.formatFingerRobotDebugProgress(0, 0, false, '∞'), '0 / 0');
  assert.equal(model.formatFingerRobotDebugProgress(0, 0, true, '∞'), '0 / ∞');
  assert.equal(model.formatFingerRobotDebugProgress(12, 0, false, '∞'), '12 / ∞');
  assert.equal(model.formatFingerRobotDebugProgress(3, 20, true, '∞'), '3 / 20');
});

test('panel cleanup invokes stop semantics without turning stop into reset', async () => {
  let stopCalls = 0;
  model.stopFingerRobotDebugOnUnmount(async () => {
    stopCalls += 1;
  });
  await Promise.resolve();
  assert.equal(stopCalls, 1);
});
