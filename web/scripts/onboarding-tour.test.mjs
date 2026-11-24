import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const webRoot = path.resolve(import.meta.dirname, '..');
const stepsFile = path.join(webRoot, 'src/pages/desktop/onboarding/steps.ts');
const anchorFile = path.join(webRoot, 'src/pages/desktop/onboarding/anchor.ts');
const tourFile = path.join(webRoot, 'src/pages/desktop/onboarding/tour.tsx');
const menuFile = path.join(webRoot, 'src/pages/desktop/menu/index.tsx');
const desktopFile = path.join(webRoot, 'src/pages/desktop/index.tsx');

function loadStepsModule() {
  const output = ts.transpileModule(fs.readFileSync(stepsFile, 'utf8'), {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
    fileName: stepsFile
  }).outputText;
  const module = { exports: {} };
  vm.runInNewContext(
    output,
    {
      exports: module.exports,
      module,
      require(id) {
        throw new Error(`unexpected module: ${id}`);
      }
    },
    { filename: stepsFile }
  );
  return module.exports;
}

const { deriveOnboardingSteps } = loadStepsModule();

const ALL_ITEMS = [
  'mouseJiggler',
  'pictureInPicture',
  'volume',
  'microphone',
  'image',
  'script',
  'terminal',
  'wol',
  'screenTimelapse',
  'power',
  'fullscreen',
  'collapse'
];

// The module under test runs in a vm sandbox; its arrays carry the sandbox
// realm's prototype, so normalize through JSON before strict comparison.
function keysOf(steps) {
  return JSON.parse(JSON.stringify(steps.map((step) => step.key)));
}

function stepByKey(steps, key) {
  const step = steps.find((entry) => entry.key === key);
  return step ? JSON.parse(JSON.stringify(step)) : undefined;
}

test('default rail (power disabled, non-WebRTC) yields the full grouped tour', () => {
  const steps = deriveOnboardingSteps({
    disabledItems: ['power'],
    videoMode: 'h264',
    screenTimelapseEnabled: false,
    isMobilePortrait: false
  });

  assert.deepEqual(keysOf(steps), ['welcome', 'display', 'input', 'tools', 'system', 'finish']);
  assert.deepEqual(stepByKey(steps, 'display').items, ['screen', 'pictureInPicture']);
  assert.deepEqual(stepByKey(steps, 'tools').items, ['image', 'script', 'terminal', 'wol']);
  assert.deepEqual(stepByKey(steps, 'system').items, ['settings', 'fullscreen', 'collapse']);
  assert.equal(stepByKey(steps, 'welcome').group, undefined);
  assert.equal(stepByKey(steps, 'finish').group, undefined);
});

test('WebRTC video mode adds the media step with both audio items', () => {
  const steps = deriveOnboardingSteps({
    disabledItems: ['power'],
    videoMode: 'h265-webrtc',
    screenTimelapseEnabled: false,
    isMobilePortrait: false
  });
  assert.deepEqual(stepByKey(steps, 'media').items, ['volume', 'microphone']);
});

test('media step mirrors the rail: dropped without WebRTC, trimmed per item', () => {
  const noWebRTC = deriveOnboardingSteps({
    disabledItems: ['power'],
    videoMode: 'h264',
    screenTimelapseEnabled: false,
    isMobilePortrait: false
  });
  assert.equal(stepByKey(noWebRTC, 'media'), undefined);

  const volumeOnly = deriveOnboardingSteps({
    disabledItems: ['power', 'microphone'],
    videoMode: 'h264-webrtc',
    screenTimelapseEnabled: false,
    isMobilePortrait: false
  });
  assert.deepEqual(stepByKey(volumeOnly, 'media').items, ['volume']);

  const bothDisabled = deriveOnboardingSteps({
    disabledItems: ['power', 'volume', 'microphone'],
    videoMode: 'h264-webrtc',
    screenTimelapseEnabled: false,
    isMobilePortrait: false
  });
  assert.equal(stepByKey(bothDisabled, 'media'), undefined);
});

test('automation step follows timelapse state and the power item', () => {
  const base = {
    disabledItems: ['power'],
    videoMode: 'h264',
    screenTimelapseEnabled: false,
    isMobilePortrait: false
  };
  assert.equal(stepByKey(deriveOnboardingSteps(base), 'automation'), undefined);

  const timelapse = deriveOnboardingSteps({ ...base, screenTimelapseEnabled: true });
  assert.deepEqual(stepByKey(timelapse, 'automation').items, ['screenTimelapse']);

  const power = deriveOnboardingSteps({ ...base, disabledItems: [] });
  assert.deepEqual(stepByKey(power, 'automation').items, ['power']);

  const both = deriveOnboardingSteps({
    disabledItems: [],
    videoMode: 'h264',
    screenTimelapseEnabled: true,
    isMobilePortrait: false
  });
  assert.deepEqual(stepByKey(both, 'automation').items, ['screenTimelapse', 'power']);

  // The service flag cannot resurrect a disabled timelapse item.
  const itemDisabled = deriveOnboardingSteps({
    disabledItems: ['power', 'screenTimelapse'],
    videoMode: 'h264',
    screenTimelapseEnabled: true,
    isMobilePortrait: false
  });
  assert.equal(stepByKey(itemDisabled, 'automation'), undefined);
});

test('everything disabled still yields welcome, display, input, system, finish', () => {
  const steps = deriveOnboardingSteps({
    disabledItems: [...ALL_ITEMS],
    videoMode: null,
    screenTimelapseEnabled: false,
    isMobilePortrait: false
  });
  assert.deepEqual(keysOf(steps), ['welcome', 'display', 'input', 'system', 'finish']);
  assert.deepEqual(stepByKey(steps, 'display').items, ['screen']);
  assert.deepEqual(stepByKey(steps, 'system').items, ['settings']);
});

test('mobile portrait rail drops the desktop-only system items', () => {
  const steps = deriveOnboardingSteps({
    disabledItems: ['power'],
    videoMode: 'h264',
    screenTimelapseEnabled: false,
    isMobilePortrait: true
  });
  assert.deepEqual(stepByKey(steps, 'system').items, ['settings']);
});

test('tour engine wiring follows the shield and layering contracts', () => {
  const tour = fs.readFileSync(tourFile, 'utf8');
  const anchor = fs.readFileSync(anchorFile, 'utf8');
  const menu = fs.readFileSync(menuFile, 'utf8');
  const desktop = fs.readFileSync(desktopFile, 'utf8');

  // Mobile exclusion: the controller is inert on mobile devices and in mobile
  // portrait (the mobile tour owns that form factor).
  assert.match(tour, /isMobileDevice/);
  assert.match(tour, /active && !isMobileDevice && !isMobilePortrait/);

  // Router suspension is registered in a passive effect with the tour owner.
  assert.match(tour, /useEffect[\s\S]*?suspendInput\('onboarding'\)/);

  // Layer discipline: named constants, never bare numbers.
  assert.match(tour, /zIndex={LAYER_ONBOARDING}/);
  assert.doesNotMatch(tour, /zIndex={\d/);
  assert.match(tour, /mask=\{\{ color: 'rgba\(8,8,12,\.62\)' \}\}/);
  assert.match(tour, /disabledInteraction/);

  // Translated skip action injected into the panel's action row.
  assert.match(tour, /actionsRender=/);
  assert.match(tour, /t\('onboarding\.skip'\)/);

  // Capture-phase keyboard navigation with propagation isolation.
  assert.match(tour, /addEventListener\('keydown', handleKeyDown, true\)/);
  assert.match(tour, /removeEventListener\('keydown', handleKeyDown, true\)/);
  assert.match(tour, /stopImmediatePropagation\(\)/);
  for (const key of ['Escape', 'ArrowRight', 'ArrowLeft', 'Enter']) {
    assert.match(tour, new RegExp(`'${key}'`));
  }

  // Focus management: move to the primary action on open, restore on close.
  assert.match(tour, /ant-tour \.ant-tour-next-btn|\.ant-tour-next-btn/);
  assert.match(tour, /isConnected/);

  // Anchor measurement: group unions measure all wrappers' children and never
  // a single node; the mobile rail box is one real element and may use a
  // single-node query.
  assert.match(anchor, /querySelectorAll\(/);
  assert.match(anchor, /wrapper\.children/);
  const groupMeasure = anchor.match(/measureMenuGroupRect[\s\S]*?\n}/);
  assert.ok(groupMeasure, 'measureMenuGroupRect exists');
  assert.doesNotMatch(groupMeasure[0], /querySelector\(/);
  assert.match(anchor, /querySelector\('\[data-menu-rail="mobile"\]'\)/);

  // The measurement proxy remounts per group AND per resize tick so rc-tour
  // re-measures on every step transition and viewport change (same-node
  // identity would freeze the mask hole). String-membership assertion: the
  // template-literal source resists regex escaping.
  const expectedKey = 'key={' + '`' + '${stepDef?.group}:${measureTick}' + '`' + '}';
  assert.ok(tour.includes(expectedKey), 'proxy must be keyed by group and measureTick');

  // The finish/skip reason survives rc-tour calling onClose inside onFinish:
  // the skip path defers to a microtask so the synchronous finish wins.
  assert.match(tour, /queueMicrotask\(/);

  // Re-measure on viewport resize while pinned.
  assert.match(tour, /addEventListener\('resize'/);
  assert.match(tour, /removeEventListener\('resize'/);

  // The controller is mounted inside the onboarding host component.
  const host = fs.readFileSync(
    path.join(webRoot, 'src/pages/desktop/onboarding/index.tsx'),
    'utf8'
  );
  assert.match(host, /<OnboardingTour onEnd={handleTourEnd} \/>/);
  assert.match(desktop, /<Onboarding \/>/);

  // The shield mirrors the mobile gate (never blocks input where the tour
  // cannot run). Membership assertion, not chain order.
  const shield = fs.readFileSync(
    path.join(webRoot, 'src/pages/desktop/onboarding/shield.tsx'),
    'utf8'
  );
  const shieldGate = shield.match(/useAtomValue\(onboardingActiveAtom\) && \(([^)]*)\)/);
  assert.ok(shieldGate, 'shield gates on the shared atom AND a form-factor union');
  assert.match(shieldGate[1], /isMobilePortrait/);
  assert.match(shieldGate[1], /!isMobileDevice/);

  // rc-tour reuses rootClassName on the full-screen mask and the target
  // placeholder, so mobile popover geometry must scope to the popover element
  // (which also carries ant-tour); a bare .nanokvm-mobile-tour max-width clamps
  // the mask to a narrow band instead of covering the viewport.
  const styles = fs.readFileSync(path.join(webRoot, 'src/assets/styles/index.css'), 'utf8');
  assert.match(styles, /\.nanokvm-mobile-tour\.ant-tour\s*\{/);
  assert.doesNotMatch(styles, /\.nanokvm-mobile-tour\s*\{/);

  // The timelapse flag is shared through the settings atom module.
  assert.match(menu, /useAtom\(\s*screenTimelapseEnabledAtom\s*\)/);
});

test('tour steps type-check against the i18n module shape', () => {
  // steps.ts item keys must exist in the i18n module's item registries.
  const stepsSource = fs.readFileSync(stepsFile, 'utf8');
  const i18nSource = fs.readFileSync(path.join(webRoot, 'src/i18n/onboarding.ts'), 'utf8');
  for (const item of [
    'screen',
    'pictureInPicture',
    'volume',
    'microphone',
    'mouseJiggler',
    'keyboard',
    'mouse',
    'image',
    'script',
    'terminal',
    'wol',
    'screenTimelapse',
    'power',
    'settings',
    'fullscreen',
    'collapse'
  ]) {
    assert.match(i18nSource, new RegExp(`${item}: `), `i18n module missing item ${item}`);
    assert.match(stepsSource, new RegExp(`'${item}'`), `step derivation missing item ${item}`);
  }
});
