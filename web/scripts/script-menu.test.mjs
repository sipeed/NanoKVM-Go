import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

const webRoot = path.resolve(import.meta.dirname, '..');
const menuSource = fs.readFileSync(
  path.join(webRoot, 'src/pages/desktop/menu/script/index.tsx'),
  'utf8'
);

function sliceBetween(startMarker, endMarker) {
  const start = menuSource.indexOf(startMarker);
  assert.notEqual(start, -1, `missing ${startMarker}`);
  const end = menuSource.indexOf(endMarker, start);
  assert.notEqual(end, -1, `missing ${endMarker} after ${startMarker}`);
  return menuSource.slice(start, end);
}

function countOccurrences(haystack, needle) {
  return haystack.split(needle).length - 1;
}

test('script menu imports the shared feedback toast surface', () => {
  assert.match(menuSource, /import \{ feedback \} from '@\/lib\/feedback\.tsx';/);
  const antdImports = [
    ...menuSource.matchAll(/import\s*(?:type\s+)?\{([^}]*)\}\s*from\s*'antd';/g)
  ];
  for (const [, specifiers] of antdImports) {
    assert.doesNotMatch(specifiers, /\bmessage\b/);
  }
  assert.doesNotMatch(menuSource, /\bimport\s+message\b/);
});

test('script upload toasts both failure paths and still clears isUploading', () => {
  const body = sliceBetween('function completeUpload', 'function executeScript');

  assert.match(body, /console\.log\(rsp\.msg\)/);
  assert.equal(
    countOccurrences(body, "feedback.error(t('script.uploadFailed'))"),
    2,
    'domain failure and transport rejection must both toast script.uploadFailed'
  );
  assert.match(body, /\.catch\(\(\) => \{[^]*feedback\.error/);
  assert.match(body, /\.finally\(\(\) => \{[^]*setIsUploading\(false\)/);
  assert.match(body, /setScripts\(\[\.\.\.scripts, rsp\.data\.file\]\)/);
});

test('background script run toasts both failure paths with the shared runFailed key', () => {
  const body = sliceBetween('function executeScript', 'function uploadFile');

  assert.match(body, /type === 'foreground'/);
  assert.match(body, /console\.log\(rsp\.msg\)/);
  assert.equal(
    countOccurrences(body, "feedback.error(t('script.runFailed'))"),
    2,
    'domain failure and transport rejection must both toast script.runFailed'
  );
  assert.match(body, /\.catch\(\(\) => \{[^]*feedback\.error/);
});

test('script delete toasts both failure paths and keeps the success path intact', () => {
  const body = sliceBetween('function completeDelete', 'function deleteScript');

  assert.match(body, /console\.log\(rsp\.msg\)/);
  assert.equal(
    countOccurrences(body, "feedback.error(t('script.deleteFailed'))"),
    2,
    'domain failure and transport rejection must both toast script.deleteFailed'
  );
  assert.match(body, /\.catch\(\(\) => \{[^]*feedback\.error/);
  assert.match(body, /setScripts\(\(prev\) => prev\.filter\(\(name\) => name !== script\)\)/);
  assert.match(body, /setCurrentScript\(\(current\) => \(current === script \? '' : current\)\)/);
});
