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

test('severe and moderate loss lower both targets at their respective thresholds', () => {
  const auto = new WebcamQuality();
  samples(auto, 2, { received: 12, missed: 13 });
  assert.deepEqual(auto.target, { mbps: 8, fps: 25 });
  samples(auto, 1, { received: 12, missed: 13 }, 2000);
  assert.deepEqual(auto.target, { mbps: 4, fps: 20 });
  const other = new WebcamQuality();
  samples(other, 5, { received: 18, missed: 2 });
  assert.deepEqual(other.target, { mbps: 8, fps: 25 });
  samples(other, 1, { received: 17, missed: 3 }, 5000);
  assert.deepEqual(other.target, { mbps: 4.5, fps: 23 });
});

test('a failed joint probe rolls back the pair and pauses upward probes for 20 seconds', () => {
  const auto = new WebcamQuality();
  samples(auto, 5);
  assert.deepEqual(auto.target, { mbps: 9.2, fps: 28 });
  samples(auto, 3, { received: 18, missed: 10 }, 5000);
  assert.deepEqual(auto.target, { mbps: 8, fps: 25 });
  samples(auto, 19, {}, 8000);
  assert.deepEqual(auto.target, { mbps: 8, fps: 25 });
  samples(auto, 1, {}, 27000);
  assert.deepEqual(auto.target, { mbps: 9.2, fps: 28 });
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
    assert.equal(auto.target.mbps, 8);
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

test('joint probes reach both ceilings in ten seconds despite occasional misses', () => {
  const auto = new WebcamQuality();
  samples(auto, 4);
  assert.deepEqual(auto.target, { mbps: 8, fps: 25 });
  samples(auto, 1, { received: 22, missed: 3 }, 4000);
  assert.deepEqual(auto.target, { mbps: 9.2, fps: 28 });
  samples(auto, 5, { received: 28 }, 5000);
  assert.deepEqual(auto.target, { mbps: 10, fps: 30 });
  const capped = new WebcamQuality();
  capped.setHostFps(25);
  samples(capped, 10);
  assert.deepEqual(capped.target, { mbps: 10, fps: 25 });
});

test('bitrate recovers after congestion with a camera capped at 20 fps', () => {
  const auto = new WebcamQuality();
  samples(auto, 3, { received: 12, missed: 8 });
  assert.deepEqual(auto.target, { mbps: 4, fps: 20 });

  samples(auto, 15, { received: 20 }, 3000);
  assert.deepEqual(auto.target, { mbps: 5.29, fps: 24 });
  samples(auto, 30, { received: 20 }, 18000);
  assert.deepEqual(auto.target, { mbps: 10, fps: 24 });
});

test('3% misses, growing queues and rising grant delay each hold a probe', () => {
  for (const [first, last] of [
    [{ received: 40 }, { received: 38, missed: 2 }],
    [{}, { queueGrew: true }],
    [{ ms: 10 }, { ms: 25 }]
  ]) {
    const auto = new WebcamQuality();
    samples(auto, 2, first);
    samples(auto, 3, last, 2000);
    assert.deepEqual(auto.target, { mbps: 8, fps: 25 });
  }
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
    [8, 10, 10, 10, 10, 10]
  );
  assert.deepEqual(final, { mbps: 10, fps: 30 });
});
test('mixed link bisects measured bounds, scales cadence and uses the final two seconds', () => {
  const trace = (limit) => calibrate((mbps) => (mbps > limit ? 'bad' : 'good'));
  const { probes, final } = trace(6);
  assert.deepEqual(
    probes.map((p) => p.mbps),
    [8, 5, 6.5, 5.75, 6.13, 6.13]
  );
  assert.deepEqual(final, { mbps: 5.75, fps: 21 });
  assert.equal(probes[1].fps, 20);
  assert.deepEqual(trace(6.2).final, { mbps: 6.13, fps: 22 });
});
test('floor is tested with reduced cadence when all probes fail', () => {
  const { probes, final } = calibrate(() => 'bad');
  assert.ok(probes.some((p) => p.mbps === 2 && p.fps === 15));
  assert.deepEqual(final, { mbps: 2, fps: 15 });
});
test('quiet or incomplete windows cannot confirm a higher target', () => {
  assert.equal(calibrate(() => 'quiet').final.mbps, 8);
  assert.equal(calibrate((_, s) => (s <= 3 ? 'good' : 'quiet')).final.mbps, 8);
  assert.equal(calibrate((_, s) => (s === 3 ? 'skip' : 'good')).probes[1].mbps, 8);
});
