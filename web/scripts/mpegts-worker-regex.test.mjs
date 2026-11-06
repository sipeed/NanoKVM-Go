import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { test } from 'node:test';

const webRoot = path.resolve(import.meta.dirname, '..');
const upstreamWrapperRegexSource =
  String.raw`/^function\s?\w*\(\w+,\s*\w+,\s*(\w+)\)/`;
const patchedWrapperRegexSource =
  String.raw`/^function\s?[\w$]*\([\w$]+,\s*[\w$]+,\s*([\w$]+)\)/`;
const upstreamWrapperRegex = /^function\s?\w*\(\w+,\s*\w+,\s*(\w+)\)/;
const patchedWrapperRegex = /^function\s?[\w$]*\([\w$]+,\s*[\w$]+,\s*([\w$]+)\)/;

function countOccurrences(source, target) {
  return source.split(target).length - 1;
}

test('official mpegts.js input contains the expected unpatched regex once', async () => {
  const packageRoot = path.join(webRoot, 'node_modules', 'mpegts.js');
  const [packageJson, bundle] = await Promise.all([
    readFile(path.join(packageRoot, 'package.json'), 'utf8'),
    readFile(path.join(packageRoot, 'dist', 'mpegts.js'), 'utf8')
  ]);

  assert.equal(JSON.parse(packageJson).version, '1.8.0');
  assert.equal(countOccurrences(bundle, upstreamWrapperRegexSource), 1);
  assert.equal(countOccurrences(bundle, patchedWrapperRegexSource), 0);
});

test('patched wrapper regex accepts dollar-sign identifiers', () => {
  const wrapper = 'function(ne,$,A)';

  assert.equal(upstreamWrapperRegex.exec(wrapper), null);
  assert.equal(patchedWrapperRegex.exec(wrapper)?.[1], 'A');
  assert.equal(patchedWrapperRegex.exec('function($worker,_exports,$require)')?.[1], '$require');
});
