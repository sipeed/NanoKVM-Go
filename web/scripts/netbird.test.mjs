import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const root = path.resolve(import.meta.dirname, '..');
const component = 'src/pages/desktop/menu/settings/remote-network/components/netbird';

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

test('NetBird distinguishes authorization, connectivity, and failed login', () => {
  const { toRemoteNetworkStatus, errorMessageKey } = load(`${component}/model.ts`);
  for (const [state, expected] of [
    ['notInstalled', 'off'],
    ['notLogin', 'off'],
    ['stopped', 'off'],
    ['installing', 'busy'],
    ['authorizing', 'busy'],
    ['connecting', 'busy'],
    ['connected', 'on'],
    ['offline', 'bad'],
    ['error', 'bad']
  ])
    assert.equal(toRemoteNetworkStatus({ state, busy: false, error: '' }), expected);
  assert.equal(toRemoteNetworkStatus({ state: 'notLogin', error: 'loginExpired' }), 'bad');
  assert.equal(toRemoteNetworkStatus({ state: 'connected', busy: true }), 'busy');
  assert.equal(errorMessageKey('loginExpired'), 'netbird.loginExpired');
  assert.equal(errorMessageKey('upstream secret details'), 'netbird.failed');
});

test('NetBird login is available only after enabling a running service', () => {
  const { isLoginAvailable, errorMessageKey } = load(`${component}/model.ts`);
  assert.equal(isLoginAvailable(undefined), false);
  for (const [enabled, running] of [
    [false, false],
    [false, true],
    [true, false]
  ]) {
    assert.equal(isLoginAvailable({ enabled, running }), false);
  }
  assert.equal(isLoginAvailable({ enabled: true, running: true }), true);
  assert.equal(errorMessageKey('startRequired'), 'netbird.startRequired');
});

test('NetBird ships complete messages for all supported languages', () => {
  const { netbirdMessages } = load(`${component}/messages.ts`);
  const localeSource = fs.readFileSync(path.join(root, 'src/i18n/locale.ts'), 'utf8');
  const supported = [
    ...localeSource.match(/supportedLanguageKeys = \[([\s\S]*?)\]/)[1].matchAll(/'([^']+)'/g)
  ].map((m) => m[1]);
  assert.deepEqual(Object.keys(netbirdMessages).sort(), supported.sort());
  const keys = Object.keys(netbirdMessages.en).sort();
  for (const [language, bundle] of Object.entries(netbirdMessages)) {
    assert.deepEqual(Object.keys(bundle).sort(), keys, language);
    for (const value of Object.values(bundle))
      assert.ok(typeof value === 'string' && value.trim(), language);
  }
});

test('NetBird protection requires verification when its policy is enabled or missing', async () => {
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
  assert.equal((await gate.resolveProtectionRequiresChallenge('netbird')).requireChallenge, true);
  policy.netbird = false;
  assert.equal((await gate.resolveProtectionRequiresChallenge('netbird')).requireChallenge, false);
  policy.netbird = true;
  assert.equal((await gate.resolveProtectionRequiresChallenge('netbird')).requireChallenge, true);
  assert.equal(
    (await gate.resolveProtectionRequiresChallenge('tailscale')).requireChallenge,
    false
  );
  enabled = false;
  assert.equal((await gate.resolveProtectionRequiresChallenge('netbird')).requireChallenge, false);
});

test('NetBird API exposes only official browser login and forwards action-bound challenges', async () => {
  const requests = [];
  const api = load('src/api/extensions/netbird.ts', {
    '@/lib/http.ts': {
      http: {
        post: (...args) => {
          requests.push(args);
          return Promise.resolve({ code: 0 });
        }
      }
    }
  });
  await api.runAction('login', 'login-proof');
  await api.runAction('login_cancel', 'cancel-proof');
  assert.equal(requests[0][0], '/api/extensions/netbird/login');
  assert.equal(JSON.stringify(requests[0][1]), JSON.stringify({ challenge: 'login-proof' }));
  assert.equal(requests[1][0], '/api/extensions/netbird/login/cancel');
});
