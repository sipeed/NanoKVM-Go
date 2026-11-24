import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const webRoot = path.resolve(import.meta.dirname, '..');
const onboardingFile = path.join(webRoot, 'src/i18n/onboarding.ts');
const localeFile = path.join(webRoot, 'src/i18n/locale.ts');

// The finger-robot menu item (persisted menu key "power") takes its visible
// label from finger-robot-text.ts; the automation step must name that label.
// Load the real module so a future rename breaks this test.
const fingerRobotTextFile = path.join(
  webRoot,
  'src/pages/desktop/menu/finger-robot/finger-robot-text.ts'
);

function loadTsModule(file, mocks = {}) {
  const output = ts.transpileModule(fs.readFileSync(file, 'utf8'), {
    compilerOptions: {
      esModuleInterop: true,
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022
    },
    fileName: file
  }).outputText;
  const module = { exports: {} };

  vm.runInNewContext(
    output,
    {
      exports: module.exports,
      module,
      require(id) {
        if (id in mocks) return mocks[id];
        throw new Error(`unexpected module: ${id}`);
      }
    },
    { filename: file }
  );
  return module.exports;
}

function loadOnboardingModule() {
  return loadTsModule(onboardingFile).default;
}

function loadFingerRobotText() {
  const localeStub = { normalizeLanguage: (language) => language };
  const mod = loadTsModule(fingerRobotTextFile, { '@/i18n/locale.ts': localeStub });
  assert.ok(typeof mod.getFingerRobotText === 'function', 'unable to load finger-robot text module');
  return mod.getFingerRobotText;
}

const fingerRobotText = loadFingerRobotText();

function supportedLanguages() {
  const source = fs.readFileSync(localeFile, 'utf8');
  const match = source.match(/supportedLanguageKeys\s*=\s*\[([\s\S]*?)\]\s*as const/);
  assert.ok(match, 'unable to find supportedLanguageKeys in locale.ts');
  return [...match[1].matchAll(/'([^']+)'/g)].map((entry) => entry[1]);
}

function flattenLeaves(value, prefix = '') {
  const entries = {};
  for (const [key, child] of Object.entries(value)) {
    const path = prefix ? `${prefix}.${key}` : key;
    if (typeof child === 'string') {
      entries[path] = child;
    } else {
      Object.assign(entries, flattenLeaves(child, path));
    }
  }
  return entries;
}

const onboardingMessages = loadOnboardingModule();
const languages = supportedLanguages();

test('the messages record declares exactly the supported languages', () => {
  // Source-level declaration check: the en fallback in the default export
  // would mask a deleted locale map at runtime, so assert the declared keys.
  const source = fs.readFileSync(onboardingFile, 'utf8');
  const recordMatch = source.match(
    /Record<SupportedLanguage, OnboardingMessages> = \{([\s\S]*?)\n\};/
  );
  assert.ok(recordMatch, 'unable to locate the messages record');
  const declared = [...recordMatch[1].matchAll(/^ {2}([a-z_]+): \{/gm)].map((entry) => entry[1]);
  assert.deepEqual([...declared].sort(), [...languages].sort());
});

test('onboarding messages resolve for every supported language with en fallback', () => {
  const en = onboardingMessages('en');
  assert.ok(en, 'english messages must exist');

  for (const language of languages) {
    assert.ok(onboardingMessages(language), `missing onboarding messages for ${language}`);
  }

  // Unknown languages fall back to English.
  assert.equal(onboardingMessages('xx'), en);
});

test('every locale has the complete, non-empty key set of English', () => {
  const reference = flattenLeaves(onboardingMessages('en'));

  for (const language of languages) {
    const leaves = flattenLeaves(onboardingMessages(language));
    assert.deepEqual(
      Object.keys(leaves).sort(),
      Object.keys(reference).sort(),
      `${language} key set diverges from en`
    );
    for (const [key, value] of Object.entries(leaves)) {
      assert.ok(value.trim().length > 0, `${language}:${key} must not be empty`);
    }
  }
});

test('no locale is derived from English with a spread', () => {
  const source = fs.readFileSync(onboardingFile, 'utf8');
  assert.doesNotMatch(source, /\.\.\.en\b/);
  assert.doesNotMatch(source, /\.\.\.messages\.en\b/);
});

test('stable technical identifiers survive translation', () => {
  for (const language of languages) {
    const messages = onboardingMessages(language);
    assert.match(messages.prompt.title, /NanoKVM/, `${language} prompt.title keeps NanoKVM`);
    assert.match(messages.steps.tools.items.script, /KVM/, `${language} tools.script keeps KVM`);
    assert.match(
      messages.steps.tools.items.terminal,
      /KVM/,
      `${language} tools.terminal keeps KVM`
    );
  }

  // Wake-on-LAN is a stable identifier only where the locale keeps the English
  // menu label; zh/zh_tw translate it (网络唤醒 / 網路喚醒).
  assert.match(onboardingMessages('en').steps.tools.items.wol, /Wake-on-LAN/);
  assert.match(onboardingMessages('zh').steps.tools.items.wol, /网络唤醒/);
  assert.match(onboardingMessages('zh_tw').steps.tools.items.wol, /網路喚醒/);
});

test('the automation step names the finger-robot label the rail actually shows', () => {
  for (const language of languages) {
    const title = fingerRobotText(language)?.title;
    assert.ok(title, `finger-robot text missing a title for ${language}`);
    assert.ok(
      onboardingMessages(language).steps.automation.items.power.includes(title),
      `${language} automation power line must name "${title}"`
    );
  }
});

// --- Item-line ↔ rail-label mirror ------------------------------------------
//
// Every step item line must name the exact label the rail shows in that
// locale, quoted in the locale file's own convention. Labels resolve from the
// canonical sources (locale module titles, the
// screen-timelapse service label, the finger-robot text module) — never from
// a re-translation inside the onboarding module.


function loadLocaleTranslation(language) {
  const mod = loadTsModule(path.join(webRoot, `src/i18n/locales/${language}.ts`));
  return mod.default.translation;
}

// Verified by codepoint dump of the locale files: pl/hu close with U+201D,
// de/cz with U+201C; fr keeps inner spaces inside « … » (trim before compare).
const ONBOARDING_QUOTE_PAIRS = {
  en: ['"', '"'],
  zh: ['「', '」'],
  zh_tw: ['「', '」'],
  ja: ['「', '」'],
  ko: ['"', '"'],
  de: ['„', '“'],
  fr: ['«', '»'],
  es: ['«', '»'],
  it: ['"', '"'],
  nl: ['"', '"'],
  pl: ['„', '”'],
  ru: ['«', '»'],
  uk: ['«', '»'],
  cz: ['„', '“'],
  da: ['"', '"'],
  hu: ['„', '”'],
  id: ['"', '"'],
  nb: ['«', '»'],
  th: ['"', '"'],
  vi: ['"', '"']
};

const ONBOARDING_ITEM_LABELS = [
  ['display', 'screen', ({ t }) => t.screen.title],
  ['display', 'pictureInPicture', ({ t }) => t.pictureInPicture.title],
  ['media', 'volume', ({ t }) => t.volume.title],
  ['media', 'microphone', ({ t }) => t.microphone.title],
  ['input', 'mouseJiggler', ({ t }) => t.settings.device.mouseJiggler.title],
  ['input', 'keyboard', ({ t }) => t.keyboard.title],
  ['input', 'mouse', ({ t }) => t.mouse.title],
  ['input', 'preview', ({ t }) => t.preview.title],
  ['tools', 'image', ({ t }) => t.image.title],
  ['tools', 'script', ({ t }) => t.script.title],
  ['tools', 'terminal', ({ t }) => t.terminal.title],
  ['tools', 'wol', ({ t }) => t.wol.title],
  ['automation', 'screenTimelapse', ({ t }) => t.settings.ai.screenTimelapseService],
  ['automation', 'power', ({ robotTitle }) => robotTitle],
  ['system', 'settings', ({ t }) => t.settings.title],
  ['system', 'fullscreen', ({ t }) => t.fullscreen.toggle],
  ['system', 'collapse', ({ t }) => t.menu.collapse]
];

function quotedSpans(line, open, close) {
  const spans = [];
  if (open === close) {
    line.split(open).forEach((part, index) => {
      if (index % 2 === 1) spans.push(part);
    });
    return spans;
  }
  let rest = line;
  while (rest.includes(open)) {
    const start = rest.indexOf(open) + open.length;
    const end = rest.indexOf(close, start);
    if (end === -1) break;
    spans.push(rest.slice(start, end));
    rest = rest.slice(end + close.length);
  }
  return spans;
}

test('every item line names the rail label as a quoted span', () => {
  for (const language of languages) {
    const pair = ONBOARDING_QUOTE_PAIRS[language];
    assert.ok(pair, `missing quote pair for ${language}`);
    const t = loadLocaleTranslation(language);
    const sources = {
      t,
      robotTitle: fingerRobotText(language)?.title
    };
    const messages = onboardingMessages(language);

    for (const [group, item, resolve] of ONBOARDING_ITEM_LABELS) {
      const label = resolve(sources);
      assert.ok(label, `${language}: no canonical label for ${group}.${item}`);
      const line = messages.steps[group].items[item];
      assert.ok(line, `${language}: onboarding line missing for ${group}.${item}`);
      const spans = quotedSpans(line, pair[0], pair[1]);
      assert.ok(
        spans.some((span) => span.trim() === label),
        `${language} ${group}.${item} must quote the exact rail label "${label}" (got: ${line})`
      );
    }
  }
});

test('the welcome card carries the input-shield notice in every locale', () => {
  // AC: users must be told input is paused during the tour. A length floor is
  // the language-neutral proxy — a one-liner cannot carry the notice. CJK copy
  // is compact, so the floor sits below the shortest CJK description (~70).
  for (const language of languages) {
    const { welcome } = onboardingMessages(language);
    assert.ok(
      welcome.description.length >= 60,
      `${language} welcome.description is too short to carry the shield notice`
    );
  }
});
