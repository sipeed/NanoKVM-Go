import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

import { parseOptions, probeSessionEvidence, statusDelta } from './input-router-firefox-probe.mjs';

const probeSource = fs.readFileSync(
  path.resolve(import.meta.dirname, 'input-router-firefox-probe.mjs'),
  'utf8'
);

test('Firefox probe parses endpoint and browser options', () => {
  const options = parseOptions(
    [
      '--url',
      'https://kvm.local/',
      '--token',
      'token',
      '--port',
      '9333',
      '--mode',
      'touch-screen',
      '--input',
      'touch',
      '--insecure'
    ],
    {}
  );
  assert.equal(options.url, 'https://kvm.local');
  assert.equal(options.token, 'token');
  assert.equal(options.port, 9333);
  assert.equal(options.mode, 'touch-screen');
  assert.equal(options.input, 'touch');
  assert.equal(options.insecure, true);
});

test('Firefox probe canonicalizes the legacy Android mode name', () => {
  const options = parseOptions(
    ['--url', 'https://kvm.local/', '--token', 'token', '--mode', 'absolute-android'],
    {}
  );

  assert.equal(options.mode, 'digitizer-stylus');
});

test('Firefox probe confirms the device profile before launching input', () => {
  assert.match(probeSource, /ensureDevicePointerMode\(options, options\.mode\)/);
  assert.match(probeSource, /devicePointerModeConfirmed: pointerMode\.confirmed/);
  assert.match(probeSource, /nanokvm-input-router-firefox-probe\/v2/);
});

test('Firefox probe attributes sources and routes to its new session', () => {
  const baseline = {
    sessions: [
      {
        id: 4,
        sources: [{ source: 'browser-touchscreen', events: 3 }],
        routes: ['hidg2/touchscreen']
      }
    ]
  };
  const active = {
    sessions: [
      ...baseline.sessions,
      {
        id: 7,
        sources: [
          { source: 'browser-keyboard', events: 4 },
          { source: 'browser-pointer', events: 4 }
        ],
        routes: ['hidg0/keyboard', 'hidg1/relative-pointer']
      }
    ]
  };

  assert.deepEqual(probeSessionEvidence(baseline, active), {
    ids: [7],
    sources: ['browser-keyboard', 'browser-pointer'],
    routes: ['hidg0/keyboard', 'hidg1/relative-pointer']
  });
});

test('Firefox probe separates keyboard source and route deltas', () => {
  const before = {
    received: 10,
    dispatched: 9,
    rejected: 1,
    decode_rejected: 1,
    dispatch_failed: 0,
    backpressure_dropped: 0,
    sources: [{ source: 'browser-keyboard', events: 4 }],
    routes: [
      {
        route: 'hidg0/keyboard',
        dispatched: 4,
        failed: 0,
        backpressure_dropped: 0
      }
    ]
  };
  const after = {
    ...before,
    received: 12,
    dispatched: 11,
    sources: [{ source: 'browser-keyboard', events: 6 }],
    routes: [
      {
        route: 'hidg0/keyboard',
        dispatched: 6,
        failed: 0,
        backpressure_dropped: 0
      }
    ]
  };
  assert.deepEqual(statusDelta(before, after), {
    received: 2,
    dispatched: 2,
    rejected: 0,
    decodeRejected: 0,
    dispatchFailed: 0,
    backpressureDropped: 0,
    sources: [{ source: 'browser-keyboard', events: 2 }],
    routes: [
      {
        route: 'hidg0/keyboard',
        dispatched: 2,
        failed: 0,
        backpressureDropped: 0
      }
    ]
  });
});
