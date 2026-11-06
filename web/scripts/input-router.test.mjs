import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const inputRouterFile = path.resolve(import.meta.dirname, '../src/lib/input-router.ts');
const inputRegionFile = path.resolve(import.meta.dirname, '../src/lib/input-region.ts');

function loadInputRegionRegistry() {
  const source = fs.readFileSync(inputRegionFile, 'utf8');
  const output = ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
    fileName: inputRegionFile
  }).outputText;
  const module = { exports: {} };

  vm.runInNewContext(output, { exports: module.exports, module }, { filename: inputRegionFile });
  return module.exports;
}

function loadInputRouter() {
  const frames = [];
  const handlers = new Map();
  const connectionStateHandlers = [];
  const source = fs.readFileSync(inputRouterFile, 'utf8');
  const output = ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
    fileName: inputRouterFile
  }).outputText;
  const module = { exports: {} };

  vm.runInNewContext(
    output,
    {
      exports: module.exports,
      module,
      TextEncoder,
      performance: { now: () => 1234 },
      require(specifier) {
        if (specifier === '@/lib/websocket.ts') {
          return {
            MessageEvent: { InputRouter: 3, InputCaptureDebug: 4 },
            client: {
              send(frame) {
                frames.push(Uint8Array.from(frame));
                return true;
              },
              on(type, handler) {
                handlers.set(type, handler);
                return () => handlers.delete(type);
              },
              onConnectionState(handler) {
                connectionStateHandlers.push(handler);
                return () => {};
              }
            }
          };
        }
        throw new Error(`Unexpected require(${specifier})`);
      }
    },
    { filename: inputRouterFile }
  );

  return { ...module.exports, frames, handlers, connectionStateHandlers };
}

function header(frame) {
  const view = new DataView(frame.buffer, frame.byteOffset, frame.byteLength);
  return {
    message: frame[0],
    version: frame[1],
    source: frame[2],
    kind: frame[3],
    flags: frame[4],
    sequence: view.getUint32(5, true),
    capturedAt: view.getUint32(9, true)
  };
}

test('encodes versioned relative pointer events', () => {
  const { InputRouterClient, InputSource, InputKind, frames } = loadInputRouter();
  const router = new InputRouterClient();

  router.sendRelativePointer(InputSource.BrowserPointer, 3, -12, 20, -1, 4321);

  assert.equal(frames.length, 1);
  assert.deepEqual(header(frames[0]), {
    message: 3,
    version: 1,
    source: InputSource.BrowserPointer,
    kind: InputKind.RelativePointer,
    flags: 0,
    sequence: 1,
    capturedAt: 4321
  });
  assert.deepEqual(Array.from(frames[0].subarray(13)), [3, 244, 255, 20, 0, 255, 255]);
});

test('encodes stable multi-touch contacts with normalized axes', () => {
  const { InputRouterClient, InputSource, InputKind, frames } = loadInputRouter();
  const router = new InputRouterClient();

  router.sendTouchscreen(
    InputSource.BrowserTouchscreen,
    [
      { id: 2, touching: true, x: 0, y: 1, pressure: 64 },
      { id: 7, touching: false, inRange: true, x: 0.5, y: 0.25, pressure: 0 }
    ],
    99
  );

  const frame = frames[0];
  assert.equal(header(frame).kind, InputKind.Touchscreen);
  assert.equal(frame[13], 2);
  assert.deepEqual(Array.from(frame.subarray(14, 22)), [2, 3, 0, 0, 255, 255, 64, 0]);
  assert.deepEqual(Array.from(frame.subarray(22, 30)), [7, 2, 0, 128, 0, 64, 0, 0]);
});

test('associates a browser PointerEvent with the routed HID frame', () => {
  const {
    InputRouterClient,
    InputSource,
    InputKind,
    frames,
    setInputCaptureDebugEnabled,
    sendBrowserPointerCapture
  } = loadInputRouter();
  const router = new InputRouterClient();
  setInputCaptureDebugEnabled(true);

  const capture = {
    api: 'PointerEvent',
    sequence: 77,
    capturedAtMS: 1000,
    type: 'pointermove',
    content: {
      pointerId: 3,
      pointerType: 'touch',
      isPrimary: true,
      clientX: 12,
      clientY: 34,
      pageX: 12,
      pageY: 34,
      screenX: 12,
      screenY: 34,
      offsetX: 1,
      offsetY: 2,
      movementX: 3,
      movementY: 4,
      button: 0,
      buttons: 1,
      pressure: 1,
      tangentialPressure: 0,
      width: 5,
      height: 6,
      tiltX: 0,
      tiltY: 0,
      twist: 0,
      altitudeAngle: 0,
      azimuthAngle: 0,
      ctrlKey: false,
      shiftKey: false,
      altKey: false,
      metaKey: false,
      timeStamp: 10,
      coalescedEventCount: 0,
      cancelable: true,
      composed: true,
      targetId: 'input'
    }
  };
  assert.equal(sendBrowserPointerCapture(capture), true);
  router.sendAbsolutePointer(
    InputSource.BrowserTouchscreen,
    0,
    0.25,
    0.5,
    0,
    1000,
    { sequence: 77, capturedAt: 1000 }
  );

  assert.equal(frames[0][0], 4);
  assert.deepEqual(JSON.parse(new TextDecoder().decode(frames[0].subarray(1))), capture);
  const routed = frames[1];
  assert.equal(header(routed).kind, InputKind.AbsolutePointer);
  assert.equal(header(routed).flags, 1);
  const view = new DataView(routed.buffer, routed.byteOffset, routed.byteLength);
  const contextOffset = routed.length - 8;
  assert.equal(view.getUint32(contextOffset, true), 77);
  assert.equal(view.getUint32(contextOffset + 4, true), 234000);
});

test('sends a complete TouchEvent snapshot through the shared debug capture channel', () => {
  const { frames, setInputCaptureDebugEnabled, sendBrowserInputCapture } = loadInputRouter();
  setInputCaptureDebugEnabled(true);
  const touch = {
    identifier: 4,
    clientX: 12,
    clientY: 34,
    pageX: 12,
    pageY: 34,
    screenX: 56,
    screenY: 78,
    radiusX: 8,
    radiusY: 9,
    rotationAngle: 0,
    force: 0.5
  };
  const capture = {
    api: 'TouchEvent',
    sequence: 78,
    capturedAtMS: 1001,
    type: 'touchmove',
    content: {
      touches: [touch],
      targetTouches: [touch],
      changedTouches: [touch],
      ctrlKey: false,
      shiftKey: false,
      altKey: false,
      metaKey: false,
      timeStamp: 11,
      cancelable: true,
      composed: true,
      targetId: 'input'
    }
  };

  assert.equal(sendBrowserInputCapture(capture), true);
  assert.deepEqual(JSON.parse(new TextDecoder().decode(frames[0].subarray(1))), capture);
});

test('pointer events classify touch separately from mouse and pen', () => {
  const source = [
    '../src/pages/desktop/mouse/pointer-events-cap.ts',
    '../src/pages/desktop/mouse/pointer-events-cap.tsx'
  ]
    .map((file) => fs.readFileSync(path.resolve(import.meta.dirname, file), 'utf8'))
    .join('\n');
  assert.match(source, /subscribePointerEvents/);
  assert.match(source, /pointerdown/);
  assert.match(source, /pointerId/);
  assert.match(source, /pointerType === 'touch'/);
  assert.match(source, /InputSource\.BrowserTouchscreen/);
  assert.match(source, /InputSource\.BrowserPointer/);
  assert.match(source, /setPointerCapture/);
  assert.match(source, /mouseMode === 'touch-screen'/);
  assert.match(source, /useNativeTouchEvents && event\.pointerType === 'touch'/);
});

test('absolute pointer preserves mode transitions and hover movement', () => {
  const source = fs.readFileSync(
    path.resolve(import.meta.dirname, '../src/pages/desktop/mouse/absolute.tsx'),
    'utf8'
  );
  const beforeLayoutEffect = source.slice(0, source.indexOf('useLayoutEffect'));

  assert.doesNotMatch(beforeLayoutEffect, /outputStylusRef\.current\s*=\s*useStylusReports/);
  assert.match(source, /event\.type === 'pointermove'/);
  assert.match(source, /event\.pointerType === 'touch'/);
  assert.match(source, /handleMouseEvent\(\{ type: 'move', x, y \}, event\.source\)/);
});

test('stylus wheel is ignored instead of reaching either HID route', () => {
  const source = fs.readFileSync(
    path.resolve(import.meta.dirname, '../src/pages/desktop/mouse/absolute.tsx'),
    'utf8'
  );
  const wheelCase = source.match(/case 'wheel':[\s\S]*?\n\s*return;/)?.[0] ?? '';
  assert.notEqual(wheelCase, '');
  assert.match(wheelCase, /if \(!stylus\) \{[\s\S]*?sendRelativePointerProxy/);
  assert.doesNotMatch(wheelCase, /sendPointerState/);
});

test('absolute movement does not synthesize an idle trailing report', () => {
  const source = fs.readFileSync(
    path.resolve(import.meta.dirname, '../src/pages/desktop/mouse/absolute.tsx'),
    'utf8'
  );

  assert.doesNotMatch(source, /pointerIdleTimerRef/);
  assert.doesNotMatch(source, /schedulePointerIdleRelease/);
  assert.doesNotMatch(source, /\}, 120\);/);
  assert.match(source, /event\.type === 'pointerleave'[\s\S]*?releaseAbsolutePointer\(\)/);
});

test('rejects malformed keyboard and oversized touch input', () => {
  const { InputRouterClient, InputSource } = loadInputRouter();
  const router = new InputRouterClient();

  assert.throws(() => router.sendKeyboard(InputSource.BrowserKeyboard, new Uint8Array(7)));
  assert.throws(() =>
    router.sendTouchscreen(
      InputSource.BrowserTouchscreen,
      Array.from({ length: 11 }, (_, index) => ({
        id: index + 1,
        touching: true,
        x: 0,
        y: 0
      }))
    )
  );
});

test('masks stylus state to descriptor-supported bits', () => {
  const { InputRouterClient, InputSource, InputKind, frames } = loadInputRouter();
  const router = new InputRouterClient();

  router.sendStylus(InputSource.BrowserPointer, 0xff, 0.5, 0.25, 255, 77);

  assert.equal(header(frames[0]).kind, InputKind.Stylus);
  assert.equal(frames[0][13], 0x03);
});

test('global input gate drops reports without consuming sequence numbers', () => {
  const { InputRouterClient, InputSource, frames, isInputRouterEnabled, setInputRouterEnabled } =
    loadInputRouter();
  const router = new InputRouterClient();

  assert.equal(isInputRouterEnabled(), true);
  setInputRouterEnabled(false);
  assert.equal(router.sendRelativePointer(InputSource.BrowserPointer, 1), false);
  assert.equal(frames.length, 0);

  setInputRouterEnabled(true);
  assert.equal(router.sendRelativePointer(InputSource.BrowserPointer, 0), true);
  assert.equal(frames.length, 1);
  assert.equal(header(frames[0]).sequence, 1);
});

test('encodes a pointer release carrying the route mask', () => {
  const {
    InputRouterClient,
    InputKind,
    InputSource,
    INPUT_ROUTER_RELEASE_RELATIVE_POINTER,
    INPUT_ROUTER_RELEASE_ABSOLUTE_POINTER,
    INPUT_ROUTER_RELEASE_STYLUS,
    INPUT_ROUTER_RELEASE_TOUCHSCREEN,
    frames,
    setInputRouterEnabled
  } = loadInputRouter();
  const router = new InputRouterClient();
  const mask =
    INPUT_ROUTER_RELEASE_RELATIVE_POINTER |
    INPUT_ROUTER_RELEASE_STYLUS |
    INPUT_ROUTER_RELEASE_TOUCHSCREEN;

  assert.equal(router.sendPointerRelease(InputSource.BrowserPointer, mask), true);
  assert.equal(frames.length, 1);
  assert.deepEqual(header(frames[0]), {
    message: 3,
    version: 1,
    source: InputSource.BrowserPointer,
    kind: InputKind.PointerRelease,
    flags: 0,
    sequence: 1,
    capturedAt: 1234
  });
  assert.equal(frames[0].length, 14);
  assert.deepEqual(Array.from(frames[0].subarray(13)), [mask]);

  setInputRouterEnabled(false);
  assert.equal(
    router.sendPointerRelease(InputSource.BrowserPointer, INPUT_ROUTER_RELEASE_ABSOLUTE_POINTER),
    false
  );
  assert.equal(frames.length, 1);

  setInputRouterEnabled(true);
  assert.equal(
    router.sendPointerRelease(InputSource.BrowserPointer, INPUT_ROUTER_RELEASE_ABSOLUTE_POINTER),
    true
  );
  assert.equal(header(frames[1]).sequence, 2);

  assert.throws(() => router.sendPointerRelease(InputSource.BrowserPointer, 0));
  assert.throws(() => router.sendPointerRelease(InputSource.BrowserPointer, 0x10));
});

test('input-region registry pushes the current element to subscribers', () => {
  const registry = loadInputRegionRegistry();
  assert.equal(registry.getInputRegionElement(), null);

  const first = {};
  const second = {};
  const seen = [];
  const unsubscribe = registry.subscribeInputRegion((element) => seen.push(element));
  assert.deepEqual(seen, [null]);

  registry.registerInputRegion(first);
  registry.registerInputRegion(first);
  registry.registerInputRegion(second);
  registry.registerInputRegion(null);
  assert.deepEqual(seen, [null, first, second, null]);
  assert.equal(registry.getInputRegionElement(), null);

  unsubscribe();
  registry.registerInputRegion(first);
  assert.deepEqual(seen, [null, first, second, null]);

  const lateSeen = [];
  const unsubscribeLate = registry.subscribeInputRegion((element) => lateSeen.push(element));
  assert.deepEqual(lateSeen, [first]);
  unsubscribeLate();
  registry.registerInputRegion(null);
});

test('input capture components bind through the shared input-region registry', () => {
  const mouseDir = path.resolve(import.meta.dirname, '../src/pages/desktop/mouse');
  const read = (file) => fs.readFileSync(path.join(mouseDir, file), 'utf8');

  const pointerCap = read('pointer-events-cap.tsx');
  const touchScreen = read('touch-screen.tsx');
  const calibrationOverlay = read('calibration-overlay.tsx');
  const inputRegion = read('input-region.tsx');

  for (const source of [pointerCap, touchScreen, calibrationOverlay]) {
    assert.doesNotMatch(source, /getElementById\s*\(\s*(?:INPUT_REGION_ID|['"]input-region['"])/);
    assert.doesNotMatch(source, /bindRetry/);
    assert.match(source, /\bsubscribeInputRegion\s*\(/);
  }
  assert.match(inputRegion, /registerInputRegion/);
});

test('absolute pointer never fabricates an unobserved position', () => {
  const mouseDir = path.resolve(import.meta.dirname, '../src/pages/desktop/mouse');
  const source = fs.readFileSync(path.join(mouseDir, 'absolute.tsx'), 'utf8');

  assert.doesNotMatch(source, /\{\s*x:\s*0\.5,\s*y:\s*0\.5\s*\}/);
  assert.match(source, /useRef<\{ x: number; y: number \} \| null>\(null\)/);
  const guards = source.match(/if \([^)]*lastPosRef\.current[^)]*\)/g) ?? [];
  assert.ok(
    guards.length >= 4,
    `expected at least 4 lastPosRef guards in absolute.tsx, got ${guards.length}`
  );
});

test('hybrid mouse releases do not emit redundant absolute positioning frames', () => {
  const source = fs.readFileSync(
    path.resolve(import.meta.dirname, '../src/pages/desktop/mouse/absolute.tsx'),
    'utf8'
  );

  assert.match(source, /if \(wasStylus && lastPosRef\.current\)/);
  const releaseCase = source.match(/default:[\s\S]*?return;/)?.[0] ?? '';
  assert.notEqual(releaseCase, '');
  assert.match(releaseCase, /if \(stylus && lastPosRef\.current\)[\s\S]*?sendPointerState/);

  const pointerLeaveRelease = source.match(
    /function releaseAbsolutePointer\(\) \{[\s\S]*?\n  \}/
  )?.[0] ?? '';
  assert.notEqual(pointerLeaveRelease, '');
  assert.match(pointerLeaveRelease, /if \(!outputStylusRef\.current \|\| !lastPosRef\.current\) return;/);
});

test('pointer components relinquish their routes after release', () => {
  const mouseDir = path.resolve(import.meta.dirname, '../src/pages/desktop/mouse');
  const read = (file) => fs.readFileSync(path.join(mouseDir, file), 'utf8');

  const absolute = read('absolute.tsx');
  const absoluteCleanup = absolute.match(/return \(\) => \{[\s\S]*?\n    \};/)?.[0] ?? '';
  assert.notEqual(absoluteCleanup, '');
  assert.match(
    absoluteCleanup,
    /releaseAllMouseButtons\(\);[\s\S]*?clearLongPressTimer\(\);[\s\S]*?sendPointerRelease\(/
  );
  assert.match(
    absolute,
    /function relinquishPointerRoutes\(\) \{[\s\S]*?outputStylusRef\.current[\s\S]*?INPUT_ROUTER_RELEASE_STYLUS[\s\S]*?INPUT_ROUTER_RELEASE_ABSOLUTE_POINTER/
  );
  const absolutePointerLeave =
    absolute.match(/if \(event\.type === 'pointerleave'\) \{[\s\S]*?\n      \}/)?.[0] ?? '';
  assert.notEqual(absolutePointerLeave, '');
  assert.match(
    absolutePointerLeave,
    /releaseAbsolutePointer\(\);[\s\S]*?sendPointerRelease\(event\.source/
  );

  const relative = read('relative.tsx');
  const relativeCleanup = relative.match(/return \(\) => \{[\s\S]*?\n    \};/)?.[0] ?? '';
  assert.notEqual(relativeCleanup, '');
  assert.match(relativeCleanup, /releaseState\(\);[\s\S]*?sendPointerRelease\(/);
  assert.match(relativeCleanup, /INPUT_ROUTER_RELEASE_RELATIVE_POINTER/);
  const relativePointerLeave =
    relative.match(/if \(event\.type === 'pointerleave'\) \{[\s\S]*?\n      \}/)?.[0] ?? '';
  assert.notEqual(relativePointerLeave, '');
  assert.match(relativePointerLeave, /buttonsRef\.current\.value === 0/);
  assert.match(
    relativePointerLeave,
    /buttonsRef\.current\.value === 0[\s\S]*?sendPointerRelease\(event\.source, INPUT_ROUTER_RELEASE_RELATIVE_POINTER\)/
  );

  const touchScreen = read('touch-screen.tsx');
  const touchScreenCleanup = touchScreen.match(/return \(\) => \{[\s\S]*?\n    \};/)?.[0] ?? '';
  assert.notEqual(touchScreenCleanup, '');
  assert.match(
    touchScreenCleanup,
    /clearContacts\(\);[\s\S]*?sendPointerRelease\([\s\S]*?INPUT_ROUTER_RELEASE_TOUCHSCREEN/
  );
});
