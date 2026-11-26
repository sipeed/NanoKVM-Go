import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { after, before, test } from 'node:test';

import { createStore } from 'jotai/vanilla';
import { createServer } from 'vite';

const webRoot = path.resolve(import.meta.dirname, '..');
const onboardingFile = path.resolve(webRoot, 'src/jotai/onboarding.ts');
const keyboardFile = path.resolve(webRoot, 'src/pages/desktop/keyboard/index.tsx');
const mouseFile = path.resolve(webRoot, 'src/pages/desktop/mouse/index.tsx');
const menuFile = path.resolve(webRoot, 'src/pages/desktop/menu/index.tsx');
const shieldFile = path.resolve(webRoot, 'src/pages/desktop/onboarding/shield.tsx');
const desktopFile = path.resolve(webRoot, 'src/pages/desktop/index.tsx');
const layersFile = path.resolve(webRoot, 'src/lib/layers.ts');
const menuVisibilityFile = path.resolve(webRoot, 'src/hooks/useMenuVisibility.ts');

let vite;
let atoms;
let hadWindow;
let previousWindow;
let hadLocalStorage;
let previousLocalStorage;

before(async () => {
  // lib/http.ts constructs its axios instance at module scope via getBaseUrl(),
  // which reads window.location, and jotai atom initializers read browser
  // storage through lib/localstorage.ts; give the SSR module graph stubs.
  hadWindow = 'window' in globalThis;
  previousWindow = globalThis.window;
  globalThis.window = {
    location: { hostname: '127.0.0.1', port: '', protocol: 'http:' }
  };

  hadLocalStorage = 'localStorage' in globalThis;
  previousLocalStorage = globalThis.localStorage;
  const storageData = new Map();
  globalThis.localStorage = {
    getItem: (key) => (storageData.has(key) ? storageData.get(key) : null),
    setItem: (key, value) => void storageData.set(key, String(value)),
    removeItem: (key) => void storageData.delete(key)
  };

  vite = await createServer({
    root: webRoot,
    appType: 'custom',
    logLevel: 'error',
    server: { middlewareMode: true, hmr: false, ws: false }
  });

  const [onboarding, keyboard, inputProfile, screen, settings] = await Promise.all([
    vite.ssrLoadModule('/src/jotai/onboarding.ts'),
    vite.ssrLoadModule('/src/jotai/keyboard.ts'),
    vite.ssrLoadModule('/src/jotai/input-profile.ts'),
    vite.ssrLoadModule('/src/jotai/screen.ts'),
    vite.ssrLoadModule('/src/jotai/settings.ts')
  ]);
  atoms = { ...onboarding, ...keyboard, ...inputProfile, ...screen, ...settings };
});

after(async () => {
  await vite.close();
  if (hadWindow) {
    globalThis.window = previousWindow;
  } else {
    delete globalThis.window;
  }
  if (hadLocalStorage) {
    globalThis.localStorage = previousLocalStorage;
  } else {
    delete globalThis.localStorage;
  }
});

function silenceWarn(fn) {
  const original = console.warn;
  console.warn = () => {};
  try {
    return fn();
  } finally {
    console.warn = original;
  }
}

test('onboarding atoms default to a fully inactive, uninitialized state', () => {
  const store = createStore();

  assert.equal(store.get(atoms.onboardingActiveAtom), false);
  assert.equal(store.get(atoms.onboardingStepAtom), 0);
  assert.equal(store.get(atoms.onboardingPromptOpenAtom), false);
  assert.equal(store.get(atoms.onboardingLaterDismissedAtom), false);
  assert.equal(store.get(atoms.menuConfigInitializedAtom), false);
  assert.equal(store.get(atoms.onboardingSessionConflictAtom), false);
});

test('keyboardInputEnabledAtom ANDs the shared flag with tour inactivity', () => {
  const store = createStore();

  for (const keyboardEnabled of [true, false]) {
    for (const tourActive of [true, false]) {
      store.set(atoms.isKeyboardEnableAtom, keyboardEnabled);
      store.set(atoms.onboardingActiveAtom, tourActive);
      assert.equal(
        store.get(atoms.keyboardInputEnabledAtom),
        keyboardEnabled && !tourActive,
        `keyboard=${keyboardEnabled} tour=${tourActive}`
      );
    }
  }

  // The derived gate never writes the shared flag back: with the flag held
  // true and the tour active, reading the gate must leave the flag untouched.
  store.set(atoms.isKeyboardEnableAtom, true);
  store.set(atoms.onboardingActiveAtom, true);
  assert.equal(store.get(atoms.keyboardInputEnabledAtom), false);
  assert.equal(store.get(atoms.isKeyboardEnableAtom), true);
});

test('onboardingSessionConflictAtom reports calibration and portal sessions', () => {
  const store = createStore();
  const session = { mode: 'absolute', corners: [] };

  for (const calibration of [null, session]) {
    for (const portalSelecting of [true, false]) {
      store.set(atoms.inputCalibrationSessionAtom, calibration);
      store.set(atoms.portalSelectingAtom, portalSelecting);
      assert.equal(
        store.get(atoms.onboardingSessionConflictAtom),
        calibration !== null || portalSelecting,
        `calibration=${calibration ? 'active' : 'idle'} portal=${portalSelecting}`
      );
    }
  }
});

test('startOnboardingAtom refuses to overlap an exclusive input session', () => {
  const store = createStore();
  store.set(atoms.inputCalibrationSessionAtom, { mode: 'absolute', corners: [] });

  const started = silenceWarn(() => store.set(atoms.startOnboardingAtom));
  assert.equal(started, false);
  assert.equal(store.get(atoms.onboardingActiveAtom), false);
});

test('startOnboardingAtom resets the step, closes the prompt, and activates', () => {
  const store = createStore();
  store.set(atoms.onboardingStepAtom, 4);
  store.set(atoms.onboardingPromptOpenAtom, true);

  const started = store.set(atoms.startOnboardingAtom);
  assert.equal(started, true);
  assert.equal(store.get(atoms.onboardingActiveAtom), true);
  assert.equal(store.get(atoms.onboardingStepAtom), 0);
  assert.equal(store.get(atoms.onboardingPromptOpenAtom), false);
});

test('startOnboardingAtom is a no-op while the tour already runs', () => {
  const store = createStore();
  store.set(atoms.onboardingActiveAtom, true);
  store.set(atoms.onboardingStepAtom, 3);

  const started = store.set(atoms.startOnboardingAtom);
  assert.equal(started, true);
  assert.equal(store.get(atoms.onboardingStepAtom), 3);
});

test('onboarding shield mounts nothing while inactive and blocks host events', () => {
  const shield = fs.readFileSync(shieldFile, 'utf8');
  const desktop = fs.readFileSync(desktopFile, 'utf8');
  const layers = fs.readFileSync(layersFile, 'utf8');

  // Gated on the shared atom; renders nothing while the tour is inactive.
  assert.match(shield, /useAtomValue\(onboardingActiveAtom\)/);
  assert.match(shield, /if \(!active\) return null;/);

  // Layer comes from the named constants, never a bare number.
  assert.match(shield, /zIndex: LAYER_ONBOARDING_BACKDROP/);
  assert.doesNotMatch(shield, /zIndex: \d/);
  assert.match(layers, /LAYER_ONBOARDING_BACKDROP = 1202/);
  assert.match(layers, /LAYER_ONBOARDING = 1210/);

  // Fully transparent (the tour mask owns dimming) and swallows pointer,
  // wheel, and context-menu events. Wheel uses a native non-passive listener
  // because React attaches onWheel passively (preventDefault would be a
  // no-op); the listener is removed on cleanup.
  assert.match(shield, /bg-transparent/);
  for (const handler of [
    'onPointerDown',
    'onPointerMove',
    'onPointerUp',
    'onPointerCancel',
    'onContextMenu'
  ]) {
    assert.match(shield, new RegExp(handler));
  }
  assert.match(shield, /addEventListener\('wheel'/);
  assert.match(shield, /passive: false/);
  assert.match(shield, /removeEventListener\('wheel'/);
  assert.doesNotMatch(shield, /onWheel=/);

  // Mounted through the onboarding host so the shield exists for every video mode.
  const host = fs.readFileSync(
    path.join(webRoot, 'src/pages/desktop/onboarding/index.tsx'),
    'utf8'
  );
  assert.match(host, /<OnboardingShield \/>/);
  assert.match(desktop, /<Onboarding \/>/);
});

test('menu rail pinning is additive and reversible', () => {
  const visibility = fs.readFileSync(menuVisibilityFile, 'utf8');
  const menu = fs.readFileSync(menuFile, 'utf8');

  // The hook consumes the shared atom, forces expanded + visible, and suspends
  // the auto-hide countdown while the tour runs.
  assert.match(visibility, /useAtomValue\(onboardingActiveAtom\)/);
  assert.match(visibility, /setIsMenuExpanded\(true\)/);
  assert.match(visibility, /stopCountdown\(\)/);
  assert.match(visibility, /!state\.onboardingActive/);

  // Deactivation restores the exact pre-tour expanded state (ref-captured),
  // never a hardcoded value.
  assert.match(visibility, /preTourExpandedRef/);
  assert.match(visibility, /setIsMenuExpanded\(restore\)/);

  // Both rails freeze dragging while the tour runs.
  assert.equal(menu.match(/disabled=\{onboardingActive\}/g)?.length, 2);

  // The mobile rail collapse is gated without touching its state.
  assert.match(menu, /isMobileRailCollapsed && !onboardingActive/);

  // Spotlight group anchors exist for every semantic rail group and are
  // layout-neutral (display: contents).
  for (const group of ['display', 'media', 'input', 'tools', 'automation', 'system']) {
    assert.match(menu, new RegExp(`renderGroup\\(\\s*['"]${group}['"]`));
  }
  assert.match(menu, /data-menu-group=\{group\}/);
  assert.match(menu, /className="contents"/);

  // The two conditional automation blocks must use distinct React keys.
  assert.match(menu, /'automation-timelapse'/);
  assert.match(menu, /'automation-power'/);
});

test('desktop input surfaces consume the onboarding gates', () => {
  const keyboard = fs.readFileSync(keyboardFile, 'utf8');
  const mouse = fs.readFileSync(mouseFile, 'utf8');
  const menu = fs.readFileSync(menuFile, 'utf8');
  const onboarding = fs.readFileSync(onboardingFile, 'utf8');

  // Keyboard reads the derived gate, never the shared flag directly; the gate
  // flip re-runs the effect whose cleanup releases held keys (releaseKeys).
  assert.match(keyboard, /useAtomValue\(keyboardInputEnabledAtom\)/);
  assert.doesNotMatch(keyboard, /useAtomValue\(isKeyboardEnableAtom\)/);
  assert.match(keyboard, /releaseKeys\(\)/);

  // Mouse producers are AND-gated at the consumption site (assert membership,
  // not operand ordering, per the source-shape guard rules).
  assert.match(mouse, /useAtomValue\(pointerModeInputEnabledAtom\)/);
  assert.match(mouse, /useAtomValue\(onboardingActiveAtom\)/);
  const inputEnabledDecl = mouse.match(/const inputEnabled = [^;]+;/)?.[0] ?? '';
  assert.match(inputEnabledDecl, /pointerInputEnabled/);
  assert.match(inputEnabledDecl, /!onboardingActive/);

  // The menu's initialized flag is lifted to the shared atom.
  assert.match(menu, /useAtom\(menuConfigInitializedAtom\)/);
  assert.doesNotMatch(menu, /\[isInitialized, [^\]]*\]\s*=\s*useState\(/);

  // The derived gate is read-only: it is defined with a getter only.
  assert.match(onboarding, /keyboardInputEnabledAtom = atom\(\s*\(get\)/);
});
