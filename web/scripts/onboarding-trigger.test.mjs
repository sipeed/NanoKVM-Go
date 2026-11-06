import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const webRoot = path.resolve(import.meta.dirname, '..');
const offerFile = path.join(webRoot, 'src/pages/desktop/onboarding/offer.ts');
const controllerFile = path.join(webRoot, 'src/pages/desktop/onboarding/index.tsx');
const replayFile = path.join(webRoot, 'src/pages/desktop/menu/settings/appearance/menu-tour.tsx');
const appearanceFile = path.join(webRoot, 'src/pages/desktop/menu/settings/appearance/index.tsx');
const desktopFile = path.join(webRoot, 'src/pages/desktop/index.tsx');

function loadOfferModule() {
  const output = ts.transpileModule(fs.readFileSync(offerFile, 'utf8'), {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
    fileName: offerFile
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
    { filename: offerFile }
  );
  return module.exports;
}

const { shouldOfferOnboarding } = loadOfferModule();

const OFFER_BASE = {
  isMobileDevice: false,
  isMobilePortrait: false,
  videoMode: 'h264-webrtc',
  menuConfigInitialized: true,
  completedVersion: '',
  laterDismissed: false,
  tourActive: false,
  currentVersion: 'v1'
};

test('offer predicate: offers only a first-run desktop with a loaded rail', () => {
  assert.equal(shouldOfferOnboarding(OFFER_BASE), true);

  // mobile devices never see the tour (owner rule)
  assert.equal(shouldOfferOnboarding({ ...OFFER_BASE, isMobileDevice: true }), false);
  // mobile portrait is owned by the mobile tour (mutual exclusion), even on
  // non-mobile user agents (large portrait touchscreens render the mobile rail)
  assert.equal(shouldOfferOnboarding({ ...OFFER_BASE, isMobilePortrait: true }), false);
  // rail not mounted yet / enablement not loaded
  assert.equal(shouldOfferOnboarding({ ...OFFER_BASE, videoMode: null }), false);
  assert.equal(shouldOfferOnboarding({ ...OFFER_BASE, menuConfigInitialized: false }), false);
  // fail-closed on unknown completion state
  assert.equal(shouldOfferOnboarding({ ...OFFER_BASE, completedVersion: null }), false);
  // same version already completed
  assert.equal(
    shouldOfferOnboarding({ ...OFFER_BASE, completedVersion: 'v1' }),
    false
  );
  // an older completed version re-offers (future tour revisions)
  assert.equal(shouldOfferOnboarding({ ...OFFER_BASE, completedVersion: 'v0' }), true);
  // session dismissal and a running tour suppress the prompt
  assert.equal(shouldOfferOnboarding({ ...OFFER_BASE, laterDismissed: true }), false);
  assert.equal(shouldOfferOnboarding({ ...OFFER_BASE, tourActive: true }), false);
});

test('first-run controller wires fetch, prompt actions, and persistence', () => {
  const controller = fs.readFileSync(controllerFile, 'utf8');

  // fetches the device-level completion state once, never on mobile or in
  // mobile portrait (the mobile tour owns that form factor). Membership
  // assertion, not chain order.
  assert.match(controller, /getOnboarding\(\)/);
  const fetchGuard = controller.match(/if \(([^)]*fetchedRef\.current[^)]*)\) return;/);
  assert.ok(fetchGuard, 'fetch-once guard exists');
  assert.match(fetchGuard[1], /isMobileDevice/);
  assert.match(fetchGuard[1], /isMobilePortrait/);
  assert.match(controller, /typeof rsp\.data\?\.completedVersion === 'string'/);

  // offer gate feeds the prompt atom in both directions: the mirror closes
  // any residue as well (a false→true flip after tour end must not reopen it)
  assert.match(controller, /shouldOfferOnboarding\(/);
  assert.match(controller, /setPromptOpen\(offered\)/);

  // completion is optimistic: the local version is set BEFORE the POST so the
  // offered gate cannot flip false→true while the request is in flight
  const persist = controller.match(/function persistCompleted\(\) \{[\s\S]*?\n  \}/);
  assert.ok(persist, 'persistCompleted exists');
  const optimisticAt = persist[0].indexOf('setCompletedVersion(ONBOARDING_VERSION)');
  const postAt = persist[0].indexOf('setOnboardingCompleted(ONBOARDING_VERSION)');
  assert.ok(optimisticAt !== -1 && postAt !== -1, 'persistCompleted updates both');
  assert.ok(optimisticAt < postAt, 'optimistic update precedes the POST');

  // Start goes through the guarded atom (which closes the prompt itself)
  assert.match(controller, /startOnboarding\(\)/);

  // Later is session-only: dismissal atom + close, no persistence call
  const later = controller.match(/function handleLater\(\) \{[\s\S]*?\n  \}/);
  assert.ok(later, 'handleLater exists');
  assert.match(later[0], /setLaterDismissed\(true\)/);
  assert.match(later[0], /setPromptOpen\(false\)/);
  assert.doesNotMatch(later[0], /persistCompleted|setOnboardingCompleted/);

  // Never and tour end persist the current version (best-effort POST)
  const never = controller.match(/function handleNever\(\) \{[\s\S]*?\n  \}/);
  assert.ok(never, 'handleNever exists');
  assert.match(never[0], /persistCompleted\(\)/);
  const tourEnd = controller.match(/function handleTourEnd\(\) \{[\s\S]*?\n  \}/);
  assert.ok(tourEnd, 'handleTourEnd exists');
  assert.match(tourEnd[0], /persistCompleted\(\)/);

  // the prompt never renders on mobile, in mobile portrait, or over a running tour
  assert.match(controller, /promptOpen && !tourActive && !isMobileDevice && !isMobilePortrait/);

  // shield + tour are hosted by the controller
  assert.match(controller, /<OnboardingShield \/>/);
  assert.match(controller, /<OnboardingTour onEnd={handleTourEnd} \/>/);

  // user-visible copy stays translated; the card uses a named layer constant
  for (const key of ['prompt.title', 'prompt.description', 'prompt.language', 'prompt.never', 'prompt.later', 'prompt.start']) {
    assert.match(controller, new RegExp(`t\\('onboarding\\.${key}'\\)`));
  }
  // the pre-tour language choice reuses the shared picker inside the card
  assert.match(controller, /import \{ LanguagePicker \} from '@\/components\/language-picker\.tsx'/);
  assert.match(controller, /<LanguagePicker \/>/);
  assert.match(controller, /zIndex: LAYER_DESKTOP_MENU_POPOVER/);
  assert.doesNotMatch(controller, /zIndex: \d/);
});

test('replay row mirrors the settings modal close and hides only in mobile landscape', () => {
  const replay = fs.readFileSync(replayFile, 'utf8');
  const appearance = fs.readFileSync(appearanceFile, 'utf8');

  // hidden only where neither tour can run: a mobile device outside portrait
  // (mobile landscape). The mobile-portrait form factor replays its own tour,
  // and the desktop row is unchanged. Membership assertion, not chain order.
  const replayGate = replay.match(/if \(([^)]*)\) return null;/);
  assert.ok(replayGate, 'replay visibility gate exists');
  assert.match(replayGate[1], /isMobileDevice/);
  assert.match(replayGate[1], /isMobilePortrait/);

  // closes the settings modal with the same atom sequence as closeModal
  // before starting the tour (the modal would cover the spotlighted rail)
  for (const call of [
    /setIsKeyboardEnable\(true\)/,
    /setIsSettingsOpen\(false\)/,
    /setIsMobileDetailOpen\(false\)/,
    /setSettingTab\('about'\)/,
    /setSubmenuOpenCount\(\(count\) => Math\.max\(0, count - 1\)\)/,
    /startOnboarding\(\)/
  ]) {
    assert.match(replay, call);
  }

  // replay never depends on the completion flag
  assert.doesNotMatch(replay, /completedVersion|getOnboarding/);

  // translated copy and appearance-tab mount
  assert.match(replay, /t\('onboarding\.replay\.label'\)/);
  assert.match(replay, /t\('onboarding\.replay\.description'\)/);
  assert.match(replay, /t\('onboarding\.replay\.action'\)/);
  assert.match(appearance, /<MenuTour \/>/);
});

test('desktop mounts the onboarding controller exactly once', () => {
  const desktop = fs.readFileSync(desktopFile, 'utf8');
  assert.equal(desktop.match(/<Onboarding \/>/g)?.length, 1);
  // shield and tour moved under the controller; no direct mounts remain
  assert.doesNotMatch(desktop, /<OnboardingShield \/>/);
  assert.doesNotMatch(desktop, /<OnboardingTour/);
});
