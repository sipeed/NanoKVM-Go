import assert from 'node:assert/strict';
import test from 'node:test';

import { WHEEL_PIXELS_PER_NOTCH, WheelAccumulator } from '../src/lib/wheel.ts';

function run(acc, events, step = 16) {
  let t = 0;
  return events.map((e) => acc.push(e, (t += step)));
}

test('touchpad pixel deltas accumulate into whole notches', () => {
  const notches = run(
    new WheelAccumulator(undefined, { acceleration: false }),
    Array(40).fill({ deltaX: 0, deltaY: 4 })
  );
  assert.equal(
    notches.reduce((a, b) => a + b, 0),
    Math.floor((40 * 4) / WHEEL_PIXELS_PER_NOTCH)
  );
});

test('up and down are symmetric for small deltas', () => {
  const sum = (dy) =>
    run(new WheelAccumulator(), Array(30).fill({ deltaX: 0, deltaY: dy })).reduce(
      (a, b) => a + b,
      0
    );
  assert.equal(sum(0.3), -sum(-0.3));
  assert.equal(sum(0.3), 1, '9 px of travel crosses the first-notch threshold once');
});

test('mostly horizontal swipes never scroll vertically', () => {
  const notches = run(new WheelAccumulator(), Array(100).fill({ deltaX: 20, deltaY: -3 }));
  assert.ok(notches.every((n) => n === 0));
});

test('a wheel mouse notch is one notch, in pixel or line mode', () => {
  assert.equal(new WheelAccumulator().push({ deltaX: 0, deltaY: 100 }, 0), 1);
  assert.equal(new WheelAccumulator().push({ deltaX: 0, deltaY: -3, deltaMode: 1 }, 0), -1);
});

test('reversing direction does not use up the old backlog first', () => {
  const acc = new WheelAccumulator(25, { acceleration: false });
  run(acc, [{ deltaX: 0, deltaY: 35 }]);
  assert.equal(acc.push({ deltaX: 0, deltaY: -45 }, 20), -1);
});

test('a pause starts a fresh gesture', () => {
  const acc = new WheelAccumulator(40, { acceleration: false });
  assert.equal(acc.push({ deltaX: 0, deltaY: 30 }, 0), 1, 'first notch after a third of a notch');
  assert.equal(acc.push({ deltaX: 0, deltaY: 30 }, 10), 0, '30 of 40 built up');
  // Without the pause the 30 built up + 10 would cross 40; after it, only 10 of the ~13 needed.
  assert.equal(acc.push({ deltaX: 0, deltaY: 10 }, 1000), 0);
});

test('the touchpad scroll setting changes the distance per notch, within bounds', () => {
  const acc = new WheelAccumulator(25, { acceleration: false });
  acc.setPixelsPerNotch(80);
  assert.equal(acc.push({ deltaX: 0, deltaY: 26 }, 0), 0, 'first notch needs ~27 (a third of 80)');
  assert.equal(acc.push({ deltaX: 0, deltaY: 2 }, 10), 1);
  assert.equal(acc.push({ deltaX: 0, deltaY: 60 }, 20), 0);
  assert.equal(acc.push({ deltaX: 0, deltaY: 30 }, 30), 1, 'then a full 80 per notch');
  acc.setPixelsPerNotch(1);
  acc.reset();
  assert.equal(acc.push({ deltaX: 0, deltaY: 3 }, 0), 0, 'clamped to the minimum of 10');
  assert.equal(acc.push({ deltaX: 0, deltaY: 1 }, 10), 1);
});

test('acceleration: the same distance scrolls further when swiped faster', () => {
  const total = (deltaY, gap, count) => {
    const acc = new WheelAccumulator(25);
    let t = 0;
    let sum = 0;
    for (let i = 0; i < count; i++) sum += acc.push({ deltaX: 0, deltaY }, (t += gap));
    return sum;
  };
  // 560 px each: slow (2 px every 7 ms), normal (8 px every 7 ms), fast (20 px every 7 ms)
  const slow = total(2, 7, 280);
  const normal = total(8, 7, 70);
  const fast = total(20, 7, 28);
  assert.ok(slow < normal && normal < fast, `${slow} < ${normal} < ${fast}`);
  assert.ok(Math.abs(normal - 560 / 25) <= 2, `normal swipe is roughly unscaled: ${normal}`);
});

test('a slow swipe starts scrolling after a short travel', () => {
  const acc = new WheelAccumulator(25);
  let t = 0;
  let travelled = 0;
  for (let i = 0; i < 100; i++) {
    travelled += 1.5;
    if (acc.push({ deltaX: 0, deltaY: 1.5 }, (t += 7)) !== 0) break;
  }
  assert.ok(travelled <= 25 / 3 + 1.5, `first notch after ${travelled} px`);
});
