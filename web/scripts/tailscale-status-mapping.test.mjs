import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const webRoot = path.resolve(import.meta.dirname, '..');
const tailscaleRoot = 'src/pages/desktop/menu/settings/remote-network/components/tailscale';

const jsxRuntime = {
  Fragment: Symbol('Fragment'),
  jsx(type, props) {
    return { type, props };
  },
  jsxs(type, props) {
    return { type, props };
  }
};

function runModule(relativePath, modules) {
  const file = path.join(webRoot, relativePath);
  const output = ts.transpileModule(fs.readFileSync(file, 'utf8'), {
    compilerOptions: {
      esModuleInterop: true,
      jsx: ts.JsxEmit.ReactJSX,
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
        if (!(id in modules)) throw new Error(`unexpected module: ${id}`);
        return modules[id];
      }
    },
    { filename: file }
  );

  return module.exports;
}

function loadStatusModel() {
  return runModule(`${tailscaleRoot}/status-model.ts`, {});
}

/** Drives TailscaleStatus with a stubbed status endpoint. */
function loadStatusComponent({ response, reject = false }) {
  const Badge = Symbol('RemoteNetworkStatusBadge');
  const statusModel = loadStatusModel();
  let state;

  const module = runModule(`${tailscaleRoot}/status.tsx`, {
    react: {
      useState(initial) {
        if (state === undefined) state = initial;
        return [
          state,
          (next) => {
            state = typeof next === 'function' ? next(state) : next;
          }
        ];
      },
      useEffect(effect) {
        effect();
      }
    },
    '@/api/extensions/tailscale.ts': {
      getStatus: () =>
        reject ? Promise.reject(new Error('unavailable')) : Promise.resolve(response)
    },
    '../../status-badge.tsx': { RemoteNetworkStatusBadge: Badge },
    './status-model.ts': statusModel,
    'react/jsx-runtime': jsxRuntime
  });

  return {
    Badge,
    /** First render: mounts the effect that requests the status. */
    mounted: module.TailscaleStatus(),
    render: () => module.TailscaleStatus(),
    settle: () => new Promise((resolve) => setTimeout(resolve, 0))
  };
}

test('Tailscale states map onto the shared home card statuses', () => {
  const { toRemoteNetworkStatus } = loadStatusModel();

  assert.equal(toRemoteNetworkStatus('running'), 'on');
  assert.equal(toRemoteNetworkStatus('installing'), 'busy');
  assert.equal(toRemoteNetworkStatus('stopped'), 'off');
  assert.equal(toRemoteNetworkStatus('notInstall'), 'off');
  assert.equal(toRemoteNetworkStatus('notRunning'), 'off');
  assert.equal(toRemoteNetworkStatus('notLogin'), 'off');
  assert.equal(toRemoteNetworkStatus('unknown-state'), 'bad');
  assert.equal(toRemoteNetworkStatus(undefined), 'bad');
  assert.equal(toRemoteNetworkStatus({ state: 'notInstall', managedState: 'enrolling' }), 'busy');
  assert.equal(
    toRemoteNetworkStatus({ state: 'notInstall', managedState: 'enabledConnecting' }),
    'busy'
  );
  assert.equal(toRemoteNetworkStatus({ state: 'running', managedState: 'enabledConnected' }), 'on');
  assert.equal(toRemoteNetworkStatus({ state: 'running', managedState: 'degraded' }), 'bad');
  assert.equal(toRemoteNetworkStatus({ state: 'running', managedState: 'invalidIdentity' }), 'bad');
});

test('the home card badge renders the authoritative Tailscale state', async () => {
  const harness = loadStatusComponent({ response: { code: 0, data: { state: 'running' } } });

  assert.equal(harness.mounted, null);
  await harness.settle();

  const badge = harness.render();
  assert.equal(badge.type, harness.Badge);
  assert.equal(badge.props.status, 'on');
});

test('a failed status response shows the error badge instead of a stale state', async () => {
  const failedResponse = loadStatusComponent({ response: { code: -1, data: undefined } });
  await failedResponse.settle();
  assert.equal(failedResponse.render().props.status, 'bad');

  const missingData = loadStatusComponent({ response: { code: 0, data: undefined } });
  await missingData.settle();
  assert.equal(missingData.render().props.status, 'bad');

  const rejected = loadStatusComponent({ response: undefined, reject: true });
  await rejected.settle();
  assert.equal(rejected.render().props.status, 'bad');
});
