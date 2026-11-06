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

function loadModel() {
  const file = path.join(webRoot, 'src/pages/desktop/menu/settings/update/model.ts');
  const output = ts.transpileModule(source('src/pages/desktop/menu/settings/update/model.ts'), {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
    fileName: file
  }).outputText;
  const module = { exports: {} };
  vm.runInNewContext(
    output,
    {
      exports: module.exports,
      module,
      require() {
        return {};
      }
    },
    { filename: file }
  );
  return module.exports;
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

const applicationApi = source('src/api/application.ts');
const tailscaleApi = source('src/api/extensions/tailscale.ts');
const page = source('src/pages/desktop/menu/settings/update/index.tsx');
const mockApplicationHandler = source('src/mocks/handlers/application.ts');
const mockTailscaleHandler = source('src/mocks/handlers/tailscale.ts');
const messages = source('src/i18n/update.ts');
const updateMessages = loadTypeScriptModule('src/i18n/update.ts').default;
const { supportedLanguageKeys } = loadTypeScriptModule('src/i18n/locale.ts', () => ({}));

test('region API is retained without a settings page selector', () => {
  assert.match(applicationApi, /http\.get\('\/api\/application\/region'\)/);
  assert.match(applicationApi, /http\.post\('\/api\/application\/region', \{ region \}\)/);
  assert.doesNotMatch(tailscaleApi, /\/api\/extensions\/tailscale\/region/);
  assert.doesNotMatch(page, /api\s*\.(getRegion|setRegion)\(/);
  assert.doesNotMatch(page, /settings\.update\.region\./);
  assert.match(messages, /confirmDesc: '.*kvmcomm/);
  assert.match(messages, /cleanupWarning/);
});

test('legacy region API preserves the managed network and does not gate modes', () => {
  assert.match(mockApplicationHandler, /\/api\/application\/region/);
  assert.match(mockApplicationHandler, /region !== 'cn' && region !== 'global'/);
  assert.doesNotMatch(mockApplicationHandler, /state\.tailscale\./);
  assert.doesNotMatch(mockTailscaleHandler, /\/api\/extensions\/tailscale\/region/);
  assert.doesNotMatch(mockTailscaleHandler, /state\.application\.region/);
  assert.match(mockTailscaleHandler, /state\.application\.tailscaleCnEnabled/);
});

test('region mutation owns the settings lock while other controls settle', () => {
  const { createSettingsBusyLock } = loadModel();
  const changes = [];
  const lock = createSettingsBusyLock((value) => changes.push(value));
  lock.setRegionBusy(true);
  lock.setPreviewBusy(true);
  lock.setRegionBusy(false);
  assert.equal(lock.isLocked(), true);
  lock.setPreviewBusy(false);
  assert.equal(lock.isLocked(), false);
  assert.deepEqual(changes, [true, false]);
});

test('region messages are complete for every supported language', () => {
  const englishRegion = updateMessages('en').region;
  const englishKeys = Object.keys(englishRegion).sort();

  assert.equal(supportedLanguageKeys.length, 20);
  for (const language of supportedLanguageKeys) {
    const region = updateMessages(language).region;
    assert.deepEqual(
      Object.keys(region).sort(),
      englishKeys,
      `${language} has the region message keys`
    );
    for (const [key, value] of Object.entries(region)) {
      assert.equal(typeof value, 'string', `${language}.region.${key} is a string`);
      assert.ok(value.trim(), `${language}.region.${key} is not empty`);
      if (language !== 'en') {
        assert.notEqual(value, englishRegion[key], `${language}.region.${key} is translated`);
      }
    }
    assert.match(region.confirmDesc, /\{\{region\}\}/, `${language} keeps {{region}}`);
  }
});
