import assert from 'node:assert/strict';
import test from 'node:test';

import { getBrowserLayoutViewportSize } from '../src/lib/mobile-layout.ts';
import {
  getOrCreatePageZoomBaseline,
  getOrCreatePageZoomBaselineData,
  getPageZoomScale
} from '../src/lib/page-zoom.ts';

class MemoryStorage {
  values = new Map();

  getItem(key) {
    return this.values.get(key) ?? null;
  }

  setItem(key, value) {
    this.values.set(key, value);
  }
}

test('keeps desktop page zoom stable across a refresh', () => {
  const storage = new MemoryStorage();
  const screen = { width: 1920, height: 1080 };

  const initialBaseline = getOrCreatePageZoomBaseline(2, screen, storage);
  assert.equal(getPageZoomScale(2, initialBaseline), 1);

  const refreshedBaseline = getOrCreatePageZoomBaseline(3, screen, storage);
  assert.equal(refreshedBaseline, 2);
  assert.equal(getPageZoomScale(3, refreshedBaseline), 1.5);
  assert.equal(getPageZoomScale(2, refreshedBaseline), 1);
});

test('rebuilds the baseline when the browser moves to a different screen', () => {
  const storage = new MemoryStorage();

  assert.equal(getOrCreatePageZoomBaseline(1, { width: 1920, height: 1080 }, storage), 1);
  assert.equal(getOrCreatePageZoomBaseline(2, { width: 2560, height: 1440 }, storage), 2);
});

test('ignores unavailable or invalid session storage', () => {
  const storage = new MemoryStorage();
  storage.setItem('nano-kvm-page-zoom-baseline', 'invalid');

  assert.equal(getOrCreatePageZoomBaseline(1.25, { width: 1920, height: 1080 }, storage), 1.25);
  assert.equal(getOrCreatePageZoomBaseline(Number.NaN, { width: 1920, height: 1080 }), 1);
});

test('uses DPR across Chrome sidebar resizes and real page zoom changes', () => {
  const chrome = 'Mozilla/5.0 AppleWebKit/537.36 Chrome/145.0.0.0 Safari/537.36';
  const baseline = { outerWidth: 1920, innerWidth: 1700 };
  for (const innerWidth of [1750, 1650, 1600, 1400, 1700]) {
    const viewport = { outerWidth: 1920, innerWidth };
    assert.equal(getPageZoomScale(2, 2, viewport, baseline, chrome), 1);
    assert.equal(getPageZoomScale(2.5, 2, viewport, baseline, chrome), 1.25);
  }
});

test('uses window geometry when DPR does not expose Safari page zoom', () => {
  const storage = new MemoryStorage();
  const screen = { width: 2560, height: 1440 };
  const baseline = getOrCreatePageZoomBaselineData(2, screen, storage, {
    innerWidth: 1440,
    outerWidth: 1500
  });

  assert.equal(
    getPageZoomScale(
      2,
      baseline.devicePixelRatio,
      { innerWidth: 960, outerWidth: 1500 },
      baseline.viewport
    ),
    1.5
  );
});

test('keeps Safari zoom relative when the first page load is already zoomed', () => {
  const storage = new MemoryStorage();
  const screen = { width: 2560, height: 1440 };
  const baseline = getOrCreatePageZoomBaselineData(2, screen, storage, {
    innerWidth: 1152,
    outerWidth: 1440
  });

  assert.equal(
    getPageZoomScale(
      2,
      baseline.devicePixelRatio,
      { innerWidth: 1152, outerWidth: 1440 },
      baseline.viewport
    ),
    1
  );
  assert.equal(
    getPageZoomScale(
      2,
      baseline.devicePixelRatio,
      { innerWidth: 960, outerWidth: 1440 },
      baseline.viewport
    ),
    1.2
  );
});

test('adds the viewport baseline to an older session record', () => {
  const storage = new MemoryStorage();
  storage.setItem(
    'nano-kvm-page-zoom-baseline',
    JSON.stringify({ devicePixelRatio: 2, screenWidth: 2560, screenHeight: 1440 })
  );

  const baseline = getOrCreatePageZoomBaselineData(2, { width: 2560, height: 1440 }, storage, {
    innerWidth: 1440,
    outerWidth: 1500
  });

  assert.deepEqual(baseline.viewport, {
    innerWidth: 1440,
    outerWidth: 1500
  });
});

test('does not mistake a Safari window resize for page zoom', () => {
  assert.equal(
    getPageZoomScale(
      2,
      2,
      { innerWidth: 940, outerWidth: 1000 },
      { innerWidth: 1440, outerWidth: 1500 }
    ),
    1
  );
});

test('keeps the mobile layout viewport stable while Safari pinch zooms', () => {
  const previousWindow = globalThis.window;
  const previousDocument = globalThis.document;
  globalThis.window = { innerWidth: 195, innerHeight: 422 };
  globalThis.document = { documentElement: { clientWidth: 390, clientHeight: 844 } };

  try {
    assert.deepEqual(getBrowserLayoutViewportSize(), { width: 390, height: 844 });
  } finally {
    if (previousWindow === undefined) delete globalThis.window;
    else globalThis.window = previousWindow;
    if (previousDocument === undefined) delete globalThis.document;
    else globalThis.document = previousDocument;
  }
});
