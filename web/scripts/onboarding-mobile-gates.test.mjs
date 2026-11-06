import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const webRoot = path.resolve(import.meta.dirname, '..');
const mobileStepsFile = path.join(webRoot, 'src/pages/desktop/onboarding/mobile/steps.ts');
const offerFile = path.join(webRoot, 'src/pages/desktop/onboarding/offer.ts');
const atomsFile = path.join(webRoot, 'src/jotai/onboarding.ts');
const mobileControllerFile = path.join(
  webRoot,
  'src/pages/desktop/onboarding/mobile/index.tsx'
);
const desktopControllerFile = path.join(webRoot, 'src/pages/desktop/onboarding/index.tsx');

// mobile/steps.ts imports ../steps.ts, so the loader must follow relative
// requires (the running specifier keeps its .ts extension). Bare/aliased ids
// stay unresolvable and fail loudly, matching the single-module loaders.
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

const moduleCache = new Map();
const { shouldOfferMobileOnboarding, MOBILE_ONBOARDING_VERSION } = loadTsModule(
  mobileStepsFile,
  moduleCache
);
const { shouldOfferOnboarding } = loadTsModule(offerFile, moduleCache);

// The all-permissive mobile input. The backend/mock default completion value
// is the empty string (never completed); null means "unknown" and must stay
// fail-closed — the two are distinct rows below.
const MOBILE_OFFER_BASE = {
  isMobilePortrait: true,
  videoMode: 'h264-webrtc',
  menuConfigInitialized: true,
  mobileCompletedVersion: '',
  laterDismissed: false,
  tourActive: false
};

const DESKTOP_OFFER_BASE = {
  isMobileDevice: false,
  isMobilePortrait: false,
  videoMode: 'h264-webrtc',
  menuConfigInitialized: true,
  completedVersion: '',
  laterDismissed: false,
  tourActive: false,
  currentVersion: 'v1'
};

test('mobile offer predicate: every fail-closed branch and the open path', () => {
  assert.equal(shouldOfferMobileOnboarding(MOBILE_OFFER_BASE), true);

  // the mobile tour owns the touch-portrait form factor exclusively
  assert.equal(
    shouldOfferMobileOnboarding({ ...MOBILE_OFFER_BASE, isMobilePortrait: false }),
    false
  );
  // the rail only exists once a video mode is selected ('' is a real falsy
  // state value, distinct from null)
  assert.equal(shouldOfferMobileOnboarding({ ...MOBILE_OFFER_BASE, videoMode: null }), false);
  assert.equal(shouldOfferMobileOnboarding({ ...MOBILE_OFFER_BASE, videoMode: '' }), false);
  // menu enablement must be loaded so derived steps match the visible rail
  assert.equal(
    shouldOfferMobileOnboarding({ ...MOBILE_OFFER_BASE, menuConfigInitialized: false }),
    false
  );
  // fail-closed on unknown completion state (not fetched / fetch failed)
  assert.equal(
    shouldOfferMobileOnboarding({ ...MOBILE_OFFER_BASE, mobileCompletedVersion: null }),
    false
  );
  // same mobile version already completed
  assert.equal(
    shouldOfferMobileOnboarding({
      ...MOBILE_OFFER_BASE,
      mobileCompletedVersion: MOBILE_ONBOARDING_VERSION
    }),
    false
  );
  // namespace separation: a completed desktop tour must not suppress the
  // mobile one, and an older mobile version re-offers (future revisions)
  assert.equal(
    shouldOfferMobileOnboarding({ ...MOBILE_OFFER_BASE, mobileCompletedVersion: 'v1' }),
    true
  );
  assert.equal(
    shouldOfferMobileOnboarding({ ...MOBILE_OFFER_BASE, mobileCompletedVersion: 'mobile-v0' }),
    true
  );
  // session dismissal and a running tour suppress the prompt
  assert.equal(
    shouldOfferMobileOnboarding({ ...MOBILE_OFFER_BASE, laterDismissed: true }),
    false
  );
  assert.equal(shouldOfferMobileOnboarding({ ...MOBILE_OFFER_BASE, tourActive: true }), false);
  // precedence: an unknown version stays fail-closed even when dismissed, and
  // the portrait gate dominates an otherwise permissive input
  assert.equal(
    shouldOfferMobileOnboarding({
      ...MOBILE_OFFER_BASE,
      mobileCompletedVersion: null,
      laterDismissed: true
    }),
    false
  );
  assert.equal(
    shouldOfferMobileOnboarding({
      ...MOBILE_OFFER_BASE,
      isMobilePortrait: false,
      mobileCompletedVersion: ''
    }),
    false
  );
});

test('desktop gate tightening regression: portrait excluded, desktop path intact', () => {
  // Full desktop branch coverage lives in onboarding-trigger.test.mjs; this
  // locks the C4 tightening itself.
  assert.equal(shouldOfferOnboarding(DESKTOP_OFFER_BASE), true);
  // a non-mobile-UA large touchscreen in portrait renders the mobile rail, so
  // the desktop tour must yield even though isMobileDevice is false
  assert.equal(shouldOfferOnboarding({ ...DESKTOP_OFFER_BASE, isMobilePortrait: true }), false);
  assert.equal(
    shouldOfferOnboarding({ ...DESKTOP_OFFER_BASE, isMobileDevice: true, isMobilePortrait: true }),
    false
  );
  // the new clause must not over-tighten the existing exclusions
  assert.equal(shouldOfferOnboarding({ ...DESKTOP_OFFER_BASE, isMobileDevice: true }), false);
  assert.equal(
    shouldOfferOnboarding({ ...DESKTOP_OFFER_BASE, completedVersion: 'v1' }),
    false
  );
});

test('desktop and mobile offer gates can never both open', () => {
  // Non-vacuity guards: the invariant below is meaningless if a refactor
  // turns both predicates into always-false.
  assert.equal(shouldOfferOnboarding(DESKTOP_OFFER_BASE), true);
  assert.equal(shouldOfferMobileOnboarding(MOBILE_OFFER_BASE), true);

  for (const isMobileDevice of [false, true]) {
    for (const isMobilePortrait of [false, true]) {
      for (const videoMode of [null, 'h264']) {
        for (const menuConfigInitialized of [true, false]) {
          for (const completedVersion of [null, '', 'v1']) {
            for (const mobileCompletedVersion of [null, '', 'mobile-v1']) {
              for (const laterDismissed of [true, false]) {
                for (const tourActive of [true, false]) {
                  const desktop = shouldOfferOnboarding({
                    isMobileDevice,
                    isMobilePortrait,
                    videoMode,
                    menuConfigInitialized,
                    completedVersion,
                    laterDismissed,
                    tourActive,
                    currentVersion: 'v1'
                  });
                  const mobile = shouldOfferMobileOnboarding({
                    isMobilePortrait,
                    videoMode,
                    menuConfigInitialized,
                    mobileCompletedVersion,
                    laterDismissed,
                    tourActive
                  });
                  assert.ok(
                    !(desktop && mobile),
                    `both gates opened: ${JSON.stringify({
                      isMobileDevice,
                      isMobilePortrait,
                      videoMode,
                      menuConfigInitialized,
                      completedVersion,
                      mobileCompletedVersion,
                      laterDismissed,
                      tourActive
                    })}`
                  );
                }
              }
            }
          }
        }
      }
    }
  }
});

test('mobile prompt atoms stay separate from the desktop pair', () => {
  const atoms = fs.readFileSync(atomsFile, 'utf8');
  const mobileController = fs.readFileSync(mobileControllerFile, 'utf8');
  const desktopController = fs.readFileSync(desktopControllerFile, 'utf8');

  // Separate session atoms so the two controllers' mirror effects can never
  // cross-talk (the desktop controller forces its prompt closed in portrait).
  assert.match(atoms, /export const onboardingMobilePromptOpenAtom = atom\(false\);/);
  assert.match(atoms, /export const onboardingMobileLaterDismissedAtom = atom\(false\);/);
  assert.doesNotMatch(atoms, /onboardingMobilePromptOpenAtom\s*=\s*onboardingPromptOpenAtom/);
  assert.doesNotMatch(atoms, /onboardingMobileLaterDismissedAtom\s*=\s*onboardingLaterDismissedAtom/);

  // Cross-talk guard: neither controller references the other form factor's
  // prompt atoms. (The Mobile prefix makes these substrings distinct.)
  assert.doesNotMatch(
    desktopController,
    /onboardingMobilePromptOpenAtom|onboardingMobileLaterDismissedAtom/
  );
  assert.doesNotMatch(mobileController, /onboardingPromptOpenAtom|onboardingLaterDismissedAtom/);

  // Both controllers share the single tour-active atom and the guarded entry,
  // so the shield and input suspension cover the mobile tour automatically.
  assert.match(mobileController, /onboardingActiveAtom/);
  assert.match(mobileController, /startOnboardingAtom/);
});

test('mobile controller wires the gate, fail-closed fetch, and persisted completion', () => {
  const controller = fs.readFileSync(mobileControllerFile, 'utf8');

  // fetches the device-level completion state once, only in mobile portrait
  assert.match(controller, /getOnboarding\(\)/);
  assert.match(controller, /if \(!isMobilePortrait \|\| fetchedRef\.current\) return;/);
  assert.match(controller, /typeof rsp\.data\?\.mobileCompletedVersion === 'string'/);

  // offer gate feeds the prompt atom in both directions (mirror contract)
  assert.match(controller, /shouldOfferMobileOnboarding\(/);
  assert.match(controller, /setPromptOpen\(offered\)/);

  // completion is optimistic: the local version is set BEFORE the POST so the
  // offered gate cannot flip false→true while the request is in flight
  const persist = controller.match(/function persistCompleted\(\) \{[\s\S]*?\n  \}/);
  assert.ok(persist, 'persistCompleted exists');
  const optimisticAt = persist[0].indexOf('setCompletedVersion(MOBILE_ONBOARDING_VERSION)');
  const postAt = persist[0].indexOf('setMobileOnboardingCompleted(MOBILE_ONBOARDING_VERSION)');
  assert.ok(optimisticAt !== -1 && postAt !== -1, 'persistCompleted updates both');
  assert.ok(optimisticAt < postAt, 'optimistic update precedes the POST');

  // a rotation abort must not persist: the early return precedes the write
  const tourEnd = controller.match(/function handleTourEnd\([\s\S]*?\n  \}/);
  assert.ok(tourEnd, 'handleTourEnd exists');
  const abortAt = tourEnd[0].indexOf("if (reason === 'abort') return;");
  const persistAt = tourEnd[0].indexOf('persistCompleted()');
  assert.ok(abortAt !== -1 && persistAt !== -1, 'handleTourEnd handles abort and persistence');
  assert.ok(abortAt < persistAt, 'abort guard precedes persistCompleted');

  // Start is conditional on the guarded atom: only a real start closes the
  // prompt, so the user can retry once an exclusive input session ends
  assert.match(controller, /if \(startOnboarding\(\)\) \{/);

  // Later is session-only: dismissal atom + close, no persistence call
  const later = controller.match(/function handleLater\(\) \{[\s\S]*?\n  \}/);
  assert.ok(later, 'handleLater exists');
  assert.match(later[0], /setLaterDismissed\(true\)/);
  assert.match(later[0], /setPromptOpen\(false\)/);
  assert.doesNotMatch(later[0], /persistCompleted|setMobileOnboardingCompleted/);

  // Never persists the current mobile version (best-effort POST)
  const never = controller.match(/function handleNever\(\) \{[\s\S]*?\n  \}/);
  assert.ok(never, 'handleNever exists');
  assert.match(never[0], /persistCompleted\(\)/);

  // the prompt never renders in landscape, over a running tour, outside
  // mobile portrait, or over the settings Modal; the card uses a named layer
  // constant that clears the rail (1000) yet stays under Select/Dropdown
  // popups (1050) so its LanguagePicker menu remains clickable
  assert.match(controller, /promptOpen && !tourActive && !settingsOpen && isMobilePortrait/);
  assert.match(controller, /isSettingsOpenAtom/);
  assert.match(controller, /zIndex: LAYER_ONBOARDING_MOBILE_PROMPT/);
  assert.doesNotMatch(controller, /zIndex: \d/);
  const layers = fs.readFileSync(path.join(webRoot, 'src/lib/layers.ts'), 'utf8');
  assert.match(layers, /LAYER_ONBOARDING_MOBILE_PROMPT = 1040/);

  // every touch target on the card meets the 44px portrait constraint: the
  // three action buttons and the pre-tour language picker
  const touchTargets = controller.match(/min-h-\[44px\]/g) ?? [];
  assert.ok(touchTargets.length >= 4, 'prompt card buttons and picker meet 44px');

  // user-visible copy stays translated across the mobile namespace and the
  // shared prompt keys
  for (const key of [
    'mobile.prompt.title',
    'mobile.prompt.description',
    'prompt.language',
    'prompt.never',
    'prompt.later',
    'prompt.start'
  ]) {
    assert.match(controller, new RegExp(`t\\('onboarding\\.${key.replaceAll('.', '\\.')}'\\)`));
  }
  // the pre-tour language choice reuses the shared picker inside the card
  assert.match(controller, /import \{ LanguagePicker \} from '@\/components\/language-picker\.tsx'/);
  assert.match(controller, /<LanguagePicker className="min-h-\[44px\]" \/>/);

  // the tour is hosted by the controller with the end-reason callback
  assert.match(controller, /<OnboardingMobileTour onEnd={handleTourEnd} \/>/);
});
