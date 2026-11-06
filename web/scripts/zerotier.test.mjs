import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const root = path.resolve(import.meta.dirname, '..');
const component = 'src/pages/desktop/menu/settings/remote-network/components/zerotier';

function load(relative, modules = {}) {
  const filename = path.join(root, relative);
  const code = ts.transpileModule(fs.readFileSync(filename, 'utf8'), {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
  }).outputText;
  const module = { exports: {} };
  vm.runInNewContext(
    code,
    {
      URL,
      atob,
      exports: module.exports,
      module,
      require(id) {
        assert.ok(id in modules, `unexpected import ${id}`);
        return modules[id];
      }
    },
    { filename }
  );
  return module.exports;
}

test('Network IDs are normalized and malformed IDs cannot be submitted', () => {
  const { normalizeNetworkID } = load(`${component}/model.ts`);
  assert.equal(normalizeNetworkID(' 0123456789ABCDEF '), '0123456789abcdef');
  for (const value of [
    '',
    '0000000000000000',
    '0123456789abcdeg',
    '0123456789abcde',
    '../0123456789abcd',
    '0123456789abcdef/extra'
  ])
    assert.equal(normalizeNetworkID(value), undefined);
});

test('ZeroTier distinguishes service availability from usable network membership', () => {
  const { toRemoteNetworkStatus, networkMessageKey, errorMessageKey } = load(
    `${component}/model.ts`
  );
  for (const [state, expected] of [
    ['notInstalled', 'off'],
    ['stopped', 'off'],
    ['noNetworks', 'off'],
    ['connecting', 'busy'],
    ['installing', 'busy'],
    ['connected', 'on'],
    ['offline', 'bad'],
    ['error', 'bad']
  ])
    assert.equal(toRemoteNetworkStatus({ state, busy: false }), expected);
  assert.equal(toRemoteNetworkStatus({ state: 'connected', busy: true }), 'busy');
  assert.equal(networkMessageKey('ACCESS_DENIED'), 'zerotier.denied');
  assert.equal(networkMessageKey('REQUESTING_CONFIGURATION'), 'zerotier.requesting');
  assert.equal(networkMessageKey('WAITING_ADDRESS'), 'zerotier.waitingAddress');
  assert.equal(networkMessageKey('unknown'), 'zerotier.error');
  assert.equal(errorMessageKey('private local API output'), 'zerotier.failed');
});

test('ZeroTier ships complete messages for all supported languages', () => {
  const { zerotierMessages } = load(`${component}/messages.ts`);
  const localeSource = fs.readFileSync(path.join(root, 'src/i18n/locale.ts'), 'utf8');
  const supported = [
    ...localeSource.match(/supportedLanguageKeys = \[([\s\S]*?)\]/)[1].matchAll(/'([^']+)'/g)
  ].map((m) => m[1]);
  assert.deepEqual(Object.keys(zerotierMessages).sort(), supported.sort());
  const keys = Object.keys(zerotierMessages.en).sort();
  for (const [language, bundle] of Object.entries(zerotierMessages)) {
    assert.deepEqual(Object.keys(bundle).sort(), keys, language);
    for (const value of Object.values(bundle))
      assert.ok(typeof value === 'string' && value.trim(), language);
  }
});

test('ZeroTier protection requires verification when its policy is enabled or missing', async () => {
  const policy = {
    ssh: false,
    tailscale: false,
    mcp: false,
    targetPower: false,
    nanoKVMRestart: false,
    deleteImage: false,
    screenTimelapse: false
  };
  let enabled = true;
  const gate = load('src/lib/protection-challenge.ts', {
    '@/api/auth.ts': {
      getTwoFactorStatus: async () => ({ code: 0, data: { enabled } }),
      getTwoFactorProtection: async () => ({
        code: 0,
        data: {
          policy,
          authorizationExpiresIn: 0,
          mcpAuthorizationExpiresIn: 0,
          screenTimelapseAuthorizationExpiresIn: 0
        }
      })
    }
  });
  assert.equal((await gate.resolveProtectionRequiresChallenge('zerotier')).requireChallenge, true);
  policy.zerotier = false;
  assert.equal((await gate.resolveProtectionRequiresChallenge('zerotier')).requireChallenge, false);
  policy.zerotier = true;
  assert.equal((await gate.resolveProtectionRequiresChallenge('zerotier')).requireChallenge, true);
  assert.equal(
    (await gate.resolveProtectionRequiresChallenge('tailscale')).requireChallenge,
    false
  );
  enabled = false;
  assert.equal((await gate.resolveProtectionRequiresChallenge('zerotier')).requireChallenge, false);
});
