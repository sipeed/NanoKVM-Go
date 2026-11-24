import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const webRoot = path.resolve(import.meta.dirname, '..');
const mobileStepsFile = path.join(webRoot, 'src/pages/desktop/onboarding/mobile/steps.ts');
const stepsFile = path.join(webRoot, 'src/pages/desktop/onboarding/steps.ts');
const mobileTourFile = path.join(webRoot, 'src/pages/desktop/onboarding/mobile/tour.tsx');
const menuFile = path.join(webRoot, 'src/pages/desktop/menu/index.tsx');
const stylesFile = path.join(webRoot, 'src/assets/styles/index.css');

// Same relative-require loader as onboarding-mobile-gates.test.mjs:
// mobile/steps.ts imports ../steps.ts with the .ts extension intact.
function loadTsModule(entryFile, cache = new Map()) {
  const resolved = path.resolve(entryFile);
  if (cache.has(resolved)) return cache.get(resolved).exports;
  const module = { exports: {} };
  cache.set(resolved, module);
  const output = ts.transpileModule(fs.readFileSync(resolved, 'utf8'), {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
    fileName: resolved
  }).outputText;
  vm.runInNewContext(
    output,
    {
      exports: module.exports,
      module,
      require(id) {
        if (typeof id !== 'string' || !id.startsWith('.')) {
          throw new Error(`unexpected module: ${id}`);
        }
        const target = path.resolve(
          path.dirname(resolved),
          id.endsWith('.ts') ? id : `${id}.ts`
        );
        return loadTsModule(target, cache);
      }
    },
    { filename: resolved }
  );
  return module.exports;
}

// One shared cache: the desktop module instance held here is the exact object
// the mobile module required, so cross-module behavior tests stay honest.
const moduleCache = new Map();
const { deriveMobileOnboardingSteps, MOBILE_ONBOARDING_VERSION } = loadTsModule(
  mobileStepsFile,
  moduleCache
);
const { deriveOnboardingSteps, ONBOARDING_VERSION } = loadTsModule(stepsFile, moduleCache);

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

// The modules under test run in a vm sandbox; their arrays carry the sandbox
// realm's prototype, so normalize through JSON before strict comparison.
const plain = (value) => JSON.parse(JSON.stringify(value));
const keysOf = (steps) => plain(steps.map((step) => step.key));
function stepByKey(steps, key) {
  const step = steps.find((entry) => entry.key === key);
  return step ? plain(step) : undefined;
}

test('version constants pin the desktop/mobile completion namespaces', () => {
  assert.equal(MOBILE_ONBOARDING_VERSION, 'mobile-v1');
  assert.equal(ONBOARDING_VERSION, 'v1');
  // the namespaces must never collapse into one (mutual suppression bug)
  assert.notEqual(MOBILE_ONBOARDING_VERSION, ONBOARDING_VERSION);
});

test('default portrait rail yields welcome, rail, grouped steps, finish in order', () => {
  const steps = deriveMobileOnboardingSteps({
    disabledItems: ['power'],
    videoMode: 'h264',
    screenTimelapseEnabled: false
  });

  // Exact order, not membership: the derivation re-wraps the desktop list via
  // slice(1, -1), so a wrong slice regresses the order loudly.
  assert.deepEqual(keysOf(steps), [
    'welcome',
    'rail',
    'display',
    'input',
    'tools',
    'system',
    'finish'
  ]);
  assert.deepEqual(stepByKey(steps, 'welcome'), { key: 'welcome', items: [] });
  assert.deepEqual(stepByKey(steps, 'rail'), { key: 'rail', anchor: 'rail', items: [] });
  assert.deepEqual(stepByKey(steps, 'display'), {
    key: 'display',
    anchor: 'display',
    items: ['screen', 'portal', 'pictureInPicture']
  });
  assert.deepEqual(stepByKey(steps, 'input'), {
    key: 'input',
    anchor: 'input',
    items: ['keyboard', 'mouse', 'mouseJiggler']
  });
  assert.deepEqual(stepByKey(steps, 'tools'), {
    key: 'tools',
    anchor: 'tools',
    items: ['image', 'script', 'terminal', 'wol']
  });
  assert.deepEqual(stepByKey(steps, 'system'), {
    key: 'system',
    anchor: 'system',
    items: ['settings']
  });
  assert.deepEqual(stepByKey(steps, 'finish'), { key: 'finish', items: [] });
  assert.equal(stepByKey(steps, 'media'), undefined);
  assert.equal(stepByKey(steps, 'automation'), undefined);
});

test('centered steps are anchorless and anchored steps carry their own key', () => {
  const steps = deriveMobileOnboardingSteps({
    disabledItems: [],
    videoMode: 'h265-webrtc',
    screenTimelapseEnabled: true
  });
  for (const step of steps) {
    if (step.key === 'welcome' || step.key === 'finish') {
      assert.equal(step.anchor, undefined, `${step.key} must stay a centered card`);
    } else {
      // locks the anchor: def.group mapping in the derivation
      assert.equal(step.anchor, step.key, `${step.key} anchors its own rail group`);
    }
  }
});

test('media step follows the WebRTC gate and per-item enablement', () => {
  // the video mode dominates: no WebRTC, no media step at all
  const noWebRTC = deriveMobileOnboardingSteps({
    disabledItems: [],
    videoMode: 'h264',
    screenTimelapseEnabled: false
  });
  assert.equal(stepByKey(noWebRTC, 'media'), undefined);

  const webrtc = deriveMobileOnboardingSteps({
    disabledItems: [],
    videoMode: 'h265-webrtc',
    screenTimelapseEnabled: false
  });
  assert.deepEqual(stepByKey(webrtc, 'media'), {
    key: 'media',
    anchor: 'media',
    items: ['volume', 'microphone']
  });
  // media sits between display and input, mirroring the rail layout
  const keys = keysOf(webrtc);
  assert.ok(keys.indexOf('media') > keys.indexOf('display'));
  assert.ok(keys.indexOf('media') < keys.indexOf('input'));

  // per-item trimming
  const volumeOnly = deriveMobileOnboardingSteps({
    disabledItems: ['power', 'microphone'],
    videoMode: 'h264-webrtc',
    screenTimelapseEnabled: false
  });
  assert.deepEqual(stepByKey(volumeOnly, 'media').items, ['volume']);

  const bothDisabled = deriveMobileOnboardingSteps({
    disabledItems: ['power', 'volume', 'microphone'],
    videoMode: 'h264-webrtc',
    screenTimelapseEnabled: false
  });
  assert.equal(stepByKey(bothDisabled, 'media'), undefined);
});

test('tools and automation steps drop when no item is enabled', () => {
  const noTools = deriveMobileOnboardingSteps({
    disabledItems: ['power', 'image', 'script', 'terminal', 'wol'],
    videoMode: 'h264',
    screenTimelapseEnabled: false
  });
  assert.equal(stepByKey(noTools, 'tools'), undefined);

  const someTools = deriveMobileOnboardingSteps({
    disabledItems: ['power', 'image', 'script'],
    videoMode: 'h264',
    screenTimelapseEnabled: false
  });
  assert.deepEqual(stepByKey(someTools, 'tools').items, ['terminal', 'wol']);

  // power (Finger Robot) alone fills the automation step
  const powerOnly = deriveMobileOnboardingSteps({
    disabledItems: [],
    videoMode: 'h264',
    screenTimelapseEnabled: false
  });
  assert.deepEqual(stepByKey(powerOnly, 'automation').items, ['power']);

  // the AI service flag cannot resurrect a disabled timelapse item
  const timelapseDisabled = deriveMobileOnboardingSteps({
    disabledItems: ['screenTimelapse'],
    videoMode: 'h264',
    screenTimelapseEnabled: true
  });
  assert.deepEqual(stepByKey(timelapseDisabled, 'automation').items, ['power']);

  const bothAutomation = deriveMobileOnboardingSteps({
    disabledItems: [],
    videoMode: 'h264',
    screenTimelapseEnabled: true
  });
  assert.deepEqual(stepByKey(bothAutomation, 'automation').items, ['screenTimelapse', 'power']);

  const noAutomation = deriveMobileOnboardingSteps({
    disabledItems: ['power', 'screenTimelapse'],
    videoMode: 'h264',
    screenTimelapseEnabled: true
  });
  assert.equal(stepByKey(noAutomation, 'automation'), undefined);
});

test('portrait system step never lists fullscreen or collapse', () => {
  // disabledItems: [] is deliberate: fullscreen/collapse are enabled here, so
  // only the forced portrait flag can strip them from the system step.
  const steps = deriveMobileOnboardingSteps({
    disabledItems: [],
    videoMode: 'h264',
    screenTimelapseEnabled: false
  });
  assert.deepEqual(stepByKey(steps, 'system').items, ['settings']);

  // contrast: the desktop derivation keeps the desktop-only rail items, so the
  // stripping above is the mobile wrapper's doing, not the shared derivation
  const desktopSteps = deriveOnboardingSteps({
    disabledItems: [],
    videoMode: 'h264',
    screenTimelapseEnabled: false,
    isMobilePortrait: false
  });
  assert.deepEqual(stepByKey(desktopSteps, 'system').items, [
    'settings',
    'fullscreen',
    'collapse'
  ]);
});

test('everything disabled still yields the minimal portrait tour', () => {
  const steps = deriveMobileOnboardingSteps({
    disabledItems: [...ALL_ITEMS],
    videoMode: null,
    screenTimelapseEnabled: false
  });
  assert.deepEqual(keysOf(steps), ['welcome', 'rail', 'display', 'input', 'system', 'finish']);
  assert.deepEqual(stepByKey(steps, 'display').items, ['screen', 'portal']);
  assert.deepEqual(stepByKey(steps, 'system').items, ['settings']);
});

test('the forced portrait flag cannot be overridden by the caller', () => {
  // JS callers may copy a desktop input object verbatim (isMobilePortrait and
  // all); the derivation spreads { ...input, isMobilePortrait: true }, so the
  // forced value must win over whatever the caller passed.
  const steps = deriveMobileOnboardingSteps({
    disabledItems: [],
    videoMode: 'h264',
    screenTimelapseEnabled: false,
    isMobilePortrait: false
  });
  assert.deepEqual(stepByKey(steps, 'system').items, ['settings']);
});

test('derivation leaves the disabledItems input untouched', () => {
  const disabledItems = ['power', 'wol'];
  const snapshot = [...disabledItems];
  deriveMobileOnboardingSteps({ disabledItems, videoMode: 'h264', screenTimelapseEnabled: false });
  assert.deepEqual(disabledItems, snapshot);
});

test('mobile tour wiring follows the portrait, layer, and abort contracts', () => {
  const tour = fs.readFileSync(mobileTourFile, 'utf8');
  const menu = fs.readFileSync(menuFile, 'utf8');

  // single derived gate: the shared active atom AND the portrait form factor
  // (membership, not exact formatting, per the source-shape guard rules)
  assert.match(tour, /const enabled = [^;]*\bactive\b[^;]*\bisMobilePortrait\b/);

  // rotation abort: leaving portrait mid-tour ends the tour without writing
  // completion (the offer gate re-prompts back in portrait); persistence lives
  // in the controller, never in the tour
  assert.match(tour, /if \(activeRef\.current && !isMobilePortrait\) \{\s*endTourRef\.current\('abort'\);/);
  assert.match(tour, /export type MobileOnboardingEndReason = 'finish' \| 'skip' \| 'abort';/);
  assert.doesNotMatch(tour, /setMobileOnboardingCompleted|persistCompleted/);

  // the reverse rotation (desktop tour in flight rotating into portrait)
  // restarts from the first mobile step instead of a stale desktop index
  assert.match(tour, /if \(enabled && !wasEnabledRef\.current\) setStep\(0\)/);

  // router-level input suspension, same owner as the desktop tour
  assert.match(tour, /suspendInput\('onboarding'\)/);

  // layer discipline: named constants, never bare numbers
  assert.match(tour, /zIndex={LAYER_ONBOARDING}/);
  assert.match(tour, /zIndex: LAYER_ONBOARDING_RING/);
  assert.doesNotMatch(tour, /zIndex[:=]\s*\{?\d/);

  // antd Tour parity with the desktop contract; no builtinPlacements override
  // (antd replaces user placements internally)
  assert.match(tour, /rootClassName="nanokvm-mobile-tour"/);
  assert.match(tour, /arrow=\{\{ pointAtCenter: true \}\}/);
  assert.match(tour, /mask=\{\{ color: 'rgba\(8,8,12,\.62\)' \}\}/);
  assert.match(tour, /disabledInteraction/);
  assert.doesNotMatch(tour, /builtinPlacements/);

  // translated skip action, suppressed on the last step; the mobile tour owns
  // the short mobile.tour.skip label (the root sentence-style skip and antd's
  // locale Prev/Next/Finish cannot fit the 268px footer row — the full label
  // contract lives in onboarding-mobile-layout.test.mjs)
  assert.match(tour, /actionsRender=/);
  assert.match(tour, /t\('onboarding\.mobile\.tour\.skip'\)/);
  assert.match(tour, /current < total - 1/);

  // rc-tour's per-step default beats the top-level prop, so every step opts
  // out explicitly (the fixed proxy is always in-viewport)
  assert.match(tour, /scrollIntoViewOptions: false/);
  assert.doesNotMatch(tour, /scrollIntoViewOptions: true/);

  // the popover parks on the side opposite the measured rail
  assert.match(tour, /anchorRect\.left < viewportWidth \/ 2/);
  assert.match(tour, /'right' : 'left'/);

  // the measurement proxy remounts per anchor step AND per resize tick so
  // rc-tour re-measures on every transition (string-membership assertion: the
  // template-literal source resists regex escaping)
  const expectedKey = 'key={' + '`' + '${stepDef?.anchor}:${measureTick}' + '`' + '}';
  assert.ok(tour.includes(expectedKey), 'proxy must be keyed by anchor and measureTick');

  // rc-tour ships no dialog semantics; the popover (scoped off the mask by
  // the ant-tour class) gets role/aria-modal, and focus moves to Next
  assert.match(tour, /document\.querySelector\('\.nanokvm-mobile-tour\.ant-tour'\)/);
  assert.match(tour, /setAttribute\('role', 'dialog'\)/);
  assert.match(tour, /setAttribute\('aria-modal', 'true'\)/);
  assert.match(tour, /\.ant-tour-next-btn/);

  // focus capture/restore on activation boundaries
  assert.match(tour, /restoreFocusRef\.current = document\.activeElement/);
  assert.match(tour, /isConnected/);

  // capture-phase keyboard navigation with propagation isolation (external
  // keyboards on phones/tablets get the desktop contract)
  assert.match(tour, /window\.addEventListener\('keydown', handleKeyDown, true\)/);
  assert.match(tour, /window\.removeEventListener\('keydown', handleKeyDown, true\)/);
  assert.match(tour, /stopImmediatePropagation\(\)/);
  for (const key of ['Escape', 'ArrowRight', 'ArrowLeft', 'Enter']) {
    assert.match(tour, new RegExp(`'${key}'`));
  }

  // re-measure on viewport changes: the mobile URL bar collapses without a
  // window resize, so visualViewport is listened to as well
  assert.match(tour, /window\.addEventListener\('resize', handleResize\)/);
  assert.match(tour, /window\.visualViewport\?\.addEventListener\('resize', handleResize\)/);
  assert.match(tour, /window\.visualViewport\?\.removeEventListener\('resize', handleResize\)/);

  // the skip reason survives rc-tour calling onClose inside onFinish: the skip
  // path defers to a microtask so the synchronous finish wins
  assert.match(tour, /queueMicrotask\(/);

  // self-rendered spotlight ring (rc-tour's mask hole has no outline)
  assert.match(tour, /nanokvm-mobile-tour-ring/);

  // the rail anchor attribute the measurement queries: drift silently kills
  // the rail step
  assert.match(menu, /data-menu-rail="mobile"/);
});

test('mobile tour CSS holds the portrait geometry and motion contracts', () => {
  const styles = fs.readFileSync(stylesFile, 'utf8');

  // self-rendered ring: the dark rail needs the extra contrast
  assert.match(styles, /\.nanokvm-mobile-tour-ring \{/);
  assert.match(styles, /outline: 2px solid rgba\(255, 255, 255, 0\.65\);/);
  assert.match(styles, /box-shadow: 0 0 18px 2px rgba\(64, 150, 255, 0\.35\);/);

  // popover geometry scoped to the popover element (rootClassName also lands
  // on the mask/placeholder — the bare-class clamp regression is guarded in
  // onboarding-tour.test.mjs); narrow-viewport cap + bottom safe area
  assert.match(styles, /\.nanokvm-mobile-tour\.ant-tour \{/);
  assert.match(styles, /max-width: min\(300px, calc\(100vw - 32px\)\);/);
  assert.match(styles, /padding-bottom: env\(safe-area-inset-bottom, 0px\);/);

  // 44px touch targets: one rule for the next/prev wrappers, one for the
  // footer buttons (skip included)
  const touchTargets = styles.match(/min-height: 44px/g) ?? [];
  assert.ok(
    touchTargets.length >= 2,
    'mobile tour buttons must meet the 44px touch target in both rules'
  );

  // reduced motion: the tour and its ring drop all transitions/animations
  const reducedMotion = styles.match(/@media \(prefers-reduced-motion: reduce\) \{([\s\S]*?)\n\}/);
  assert.ok(reducedMotion, 'reduced-motion block exists');
  assert.match(reducedMotion[1], /\.nanokvm-mobile-tour/);
  assert.match(reducedMotion[1], /\.nanokvm-mobile-tour-ring/);
  assert.match(reducedMotion[1], /transition: none !important;/);
  assert.match(reducedMotion[1], /animation: none !important;/);
});
