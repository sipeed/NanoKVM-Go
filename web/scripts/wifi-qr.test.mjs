import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const webRoot = path.resolve(import.meta.dirname, '..');
const helperFile = path.join(webRoot, 'src/pages/wifi/wifi-qr.ts');

function loadHelper() {
  const source = fs.readFileSync(helperFile, 'utf8');
  const output = ts.transpileModule(source, {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022
    },
    fileName: helperFile
  }).outputText;
  const context = { exports: {}, module: { exports: {} }, URL, URLSearchParams, TextEncoder };
  vm.runInNewContext(output, context, { filename: helperFile });
  return context.exports;
}

const { parseWifiQrText, normalizeQrWifi } = loadHelper();
const plain = (value) => (value ? { ...value } : value);

test('WIFI: payloads from phone sharing are parsed in any field order', () => {
  assert.deepEqual(plain(parseWifiQrText('WIFI:T:WPA;S:Office;P:hunter22;;')), {
    ssid: 'Office',
    password: 'hunter22'
  });
  assert.deepEqual(plain(parseWifiQrText('WIFI:S:Office;T:SAE;P:hunter22;H:false;;')), {
    ssid: 'Office',
    password: 'hunter22'
  });
  assert.deepEqual(plain(parseWifiQrText('wifi:t:WPA;s:Lower;p:pw123456;;')), {
    ssid: 'Lower',
    password: 'pw123456'
  });
});

test('WIFI: escapes are decoded', () => {
  assert.deepEqual(plain(parseWifiQrText('WIFI:T:WPA;S:a\\;b\\:c\\\\d;P:x\\,y\\"z;;')), {
    ssid: 'a;b:c\\d',
    password: 'x,y"z'
  });
});

test('WIFI: quoted values and an unterminated last field are accepted', () => {
  assert.deepEqual(plain(parseWifiQrText('WIFI:T:WPA;S:"Quoted Net";P:"semi;colon";;')), {
    ssid: 'Quoted Net',
    password: 'semi;colon'
  });
  assert.deepEqual(plain(parseWifiQrText('WIFI:T:WPA;P:lastpass;S:Tail')), {
    ssid: 'Tail',
    password: 'lastpass'
  });
});

test('WIFI: open networks ignore any password field', () => {
  assert.deepEqual(plain(parseWifiQrText('WIFI:T:nopass;S:Guest;P:ignored;;')), {
    ssid: 'Guest',
    password: ''
  });
  assert.deepEqual(plain(parseWifiQrText('WIFI:S:Guest;;')), { ssid: 'Guest', password: '' });
});

test('provisioning URLs are parsed from photos too', () => {
  assert.deepEqual(
    plain(parseWifiQrText('http://192.168.12.1/#/wifi?ssid=%E5%AE%9E%E9%AA%8C%E5%AE%A4&psk=a%26b')),
    { ssid: '实验室', password: 'a&b' }
  );
});

test('unrelated or invalid codes are rejected', () => {
  for (const text of [
    'https://example.com/',
    'http://192.168.12.1/#/settings?ssid=x',
    'http://192.168.12.1/#/wifi?psk=only',
    'WIFI:T:WPA;P:nossid;;',
    'WIFI:T:WPA;S:   ;P:x;;',
    `WIFI:T:WPA;S:${'a'.repeat(33)};P:x;;`,
    `WIFI:T:WPA;S:ok;P:${'p'.repeat(129)};;`,
    'not a url'
  ]) {
    assert.equal(parseWifiQrText(text), null, text);
  }
});

test('SSID length is limited in UTF-8 bytes, not characters', () => {
  assert.ok(normalizeQrWifi('实'.repeat(10), ''));
  assert.equal(normalizeQrWifi('实'.repeat(11), ''), null);
});
