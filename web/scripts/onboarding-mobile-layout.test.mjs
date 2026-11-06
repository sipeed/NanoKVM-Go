import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const webRoot = path.resolve(import.meta.dirname, '..');
const mobileTourFile = path.join(webRoot, 'src/pages/desktop/onboarding/mobile/tour.tsx');
const desktopTourFile = path.join(webRoot, 'src/pages/desktop/onboarding/tour.tsx');
const mobileStepsFile = path.join(webRoot, 'src/pages/desktop/onboarding/mobile/steps.ts');
const messagesFile = path.join(webRoot, 'src/i18n/onboarding.ts');
const localeFile = path.join(webRoot, 'src/i18n/locale.ts');
const stylesFile = path.join(webRoot, 'src/assets/styles/index.css');

// mobile/steps.ts imports ../steps.ts, so the loader follows relative
// requires (same pattern as onboarding-mobile-gates.test.mjs); bare/aliased
// ids fail loudly.
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

const mobileTour = fs.readFileSync(mobileTourFile, 'utf8');
const desktopTour = fs.readFileSync(desktopTourFile, 'utf8');
const styles = fs.readFileSync(stylesFile, 'utf8');

// Extract a flat CSS rule block by its selector (the mobile-tour rules have
// no nested braces). The block opens at the next '{' after the selector, so
// comma-grouped selectors spanning lines resolve to their shared block;
// ordered rules keep prefix selectors (footer vs its descendants) distinct.
function cssRule(selector) {
  const start = styles.indexOf(selector);
  assert.ok(start !== -1, `missing CSS rule for ${selector}`);
  const open = styles.indexOf('{', start);
  const close = styles.indexOf('}', open);
  return styles.slice(open + 1, close);
}

// The 268px footer row cannot fit antd's locale Prev/Next/Finish (fr "Étape
// précédente") or the root sentence-style skip, so the mobile tour owns four
// short labels under onboarding.mobile.tour (see i18n/onboarding.ts).
test('mobile tour consumes the four owned button labels', () => {
  for (const key of ['skip', 'prev', 'next', 'end']) {
    assert.ok(
      mobileTour.includes(`t('onboarding.mobile.tour.${key}')`),
      `mobile tour must consume the literal key onboarding.mobile.tour.${key}`
    );
  }
  // the labels reach the buttons through the step-level button props
  assert.match(mobileTour, /prevButtonProps/);
  assert.match(mobileTour, /nextButtonProps/);
  // the root sentence-style skip belongs to the desktop tour only
  assert.ok(
    !mobileTour.includes("t('onboarding.skip')"),
    'mobile tour must not consume the root onboarding.skip key'
  );
});

test('desktop tour keeps the root skip and no mobile.tour keys', () => {
  assert.ok(
    desktopTour.includes("t('onboarding.skip')"),
    'desktop tour keeps the root onboarding.skip key'
  );
  assert.ok(!desktopTour.includes('mobile.tour'), 'desktop tour must not reference mobile.tour');
  // the desktop popover stays free of a custom indicator renderer (R4)
  assert.ok(!desktopTour.includes('indicatorsRender'), 'desktop tour must not add indicatorsRender');
});

test('mobile tour footer stacks into indicator and button rows', () => {
  // full selector chains beat antd's cssinjs footer rules
  // (.ant-tour .ant-tour-inner .ant-tour-footer (0,3,0), containers (0,4,0),
  // the per-button margin (0,5,0)) without injection-order luck
  const footerSelector = '.nanokvm-mobile-tour.ant-tour .ant-tour-inner .ant-tour-footer';

  const footer = cssRule(footerSelector);
  assert.match(footer, /flex-direction:\s*column/);
  assert.match(footer, /row-gap:\s*8px/);

  // dots stay left: the footer's text-align: end would right-align them once
  // the column container stretches the indicators full width
  const indicators = cssRule(`${footerSelector} .ant-tour-indicators`);
  assert.match(indicators, /text-align:\s*start/);

  const buttons = cssRule(`${footerSelector} .ant-tour-buttons`);
  assert.match(buttons, /display:\s*flex/);
  assert.match(buttons, /flex-wrap:\s*nowrap/);
  assert.match(buttons, /justify-content:\s*flex-end/);
  assert.match(buttons, /column-gap:\s*8px/);
  assert.match(buttons, /margin-inline-start:\s*0/);

  // the row never wraps and never shrink-clips a label: an overflowing
  // locale must fail loudly, not truncate silently
  const buttonItems = cssRule(`${footerSelector} .ant-tour-buttons .ant-btn`);
  assert.match(buttonItems, /margin-inline-start:\s*0/);
  assert.match(buttonItems, /flex-shrink:\s*0/);
});

test('mobile tour buttons keep 44px touch targets at 8px inline padding', () => {
  // 8px keeps the three-button row inside the 268px content width for all 20
  // locales (probed headless: worst middle-step trio de ≈ 237px); the 300px
  // popover geometry stays guard-locked in onboarding-mobile-steps.test.mjs
  const nextBtn = cssRule('.nanokvm-mobile-tour .ant-tour-next-btn,');
  assert.match(nextBtn, /min-height:\s*44px/);
  assert.match(nextBtn, /padding-inline:\s*8px/);
  assert.doesNotMatch(nextBtn, /padding-inline:\s*16px/);

  const footerBtn = cssRule('.nanokvm-mobile-tour .ant-tour-footer .ant-btn');
  assert.match(footerBtn, /min-height:\s*44px/);
  assert.match(footerBtn, /padding-inline:\s*8px/);
});

test('welcome stays first and finish last (label wiring preconditions)', () => {
  const { deriveMobileOnboardingSteps } = loadTsModule(mobileStepsFile);
  const steps = deriveMobileOnboardingSteps({
    disabledItems: [],
    videoMode: 'h264',
    screenTimelapseEnabled: false
  });
  // the exact full order is guarded in onboarding-mobile-steps.test.mjs; here
  // only the boundary anchors the label wiring relies on
  assert.equal(steps[0].key, 'welcome', 'welcome must stay step 0 (boundary label wiring)');
  assert.equal(steps[steps.length - 1].key, 'finish', 'finish must stay the last step (end label)');
});

test('mobile.tour owns exactly four short labels with skip under the root sentence', () => {
  // the welcome relabel retired tour.start: the namespace owns exactly
  // skip/prev/next/end, and the short skip must stay clearly shorter than the
  // root sentence form or the 268px footer row fit silently breaks
  const onboardingMessages = loadTsModule(messagesFile).default;
  // same source-of-truth extraction as onboarding-messages.test.mjs
  const localeSource = fs.readFileSync(localeFile, 'utf8');
  const langMatch = localeSource.match(/supportedLanguageKeys\s*=\s*\[([\s\S]*?)\]\s*as const/);
  assert.ok(langMatch, 'unable to find supportedLanguageKeys in locale.ts');
  const langs = [...langMatch[1].matchAll(/'(\w+)'/g)].map((entry) => entry[1]);
  assert.equal(langs.length, 20, 'onboarding messages must cover all 20 locales');
  for (const lang of langs) {
    const entry = onboardingMessages(lang);
    assert.deepEqual(
      Object.keys(entry.mobile.tour).sort(),
      ['end', 'next', 'prev', 'skip'],
      `${lang}: mobile.tour must own exactly skip/prev/next/end`
    );
    assert.ok(
      entry.mobile.tour.skip.length <= entry.skip.length,
      `${lang}: mobile.tour.skip must not exceed the root skip length`
    );
  }
});
