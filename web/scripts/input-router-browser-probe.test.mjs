import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

import { parseOptions, statusDelta } from './input-router-browser-probe.mjs';

const probeSource = fs.readFileSync(
  path.resolve(import.meta.dirname, 'input-router-browser-probe.mjs'),
  'utf8'
);

test('browser probe parses touch mode and browser options', () => {
  const options = parseOptions(
    [
      '--insecure',
      '--browser',
      '/usr/bin/chromium',
      '--port',
      '9334',
      '--mode',
      'relative',
      '--switch-from',
      'digitizer-stylus',
      '--input',
      'mouse',
      '--contacts',
      '2'
    ],
    { NANOKVM_URL: 'https://192.0.2.1/', NANOKVM_TOKEN: 'secret' }
  );

  assert.equal(options.url, 'https://192.0.2.1');
  assert.equal(options.browser, '/usr/bin/chromium');
  assert.equal(options.port, 9334);
  assert.equal(options.mode, 'relative');
  assert.equal(options.switchFrom, 'digitizer-stylus');
  assert.equal(options.input, 'mouse');
  assert.equal(options.contacts, 2);
  assert.equal(options.insecure, true);
});

test('browser probe canonicalizes the legacy Android mode name', () => {
  const options = parseOptions(
    ['--mode', 'absolute-android', '--switch-from', 'absolute-android'],
    { NANOKVM_URL: 'https://192.0.2.1/', NANOKVM_TOKEN: 'secret' }
  );

  assert.equal(options.mode, 'digitizer-stylus');
  assert.equal(options.switchFrom, 'digitizer-stylus');
});

test('browser probe confirms the device profile before input and after UI switching', () => {
  assert.match(probeSource, /ensureDevicePointerMode\(options, initialBrowserMode\)/);
  assert.match(probeSource, /observeDevicePointerMode\(options, mode, previousDeviceStatus\)/);
  assert.match(probeSource, /devicePointerModeConfirmed: finalPointerMode\.confirmed/);
  assert.match(probeSource, /nanokvm-input-router-browser-probe\/v2/);
});

test('browser probe separates source and route counters', () => {
  const before = {
    received: 10,
    dispatched: 9,
    rejected: 1,
    decode_rejected: 1,
    dispatch_failed: 0,
    backpressure_dropped: 0,
    sources: [{ source: 'browser-touchscreen', events: 4 }],
    routes: [{ route: 'hidg2/touchscreen', dispatched: 4, failed: 0, backpressure_dropped: 0 }]
  };
  const after = {
    received: 12,
    dispatched: 11,
    rejected: 1,
    decode_rejected: 1,
    dispatch_failed: 0,
    backpressure_dropped: 0,
    sources: [{ source: 'browser-touchscreen', events: 6 }],
    routes: [{ route: 'hidg2/touchscreen', dispatched: 6, failed: 0, backpressure_dropped: 0 }]
  };

  assert.deepEqual(statusDelta(before, after), {
    received: 2,
    dispatched: 2,
    rejected: 0,
    decodeRejected: 0,
    dispatchFailed: 0,
    backpressureDropped: 0,
    sources: [{ source: 'browser-touchscreen', events: 2 }],
    routes: [{ route: 'hidg2/touchscreen', dispatched: 2, failed: 0, backpressureDropped: 0 }]
  });
});
