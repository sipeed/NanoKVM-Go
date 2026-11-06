import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const webRoot = path.resolve(import.meta.dirname, '..');
const tailscaleRoot = 'src/pages/desktop/menu/settings/remote-network/components/tailscale';

function transpile(relativePath, replacements = []) {
  const file = path.join(webRoot, relativePath);
  let source = fs.readFileSync(file, 'utf8');
  for (const [pattern, replacement] of replacements)
    source = source.replaceAll(pattern, replacement);
  return {
    file,
    output: ts.transpileModule(source, {
      compilerOptions: {
        esModuleInterop: true,
        jsx: ts.JsxEmit.ReactJSX,
        module: ts.ModuleKind.CommonJS,
        target: ts.ScriptTarget.ES2022
      },
      fileName: file
    }).outputText
  };
}

function evaluate({ file, output }, modules, globals = {}) {
  const module = { exports: {} };
  vm.runInNewContext(
    output,
    {
      ...globals,
      exports: module.exports,
      module,
      require(id) {
        if (id in modules) return modules[id];
        throw new Error(`unexpected module: ${id}`);
      }
    },
    { filename: file }
  );
  return module.exports;
}

function loadCapability() {
  return evaluate(transpile(`${tailscaleRoot}/capability.ts`), {});
}

const icon = Symbol('Icon');

function loadModes(capability) {
  return evaluate(transpile(`${tailscaleRoot}/modes.ts`), {
    '@/components/icons/sipeed': { Sipeed: icon },
    '@/components/icons/tailscale': { Tailscale: icon },
    './capability.ts': capability
  });
}

test('backend capability is authoritative and missing capability fails closed', () => {
  const capability = loadCapability();
  assert.deepEqual(
    [...capability.getAllowedModeIds(['official', 'sipeed'])],
    ['official', 'sipeed']
  );
  assert.deepEqual([...capability.getAllowedModeIds(undefined)], ['official']);
  assert.deepEqual([...capability.getAllowedModeIds([])], ['official']);
  assert.deepEqual([...capability.getAllowedModeIds(['official'])], ['official']);
  assert.deepEqual([...capability.getAllowedModeIds(['sipeed'])], ['sipeed']);
});

test('mode metadata is filtered by backend capability', () => {
  const capability = loadCapability();
  const { getDefaultTailscaleModeId, getTailscaleMode, getTailscaleModes } = loadModes(capability);
  const both = getTailscaleModes(['official', 'sipeed']);
  const officialOnly = getTailscaleModes(['official']);

  assert.deepEqual(
    [...both].map((mode) => mode.id),
    ['sipeed', 'official']
  );
  assert.deepEqual(
    [...officialOnly].map((mode) => mode.id),
    ['official']
  );
  assert.equal(getDefaultTailscaleModeId(both), 'sipeed');
  assert.equal(getTailscaleMode(both, 'sipeed').recommended, true);
  assert.equal(getTailscaleMode(officialOnly, 'sipeed').id, 'official');

  // The manifest-off payload is official-only, so the mode picker must not
  // render: the view derives hasModeChoice from the backend mode count.
  assert.equal(officialOnly.length, 1);
  const view = fs.readFileSync(path.join(webRoot, `${tailscaleRoot}/view.tsx`), 'utf8');
  const declaration = view.match(/const hasModeChoice = ([^;]+);/);
  assert.ok(declaration, 'view.tsx must declare hasModeChoice');
  assert.ok(
    declaration[1].includes('availableModes'),
    'hasModeChoice must follow the backend mode list'
  );
  assert.match(declaration[1], /\.length\s*>\s*1/, 'hasModeChoice must need more than one mode');
});

test('legacy region-mode variables are absent and the pairing server is explicit', () => {
  const files = [
    'package.json',
    'README.md',
    ...fs
      .readdirSync(path.join(webRoot, `${tailscaleRoot}`), { recursive: true })
      .filter((entry) => typeof entry === 'string' && /\.(?:ts|tsx)$/.test(entry))
      .map((entry) => path.join(tailscaleRoot, entry))
  ];
  const source = files.map((file) => fs.readFileSync(path.join(webRoot, file), 'utf8')).join('\n');
  assert.doesNotMatch(source, /VITE_REMOTE_NETWORK_MODES/);
  assert.doesNotMatch(source, /NANOKVM_BUILD_REGION/);

  const overview = fs.readFileSync(
    path.join(webRoot, `${tailscaleRoot}/sipeed-overview.tsx`),
    'utf8'
  );
  assert.match(overview, /const SIPEED_CONTROL_URL = 'https:\/\/control\.tailnet\.sipeed\.com'/);
});

test('one build has no region-specific scripts', () => {
  const scripts = JSON.parse(fs.readFileSync(path.join(webRoot, 'package.json'), 'utf8')).scripts;
  assert.equal(scripts['dev:cn'], undefined);
  assert.equal(scripts['build:cn'], undefined);
});

test('sipeed overview derives its Intl locale from the app language', () => {
  const overview = fs.readFileSync(
    path.join(webRoot, `${tailscaleRoot}/sipeed-overview.tsx`),
    'utf8'
  );
  assert.match(overview, /getIntlLocale\(i18n\.resolvedLanguage \|\| i18n\.language\)/);
  assert.doesNotMatch(overview, /navigator\.language/);
});
