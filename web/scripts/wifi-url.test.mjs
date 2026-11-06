import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const webRoot = path.resolve(import.meta.dirname, '..');
const helperFile = path.join(webRoot, 'src/pages/wifi/wifi-url.ts');
const wifiSource = fs.readFileSync(path.join(webRoot, 'src/pages/wifi/index.tsx'), 'utf8');

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
    module: { exports: {} },
    URL,
    URLSearchParams
  };
  vm.runInNewContext(output, context, { filename: helperFile });
  return context.exports;
}

const { consumeWifiUrlParams } = loadHelper();
const consumeWifiPassword = (url, replace) => consumeWifiUrlParams(url, replace).apPassword;

test('legacy lowercase and uppercase AP password parameters are consumed and scrubbed', () => {
  for (const [url, expected] of [
    ['http://device.local/#/wifi?p=lower-secret', 'lower-secret'],
    ['http://device.local/#/wifi?P=upper-secret', 'upper-secret']
  ]) {
    const replaced = [];
    assert.equal(
      consumeWifiPassword(url, (safeUrl) => replaced.push(safeUrl)),
      expected
    );
    assert.equal(replaced.length, 1);
    assert.equal(new URL(replaced[0]).hash, '#/wifi');
    assert.doesNotMatch(replaced[0], /lower-secret|upper-secret|[?&][pP]=/);
  }
});

test('both password spellings are removed while safe route and page queries survive', () => {
  const replaced = [];
  const password = consumeWifiPassword(
    'http://device.local/?language=en#/wifi?mode=setup&p=first&P=second&step=scan',
    (safeUrl) => replaced.push(safeUrl)
  );

  assert.equal(password, 'first');
  const safeUrl = new URL(replaced[0]);
  assert.equal(safeUrl.search, '?language=en');
  assert.equal(safeUrl.hash, '#/wifi?mode=setup&step=scan');
});

test('manual entry routes do not alter browser history', () => {
  for (const url of ['http://device.local/#/wifi', 'http://device.local/#/wifi?mode=setup']) {
    let replaced = false;
    assert.equal(
      consumeWifiPassword(url, () => {
        replaced = true;
      }),
      ''
    );
    assert.equal(replaced, false);
  }
});

test('QR provisioning credentials are consumed and scrubbed with the AP password', () => {
  const replaced = [];
  const params = consumeWifiUrlParams(
    'http://192.168.12.1/#/wifi?ssid=Office%20Wi-Fi&psk=s3cret%26%3D&step=scan',
    (safeUrl) => replaced.push(safeUrl)
  );

  assert.equal(params.apPassword, '');
  assert.equal(params.ssid, 'Office Wi-Fi');
  assert.equal(params.psk, 's3cret&=');
  assert.equal(replaced.length, 1);
  assert.equal(new URL(replaced[0]).hash, '#/wifi?step=scan');
  assert.doesNotMatch(replaced[0], /Office|s3cret|ssid=|psk=/);
});

test('an open network QR code yields an empty psk and a missing ssid stays null', () => {
  const open = consumeWifiUrlParams('http://192.168.12.1/#/wifi?ssid=Guest', () => {});
  assert.equal(open.ssid, 'Guest');
  assert.equal(open.psk, '');

  const passwordOnly = consumeWifiUrlParams('http://192.168.12.1/#/wifi?p=ap', () => {});
  assert.equal(passwordOnly.ssid, null);
});

test('component scrubs synchronously before automatic verification and keeps manual retry flow', () => {
  const consumeMatch = wifiSource.match(/const (\w+) = consumeWifiUrlParamsFromCurrentUrl\(\);/);
  assert.ok(consumeMatch, 'component must consume the URL secrets synchronously');

  const consumeIndex = wifiSource.indexOf(consumeMatch[0]);
  const verifyIndex = wifiSource.indexOf(
    `verifyPassword(${consumeMatch[1]}.apPassword`,
    consumeIndex
  );

  assert.ok(
    verifyIndex > consumeIndex,
    'URL scrubbing must happen before the verification API path'
  );
  assert.match(wifiSource, /onFinish=\{onVerifyFinish\}/);
  assert.match(wifiSource, /readInputString\(values\.apPassword\)/);
  assert.match(wifiSource, /await verifyPassword\(candidate\);/);
  assert.match(wifiSource, /else if \(rsp\?\.code === -1\) \{\s*setVerifyState\('denied'\);/);
  assert.match(wifiSource, /else if \(!silent\) \{\s*setVerifyState\('failed'\);/);

  // The page must never persist or log AP/Wi-Fi passwords. It may not touch
  // browser storage directly; the only sanctioned storage surface is the
  // view-mode preference helpers from lib/localstorage.ts, which must never
  // receive a password value.
  assert.doesNotMatch(wifiSource, /(?:localStorage|sessionStorage)\s*\./);
  assert.doesNotMatch(wifiSource, /console\.(?:log|error)\([^)]*pass/i);
  assert.doesNotMatch(wifiSource, /setWifiPortalViewMode\([^)]*pass/i);

  const storageImport = wifiSource.match(/import \{([^}]*)\} from '@\/lib\/localstorage\.ts';/);
  if (storageImport) {
    const names = storageImport[1]
      .split(',')
      .map((name) => name.trim())
      .sort();
    assert.deepEqual(names, ['getWifiPortalViewMode', 'setWifiPortalViewMode']);
  }
});
