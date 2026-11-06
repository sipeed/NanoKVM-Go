import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const LOGIN_PASSWORD_KEY = 'nano-kvm-login-password';
const TTL_MS = 30 * 24 * 60 * 60 * 1000;

function loadLocalStorageModule({
  entries = [],
  now = Date.UTC(2026, 0, 2),
  throwingOperations = []
} = {}) {
  const values = new Map(entries);
  const throws = new Set(throwingOperations);
  const localStorage = {
    getItem(key) {
      if (throws.has('getItem')) throw new Error('mock getItem failure');
      return values.has(key) ? values.get(key) : null;
    },
    removeItem(key) {
      if (throws.has('removeItem')) throw new Error('mock removeItem failure');
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

  vm.runInNewContext(
    output,
    { Date: { now: () => now }, exports: module.exports, localStorage, module },
    { filename: file }
  );

  return { localStorage: module.exports, values };
}

test('writes and reads a 30-day remembered-password envelope', () => {
  const now = Date.UTC(2026, 0, 2, 3, 4, 5);
  const { localStorage, values } = loadLocalStorageModule({ now });

  assert.equal(localStorage.setLoginPassword('mock-remembered-password'), true);
  assert.deepEqual(JSON.parse(values.get(LOGIN_PASSWORD_KEY)), {
    value: 'mock-remembered-password',
    expiry: now + TTL_MS
  });
  assert.equal(localStorage.getLoginPassword(), 'mock-remembered-password');
  assert.equal(localStorage.setLoginUsername('mock-user'), true);
  assert.equal(localStorage.getLoginUsername(), 'mock-user');
});

test('removes unavailable, expired, malformed, and legacy password records', () => {
  const now = Date.UTC(2026, 0, 2, 3, 4, 5);
  const invalidRecords = [
    ['expired at the exact boundary', JSON.stringify({ value: 'mock-value', expiry: now })],
    ['expired in the past', JSON.stringify({ value: 'mock-value', expiry: now - 1 })],
    ['empty value', JSON.stringify({ value: '', expiry: now + TTL_MS })],
    ['missing expiry', JSON.stringify({ value: 'mock-value' })],
    ['invalid expiry', JSON.stringify({ value: 'mock-value', expiry: null })],
    ['non-finite expiry', '{"value":"mock-value","expiry":1e999}'],
    ['array instead of envelope', JSON.stringify(['mock-value', now + TTL_MS])],
    ['empty stored value', ''],
    ['malformed json', '{mock-value'],
    ['legacy raw value', 'mock-legacy-password']
  ];

  for (const [name, record] of invalidRecords) {
    const { localStorage, values } = loadLocalStorageModule({
      entries: [
        [LOGIN_PASSWORD_KEY, record],
        ['nano-kvm-unrelated-setting', 'keep-me']
      ],
      now
    });

    assert.equal(localStorage.getLoginPassword(), '', name);
    assert.equal(values.has(LOGIN_PASSWORD_KEY), false, name);
    assert.equal(values.get('nano-kvm-unrelated-setting'), 'keep-me', name);
  }

  const { localStorage, values } = loadLocalStorageModule({
    entries: [['nano-kvm-unrelated-setting', 'keep-me']],
    now
  });
  assert.equal(localStorage.getLoginPassword(), '');
  assert.equal(values.get('nano-kvm-unrelated-setting'), 'keep-me');
});

test('replaces remembered passwords and removes only the exact key', () => {
  const now = Date.UTC(2026, 0, 2, 3, 4, 5);
  const { localStorage, values } = loadLocalStorageModule({
    entries: [
      [LOGIN_PASSWORD_KEY, JSON.stringify({ value: 'mock-old-password', expiry: now + 1 })],
      ['nano-kvm-login-username', 'mock-user'],
      ['nano-kvm-unrelated-setting', 'keep-me']
    ],
    now
  });

  assert.equal(localStorage.setLoginPassword('mock-new-password'), true);
  assert.deepEqual(JSON.parse(values.get(LOGIN_PASSWORD_KEY)), {
    value: 'mock-new-password',
    expiry: now + TTL_MS
  });

  assert.equal(localStorage.setLoginPassword(''), true);
  assert.equal(values.has(LOGIN_PASSWORD_KEY), false);
  assert.equal(values.get('nano-kvm-login-username'), 'mock-user');
  assert.equal(values.get('nano-kvm-unrelated-setting'), 'keep-me');

  values.set(
    LOGIN_PASSWORD_KEY,
    JSON.stringify({ value: 'mock-new-password', expiry: now + TTL_MS })
  );
  assert.equal(localStorage.removeLoginPassword(), true);
  assert.equal(values.has(LOGIN_PASSWORD_KEY), false);
  assert.equal(values.get('nano-kvm-unrelated-setting'), 'keep-me');
});

test('contains localStorage failures without affecting unrelated records', () => {
  const now = Date.UTC(2026, 0, 2, 3, 4, 5);
  const initialEntries = [
    [LOGIN_PASSWORD_KEY, JSON.stringify({ value: 'mock-existing-password', expiry: now + TTL_MS })],
    ['nano-kvm-unrelated-setting', 'keep-me']
  ];

  const readFailure = loadLocalStorageModule({
    entries: initialEntries,
    now,
    throwingOperations: ['getItem']
  });
  assert.equal(readFailure.localStorage.getLoginPassword(), '');
  assert.equal(readFailure.localStorage.getLoginUsername(), '');
  assert.equal(readFailure.values.get('nano-kvm-unrelated-setting'), 'keep-me');

  const writeFailure = loadLocalStorageModule({
    entries: initialEntries,
    now,
    throwingOperations: ['setItem']
  });
  assert.equal(writeFailure.localStorage.setLoginPassword('mock-new-password'), false);
  assert.equal(writeFailure.localStorage.setLoginUsername('mock-user'), false);
  assert.equal(writeFailure.values.get('nano-kvm-unrelated-setting'), 'keep-me');

  const removeFailure = loadLocalStorageModule({
    entries: initialEntries,
    now,
    throwingOperations: ['removeItem']
  });
  assert.equal(removeFailure.localStorage.removeLoginPassword(), false);
  assert.equal(removeFailure.localStorage.setLoginPassword(''), false);
  assert.equal(removeFailure.localStorage.getLoginPassword(), 'mock-existing-password');
  assert.equal(removeFailure.values.get('nano-kvm-unrelated-setting'), 'keep-me');

  const invalidRemoveFailure = loadLocalStorageModule({
    entries: [
      [LOGIN_PASSWORD_KEY, 'mock-legacy-password'],
      ['nano-kvm-unrelated-setting', 'keep-me']
    ],
    throwingOperations: ['removeItem']
  });
  assert.equal(invalidRemoveFailure.localStorage.getLoginPassword(), '');
  assert.equal(invalidRemoveFailure.values.get('nano-kvm-unrelated-setting'), 'keep-me');
});
