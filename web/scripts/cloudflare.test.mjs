import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const root = path.resolve(import.meta.dirname, '..');
const component = 'src/pages/desktop/menu/settings/remote-network/components/cloudflare';

function load(relative, modules = {}, globals = {}) {
  const filename = path.join(root, relative);
  const code = ts.transpileModule(fs.readFileSync(filename, 'utf8'), {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022,
      jsx: ts.JsxEmit.ReactJSX
    }
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
      },
      ...globals
    },
    { filename }
  );
  return module.exports;
}

function mountView(
  initialStatus,
  configPromise = Promise.resolve({
    code: 0,
    data: { mode: initialStatus.mode, accessUrl: 'https://kvm.example.com', tokenSet: true }
  })
) {
  const slots = [];
  let cursor = 0;
  let timer;
  let status = initialStatus;
  let modeWrites = 0;
  const react = {
    useState(initial) {
      const slot = (slots[cursor++] ??= { value: initial });
      return [
        slot.value,
        (next) => {
          slot.value = typeof next === 'function' ? next(slot.value) : next;
        }
      ];
    },
    useRef(initial) {
      const slot = (slots[cursor++] ??= { value: { current: initial } });
      return slot.value;
    },
    useCallback(callback, deps) {
      const slot = (slots[cursor++] ??= {});
      if (!slot.deps || deps.some((dep, index) => dep !== slot.deps[index])) {
        slot.value = callback;
        slot.deps = deps;
      }
      return slot.value;
    },
    useEffect(effect, deps) {
      const slot = (slots[cursor++] ??= {});
      if (slot.deps && deps.every((dep, index) => dep === slot.deps[index])) return;
      slot.cleanup?.();
      slot.deps = deps;
      slot.cleanup = effect();
    }
  };
  const controls = Object.fromEntries(
    [
      'TunnelAccessLink',
      'TunnelConfigPanel',
      'TunnelConnection',
      'TunnelHeader',
      'TunnelInstall',
      'TunnelTokenField'
    ].map((name) => [name, Symbol(name)])
  );
  const ChooseMode = Symbol('ChooseMode');
  const jsx = (type, props) => ({ type, props });
  const module = load(
    `${component}/view.tsx`,
    {
      react,
      antd: Object.fromEntries(
        ['Alert', 'Button', 'Form', 'Input', 'Modal', 'Spin'].map((name) => [name, Symbol(name)])
      ),
      'lucide-react': Object.fromEntries(
        ['CloudIcon', 'SaveIcon', 'Trash2Icon'].map((name) => [name, Symbol(name)])
      ),
      '@/api/extensions/cloudflare.ts': {
        getStatus: async () => ({ code: 0, data: status }),
        getConfig: () => configPromise,
        setMode: async () => {
          modeWrites++;
          return { code: 0 };
        }
      },
      '@/hooks/useProtectionChallengeGate.tsx': {
        useProtectionChallengeGate: () => ({
          isGateBusy: false,
          modal: null,
          runWithChallenge: (_, __, callback) => callback()
        })
      },
      '../../messages.ts': { useRemoteNetworkMessages: () => (key) => key },
      '../../tunnel-controls.tsx': controls,
      './choose-mode.tsx': { ChooseMode },
      './model.ts': load(`${component}/model.ts`),
      'react/jsx-runtime': { Fragment: Symbol('Fragment'), jsx, jsxs: jsx }
    },
    {
      AbortController,
      setTimeout(callback) {
        timer = callback;
        return 1;
      },
      clearTimeout() {
        timer = undefined;
      }
    }
  );
  let backCalls = 0;
  const props = {
    onBack() {
      backCalls++;
    },
    setLocked() {}
  };
  return {
    ...controls,
    ChooseMode,
    get modeWrites() {
      return modeWrites;
    },
    get backCalls() {
      return backCalls;
    },
    render() {
      cursor = 0;
      return module.CloudflareView(props);
    },
    async poll(next = status) {
      status = next;
      await timer();
    },
    unmount() {
      for (const slot of slots) slot?.cleanup?.();
    }
  };
}

function findByType(node, type) {
  if (Array.isArray(node)) {
    for (const child of node) {
      const found = findByType(child, type);
      if (found) return found;
    }
    return;
  }
  if (!node || typeof node !== 'object') return;
  if (node.type === type) return node;
  for (const value of Object.values(node.props ?? {})) {
    const found = findByType(value, type);
    if (found) return found;
  }
}

const settle = () => new Promise((resolve) => setImmediate(resolve));

test('active Quick and Named tunnels restore their connection page and return directly home', async () => {
  for (const mode of ['quick', 'named']) {
    for (const state of ['connected', 'connecting', 'error']) {
      const status = {
        mode,
        state,
        enabled: true,
        running: state !== 'error',
        busy: false,
        accessUrl: 'https://kvm.example.com'
      };
      let resolveConfig;
      const harness = mountView(
        status,
        new Promise((resolve) => {
          resolveConfig = resolve;
        })
      );
      try {
        let tree = harness.render();
        await settle();
        tree = harness.render();
        assert.equal(
          findByType(tree, harness.ChooseMode),
          undefined,
          `${mode}/${state} before config arrives`
        );
        resolveConfig({ code: 0, data: { mode, accessUrl: status.accessUrl, tokenSet: true } });
        await settle();
        tree = harness.render();
        assert.ok(findByType(tree, harness.TunnelConnection), `${mode}/${state}`);
        assert.equal(findByType(tree, harness.ChooseMode), undefined);
        assert.equal(harness.modeWrites, 0);

        findByType(tree, harness.TunnelHeader).props.onBack();
        assert.equal(harness.backCalls, 1, `${mode}/${state} back should leave Cloudflare`);
        tree = harness.render();
        assert.equal(findByType(tree, harness.ChooseMode), undefined);
        await harness.poll();
        tree = harness.render();
        assert.ok(findByType(tree, harness.TunnelConnection));
        assert.equal(findByType(tree, harness.ChooseMode), undefined);
        assert.equal(harness.modeWrites, 0);
      } finally {
        harness.unmount();
      }
    }
  }
});

test('inactive tunnels still choose a mode and loading back returns to the home page', async () => {
  const harness = mountView({
    mode: 'named',
    state: 'notConfigured',
    enabled: false,
    running: false,
    busy: false
  });
  try {
    let tree = harness.render();
    findByType(tree, harness.TunnelHeader).props.onBack();
    assert.equal(harness.backCalls, 1);
    await settle();
    tree = harness.render();
    const picker = findByType(tree, harness.ChooseMode);
    assert.ok(picker);
    picker.props.onConfirm();
    tree = harness.render();
    assert.ok(findByType(tree, harness.TunnelConnection));
    await harness.poll();
    assert.equal(findByType(harness.render(), harness.ChooseMode), undefined);
    findByType(harness.render(), harness.TunnelHeader).props.onBack();
    tree = harness.render();
    const selection = findByType(tree, harness.ChooseMode);
    assert.ok(selection);
    selection.props.onChange('quick');
    await harness.poll({
      mode: 'named',
      state: 'connected',
      enabled: true,
      running: true,
      busy: false
    });
    assert.equal(findByType(harness.render(), harness.ChooseMode).props.value, 'quick');
  } finally {
    harness.unmount();
  }
});

test('Cloudflare badge distinguishes installation, pending connection and active forwarding', () => {
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

test('Tunnel Token and optional access address validation', () => {
  const { validateConfig, errorMessageKey } = load(`${component}/model.ts`);
  const token = btoa(
    JSON.stringify({ a: 'account', t: 'd1838763-3549-430f-a54e-692caf9f1653', s: btoa('secret') })
  );
  const form = { accessUrl: 'https://kvm.example.com', token };
  assert.equal(validateConfig(form), undefined);
  assert.equal(validateConfig({ ...form, accessUrl: '' }), undefined);
  assert.equal(validateConfig({ accessUrl: '' }, true), undefined);
  assert.equal(validateConfig({ accessUrl: '' }), 'invalidToken');
  for (const value of ['', 'account-api-token', 'cloudflared service install token', btoa('{}')])
    assert.equal(validateConfig({ ...form, token: value }, true), 'invalidToken');
  for (const value of [
    'http://kvm.example.com',
    'https://user:pass@kvm.example.com',
    'https://kvm.example.com/path',
    'https://kvm.example.com?token=secret',
    'https://kvm.example.com/#fragment',
    'https://kvm.example.com:99999',
    'javascript:alert(1)'
  ])
    assert.equal(validateConfig({ ...form, accessUrl: value }), 'invalidURL');
  assert.equal(errorMessageKey('untrusted cloudflared output'), 'cloudflare.failed');
});

test('URL-only saves do not send or replace the stored credential', () => {
  const calls = [];
  const api = load('src/api/extensions/cloudflare.ts', {
    '@/lib/http.ts': { http: { post: (...args) => calls.push(args) } }
  });
  api.setConfig({ accessUrl: 'https://kvm.example.com' }, 'challenge');
  assert.equal(calls[0][0], '/api/extensions/cloudflare/config');
  assert.equal(Object.hasOwn(calls[0][1], 'token'), false);
  assert.equal(calls[0][1].challenge, 'challenge');
});

test('Cloudflare ships complete messages for all supported languages', () => {
  const { cloudflareMessages } = load(`${component}/messages.ts`);
  const localeSource = fs.readFileSync(path.join(root, 'src/i18n/locale.ts'), 'utf8');
  const supported = [
    ...localeSource.match(/supportedLanguageKeys = \[([\s\S]*?)\]/)[1].matchAll(/'([^']+)'/g)
  ].map((m) => m[1]);
  assert.deepEqual(Object.keys(cloudflareMessages).sort(), supported.sort());
  const keys = Object.keys(cloudflareMessages.en).sort();
  for (const [language, bundle] of Object.entries(cloudflareMessages)) {
    assert.deepEqual(Object.keys(bundle).sort(), keys, language);
    for (const value of Object.values(bundle))
      assert.ok(typeof value === 'string' && value.trim(), language);
  }
});

test('Cloudflare protection requires verification when its policy is enabled or missing', async () => {
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
  assert.equal(
    (await gate.resolveProtectionRequiresChallenge('cloudflare')).requireChallenge,
    true
  );
  policy.cloudflare = false;
  assert.equal(
    (await gate.resolveProtectionRequiresChallenge('cloudflare')).requireChallenge,
    false
  );
  policy.cloudflare = true;
  assert.equal(
    (await gate.resolveProtectionRequiresChallenge('cloudflare')).requireChallenge,
    true
  );
  assert.equal(
    (await gate.resolveProtectionRequiresChallenge('tailscale')).requireChallenge,
    false
  );
  enabled = false;
  assert.equal(
    (await gate.resolveProtectionRequiresChallenge('cloudflare')).requireChallenge,
    false
  );
});
