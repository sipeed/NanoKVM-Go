import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

const webRoot = path.resolve(import.meta.dirname, '..');

function source(relativePath) {
  return fs.readFileSync(path.join(webRoot, relativePath), 'utf8');
}

const imagesSource = source('src/pages/desktop/menu/image/images.tsx');

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

const deleteBody = sliceBetween(imagesSource, 'function deleteImage()', 'function openNotification');

test('image delete keeps the protection challenge gate around the executor', () => {
  assert.match(
    deleteBody,
    /runWithChallenge\('delete_image', 'deleteImage', async \(challenge\) =>/
  );
});

test('image delete toasts the domain failure next to the diagnostic log', () => {
  assert.match(deleteBody, /console\.log\(rsp\.msg\)/);
  assert.match(
    deleteBody,
    /else \{[^]*console\.log\(rsp\.msg\);[^]*feedback\.error\(t\('image\.deleteFailed'\)\);/
  );
});

test('image delete toasts the transport rejection inside the executor catch', () => {
  assert.match(
    deleteBody,
    /\} catch \{[^]*feedback\.error\(t\('image\.deleteFailed'\)\);[^]*return undefined;[^]*\} finally \{[^]*setDeletingImage\(''\);/
  );
  assert.equal(
    countOccurrences(deleteBody, "feedback.error(t('image.deleteFailed'))"),
    2,
    'domain failure and transport rejection must both toast image.deleteFailed'
  );
});

test('image delete does not import message from antd', () => {
  const antdImports = [
    ...imagesSource.matchAll(/import\s*(?:type\s+)?\{([^}]*)\}\s*from\s*'antd';/g)
  ];
  for (const [, specifiers] of antdImports) {
    assert.doesNotMatch(specifiers, /\bmessage\b/);
  }
  assert.doesNotMatch(imagesSource, /\bimport\s+message\b/);
  assert.match(imagesSource, /import \{ feedback \} from '@\/lib\/feedback\.tsx';/);
});

test('image.deleteFailed has non-empty copy in the baseline locales', () => {
  for (const locale of ['en', 'zh', 'ja']) {
    const localeSource = source(`src/i18n/locales/${locale}.ts`);
    assert.match(
      localeSource,
      /deleteFailed: '.+',\n\s*okBtn:/,
      `${locale}.ts must define a non-empty image.deleteFailed next to deleteConfirm`
    );
  }
});
