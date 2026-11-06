import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const webRoot = path.resolve(import.meta.dirname, '..');
const tailscaleRoot = 'src/pages/desktop/menu/settings/remote-network/components/tailscale';
const viewFile = path.join(webRoot, `${tailscaleRoot}/view.tsx`);
const installFile = path.join(webRoot, `${tailscaleRoot}/install.tsx`);
const typesFile = path.join(webRoot, `${tailscaleRoot}/types.ts`);
const indexSource = fs.readFileSync(viewFile, 'utf8');
const typesSource = fs.readFileSync(typesFile, 'utf8');

function loadInstall({ serverInstalling }) {
  const source = fs.readFileSync(installFile, 'utf8');
  const output = ts.transpileModule(source, {
    compilerOptions: {
      esModuleInterop: true,
      jsx: ts.JsxEmit.ReactJSX,
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022
    },
    fileName: installFile
  }).outputText;

  let apiInstallCalls = 0;
  let challengeCalls = 0;
  let lockCalls = [];
  let onSuccessCalls = 0;
  let pendingChallenge;

  const Button = Symbol('Button');
  const module = { exports: {} };
  const modules = {
    react: {
      useState(initialValue) {
        let value = initialValue;
        return [
          value,
          (nextValue) => {
            value = typeof nextValue === 'function' ? nextValue(value) : nextValue;
          }
        ];
      }
    },
    '@ant-design/icons': {
      DownloadOutlined: Symbol('DownloadOutlined'),
      InfoCircleOutlined: Symbol('InfoCircleOutlined')
    },
    antd: {
      Button,
      Card: Symbol('Card'),
      Result: Symbol('Result')
    },
    'react-i18next': {
      useTranslation() {
        return { t: (key) => key };
      }
    },
    '@/api/extensions/tailscale.ts': {
      async install() {
        apiInstallCalls += 1;
        return { code: 0 };
      }
    },
    '@/hooks/useProtectionChallengeGate.tsx': {
      useProtectionChallengeGate() {
        return {
          isGateBusy: false,
          modal: null,
          runWithChallenge(_action, _scope, callback) {
            challengeCalls += 1;
            pendingChallenge = callback(undefined);
            return pendingChallenge;
          }
        };
      }
    },
    '../../messages.ts': {
      useRemoteNetworkMessages() {
        return (key) => key;
      }
    },
    'react/jsx-runtime': {
      Fragment: Symbol('Fragment'),
      jsx(type, props) {
        return { type, props };
      },
      jsxs(type, props) {
        return { type, props };
      }
    }
  };

  vm.runInNewContext(
    output,
    {
      exports: module.exports,
      module,
      require(id) {
        if (!(id in modules)) throw new Error(`unexpected module: ${id}`);
        return modules[id];
      }
    },
    { filename: installFile }
  );

  const tree = module.exports.Install({
    isInstalling: serverInstalling,
    onBusyChange(value) {
      lockCalls.push(value);
    },
    async onSuccess() {
      onSuccessCalls += 1;
    }
  });

  function find(node, type) {
    if (!node || typeof node !== 'object') return undefined;
    if (node.type === type) return node;

    for (const value of Object.values(node.props ?? {})) {
      for (const child of Array.isArray(value) ? value : [value]) {
        const result = find(child, type);
        if (result) return result;
      }
    }
    return undefined;
  }

  return {
    button: find(tree, Button),
    calls: {
      get apiInstall() {
        return apiInstallCalls;
      },
      get challenge() {
        return challengeCalls;
      },
      get locks() {
        return lockCalls;
      },
      get onSuccess() {
        return onSuccessCalls;
      }
    },
    waitForChallenge: async () => pendingChallenge
  };
}

test('the Tailscale status contract includes the installing state', () => {
  assert.match(typesSource, /State = [^;]*'installing'/);
});

test('an installation recovered from the server shares the serial status poller', () => {
  assert.match(indexSource, /status\?\.state !== 'installing' &&/);
  assert.match(indexSource, /!isManagedTransition\(status\)/);
  assert.match(indexSource, /await getStatus\(false\);/);
  assert.match(indexSource, /STATUS_POLL_INTERVAL_MS = 1500/);
  assert.match(indexSource, /window\.setTimeout\(poll, STATUS_POLL_INTERVAL_MS\)/);
  assert.match(indexSource, /window\.clearTimeout\(timer\)/);
  assert.match(
    indexSource,
    /if \(statusRequestInFlightRef\.current\) \{\s*statusRefreshQueuedRef\.current = true;\s*return;\s*\}/
  );
  assert.match(indexSource, /statusRequestInFlightRef\.current = false;/);
  assert.match(indexSource, /isInstallRequestRunning \|\| status\?\.state === 'installing'/);
  assert.match(indexSource, /setLocked\(isInstalling \|\| isConnectionBusy\)/);
  assert.match(indexSource, /onBusyChange=\{setIsInstallRequestRunning\}/);
  assert.match(indexSource, /status\?\.state === 'installing' \|\|/);
  assert.match(indexSource, /isSipeedMode && status\?\.managedState !== 'enabledConnected'/);
  assert.match(indexSource, /isInstalling=\{status\.state === 'installing'\}/);
});

test('the recovered installing UI cannot start a second installation', async () => {
  const harness = loadInstall({ serverInstalling: true });
  assert.ok(harness.button);
  assert.equal(harness.button.props.loading, true);
  assert.equal(harness.button.props.disabled, true);
  assert.equal(harness.button.props.children, 'settings.tailscale.installing');

  harness.button.props.onClick();
  await harness.waitForChallenge();

  assert.equal(harness.calls.challenge, 0);
  assert.equal(harness.calls.apiInstall, 0);
  assert.deepEqual(harness.calls.locks, []);
  assert.equal(harness.calls.onSuccess, 0);
});

test('a user-started installation refreshes status after the request succeeds', async () => {
  const harness = loadInstall({ serverInstalling: false });
  assert.ok(harness.button);
  assert.equal(harness.button.props.disabled, false);

  harness.button.props.onClick();
  await harness.waitForChallenge();

  assert.equal(harness.calls.challenge, 1);
  assert.equal(harness.calls.apiInstall, 1);
  assert.equal(harness.calls.onSuccess, 1);
  assert.deepEqual(harness.calls.locks, [true, false]);
});
