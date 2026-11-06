import assert from 'node:assert/strict';
import test from 'node:test';

import {
  activeTouchPayload,
  encodeFrame,
  evaluateProbe,
  hid2ProbeFrame,
  keyboardModifierPayload,
  parseOptions,
  relativePointerPayload,
  statusDelta
} from './input-router-device-probe.mjs';

test('device probe encodes transport and semantic headers', () => {
  const frame = encodeFrame(3, 5, Buffer.from([0]), 0x12345678, 0x87654321);

  assert.equal(frame.length, 14);
  assert.deepEqual(Array.from(frame.subarray(0, 5)), [3, 1, 3, 5, 0]);
  assert.equal(frame.readUInt32LE(5), 0x12345678);
  assert.equal(frame.readUInt32LE(9), 0x87654321);
  assert.equal(frame[13], 0);
});

test('device probe builds two stable active touch contacts', () => {
  const payload = activeTouchPayload(3);

  assert.equal(payload.length, 17);
  assert.equal(payload[0], 2);
  assert.deepEqual(Array.from(payload.subarray(1, 9)), [1, 3, 0, 80, 0, 112, 255, 0]);
  assert.deepEqual(Array.from(payload.subarray(9, 17)), [2, 3, 0, 176, 0, 112, 255, 0]);
});

test('device probe builds a modifier-only keyboard report', () => {
  assert.deepEqual(Array.from(keyboardModifierPayload(2)), [2, 0, 0, 0, 0, 0, 0, 0]);
});

test('device probe builds a no-button relative pointer report', () => {
  assert.deepEqual(Array.from(relativePointerPayload(-1)), [0, 255, 255, 0, 0, 0, 0]);
});

test('device probe selects only the active HID2 profile frame', () => {
  const expected = {
    absolute: { source: 2, kind: 3, route: 'hidg2/absolute-pointer', payloadLength: 7 },
    'digitizer-stylus': { source: 2, kind: 4, route: 'hidg2/stylus', payloadLength: 6 },
    'touch-screen': { source: 3, kind: 5, route: 'hidg2/touchscreen', payloadLength: 1 }
  };

  for (const [mode, want] of Object.entries(expected)) {
    const frame = hid2ProbeFrame(mode);
    assert.deepEqual(
      {
        source: frame.source,
        kind: frame.kind,
        route: frame.route,
        payloadLength: frame.payload.length
      },
      want
    );
  }
  assert.throws(() => hid2ProbeFrame('unknown'), /unsupported active pointer mode/);
});

test('device probe separates route and failure deltas', () => {
  const before = runtime({ received: 10, dispatched: 9 });
  const after = runtime({ received: 20, dispatched: 19 });
  after.routes[0].dispatched = 7;
  after.routes[0].write_retries = 2;

  assert.deepEqual(statusDelta(before, after), {
    received: 10,
    dispatched: 10,
    rejected: 0,
    decodeRejected: 0,
    dispatchFailed: 0,
    backpressureDropped: 0,
    routes: [
      {
        route: 'hidg0/keyboard',
        dispatched: 7,
        failed: 0,
        writeRetries: 2,
        backpressureDropped: 0
      }
    ]
  });
});

test('device probe verifies marked sessions are released', () => {
  const before = runtime({ received: 0, dispatched: 0 });
  const middle = runtime({ received: 5, dispatched: 5 });
  middle.sessions = [
    { id: 11, last_source_sequence: 100 },
    { id: 12, last_source_sequence: 200 }
  ];
  const after = runtime({ received: 5, dispatched: 5 });

  const result = evaluateProbe({
    before,
    middle,
    after,
    sent: 5,
    probeSessionIDs: [11, 12]
  });

  assert.equal(result.pass, true);
  assert.equal(result.outcome, 'pass');
  assert.equal(result.checks.noRejectedFrames, true);
  assert.deepEqual(result.probeSessionIDs, [11, 12]);
});

test('device probe marks concurrent external input as inconclusive', () => {
  const before = runtime({ received: 0, dispatched: 0 });
  before.sessions = [{ id: 1, events: 2, last_source_sequence: 1 }];
  const middle = runtime({ received: 6, dispatched: 6 });
  middle.sessions = [
    { id: 1, events: 3, last_source_sequence: 2 },
    { id: 8, events: 3, last_source_sequence: 90 },
    { id: 9, events: 2, last_source_sequence: 91 }
  ];
  const after = runtime({ received: 6, dispatched: 6 });
  after.sessions = [{ id: 1, events: 3, last_source_sequence: 2 }];

  const result = evaluateProbe({
    before,
    middle,
    after,
    sent: 5,
    probeSessionIDs: [8, 9]
  });

  assert.equal(result.pass, false);
  assert.equal(result.outcome, 'inconclusive');
  assert.deepEqual(result.probeSessionIDs, [8, 9]);
  assert.deepEqual(result.diagnostics.changedPreexistingSessionIDs, [1]);
});

test('device probe reads endpoint and token from environment', () => {
  const options = parseOptions(
    [
      '--insecure',
      '--active-touch',
      '--active-pointer',
      '--hid2-frames',
      '25',
      '--target-os',
      'Android'
    ],
    { NANOKVM_URL: 'https://192.0.2.1/', NANOKVM_TOKEN: 'secret' }
  );

  assert.equal(options.url, 'https://192.0.2.1');
  assert.equal(options.token, 'secret');
  assert.equal(options.insecure, true);
  assert.equal(options.activeTouch, true);
  assert.equal(options.activePointer, true);
  assert.equal(options.hid2Frames, 25);
  assert.equal(options.targetOS, 'Android');

  const legacy = parseOptions(['--touch-frames', '7'], {
    NANOKVM_URL: 'https://192.0.2.1/',
    NANOKVM_TOKEN: 'secret'
  });
  assert.equal(legacy.hid2Frames, 7);
});

function runtime({ received, dispatched }) {
  return {
    received,
    dispatched,
    rejected: 0,
    decode_rejected: 0,
    dispatch_failed: 0,
    backpressure_dropped: 0,
    sessions: [],
    routes: [
      {
        route: 'hidg0/keyboard',
        dispatched: 0,
        failed: 0,
        write_retries: 0,
        backpressure_dropped: 0
      }
    ]
  };
}
