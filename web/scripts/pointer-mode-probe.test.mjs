import assert from 'node:assert/strict';
import test from 'node:test';

import {
  ensureDevicePointerMode,
  observeDevicePointerMode,
  parsePointerModeResponse,
  requiredDevicePointerMode
} from './pointer-mode-probe.mjs';

const options = {
  url: 'https://kvm.example',
  token: 'test-token',
  insecure: true
};

function response(mode, overrides = {}) {
  return {
    code: 0,
    msg: 'success',
    data: {
      mode,
      switching: false,
      ...overrides
    }
  };
}

function scriptedRequest(results) {
  const calls = [];
  const request = async (url, init, insecure) => {
    calls.push({ url, init, insecure });
    if (results.length === 0) throw new Error('unexpected pointer mode request');
    const result = results.shift();
    if (result instanceof Error) throw result;
    return result;
  };
  return { calls, request };
}

const noWait = async () => undefined;

test('maps browser modes to device profiles and keeps relative local', () => {
  assert.equal(requiredDevicePointerMode('absolute'), 'absolute');
  assert.equal(requiredDevicePointerMode('digitizer-stylus'), 'digitizer-stylus');
  assert.equal(requiredDevicePointerMode('touch-screen'), 'touch-screen');
  assert.equal(requiredDevicePointerMode('relative'), null);
  assert.throws(() => requiredDevicePointerMode('absolute-android'), /unsupported/);
});

test('strictly parses healthy and degraded GET responses', () => {
  assert.deepEqual(parsePointerModeResponse(response('absolute')), {
    mode: 'absolute',
    switching: false,
    degraded: false,
    error: ''
  });
  assert.deepEqual(
    parsePointerModeResponse(response('unknown', { degraded: true, error: 'bad' })),
    {
      mode: 'unknown',
      switching: false,
      degraded: true,
      error: 'bad'
    }
  );
  assert.throws(() => parsePointerModeResponse(response('relative')), /invalid mode/);
  assert.throws(
    () => parsePointerModeResponse({ code: 0, data: { mode: 'absolute', switching: 'no' } }),
    /invalid switching/
  );
  assert.throws(
    () =>
      parsePointerModeResponse({
        code: 0,
        data: { mode: 'absolute', switching: false, degraded: 'no' }
      }),
    /invalid degraded/
  );
  assert.throws(
    () =>
      parsePointerModeResponse({
        code: 0,
        data: { mode: 'absolute', switching: false, error: 1 }
      }),
    /invalid error/
  );
  assert.throws(() => parsePointerModeResponse({ code: -1, msg: 'failed' }), /failed/);
});

test('same device profile performs GET only', async () => {
  const transport = scriptedRequest([response('digitizer-stylus')]);
  const evidence = await ensureDevicePointerMode(options, 'digitizer-stylus', {
    request: transport.request,
    sleep: noWait
  });

  assert.equal(evidence.confirmed, true);
  assert.equal(evidence.postAttempted, false);
  assert.deepEqual(
    transport.calls.map((call) => call.init.method),
    ['GET']
  );
});

test('relative mode confirms the current profile without POST', async () => {
  const transport = scriptedRequest([response('touch-screen')]);
  const evidence = await ensureDevicePointerMode(options, 'relative', {
    request: transport.request,
    sleep: noWait
  });

  assert.equal(evidence.targetMode, null);
  assert.equal(evidence.after.mode, 'touch-screen');
  assert.equal(evidence.confirmed, true);
  assert.deepEqual(
    transport.calls.map((call) => call.init.method),
    ['GET']
  );
});

test('different profile POSTs once and tolerates temporary GET disconnection', async () => {
  const transport = scriptedRequest([
    response('absolute'),
    { code: 0, msg: 'success', data: response('touch-screen').data },
    new Error('USB network disconnected'),
    response('absolute', { switching: true }),
    response('touch-screen')
  ]);
  const sleeps = [];
  const evidence = await ensureDevicePointerMode(options, 'touch-screen', {
    request: transport.request,
    sleep: async (milliseconds) => sleeps.push(milliseconds),
    attempts: 4,
    intervalMs: 5
  });

  assert.equal(evidence.confirmed, true);
  assert.equal(evidence.postAttempted, true);
  assert.equal(evidence.post.received, true);
  assert.deepEqual(
    transport.calls.map((call) => call.init.method),
    ['GET', 'POST', 'GET', 'GET', 'GET']
  );
  assert.deepEqual(JSON.parse(transport.calls[1].init.body), { mode: 'touch-screen' });
  assert.equal(transport.calls[1].init.headers.Authorization, 'Bearer test-token');
  assert.deepEqual(sleeps, [5, 5]);
});

test('lost POST response is accepted only after GET confirms the target', async () => {
  const transport = scriptedRequest([
    response('absolute'),
    new Error('socket reset'),
    response('absolute', { switching: true }),
    response('digitizer-stylus')
  ]);
  const evidence = await ensureDevicePointerMode(options, 'digitizer-stylus', {
    request: transport.request,
    sleep: noWait,
    attempts: 2
  });

  assert.equal(evidence.confirmed, true);
  assert.deepEqual(evidence.post, {
    received: false,
    code: null,
    message: 'socket reset'
  });
});

test('failed switch and degraded state cannot be reported as confirmed', async () => {
  const failed = scriptedRequest([
    response('absolute'),
    { code: -5, msg: 'rebuild failed', data: response('absolute').data },
    response('absolute')
  ]);
  await assert.rejects(
    ensureDevicePointerMode(options, 'touch-screen', {
      request: failed.request,
      sleep: noWait,
      attempts: 1
    }),
    /was not activated/
  );

  const degraded = scriptedRequest([
    response('absolute'),
    { code: -8, msg: 'degraded', data: response('unknown', { degraded: true }).data },
    response('unknown', { degraded: true, error: 'rollback failed' })
  ]);
  await assert.rejects(
    ensureDevicePointerMode(options, 'digitizer-stylus', {
      request: degraded.request,
      sleep: noWait,
      attempts: 1
    }),
    /was not activated/
  );
});

test('browser observation verifies HID2 target and relative profile stability', async () => {
  const hid2 = scriptedRequest([
    response('absolute', { switching: true }),
    response('touch-screen')
  ]);
  const switched = await observeDevicePointerMode(
    options,
    'touch-screen',
    parsePointerModeResponse(response('absolute')),
    { request: hid2.request, sleep: noWait, attempts: 2 }
  );
  assert.equal(switched.confirmed, true);

  const relative = scriptedRequest([response('touch-screen')]);
  const unchanged = await observeDevicePointerMode(
    options,
    'relative',
    parsePointerModeResponse(response('touch-screen')),
    { request: relative.request, sleep: noWait }
  );
  assert.equal(unchanged.confirmed, true);
  assert.equal(unchanged.after.mode, 'touch-screen');
  assert.ok(relative.calls.every((call) => call.init.method === 'GET'));
});
