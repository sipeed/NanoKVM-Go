import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { after, before, test } from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';
import { createServer } from 'vite';

const webRoot = path.resolve(import.meta.dirname, '..');
let vite;
let model;

before(async () => {
  vite = await createServer({
    root: webRoot,
    appType: 'custom',
    logLevel: 'error',
    server: { middlewareMode: true, hmr: false }
  });
  model = await vite.ssrLoadModule(
    '/src/pages/desktop/menu/settings/ai/screen-timelapse-recording-model.ts'
  );
});

after(async () => {
  await vite.close();
});

test('recording QP model accepts the configured fixed-QP range', () => {
  assert.equal(model.parseScreenTimelapseQP(36), 36);
  assert.equal(model.parseScreenTimelapseQP(42), 42);
  assert.equal(model.parseScreenTimelapseQP(51), 51);
});

test('recording QP model rejects invalid values and normalizes to 42', () => {
  for (const value of [35, 52, 8.5, '42']) {
    assert.equal(model.parseScreenTimelapseQP(value), null);
    assert.equal(model.normalizeScreenTimelapseQP(value), 42);
  }
});

test('recording quality maps cleanly onto the internal fixed-QP range', () => {
  assert.deepEqual(
    model.screenTimelapseQualitySteps.map((step) => step.quality),
    [0, 7, 13, 20, 27, 33, 40, 47, 53, 60, 67, 73, 80, 87, 93, 100]
  );
  assert.deepEqual(
    model.screenTimelapseQualitySteps.map((step) => step.qp),
    [51, 50, 49, 48, 47, 46, 45, 44, 43, 42, 41, 40, 39, 38, 37, 36]
  );
  assert.equal(model.screenTimelapseQPToQuality(51), 0);
  assert.equal(model.screenTimelapseQPToQuality(42), 60);
  assert.equal(model.screenTimelapseQPToQuality(36), 100);
  assert.equal(model.screenTimelapseQualityDefault, 60);
  assert.equal(model.screenTimelapseQualityToQP(0), 51);
  assert.equal(model.screenTimelapseQualityToQP(60), 42);
  assert.equal(model.screenTimelapseQualityToQP(100), 36);

  for (let qp = 36; qp <= 51; qp += 1) {
    assert.equal(model.screenTimelapseQualityToQP(model.screenTimelapseQPToQuality(qp)), qp);
  }
});

test('recording quality rejects invalid and unsupported positions', () => {
  for (const value of [-1, 101, 18.5, '18']) {
    assert.equal(model.parseScreenTimelapseQuality(value), null);
    assert.equal(model.screenTimelapseQualityToQP(value), null);
  }
  assert.equal(model.parseScreenTimelapseQuality(50), 50);
  assert.equal(model.screenTimelapseQualityToQP(50), null);
});

test('screen timelapse settings use a dedicated recording API and panel', () => {
  const apiSource = fs.readFileSync(path.join(webRoot, 'src/api/ai.ts'), 'utf8');
  const settingsSource = fs.readFileSync(
    path.join(webRoot, 'src/pages/desktop/menu/settings/ai/screen-timelapse.tsx'),
    'utf8'
  );
  const panelSource = fs.readFileSync(
    path.join(webRoot, 'src/pages/desktop/menu/settings/ai/screen-timelapse-recording.tsx'),
    'utf8'
  );
  const localeSource = fs.readFileSync(
    path.join(webRoot, 'src/i18n/timelapse-recording.ts'),
    'utf8'
  );
  const stylesSource = fs.readFileSync(path.join(webRoot, 'src/assets/styles/index.css'), 'utf8');

  assert.match(apiSource, /\/api\/ai\/screen-timelapse\/recording/);
  assert.match(settingsSource, /ScreenTimelapseRecordingPanel/);
  assert.match(panelSource, /setScreenTimelapseRecording\(qp\)/);
  assert.match(panelSource, /screenTimelapseQPToQuality\(nextConfig\.qp\)/);
  assert.match(panelSource, /screenTimelapseQualityToQP\(draftQuality\)/);
  assert.match(panelSource, /<Slider/);
  assert.match(panelSource, /step=\{null\}/);
  assert.match(panelSource, /marks=\{qualityMarks\}/);
  assert.match(panelSource, /value === screenTimelapseQualityDefault/);
  assert.match(panelSource, /text\('default', 'Default'\)/);
  assert.match(panelSource, /tooltip=\{\{ open: false \}\}/);
  assert.match(panelSource, /styles=\{\{ rail: screenTimelapseSliderRailStyle \}\}/);
  assert.match(panelSource, /className="screen-timelapse-quality-slider"/);
  assert.match(
    stylesSource,
    /\.screen-timelapse-quality-slider:not\(\.ant-slider-disabled\) \.ant-slider-dot \{/
  );
  assert.match(stylesSource, /\.screen-timelapse-quality-slider[^}]+\.ant-slider-dot-active/);
  assert.doesNotMatch(panelSource, /\{draftQuality\}%/);
  assert.doesNotMatch(panelSource, /<InputNumber|<Form|Fixed QP|text\('qp'/);
  assert.doesNotMatch(localeSource, /\bQP\b/);
  assert.doesNotMatch(panelSource, /setQuality|localstorage|videoParametersAtom/);
});

// --- Feature-scoped hydrated message module guard ---------------------------
//
// `src/i18n/timelapse-recording.ts` is not a file under `src/i18n/locales/`.
// `i18n/index.ts` hydrates the `settings.ai.screenTimelapseRecording` namespace
// at runtime and the panel reads keys through dynamic template literals, so
// `pnpm i18n:audit` structurally cannot see this module (audit-i18n.mjs never
// imports it). The focused tests below are therefore the only automated gate:
// the runtime en fallback masks a deleted locale map, so the declared keys are
// asserted at the source level as well.

const timelapseRecordingFile = path.join(webRoot, 'src/i18n/timelapse-recording.ts');
const retentionFile = path.join(webRoot, 'src/i18n/retention.ts');
const localeFile = path.join(webRoot, 'src/i18n/locale.ts');

const RECORDING_KEYS = [
  'title',
  'low',
  'default',
  'high',
  'activeHint',
  'loadFailed',
  'saveFailed',
  'saved',
  'savedAndRestarted',
  'noChanges',
  'save'
];

// `import type { SupportedLanguage }` is elided by ts.transpileModule, so both
// message modules stay require-free; a value import would emit
// require('./locale.ts') and make the throwing sandbox below reject the module.
function loadTsModule(file) {
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
        throw new Error(`unexpected module: ${id}`);
      }
    },
    { filename: file }
  );
  return module.exports;
}

function supportedLanguages() {
  const source = fs.readFileSync(localeFile, 'utf8');
  const match = source.match(/supportedLanguageKeys\s*=\s*\[([\s\S]*?)\]\s*as const/);
  assert.ok(match, 'unable to find supportedLanguageKeys in locale.ts');
  return [...match[1].matchAll(/'([^']+)'/g)].map((entry) => entry[1]);
}

const recordingMessages = loadTsModule(timelapseRecordingFile).default;
const retentionMessages = loadTsModule(retentionFile).default;
const languages = supportedLanguages();

test('the recording messages record declares exactly the supported languages', () => {
  // Source-level declaration check: the en fallback in the default export
  // would mask a deleted locale map at runtime, so assert the declared keys.
  const source = fs.readFileSync(timelapseRecordingFile, 'utf8');
  const recordMatch = source.match(
    /Record<SupportedLanguage, ScreenTimelapseRecordingMessages> = \{([\s\S]*?)\n\};/
  );
  assert.ok(recordMatch, 'unable to locate the recording messages record');
  const declared = [...recordMatch[1].matchAll(/^ {2}([a-z_]+): \{/gm)].map((entry) => entry[1]);
  assert.deepEqual([...declared].sort(), [...languages].sort());
});

test('recording messages resolve for every supported language with en fallback', () => {
  const en = recordingMessages('en');
  assert.ok(en, 'english messages must exist');

  for (const language of languages) {
    assert.ok(recordingMessages(language), `missing recording messages for ${language}`);
  }

  // Unknown languages fall back to the identical English map object.
  assert.equal(recordingMessages('xx'), en);
});

test('every recording locale has the complete, non-empty key set of English', () => {
  assert.deepEqual(Object.keys(recordingMessages('en')).sort(), [...RECORDING_KEYS].sort());

  for (const language of languages) {
    const messages = recordingMessages(language);
    assert.deepEqual(
      Object.keys(messages).sort(),
      [...RECORDING_KEYS].sort(),
      `${language} key set diverges from en`
    );
    for (const key of RECORDING_KEYS) {
      assert.ok(messages[key].trim().length > 0, `${language}:${key} must not be empty`);
    }
  }
});

test('no recording locale is derived from English with a spread', () => {
  const source = fs.readFileSync(timelapseRecordingFile, 'utf8');
  assert.doesNotMatch(source, /\.\.\.en\b/);
  assert.doesNotMatch(source, /\.\.\.messages\.en\b/);
});

test('shared save terms stay identical across the timelapse message modules', () => {
  // `save` and `noChanges` are generic action/state labels shared with the
  // sibling retention module, so the spec requires them byte-identical per
  // language. `saved`/`saveFailed`/`loadFailed` are full context sentences in
  // retention ("Automatic cleanup settings saved.") and intentionally differ
  // from the recording-quality sentences, so they are excluded here.
  for (const language of languages) {
    const recording = recordingMessages(language);
    const retention = retentionMessages[language]?.retention;
    assert.ok(retention, `missing retention messages for ${language}`);
    assert.equal(recording.save, retention.save, `${language} save must match retention.ts`);
    assert.equal(
      recording.noChanges,
      retention.noChanges,
      `${language} noChanges must match retention.ts`
    );
  }
});
