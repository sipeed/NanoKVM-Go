import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const PORTAL_MODE_KEY = 'nano-kvm-portal-mode';
const PORTAL_BORDER_ENABLED_KEY = 'nano-kvm-portal-border-enabled';
const INPUT_PROFILES_KEY = 'nano-kvm-input-profiles';

function loadLocalStorageModule({ entries = [], throwingOperations = [] } = {}) {
  const values = new Map(entries);
  const throws = new Set(throwingOperations);
  const localStorage = {
    getItem(key) {
      if (throws.has('getItem')) throw new Error('mock getItem failure');
      return values.has(key) ? values.get(key) : null;
    },
    removeItem(key) {
      values.delete(key);
    },
    setItem(key, value) {
      if (throws.has('setItem')) throw new Error('mock setItem failure');
      values.set(key, String(value));
    }
  };
  const file = path.resolve(import.meta.dirname, '../src/lib/localstorage.ts');
  const source = fs.readFileSync(file, 'utf8');
  const output = ts.transpileModule(source, {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022
    },
    fileName: file
  }).outputText;
  const module = { exports: {} };

  vm.runInNewContext(output, { exports: module.exports, localStorage, module }, { filename: file });

  return { localStorage: module.exports, values };
}

test('saves and restores all Portal modes', () => {
  const { localStorage, values } = loadLocalStorageModule();

  for (const mode of ['auto', 'manual', 'off']) {
    localStorage.setPortalMode(mode);
    assert.equal(values.get(PORTAL_MODE_KEY), mode);
    assert.equal(localStorage.getPortalMode(), mode);
  }
});

test('migrates legacy on and defaults missing or invalid values to auto', () => {
  const missing = loadLocalStorageModule();
  assert.equal(missing.localStorage.getPortalMode(), 'auto');
  assert.equal(missing.values.has(PORTAL_MODE_KEY), false);

  for (const mode of ['on', '', 'enabled', 'disabled']) {
    const { localStorage, values } = loadLocalStorageModule({
      entries: [[PORTAL_MODE_KEY, mode]]
    });
    assert.equal(localStorage.getPortalMode(), 'auto');
    assert.equal(values.get(PORTAL_MODE_KEY), mode);
  }
});

test('reads a stored mode without creating a default', () => {
  const missing = loadLocalStorageModule();
  assert.equal(missing.localStorage.getStoredPortalMode(), null);

  const legacy = loadLocalStorageModule({ entries: [[PORTAL_MODE_KEY, 'on']] });
  assert.equal(legacy.localStorage.getStoredPortalMode(), 'auto');
});

test('keeps Portal usable when browser storage is unavailable', () => {
  const readFailure = loadLocalStorageModule({ throwingOperations: ['getItem'] });
  assert.equal(readFailure.localStorage.getPortalMode(), 'auto');

  const writeFailure = loadLocalStorageModule({ throwingOperations: ['setItem'] });
  assert.doesNotThrow(() => writeFailure.localStorage.setPortalMode('off'));
});

test('defaults the Portal border to off and restores the saved state', () => {
  const missing = loadLocalStorageModule();
  assert.equal(missing.localStorage.getPortalBorderEnabled(), false);
  assert.equal(missing.values.has(PORTAL_BORDER_ENABLED_KEY), false);

  for (const enabled of [true, false]) {
    const { localStorage, values } = loadLocalStorageModule();
    localStorage.setPortalBorderEnabled(enabled);
    assert.equal(values.get(PORTAL_BORDER_ENABLED_KEY), String(enabled));
    assert.equal(localStorage.getPortalBorderEnabled(), enabled);
  }

  const readFailure = loadLocalStorageModule({ throwingOperations: ['getItem'] });
  assert.equal(readFailure.localStorage.getPortalBorderEnabled(), false);
});

test('caches input profiles in browser storage and tolerates invalid data', () => {
  const { localStorage, values } = loadLocalStorageModule();
  const store = { version: 1, selectedProfileId: '', profiles: [] };

  localStorage.setInputProfileStore(store);
  assert.equal(values.get(INPUT_PROFILES_KEY), JSON.stringify(store));
  assert.equal(JSON.stringify(localStorage.getInputProfileStore()), JSON.stringify(store));

  const invalid = loadLocalStorageModule({ entries: [[INPUT_PROFILES_KEY, '{invalid']] });
  assert.equal(invalid.localStorage.getInputProfileStore(), null);

  const unavailable = loadLocalStorageModule({ throwingOperations: ['setItem'] });
  assert.doesNotThrow(() => unavailable.localStorage.setInputProfileStore(store));
});
