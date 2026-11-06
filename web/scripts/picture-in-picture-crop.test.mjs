import assert from 'node:assert/strict';
import test from 'node:test';

const { getFrameCropRect } = await import('../src/lib/picture-in-picture-crop.ts');

const crop = (left, top, width, height, sourceWidth = 1920, sourceHeight = 1080) => ({
  sourceWidth,
  sourceHeight,
  contentRect: { left, top, width, height },
  contentAspectRatio: width / height,
  updatedAt: 1
});

test('no crop, an unready crop, or a full-frame crop show the whole frame', () => {
  assert.equal(getFrameCropRect(null, 1920, 1080), null);
  assert.equal(getFrameCropRect({ ...crop(0, 0, 10, 10), contentRect: null }, 1920, 1080), null);
  assert.equal(getFrameCropRect(crop(0, 0, 10, 10, 0, 0), 1920, 1080), null);
  assert.equal(getFrameCropRect(crop(0, 0, 1920, 1080), 1920, 1080), null);
  assert.equal(getFrameCropRect(crop(100, 100, 400, 300), 0, 0), null);
});

test('crop maps 1:1 when the frame matches the source', () => {
  assert.deepEqual(getFrameCropRect(crop(420, 0, 1080, 1080), 1920, 1080), {
    x: 420,
    y: 0,
    width: 1080,
    height: 1080
  });
});

test('crop scales to a smaller decoded frame and stays even-aligned', () => {
  // 1920x1080 source decoded at 1280x720: factor 2/3.
  const rect = getFrameCropRect(crop(301, 151, 999, 601), 1280, 720);
  assert.deepEqual(rect, { x: 200, y: 100, width: 666, height: 400 });
  for (const value of Object.values(rect)) assert.equal(value % 2, 0);
});

test('crop is clamped to the frame bounds', () => {
  assert.deepEqual(getFrameCropRect(crop(-50, -20, 1000, 2000), 1920, 1080), {
    x: 0,
    y: 0,
    width: 950,
    height: 1080
  });
  assert.deepEqual(getFrameCropRect(crop(1919, 1079, 50, 50), 1920, 1080), {
    x: 1918,
    y: 1078,
    width: 2,
    height: 2
  });
  assert.equal(getFrameCropRect(crop(1920, 1080, 50, 50), 1920, 1080), null);
});
