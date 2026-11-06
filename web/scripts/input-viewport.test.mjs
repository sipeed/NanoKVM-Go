import assert from 'node:assert/strict';
import test from 'node:test';

import {
  createInputViewport,
  createTouchPort,
  mapClientPointToTouchPort,
  mapContentPointToTouchPort,
  mapLocalPointToTouchPort
} from '../src/lib/input-viewport.ts';

test('maps the browser capture rectangle into normalized content coordinates', () => {
  const touchPort = createTouchPort({ left: 10, top: 20, width: 101, height: 201 }, null);

  assert.deepEqual(mapClientPointToTouchPort(touchPort, 10, 20), {
    content: { x: 0, y: 0 },
    source: { x: 0, y: 0 }
  });
  assert.deepEqual(mapClientPointToTouchPort(touchPort, 110, 220), {
    content: { x: 1, y: 1 },
    source: { x: 1, y: 1 }
  });
});

test('maps cropped content into the complete source viewport', () => {
  const viewport = createInputViewport({
    sourceWidth: 1920,
    sourceHeight: 1080,
    contentRect: { left: 720, top: 0, width: 480, height: 1080 },
    contentAspectRatio: 480 / 1080,
    updatedAt: 1
  });

  assert.deepEqual(mapContentPointToTouchPort(viewport, 0, 0).source, { x: 0.375, y: 0 });
  assert.deepEqual(mapContentPointToTouchPort(viewport, 0.5, 0.5).source, {
    x: 0.5,
    y: 0.5
  });
  assert.deepEqual(mapContentPointToTouchPort(viewport, 1, 1).source, { x: 0.625, y: 1 });
  assert.equal(viewport.contentAspectRatio, 480 / 1080);
});

test('keeps a touch port stable when live crop state changes during a gesture', () => {
  const crop = {
    sourceWidth: 1920,
    sourceHeight: 1080,
    contentRect: { left: 240, top: 0, width: 1440, height: 1080 },
    contentAspectRatio: 4 / 3,
    updatedAt: 1
  };
  const touchPort = createTouchPort({ left: 0, top: 0, width: 101, height: 101 }, crop);

  crop.contentRect.left = 0;
  crop.contentRect.width = 1920;

  assert.deepEqual(mapClientPointToTouchPort(touchPort, 0, 50).source, {
    x: 0.125,
    y: 0.5
  });
});

test('clamps client and content coordinates to their viewport bounds', () => {
  const touchPort = createTouchPort({ left: 10, top: 20, width: 100, height: 100 }, null);

  assert.deepEqual(mapClientPointToTouchPort(touchPort, -100, 500).content, { x: 0, y: 1 });
  assert.deepEqual(mapContentPointToTouchPort(touchPort.viewport, 2, -1).source, {
    x: 1,
    y: 0
  });
});

test('maps target-local coordinates without relying on the browser viewport origin', () => {
  const touchPort = createTouchPort({ left: 300, top: 200, width: 101, height: 201 }, null);

  assert.deepEqual(mapLocalPointToTouchPort(touchPort, 50, 100, 101, 201), {
    content: { x: 0.5, y: 0.5 },
    source: { x: 0.5, y: 0.5 }
  });
});
