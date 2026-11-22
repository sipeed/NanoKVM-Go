import assert from 'node:assert/strict';
import test from 'node:test';

import { FramePacer, WebcamQuality } from '../src/pages/desktop/menu/webcam-quality.ts';

function samples(
  auto,
  n,
  {
    received = 25,
    size = 25000,
    ms = 35,
    missed = 0,
    eligible = received + missed,
    queueGrew = false
  } = {},
  start = 0
) {
  for (let i = 1; i <= n; i++)
    auto.observe(
      {
        elapsedMs: 1000,
        received,
        bytes: received * size,
        grantMs: received * ms,
        eligible,
        missed,
        queueGrew
      },
      start + i * 1000
    );
}

test('three bad seconds cut bitrate and a flat grant cycle with smaller JPEGs cuts fps', () => {
  const auto = new WebcamQuality();
  samples(auto, 2, { received: 12, missed: 13, ms: 80 });
  assert.equal(auto.target.mbps, 5);
  samples(auto, 1, { received: 12, missed: 13, ms: 80 }, 2000);
  assert.deepEqual(auto.target, { mbps: 4, fps: 30 });
  samples(auto, 3, { received: 12, size: 20000, missed: 13, ms: 79 }, 3000);
  assert.deepEqual(auto.target, { mbps: 2, fps: 15 });
  samples(auto, 10, { received: 15, size: 20000, ms: 60 }, 6000);
  assert.deepEqual(auto.target, { mbps: 2, fps: 17 });
});
test('fresh ten-second windows hold at 10% and trim above 10%; improving delivery keeps fps', () => {
  const auto = new WebcamQuality();
  samples(auto, 10, { received: 18, missed: 2 });
  assert.equal(auto.target.mbps, 5);
  samples(auto, 10, { received: 21, missed: 4 }, 10000);
  assert.equal(auto.target.mbps, 4.5);
  const other = new WebcamQuality();
  samples(other, 3, { received: 12, missed: 13, ms: 80 });
  samples(other, 3, { received: 17, missed: 8, size: 20000, ms: 55 }, 3000);
  assert.deepEqual(other.target, { mbps: 3.2, fps: 30 });
});
test('failed probe restores last target, blocks probing for twenty seconds', () => {
  const auto = new WebcamQuality();
  samples(auto, 3, { received: 18, missed: 7 });
  samples(auto, 10, { size: 20000 }, 3000);
  assert.equal(auto.target.mbps, 4.4);
  samples(auto, 3, { received: 18, missed: 7, size: 22000 }, 13000);
  assert.equal(auto.target.mbps, 4);
  samples(auto, 19, { size: 20000 }, 16000);
  assert.equal(auto.target.mbps, 4);
  samples(auto, 1, { size: 20000 }, 35000);
  assert.equal(auto.target.mbps, 4.4);
});
test('empty observations, idle camera and suspended timers never establish a loss streak', () => {
  const auto = new WebcamQuality();
  for (const elapsedMs of [1000, 6000]) {
    samples(auto, 2, { received: 12, missed: 13 });
    auto.observe(
      {
        elapsedMs,
        eligible: elapsedMs === 1000 ? 0 : 25,
        missed: 13,
        received: 0,
        bytes: 0,
        grantMs: 0,
        queueGrew: false
      },
      5000
    );
    samples(auto, 1, { received: 12, missed: 13 }, 6000);
    assert.equal(auto.target.mbps, 5);
    auto.observe(
      {
        elapsedMs: 1000,
        eligible: 0,
        received: 0,
        missed: 0,
        bytes: 0,
        grantMs: 0,
        queueGrew: false
      },
      8000
    );
  }
});
test('pacer respects the selected capture rate', () => {
  for (const [camera, target, expected] of [
    [25, 12, 36],
    [25, 15, 45],
    [25, 25, 75],
    [10, 25, 30]
  ]) {
    const pacer = new FramePacer();
    let count = 0;
    for (let i = 0; i < 3 * camera; i++) if (pacer.accept((i * 1000) / camera, target)) count++;
    assert.equal(count, expected);
  }
});

test('ordinary camera timestamp jitter never halves a full-rate LAN stream', () => {
  const pacer = new FramePacer();
  const timestamps = Array.from({ length: 250 }, (_, i) => i * 40 - (i % 2) * 2);
  assert.equal(timestamps.filter((ms) => pacer.accept(ms, 25)).length, 250);
  const reduced = new FramePacer();
  assert.equal(timestamps.filter((ms) => reduced.accept(ms, 15)).length, 150);
});

test('native capture can run faster than the host and never needs duplicate frames', () => {
  for (const [input, target, expected] of [
    [30, 25, 250],
    [60, 25, 250],
    [20, 25, 200],
    [30, 12, 120]
  ]) {
    const pacer = new FramePacer();
    const captured = Array.from({ length: input * 10 }, (_, i) => (i * 1000) / input);
    assert.equal(captured.filter((ms) => pacer.accept(ms, target)).length, expected);
  }
});

test('an irregular 20 fps camera can use all available slots at a 25 fps target', () => {
  for (const missing of [0, 1, 2]) {
    const pacer = new FramePacer();
    const timestamps = Array.from({ length: 300 }, (_, i) => i).filter((i) => i % 3 !== missing);
    const passed = timestamps.filter((i) => pacer.accept((i * 1000) / 30, 25)).length;
    assert.equal(passed, 200);
  }
});

test('clean windows reach the 8 Mbps and 30 fps ceilings', () => {
  const auto = new WebcamQuality();
  let time = 0;
  function tick(n, overrides = {}) {
    samples(auto, n, { received: 20, size: 25000, ...overrides }, time);
    time += n * 1000;
  }
  assert.deepEqual(auto.target, { mbps: 5, fps: 30 });
  tick(9);
  tick(1, { missed: 1 });
  tick(9);
  assert.equal(auto.target.mbps, 5);
  tick(1);
  assert.equal(auto.target.mbps, 5.5);
  tick(10);
  assert.equal(auto.target.mbps, 6.05);
  tick(60, { received: 30 });
  assert.deepEqual(auto.target, { mbps: 8, fps: 30 });
  auto.setHostFps(25);
  tick(60, { received: 25 });
  assert.deepEqual(auto.target, { mbps: 8, fps: 25 });
});

test('sustained congestion stops at 2 Mbps and 15 fps', () => {
  const auto = new WebcamQuality();
  samples(auto, 90, { received: 6, missed: 15, size: 8000, ms: 150 });
  assert.deepEqual(auto.target, { mbps: 2, fps: 15 });
});

function calibrate(verdict) {
  const auto = new WebcamQuality();
  const probes = [];
  for (let second = 1; second <= 11; second++) {
    const { mbps } = auto.target;
    const result = verdict(mbps, second);
    const sample = {
      elapsedMs: 1000,
      received: 30,
      eligible: 30,
      missed: result === 'bad' ? 6 : 0,
      bytes: mbps * 125000 * (result === 'quiet' ? 0.4 : 1),
      grantMs: 900,
      queueGrew: false
    };
    if (result !== 'skip') auto.observeStartup(sample, second);
    if ([1, 3, 5, 7, 9, 11].includes(second)) probes.push(auto.target);
  }
  auto.finishStartup();
  return { probes, final: auto.target };
}
test('warmup misses ignored, a loaded healthy link probes the actual ceiling', () => {
  const { probes, final } = calibrate((_, second) => (second === 1 ? 'bad' : 'good'));
  assert.deepEqual(
    probes.map((p) => p.mbps),
    [5, 8, 8, 8, 8, 8]
  );
  assert.deepEqual(final, { mbps: 8, fps: 30 });
});
test('mixed link bisects measured bounds, scales cadence and uses the final two seconds', () => {
  const trace = (limit) => calibrate((mbps) => (mbps > limit ? 'bad' : 'good'));
  const { probes, final } = trace(4);
  assert.deepEqual(
    probes.map((p) => p.mbps),
    [5, 3.5, 4.25, 3.88, 4.07, 4.07]
  );
  assert.deepEqual(final, { mbps: 3.88, fps: 26 });
  assert.equal(probes[1].fps, 25);
  assert.deepEqual(trace(4.1).final, { mbps: 4.07, fps: 27 });
});
test('floor is tested with reduced cadence when all probes fail', () => {
  const { probes, final } = calibrate(() => 'bad');
  assert.ok(probes.some((p) => p.mbps === 2 && p.fps === 15));
  assert.deepEqual(final, { mbps: 2, fps: 15 });
});
test('quiet or incomplete windows cannot confirm a higher target', () => {
  assert.equal(calibrate(() => 'quiet').final.mbps, 5);
  assert.equal(calibrate((_, s) => (s <= 3 ? 'good' : 'quiet')).final.mbps, 5);
  assert.equal(calibrate((_, s) => (s === 3 ? 'skip' : 'good')).probes[1].mbps, 5);
});
