import assert from 'node:assert/strict';
import { access, mkdir, mkdtemp, readFile, rm } from 'node:fs/promises';
import path from 'node:path';
import test from 'node:test';
import { build, createServer } from 'vite';

const webRoot = path.resolve(import.meta.dirname, '..');
const cacheRoot = path.join(webRoot, '..', '.cache');

for (const mode of ['production', 'mocked']) {
  test(`release build excludes mock code and worker with mode=${mode}`, async () => {
    await mkdir(cacheRoot, { recursive: true });
    const outDir = await mkdtemp(path.join(cacheRoot, 'mocked-build-test-'));
    try {
      const result = await build({
        root: webRoot,
        mode,
        logLevel: 'silent',
        build: { outDir, emptyOutDir: true, reportCompressedSize: false }
      });
      const outputs = (Array.isArray(result) ? result : [result]).flatMap(
        (bundle) => bundle.output
      );
      const mockModules = outputs
        .filter((output) => output.type === 'chunk')
        .flatMap((chunk) => Object.keys(chunk.modules))
        .filter((id) =>
          /\/(?:src\/mocks|node_modules\/(?:msw|@mswjs))\//.test(id.replaceAll('\\', '/'))
        );

      assert.deepEqual(mockModules, [], 'mock dependencies must not enter release chunks');
      await assert.rejects(access(path.join(outDir, 'mockServiceWorker.js')), { code: 'ENOENT' });
      await access(path.join(outDir, 'index.html'));
      for (const file of ['sipeed.ico', 'manifest.webmanifest', 'input-debug.html']) {
        assert.deepEqual(
          await readFile(path.join(outDir, file)),
          await readFile(path.join(webRoot, 'public', file))
        );
      }
    } finally {
      await rm(outDir, { recursive: true, force: true });
    }
  });
}

test('mocked development still serves its service worker', async () => {
  const server = await createServer({
    root: webRoot,
    mode: 'mocked',
    logLevel: 'silent',
    server: { host: '127.0.0.1', port: 0 }
  });
  try {
    await server.listen();
    const response = await fetch(new URL('/mockServiceWorker.js', server.resolvedUrls.local[0]));
    assert.equal(response.status, 200);
    assert.equal(
      await response.text(),
      await readFile(path.join(webRoot, 'public/mockServiceWorker.js'), 'utf8')
    );
  } finally {
    await server.close();
  }
});
