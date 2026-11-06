import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const webRoot = path.resolve(import.meta.dirname, '..');
const helperFile = path.join(webRoot, 'src/pages/wifi/wifi-visual.ts');

function loadHelper() {
  const source = fs.readFileSync(helperFile, 'utf8');
  const output = ts.transpileModule(source, {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022
    },
    fileName: helperFile
  }).outputText;
  const context = {
    exports: {},
    module: { exports: {} }
  };
  vm.runInNewContext(output, context, { filename: helperFile });
  return context.exports;
}

const { getWifiSignalLevel, getWifiBand } = loadHelper();

test('signal level maps dBm into four documented bands', () => {
  // strong: -50 dBm or better
  assert.equal(getWifiSignalLevel(-30), 'full');
  assert.equal(getWifiSignalLevel(-50), 'full');
  // good: -60..-51
  assert.equal(getWifiSignalLevel(-51), 'high');
  assert.equal(getWifiSignalLevel(-60), 'high');
  // weak: -70..-61
  assert.equal(getWifiSignalLevel(-61), 'low');
  assert.equal(getWifiSignalLevel(-70), 'low');
  // marginal: below -70
  assert.equal(getWifiSignalLevel(-71), 'zero');
  assert.equal(getWifiSignalLevel(-95), 'zero');
});

test('signal level degrades invalid readings to the weakest icon', () => {
  assert.equal(getWifiSignalLevel(Number.NaN), 'zero');
  assert.equal(getWifiSignalLevel(Number.POSITIVE_INFINITY), 'zero');
  assert.equal(getWifiSignalLevel(Number.NEGATIVE_INFINITY), 'zero');
});

test('band chip parses GHz labels from mocked and backend strings', () => {
  assert.equal(getWifiBand('2.4 GHz'), '2.4G');
  assert.equal(getWifiBand('5 GHz'), '5G');
  assert.equal(getWifiBand('6 GHz'), '6G');
  assert.equal(getWifiBand('5GHz'), '5G');
  assert.equal(getWifiBand(' 2.4 ghz '), '2.4G');
});

test('band chip parses normalized MHz channel frequencies', () => {
  assert.equal(getWifiBand('2412 MHz'), '2.4G');
  assert.equal(getWifiBand('2484 MHz'), '2.4G');
  assert.equal(getWifiBand('5180 MHz'), '5G');
  assert.equal(getWifiBand('5825 MHz'), '5G');
  assert.equal(getWifiBand('5955 MHz'), '6G');
  assert.equal(getWifiBand('7115 MHz'), '6G');
});

test('band chip stays hidden for unknown or out-of-range frequencies', () => {
  assert.equal(getWifiBand(''), null);
  assert.equal(getWifiBand('   '), null);
  assert.equal(getWifiBand('unknown'), null);
  assert.equal(getWifiBand('900 MHz'), null);
  assert.equal(getWifiBand('8000 MHz'), null);
  assert.equal(getWifiBand('4 GHz'), null);
});
