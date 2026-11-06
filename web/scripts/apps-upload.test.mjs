import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const webRoot = path.resolve(import.meta.dirname, '..');

function source(relativePath) {
  return fs.readFileSync(path.join(webRoot, relativePath), 'utf8');
}

function loadTypeScriptModule(relativePath, requireModule) {
  const file = path.join(webRoot, relativePath);
  const output = ts.transpileModule(source(relativePath), {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
    fileName: file
  }).outputText;
  const module = { exports: {} };
  const require =
    requireModule ??
    ((specifier) => {
      throw new Error(`Unexpected require(${specifier}) while loading ${relativePath}`);
    });

  vm.runInNewContext(output, { exports: module.exports, module, require }, { filename: file });
  return module.exports;
}

const page = source('src/pages/desktop/menu/settings/apps/index.tsx');
const confirmation = loadTypeScriptModule(
  'src/pages/desktop/menu/settings/apps/upload-confirmation.ts'
);
const messages = loadTypeScriptModule('src/i18n/apps.ts').default;
const { supportedLanguageKeys } = loadTypeScriptModule('src/i18n/locale.ts', () => ({}));

test('large App uploads require confirmation only above 100 MiB', () => {
  const threshold = 100 * 1024 * 1024;

  assert.equal(confirmation.LARGE_APP_UPLOAD_THRESHOLD_BYTES, threshold);
  assert.equal(confirmation.requiresLargeAppUploadConfirmation(threshold), false);
  assert.equal(confirmation.requiresLargeAppUploadConfirmation(threshold + 1), true);
});

test('large upload confirmation gates inspection and defaults focus to cancel', () => {
  assert.match(page, /Modal\.useModal\(\)/);
  assert.match(page, /requiresLargeAppUploadConfirmation\(file\.size\)/);
  assert.match(
    page,
    /modal\.confirm\(\{[\s\S]*?autoFocusButton: 'cancel'[\s\S]*?onOk: \(\) => void inspectUpload\(file\)/
  );
  assert.match(page, /\{modalContextHolder\}/);
});

test('App messages are complete for every supported language', () => {
  const englishKeys = Object.keys(messages('en')).sort();

  assert.equal(supportedLanguageKeys.length, 20);
  for (const language of supportedLanguageKeys) {
    const labels = messages(language);
    assert.deepEqual(Object.keys(labels).sort(), englishKeys, `${language} has the Apps message keys`);
    for (const [key, value] of Object.entries(labels)) {
      assert.equal(typeof value, 'string', `${language}.${key} is a string`);
      assert.ok(value.trim(), `${language}.${key} is not empty`);
    }
    assert.ok(labels.largeUploadTitle);
    assert.match(labels.largeUploadDesc, /\{\{name\}\}/);
    assert.match(labels.largeUploadDesc, /\{\{size\}\}/);
    assert.match(labels.largeUploadDesc, /MiB/);
    assert.ok(labels.largeUploadConfirm);
    assert.ok(labels.largeUploadCancel);
    assert.match(labels.zipHint, /ZIP/);
    assert.match(labels.zipHint, /app\.json/);
    assert.match(labels.zipHint, /main\.py/);
  }
});
