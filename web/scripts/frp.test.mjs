import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const root = path.resolve(import.meta.dirname, '..');
const component = 'src/pages/desktop/menu/settings/remote-network/components/frp';

function load(relative, modules = {}) {
  const filename = path.join(root, relative);
  const code = ts.transpileModule(fs.readFileSync(filename, 'utf8'), {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 }
  }).outputText;
  const module = { exports: {} };
  vm.runInNewContext(
    code,
    {
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

test('FRP badge distinguishes installation, pending connection and active forwarding', () => {
  const { toRemoteNetworkStatus } = load(`${component}/model.ts`);
  for (const [state, expected] of [
    ['notInstalled', 'off'],
    ['notConfigured', 'off'],
    ['stopped', 'off'],
    ['installing', 'busy'],
    ['connecting', 'busy'],
    ['connected', 'on'],
    ['error', 'bad']
  ])
    assert.equal(toRemoteNetworkStatus({ state, busy: false }), expected);
  assert.equal(toRemoteNetworkStatus({ state: 'connected', busy: true }), 'busy');
});

test('form rejects missing server, invalid ports and conflicting control and public ports', () => {
  const { validateConfig, errorMessageKey } = load(`${component}/model.ts`);
  const valid = { serverAddr: 'frp.example.com', serverPort: 7000, remotePort: 8443 };
  assert.equal(validateConfig(valid), undefined);
  assert.equal(validateConfig({ ...valid, serverAddr: ' ' }), 'invalidServer');
  for (const remotePort of [0, -1, 65536, 1.5, 7000, NaN]) {
    assert.equal(validateConfig({ ...valid, remotePort }), 'invalidPort');
  }
  assert.equal(errorMessageKey('untrusted server response'), 'frp.failed');
});

test('saving unrelated fields omits the stored secret while an explicit empty token clears it', () => {
  const calls = [];
  const api = load('src/api/extensions/frp.ts', {
    '@/lib/http.ts': { http: { post: (...args) => calls.push(args) } }
  });
  const form = { serverAddr: 'frp.example.com', serverPort: 7000, remotePort: 8443 };
  api.setConfig(form, 'challenge');
  assert.equal(calls[0][0], '/api/extensions/frp/config');
  assert.equal(Object.hasOwn(calls[0][1], 'token'), false);
  assert.equal(calls[0][1].challenge, 'challenge');
  api.setConfig({ ...form, token: '' });
  assert.equal(calls[1][1].token, '');
});

test('FRP ships complete messages for all supported languages', () => {
  const { frpMessages } = load(`${component}/messages.ts`);
  const localeSource = fs.readFileSync(path.join(root, 'src/i18n/locale.ts'), 'utf8');
  const supported = [
    ...localeSource.match(/supportedLanguageKeys = \[([\s\S]*?)\]/)[1].matchAll(/'([^']+)'/g)
  ].map((m) => m[1]);
  assert.deepEqual(Object.keys(frpMessages).sort(), supported.sort());
  const keys = Object.keys(frpMessages.en).sort();
  for (const [language, bundle] of Object.entries(frpMessages)) {
    assert.deepEqual(Object.keys(bundle).sort(), keys, language);
    for (const value of Object.values(bundle))
      assert.ok(typeof value === 'string' && value.trim(), language);
  }
});

test('FRP protection requires verification when its policy is enabled or missing', async () => {
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
  assert.equal((await gate.resolveProtectionRequiresChallenge('frp')).requireChallenge, true);
  policy.frp = false;
  assert.equal((await gate.resolveProtectionRequiresChallenge('frp')).requireChallenge, false);
  policy.frp = true;
  assert.equal((await gate.resolveProtectionRequiresChallenge('frp')).requireChallenge, true);
  assert.equal(
    (await gate.resolveProtectionRequiresChallenge('tailscale')).requireChallenge,
    false
  );
  enabled = false;
  assert.equal((await gate.resolveProtectionRequiresChallenge('frp')).requireChallenge, false);
});
