import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

// Focused coverage for the pre-tour language choice in the first-run
// onboarding prompt: the picker must offer every supported language a
// canonical native label, the prompt card must render the shared picker
// before its action buttons (selection happens before Start), and the shared
// picker must persist the choice only after i18next accepts it.

const webRoot = path.resolve(import.meta.dirname, '..');
const onboardingFile = path.join(webRoot, 'src/i18n/onboarding.ts');
const localeFile = path.join(webRoot, 'src/i18n/locale.ts');
const controllerFile = path.join(webRoot, 'src/pages/desktop/onboarding/index.tsx');
const pickerFile = path.join(webRoot, 'src/components/language-picker.tsx');

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

const onboardingMessages = loadTsModule(onboardingFile).default;

function supportedLanguages() {
  const source = fs.readFileSync(localeFile, 'utf8');
  const match = source.match(/supportedLanguageKeys\s*=\s*\[([\s\S]*?)\]\s*as const/);
  assert.ok(match, 'unable to find supportedLanguageKeys in locale.ts');
  return [...match[1].matchAll(/'([^']+)'/g)].map((entry) => entry[1]);
}

// The canonical "Language" term of a locale is the first `language:` leaf in
// its locales/<lang>.ts module (auth.language; the appearance settings label
// repeats the same word). The onboarding label must reuse it verbatim instead
// of inventing new vocabulary (i18n spec shared-term rule).
function canonicalLanguageWord(language) {
  const source = fs.readFileSync(path.join(webRoot, 'src/i18n/locales', `${language}.ts`), 'utf8');
  const match = source.match(/language:\s*'([^']+)'/);
  assert.ok(match, `no canonical language term found for ${language}`);
  return match[1];
}

test('every locale labels the prompt language row with its canonical term', () => {
  const languages = supportedLanguages();
  assert.equal(languages.length, 20, 'expected the 20-language contract');

  for (const language of languages) {
    const messages = onboardingMessages(language);
    assert.ok(messages?.prompt, `missing prompt messages for ${language}`);
    assert.equal(
      messages.prompt.language,
      canonicalLanguageWord(language),
      `onboarding.prompt.language must reuse the canonical term of ${language}`
    );
  }
});

test('the prompt card renders the shared picker before its action buttons', () => {
  const controller = fs.readFileSync(controllerFile, 'utf8');

  // the card keeps its existing render gate (desktop-only, not over a tour)
  assert.match(controller, /promptOpen && !tourActive && !isMobileDevice/);

  // the shared picker is reused, not reimplemented
  assert.match(controller, /import \{ LanguagePicker \} from '@\/components\/language-picker\.tsx'/);
  assert.match(controller, /<LanguagePicker \/>/);

  // ordering: description -> language row -> action buttons (Never/Later/Start)
  const description = controller.indexOf("t('onboarding.prompt.description')");
  const language = controller.indexOf("t('onboarding.prompt.language')");
  const picker = controller.indexOf('<LanguagePicker />');
  const firstAction = controller.indexOf("t('onboarding.prompt.never')");
  assert.ok(description > -1 && language > description, 'language row follows the description');
  assert.ok(picker > language, 'picker sits on the language row');
  assert.ok(firstAction > picker, 'language choice precedes the action buttons');
});

test('the shared picker persists only after i18next accepts the language', () => {
  const picker = fs.readFileSync(pickerFile, 'utf8');

  assert.match(picker, /import \{ setLanguage \} from '@\/lib\/localstorage\.ts'/);
  const change = picker.match(/async function changeLanguage\([\s\S]*?\n  \}/);
  assert.ok(change, 'changeLanguage exists');
  const accepted = change[0].indexOf('await i18n.changeLanguage(nextLanguage)');
  const persisted = change[0].indexOf('setLanguage(nextLanguage)');
  assert.ok(accepted > -1, 'i18next change is awaited');
  assert.ok(persisted > accepted, 'localStorage persistence follows the accepted change');
});
