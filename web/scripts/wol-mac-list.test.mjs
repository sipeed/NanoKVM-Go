import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const webRoot = path.resolve(import.meta.dirname, '..');

function source(relativePath) {
  return fs.readFileSync(path.join(webRoot, relativePath), 'utf8');
}

const wolSource = source('src/pages/desktop/menu/wol/index.tsx');
const mockStateSource = source('src/mocks/state.ts');
const mockDeviceSource = source('src/mocks/handlers/device.ts');

function sliceBetween(haystack, startMarker, endMarker) {
  const start = haystack.indexOf(startMarker);
  assert.notEqual(start, -1, `missing ${startMarker}`);
  const end = haystack.indexOf(endMarker, start);
  assert.notEqual(end, -1, `missing ${endMarker} after ${startMarker}`);
  return haystack.slice(start, end);
}

function countOccurrences(haystack, needle) {
  return haystack.split(needle).length - 1;
}

const getMacsBody = sliceBetween(wolSource, 'async function getMacs()', 'function toggleShow');
const setMacNameBody = sliceBetween(wolSource, 'async function setMacName', 'function deleteMac');
const deleteMacBody = sliceBetween(wolSource, 'function deleteMac', 'function wake');

test('getMacs toasts load failures on a nonzero code and on a transport rejection', () => {
  assert.match(getMacsBody, /try \{/);
  assert.match(
    getMacsBody,
    /rsp\.code !== 0[^]*console\.log\(rsp\.msg\);[^]*feedback\.error\(t\('wol\.errors\.loadFailed'\)\);/
  );
  assert.match(
    getMacsBody,
    /\} catch \(err\) \{[^]*console\.log\(err\);[^]*feedback\.error\(t\('wol\.errors\.loadFailed'\)\);/
  );
  assert.equal(
    countOccurrences(getMacsBody, "feedback.error(t('wol.errors.loadFailed'))"),
    3,
    'nonzero code, non-array payload, and transport rejection must all toast wol.errors.loadFailed'
  );
});

test('getMacs narrows the macs payload before parsing and drops the double write', () => {
  assert.match(getMacsBody, /const macs: unknown = rsp\.data\?\.macs;/);
  assert.match(getMacsBody, /if \(!Array\.isArray\(macs\)\)/);
  assert.equal(countOccurrences(getMacsBody, 'setMacList('), 1);
});

test('setMacName toasts rename failures inside the async body and keeps the binding voided', () => {
  assert.match(
    setMacNameBody,
    /rsp\.code !== 0[^]*console\.log\(rsp\.msg\);[^]*feedback\.error\(t\('wol\.errors\.renameFailed'\)\);/
  );
  assert.match(
    setMacNameBody,
    /\} catch \(err\) \{[^]*console\.log\(err\);[^]*feedback\.error\(t\('wol\.errors\.renameFailed'\)\);/
  );
  assert.equal(
    countOccurrences(setMacNameBody, "feedback.error(t('wol.errors.renameFailed'))"),
    2,
    'nonzero code and transport rejection must both toast wol.errors.renameFailed'
  );
  assert.match(wolSource, /onPressEnter=\{\(e\) => void setMacName\(e, item\.mac\)\}/);
});

test('deleteMac toasts delete failures on a nonzero code and on a transport rejection', () => {
  assert.match(
    deleteMacBody,
    /rsp\.code !== 0[^]*console\.log\(rsp\.msg\);[^]*feedback\.error\(t\('wol\.errors\.deleteFailed'\)\);/
  );
  assert.match(
    deleteMacBody,
    /\.catch\(\(err\) => \{[^]*console\.log\(err\);[^]*feedback\.error\(t\('wol\.errors\.deleteFailed'\)\);/
  );
  assert.equal(
    countOccurrences(deleteMacBody, "feedback.error(t('wol.errors.deleteFailed'))"),
    2,
    'nonzero code and transport rejection must both toast wol.errors.deleteFailed'
  );
});

test('wol menu does not import message from antd', () => {
  const antdImports = [...wolSource.matchAll(/import\s*(?:type\s+)?\{([^}]*)\}\s*from\s*'antd';/g)];
  for (const [, specifiers] of antdImports) {
    assert.doesNotMatch(specifiers, /\bmessage\b/);
  }
  assert.doesNotMatch(wolSource, /\bimport\s+message\b/);
  assert.match(wolSource, /import \{ feedback \} from '@\/lib\/feedback\.tsx';/);
});

test('WoL MAC mock state follows the backend string-list contract', () => {
  assert.match(mockStateSource, /wolMacs: string\[\];/);
  assert.doesNotMatch(mockStateSource, /wolMacs: Array<\{/);
  assert.match(mockStateSource, /wolMacs: \['02:00:00:00:00:10 Mock workstation'\]/);
  assert.match(mockDeviceSource, /line\.split\(' '\)\[0\] !== mac/);
  assert.match(mockDeviceSource, /line\.split\(' '\)\[0\] === mac/);
});

test('wol failure keys have non-empty copy in the baseline locales', () => {
  for (const locale of ['en', 'zh', 'ja']) {
    const file = path.join(webRoot, `src/i18n/locales/${locale}.ts`);
    const output = ts.transpileModule(fs.readFileSync(file, 'utf8'), {
      compilerOptions: { module: ts.ModuleKind.CommonJS },
      fileName: file
    }).outputText;
    const module = { exports: {} };
    vm.runInNewContext(output, { exports: module.exports, module }, { filename: file });

    for (const key of ['loadFailed', 'renameFailed', 'deleteFailed']) {
      const value = module.exports.default.translation.wol.errors[key];
      const message = `${locale}.ts must define a non-empty wol.errors.${key}`;
      assert.equal(typeof value, 'string', message);
      assert.ok(value.trim(), message);
    }
  }
});
