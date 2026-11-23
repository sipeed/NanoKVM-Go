import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const portal = loadTypeScriptModule('../src/lib/portal-detection.ts');
const crop = loadTypeScriptModule('../src/lib/screen-crop.ts');

test('detects symmetric pillarbox borders', () => {
  const image = makeImage(192, 108, { left: 24, top: 0, width: 144, height: 108 });
  const area = portal.detectPortalEffectiveAreaFromImageData(image, image.width, image.height);

  assert.deepEqual({ ...area.contentRect }, { left: 24, top: 0, width: 144, height: 108 });
});

test('detects portrait content with wide landscape side borders', () => {
  const image = makeImage(192, 108, { left: 65, top: 0, width: 62, height: 108 });
  const area = portal.detectPortalEffectiveAreaFromImageData(image, image.width, image.height);

  assert.deepEqual({ ...area.contentRect }, { left: 65, top: 0, width: 62, height: 108 });
});

test('detects content when symmetric borders exceed 40 percent', () => {
  const image = makeImage(520, 218, { left: 214, top: 0, width: 92, height: 218 });
  const area = portal.detectPortalEffectiveAreaFromImageData(image, image.width, image.height);

  assert.ok(area.contentRect);
  assert.equal(area.contentRect.top, 0);
  assert.equal(area.contentRect.height, 218);
  assert.ok(area.contentRect.left / image.width > 0.4);
  assert.ok((image.width - area.contentRect.left - area.contentRect.width) / image.width > 0.4);
});

test('tracker quickly activates a static portrait pillarbox', () => {
  const tracker = new portal.PortalDetectionTracker();
  const image = makeImage(192, 108, { left: 65, top: 0, width: 62, height: 108 });
  const area = portal.detectPortalEffectiveAreaFromImageData(image, 192, 108);
  const signature = portal.getPortalContentSignature(image, area);
  let update;

  for (let frame = 0; frame < 3; frame += 1) {
    update = tracker.update({ ...area, updatedAt: frame }, signature);
  }
  assert.equal(update.phase, 'learning');

  update = tracker.update({ ...area, updatedAt: 4 }, signature);
  assert.equal(update.phase, 'active');
  assert.deepEqual({ ...update.area.contentRect }, { left: 65, top: 0, width: 62, height: 108 });
});

test('keeps black, dark, and asymmetric frames uncropped', () => {
  for (const image of [
    makeImage(192, 108, null),
    makeImage(192, 108, { left: 0, top: 0, width: 192, height: 108 }, 18),
    makeImage(192, 108, { left: 24, top: 0, width: 168, height: 108 })
  ]) {
    const area = portal.detectPortalEffectiveAreaFromImageData(image, image.width, image.height);
    assert.equal(area.contentRect, null);
  }
});

test('preserves measurable near-black content edges', () => {
  const image = makeImage(192, 108, { left: 65, top: 0, width: 62, height: 108 });
  paintRect(image, { left: 65, top: 0, width: 62, height: 108 }, 18);
  paintRect(image, { left: 71, top: 0, width: 50, height: 108 }, 100);
  const area = portal.detectPortalEffectiveAreaFromImageData(image, 192, 108);
  assert.deepEqual({ ...area.contentRect }, { left: 65, top: 0, width: 62, height: 108 });
});

test('downsampled detection encloses visible edges', () => {
  const expected = { left: 944, top: 0, width: 668, height: 1440 };
  const image = makeImage(2560, 1440, expected);
  const area = portal.detectPortalEffectiveAreaFromImageData(image, 2560, 1440);
  assert.ok(area.contentRect.left <= expected.left);
  assert.ok(area.contentRect.left + area.contentRect.width >= expected.left + expected.width);
});

test('downsampled detection includes faint interpolated edge pixels', () => {
  const image = makeImage(520, 293, null);
  paintRect(image, { left: 0, top: 0, width: 520, height: 293 }, 16);
  paintRect(image, { left: 192, top: 0, width: 136, height: 293 }, 20);
  paintRect(image, { left: 193, top: 0, width: 134, height: 293 }, 160);
  const area = portal.detectPortalEffectiveAreaFromImageData(image, 520, 293);
  assert.ok(area.contentRect.left <= 192);
  assert.ok(area.contentRect.left + area.contentRect.width >= 328);
});

test('tracker preserves learned edges through dark margins and black frames', () => {
  const tracker = new portal.PortalDetectionTracker();
  const home = makeImage(192, 108, { left: 65, top: 0, width: 62, height: 108 });
  const dark = makeImage(192, 108, { left: 71, top: 0, width: 50, height: 108 });
  const homeArea = trackFrames(tracker, home, 8).area;
  for (const image of [dark, makeImage(192, 108, null), dark, home]) {
    const update = trackFrames(tracker, image, 12);
    assert.equal(update.phase, 'active');
    assert.deepEqual({ ...update.area.contentRect }, { ...homeArea.contentRect });
  }
});

test('relearns a full-height landscape crop that rotates to portrait', () => {
  const tracker = new portal.PortalDetectionTracker();
  const landscape = { left: 24, top: 0, width: 144, height: 108 };
  const portrait = { left: 65, top: 0, width: 62, height: 108 };
  trackFrames(tracker, makeImage(192, 108, landscape), 8);
  const update = trackFrames(tracker, makeImage(192, 108, portrait), 8);
  assert.deepEqual({ ...update.area.contentRect }, portrait);
});

test('tracker accepts expansion, rotation, full desktop, and new source dimensions', () => {
  const tracker = new portal.PortalDetectionTracker();
  trackFrames(tracker, makeImage(192, 108, { left: 71, top: 0, width: 50, height: 108 }), 8);
  const expanded = trackFrames(
    tracker,
    makeImage(192, 108, { left: 65, top: 0, width: 62, height: 108 }),
    8
  );
  assert.equal(expanded.area.contentRect.width, 62);
  const rotated = trackFrames(
    tracker,
    makeImage(192, 108, { left: 24, top: 14, width: 144, height: 80 }),
    8
  );
  assert.deepEqual({ ...rotated.area.contentRect }, { left: 24, top: 14, width: 144, height: 80 });
  const desktop = trackFrames(
    tracker,
    makeImage(192, 108, { left: 0, top: 0, width: 192, height: 108 }),
    4
  );
  assert.equal(desktop.phase, 'idle');
  assert.equal(desktop.area.contentRect, null);
  const nextSource = trackFrames(
    tracker,
    makeImage(160, 100, { left: 20, top: 0, width: 120, height: 100 }),
    8
  );
  assert.deepEqual(
    { ...nextSource.area.contentRect },
    { left: 20, top: 0, width: 120, height: 100 }
  );
});

test('a cursor on a black screen does not erase learned bounds', () => {
  const tracker = new portal.PortalDetectionTracker();
  const home = makeImage(192, 108, { left: 65, top: 0, width: 62, height: 108 });
  const learned = trackFrames(tracker, home, 8).area;
  const blank = makeImage(192, 108, null);
  paintRect(blank, { left: 90, top: 40, width: 3, height: 3 }, 180);
  const update = trackFrames(tracker, blank, 8);
  assert.equal(update.phase, 'active');
  assert.deepEqual({ ...update.area.contentRect }, { ...learned.contentRect });
});

test('an explicit reset learns a narrower source at the same resolution', () => {
  const tracker = new portal.PortalDetectionTracker();
  trackFrames(tracker, makeImage(192, 108, { left: 65, top: 0, width: 62, height: 108 }), 8);
  tracker.reset();
  const update = trackFrames(
    tracker,
    makeImage(192, 108, { left: 71, top: 0, width: 50, height: 108 }),
    8
  );
  assert.equal(update.phase, 'active');
  assert.deepEqual({ ...update.area.contentRect }, { left: 71, top: 0, width: 50, height: 108 });
});

test('retains bounds when the measured video black level is above zero', () => {
  const tracker = new portal.PortalDetectionTracker();
  const image = makeImage(192, 108, null);
  paintRect(image, { left: 0, top: 0, width: 192, height: 108 }, 16);
  paintRect(image, { left: 65, top: 0, width: 62, height: 108 }, 160);
  const learned = trackFrames(tracker, image, 8).area;
  paintRect(image, { left: 65, top: 0, width: 62, height: 108 }, 16);
  paintRect(image, { left: 90, top: 40, width: 3, height: 3 }, 180);
  const update = trackFrames(tracker, image, 8);
  assert.equal(update.phase, 'active');
  assert.deepEqual({ ...update.area.contentRect }, { ...learned.contentRect });
});

test('tracker requires stable borders and changing content', () => {
  const tracker = new portal.PortalDetectionTracker();
  let update;

  for (let frame = 0; frame < 4; frame += 1) {
    const image = makeImage(
      192,
      108,
      { left: 24, top: 0, width: 144, height: 108 },
      120,
      frame >= 2 ? 180 : 120
    );
    const area = portal.detectPortalEffectiveAreaFromImageData(image, 192, 108);
    update = tracker.update(area, portal.getPortalContentSignature(image, area));
  }

  assert.equal(update.phase, 'active');
  assert.deepEqual({ ...update.area.contentRect }, { left: 24, top: 0, width: 144, height: 108 });
});

test('tracker delays activation for a static centered panel', () => {
  const tracker = new portal.PortalDetectionTracker();
  const image = makeImage(192, 108, { left: 24, top: 0, width: 144, height: 108 });
  const area = portal.detectPortalEffectiveAreaFromImageData(image, 192, 108);
  const signature = portal.getPortalContentSignature(image, area);
  let update;

  for (let frame = 0; frame < 7; frame += 1) {
    update = tracker.update({ ...area, updatedAt: frame }, signature);
  }

  assert.equal(update.phase, 'learning');
  assert.equal(update.area.contentRect, null);

  update = tracker.update({ ...area, updatedAt: 8 }, signature);
  assert.equal(update.phase, 'active');
});

test('tracker releases a locked crop after four full frames', () => {
  const tracker = new portal.PortalDetectionTracker();
  const changingImages = [120, 120, 180, 180].map((change) =>
    makeImage(192, 108, { left: 24, top: 0, width: 144, height: 108 }, 120, change)
  );
  for (const image of changingImages) {
    const area = portal.detectPortalEffectiveAreaFromImageData(image, 192, 108);
    tracker.update(area, portal.getPortalContentSignature(image, area));
  }

  const full = portal.detectPortalEffectiveAreaFromImageData(
    makeImage(192, 108, { left: 0, top: 0, width: 192, height: 108 }),
    192,
    108
  );
  assert.equal(tracker.update(full).phase, 'active');
  assert.equal(tracker.update(full).phase, 'active');
  assert.equal(tracker.update(full).phase, 'active');
  const released = tracker.update(full);
  assert.equal(released.phase, 'idle');
  assert.equal(released.area.contentRect, null);
});

test('manual regions scale and clamp to the current source', () => {
  const area = crop.getScreenCropFromManualRegion(
    {
      id: 'region-1',
      sourceWidth: 1920,
      sourceHeight: 1080,
      contentRect: { left: 240, top: 0, width: 1440, height: 1080 }
    },
    1280,
    720
  );

  assert.deepEqual({ ...area.contentRect }, { left: 160, top: 0, width: 960, height: 720 });
  assert.equal(
    crop.getScreenCropFromManualRegion(
      {
        id: 'region-2',
        sourceWidth: 1920,
        sourceHeight: 1080,
        contentRect: { left: 0, top: 0, width: 1920, height: 1080 }
      },
      1024,
      768
    ).contentRect,
    null
  );
});

test('resolution presets produce a centered crop', () => {
  const area = crop.getScreenCropFromResolution({ width: 4, height: 3 }, 1920, 1080);
  assert.deepEqual({ ...area.contentRect }, { left: 240, top: 0, width: 1440, height: 1080 });
});

test('maps cropped content coordinates back into the complete source frame', () => {
  const area = crop.getScreenCropFromResolution({ width: 4, height: 3 }, 1920, 1080);

  const topLeft = crop.mapNormalizedContentPointToSource(0, 0, area);
  const center = crop.mapNormalizedContentPointToSource(0.5, 0.5, area);
  const bottomRight = crop.mapNormalizedContentPointToSource(1, 1, area);
  assert.equal(topLeft.x, 0.125);
  assert.equal(topLeft.y, 0);
  assert.equal(center.x, 0.5);
  assert.equal(center.y, 0.5);
  assert.equal(bottomRight.x, 0.875);
  assert.equal(bottomRight.y, 1);
});

test('input source size falls back to EDID before the first media frame', () => {
  const selected = crop.selectScreenSourceSize(
    { width: 0, height: 0 },
    { width: 0, height: 0 },
    { width: 1920, height: 1080 }
  );
  assert.deepEqual({ ...selected }, { width: 1920, height: 1080 });

  const actual = crop.selectScreenSourceSize(
    { width: 1280, height: 720 },
    { width: 0, height: 0 },
    { width: 1920, height: 1080 }
  );
  assert.deepEqual({ ...actual }, { width: 1280, height: 720 });
});

function makeImage(width, height, contentRect, base = 120, changed = base) {
  const data = new Uint8ClampedArray(width * height * 4);
  for (let y = 0; y < height; y += 1) {
    for (let x = 0; x < width; x += 1) {
      const inside =
        contentRect &&
        x >= contentRect.left &&
        x < contentRect.left + contentRect.width &&
        y >= contentRect.top &&
        y < contentRect.top + contentRect.height;
      const offset = (y * width + x) * 4;
      const value = inside ? (x > contentRect.left + contentRect.width / 2 ? changed : base) : 0;
      data[offset] = value;
      data[offset + 1] = inside ? Math.min(255, value + 30) : 0;
      data[offset + 2] = inside ? Math.max(0, value - 25) : 0;
      data[offset + 3] = 255;
    }
  }
  return { width, height, data };
}

function paintRect(image, rect, value) {
  for (let y = rect.top; y < rect.top + rect.height; y += 1) {
    for (let x = rect.left; x < rect.left + rect.width; x += 1) {
      const offset = (y * image.width + x) * 4;
      image.data.fill(value, offset, offset + 3);
    }
  }
}

function trackFrames(tracker, image, count) {
  let update;
  for (let frame = 0; frame < count; frame += 1) {
    const area = portal.detectPortalEffectiveAreaFromImageData(image, image.width, image.height);
    update = tracker.update(area, portal.getPortalContentSignature(image, area));
  }
  return update;
}

function loadTypeScriptModule(relativePath) {
  const file = path.resolve(import.meta.dirname, relativePath);
  const source = fs.readFileSync(file, 'utf8');
  const output = ts.transpileModule(source, {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022
    },
    fileName: file
  }).outputText;
  const module = { exports: {} };
  vm.runInNewContext(output, { exports: module.exports, module }, { filename: file });
  return module.exports;
}
