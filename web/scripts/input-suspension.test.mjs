import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const inputRouterFile = path.resolve(import.meta.dirname, '../src/lib/input-router.ts');
const inputSuspensionFile = path.resolve(import.meta.dirname, '../src/lib/input-suspension.ts');
const controllerFile = path.resolve(
  import.meta.dirname,
  '../src/pages/desktop/pointer-mode-controller.tsx'
);
const selectionFile = path.resolve(import.meta.dirname, '../src/hooks/usePointerModeSelection.ts');

function transpile(file) {
  const source = fs.readFileSync(file, 'utf8');
  return ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
    fileName: file
  }).outputText;
}

function loadInputRouter() {
  const frames = [];
  const module = { exports: {} };

  vm.runInNewContext(
    transpile(inputRouterFile),
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
              on() {
                return () => {};
              },
              onConnectionState() {
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

  return { ...module.exports, frames };
}

function loadInputSuspension() {
  const router = loadInputRouter();
  const module = { exports: {} };

  vm.runInNewContext(
    transpile(inputSuspensionFile),
    {
      exports: module.exports,
      module,
      require(specifier) {
        if (specifier === '@/lib/input-router.ts') return router;
        throw new Error(`Unexpected require(${specifier})`);
      }
    },
    { filename: inputSuspensionFile }
  );

  return { suspension: module.exports, router };
}

test('base enable state drives the router while unsuspended', () => {
  const { suspension, router } = loadInputSuspension();
  const sender = new router.InputRouterClient();

  assert.equal(router.isInputRouterEnabled(), true);
  assert.equal(suspension.isInputSuspended(), false);

  suspension.setBaseInputEnabled(false);
  assert.equal(router.isInputRouterEnabled(), false);
  assert.equal(sender.sendRelativePointer(router.InputSource.BrowserPointer, 1), false);
  assert.equal(router.frames.length, 0);

  suspension.setBaseInputEnabled(true);
  assert.equal(router.isInputRouterEnabled(), true);
  assert.equal(sender.sendRelativePointer(router.InputSource.BrowserPointer, 0), true);
  assert.equal(router.frames.length, 1);
});

test('a held suspension drops every input source until release', () => {
  const { suspension, router } = loadInputSuspension();
  const sender = new router.InputRouterClient();

  const release = suspension.suspendInput('onboarding');
  assert.equal(router.isInputRouterEnabled(), false);
  assert.equal(suspension.isInputSuspended(), true);

  for (const source of [
    router.InputSource.BrowserKeyboard,
    router.InputSource.BrowserPointer,
    router.InputSource.BrowserTouchscreen,
    router.InputSource.VirtualKeyboard,
    router.InputSource.VirtualPointer,
    router.InputSource.WebHID
  ]) {
    assert.equal(sender.sendRelativePointer(source, 1), false);
  }
  assert.equal(router.frames.length, 0);

  release();
  assert.equal(router.isInputRouterEnabled(), true);
  assert.equal(suspension.isInputSuspended(), false);
});

test('base re-derives and unmount-style resets cannot stomp a held suspension', () => {
  const { suspension, router } = loadInputSuspension();

  const release = suspension.suspendInput('onboarding');
  suspension.setBaseInputEnabled(false);
  assert.equal(router.isInputRouterEnabled(), false);

  // Pointer-mode controller unmount historically forced the router back on.
  suspension.setBaseInputEnabled(true);
  assert.equal(router.isInputRouterEnabled(), false);

  // Release restores the current base state, not a stale snapshot.
  release();
  assert.equal(router.isInputRouterEnabled(), true);
});

test('release restores a still-disabled base state instead of forcing input on', () => {
  const { suspension, router } = loadInputSuspension();

  suspension.setBaseInputEnabled(false);
  const release = suspension.suspendInput('onboarding');
  release();

  assert.equal(router.isInputRouterEnabled(), false);
  suspension.setBaseInputEnabled(true);
  assert.equal(router.isInputRouterEnabled(), true);
});

test('interleaved owners require every holder to release', () => {
  const { suspension, router } = loadInputSuspension();

  const releaseTour = suspension.suspendInput('onboarding');
  const releaseCalibration = suspension.suspendInput('calibration');
  assert.equal(router.isInputRouterEnabled(), false);

  releaseTour();
  assert.equal(router.isInputRouterEnabled(), false);
  assert.equal(suspension.isInputSuspended(), true);

  releaseCalibration();
  assert.equal(router.isInputRouterEnabled(), true);
});

test('release handles are idempotent', () => {
  const { suspension, router } = loadInputSuspension();

  const first = suspension.suspendInput('onboarding');
  const second = suspension.suspendInput('calibration');
  first();
  first();
  assert.equal(router.isInputRouterEnabled(), false);

  second();
  assert.equal(router.isInputRouterEnabled(), true);
});

test('suspendInput rejects an empty owner', () => {
  const { suspension, router } = loadInputSuspension();

  assert.throws(() => suspension.suspendInput(''), /non-empty owner/);
  assert.equal(router.isInputRouterEnabled(), true);
});

test('former setInputRouterEnabled writers migrate to the suspension facade', () => {
  const controller = fs.readFileSync(controllerFile, 'utf8');
  const selection = fs.readFileSync(selectionFile, 'utf8');
  const suspensionSource = fs.readFileSync(inputSuspensionFile, 'utf8');

  // All former writers express base state through the suspension module.
  assert.doesNotMatch(controller, /setInputRouterEnabled/);
  assert.doesNotMatch(selection, /setInputRouterEnabled/);
  assert.match(controller, /setBaseInputEnabled/);
  assert.match(selection, /setInputEnabled: setBaseInputEnabled/);

  // The router's boolean stays the single funnel, driven only by the facade
  // (assert membership, not operand ordering, per the source-shape guard rules).
  assert.match(suspensionSource, /setInputRouterEnabled\(/);
  assert.match(suspensionSource, /baseEnabled && suspensionOwners\.size === 0/);
  assert.equal(suspensionSource.match(/setInputRouterEnabled\(/g)?.length, 1);
});

test('a held suspension drops the full HID surface for every producer type', () => {
  const { suspension, router } = loadInputSuspension();
  const sender = new router.InputRouterClient();
  const release = suspension.suspendInput('onboarding');

  const sources = [
    router.InputSource.BrowserKeyboard,
    router.InputSource.BrowserPointer,
    router.InputSource.BrowserTouchscreen,
    router.InputSource.VirtualKeyboard,
    router.InputSource.VirtualPointer,
    router.InputSource.WebHID
  ];

  for (const source of sources) {
    assert.equal(sender.sendKeyboard(source, new Uint8Array(8)), false, `keyboard ${source}`);
    assert.equal(sender.sendRelativePointer(source, 1, 4, -4, 1), false, `relative ${source}`);
    assert.equal(sender.sendAbsolutePointer(source, 1, 0.5, 0.5), false, `absolute ${source}`);
    assert.equal(sender.sendStylus(source, 1, 0.5, 0.5, 128), false, `stylus ${source}`);
    assert.equal(
      sender.sendTouchscreen(source, [{ id: 1, touching: true, x: 0.5, y: 0.5 }]),
      false,
      `touchscreen ${source}`
    );
    assert.equal(sender.sendPointerRelease(source, 1), false, `release ${source}`);
  }
  assert.equal(router.frames.length, 0, 'zero HID frames at the WS boundary while suspended');

  release();
  assert.equal(sender.sendRelativePointer(router.InputSource.BrowserPointer, 0), true);
  assert.equal(router.frames.length, 1);
});

// C3 check note N3: sendBrowserInputCapture telemetry frames are a debug
// channel (MessageEvent.InputCaptureDebug), not HID reports, and bypass the
// router gate by design. Pin that exception so it is never mistaken for an
// HID leak — the gate must drop InputRouter frames while letting the
// explicitly enabled debug channel through unchanged.
test('capture-debug telemetry bypasses the suspension gate by design', () => {
  const { suspension, router } = loadInputSuspension();
  const sender = new router.InputRouterClient();

  router.setInputCaptureDebugEnabled(true);
  const release = suspension.suspendInput('onboarding');

  assert.equal(sender.sendRelativePointer(router.InputSource.BrowserPointer, 1), false);
  assert.equal(
    router.sendBrowserPointerCapture({ kind: 'pointer', detail: 'probe' }),
    true,
    'debug channel stays open while HID is suspended'
  );
  assert.equal(router.frames.length, 1);
  assert.equal(router.frames[0][0], 4, 'only MessageEvent.InputCaptureDebug frames flow');
  assert.notEqual(router.frames[0][0], 3, 'no MessageEvent.InputRouter frame leaks');

  release();
  router.setInputCaptureDebugEnabled(false);
});
