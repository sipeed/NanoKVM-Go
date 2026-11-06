import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const webRoot = path.resolve(import.meta.dirname, '..');
const progressModelFile = path.join(webRoot, 'src/pages/desktop/menu/image/download-progress.ts');
const downloader = fs.readFileSync(
  path.join(webRoot, 'src/pages/desktop/menu/image/downloadImage.tsx'),
  'utf8'
);
const storageApi = fs.readFileSync(path.join(webRoot, 'src/api/storage.ts'), 'utf8');
const deviceMocks = fs.readFileSync(path.join(webRoot, 'src/mocks/handlers/device.ts'), 'utf8');

function loadProgressModel() {
  const source = fs.readFileSync(progressModelFile, 'utf8');
  const output = ts.transpileModule(source, {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022
    },
    fileName: progressModelFile
  }).outputText;
  const module = { exports: {} };

  vm.runInNewContext(output, { exports: module.exports, module }, { filename: progressModelFile });
  return module.exports;
}

const progressModel = loadProgressModel();

test('image download polling clears stale timer IDs and restarts after acceptance', () => {
  assert.match(
    downloader,
    /function stopStatusPolling\(\) \{[\s\S]*?clearInterval\(intervalId\.current\);[\s\S]*?intervalId\.current = undefined;/
  );
  assert.match(
    downloader,
    /function startStatusPolling\(\) \{[\s\S]*?stopStatusPolling\(\);[\s\S]*?intervalId\.current = setInterval\(getDownloadStatus, 2500\);/
  );
  assert.match(
    downloader,
    /downloadImage\(url\)[\s\S]*?\.then\(\(rsp\) => \{[\s\S]*?getDownloadStatus\(\);[\s\S]*?startStatusPolling\(\);/
  );
});

test('image download unmount cleanup stops polling, invalidates requests, and restores keyboard', () => {
  const effectMatch = downloader.match(
    /useEffect\(\(\) => \{[^]*?checkDiskEnabled\(\);[^]*?return \(\) => \{([^]*?)\};[^]*?\}, \[setIsKeyboardEnable\]\);/
  );
  assert.ok(effectMatch, 'mount effect must return an unmount cleanup');
  const cleanup = effectMatch[1];
  assert.match(cleanup, /stopStatusPolling\(\);/);
  assert.match(cleanup, /invalidateStatusRequests\(\);/);
  assert.match(cleanup, /setIsKeyboardEnable\(true\);/);
  assert.ok(
    cleanup.indexOf('stopStatusPolling()') < cleanup.indexOf('setIsKeyboardEnable(true)'),
    'polling stops before the keyboard atom is restored'
  );
  assert.ok(
    cleanup.indexOf('invalidateStatusRequests()') < cleanup.indexOf('setIsKeyboardEnable(true)'),
    'in-flight requests are invalidated before the keyboard atom is restored'
  );
});

test('image download hides the green URL status while preserving failures', () => {
  assert.doesNotMatch(downloader, /text-green-500/);
  assert.match(downloader, /status === 'failed' && \([\s\S]*?text-red-500[\s\S]*?\{log\}/);
  assert.match(
    downloader,
    /downloadActive && \(progress !== undefined \|\| transferSummary\)[\s\S]*?<Progress/
  );
});

test('image download maps only the known storage response to localized storage feedback', () => {
  assert.match(downloader, /const insufficientStorageResponseMessage = 'insufficient storage';/);
  assert.match(
    downloader,
    /rsp\.msg === insufficientStorageResponseMessage\s*\?\s*'download\.insufficientStorage'\s*:\s*'download\.failed'/
  );
  assert.doesNotMatch(downloader, /setLog\(rsp\.msg\)/);
});

test('image download modal uses one balanced full-width layout', () => {
  assert.match(downloader, /className="flex w-full flex-col gap-4 pt-2"/);
  assert.match(downloader, /className="min-w-0 flex-1"/);
  assert.match(downloader, /className="shrink-0"/);
  assert.doesNotMatch(downloader, /<Divider|min-w-\[300px\]|className="py-2"/);
});

test('image download formats speed, transferred size, and remaining time', () => {
  const details = progressModel.getDownloadTransferDetails(
    {
      downloadedBytes: Math.round(32.3 * 1024 * 1024),
      totalBytes: Math.round(2.7 * 1024 * 1024 * 1024),
      bytesPerSecond: Math.round(12.4 * 1024 * 1024),
      remainingSeconds: 15 * 60
    },
    'en-US'
  );

  assert.deepEqual(
    { ...details },
    {
      speed: '12.4 MB/s',
      transferred: '32.3 MB/2.7 GB',
      remaining: '15 minutes'
    }
  );
  assert.equal(progressModel.formatRemainingDuration(15 * 60, 'zh-CN'), '15 分钟');
  assert.equal(progressModel.formatRemainingDuration(32, 'zh-CN'), '32 秒钟');
  assert.equal(
    progressModel.formatDownloadTransferSummary(
      {
        speed: '4.8 MB/s',
        transferred: '34.3 MB/186.9 MB',
        remaining: '32 秒钟'
      },
      'zh-CN',
      '剩余 32 秒钟'
    ),
    '4.8 MB/s - 34.3 MB/186.9 MB，剩余 32 秒钟'
  );
  assert.equal(
    progressModel.formatDownloadTransferSummary(
      {
        speed: '4.8 MB/s',
        transferred: '34.3 MB/186.9 MB',
        remaining: '32 seconds'
      },
      'en-US',
      '32 seconds remaining'
    ),
    '4.8 MB/s - 34.3 MB/186.9 MB, 32 seconds remaining'
  );
});

test('image download preserves useful details when the total size is unknown or stalled', () => {
  const unknownTotal = progressModel.getDownloadTransferDetails(
    {
      downloadedBytes: 32 * 1024 * 1024,
      totalBytes: -1,
      bytesPerSecond: 4 * 1024 * 1024,
      remainingSeconds: -1
    },
    'en-US'
  );
  assert.deepEqual(
    { ...unknownTotal },
    { speed: '4 MB/s', transferred: '32 MB', remaining: undefined }
  );

  const stalled = progressModel.getDownloadTransferDetails(
    {
      downloadedBytes: 32 * 1024 * 1024,
      totalBytes: 64 * 1024 * 1024,
      bytesPerSecond: 0,
      remainingSeconds: -1
    },
    'en-US'
  );
  assert.deepEqual(
    { ...stalled },
    { speed: undefined, transferred: '32 MB/64 MB', remaining: undefined }
  );
});

test('image download resets transfer details and consumes typed backend metrics', () => {
  assert.match(storageApi, /downloadedBytes: number;/);
  assert.match(storageApi, /totalBytes: number;/);
  assert.match(storageApi, /bytesPerSecond: number;/);
  assert.match(storageApi, /remainingSeconds: number;/);
  assert.match(downloader, /setTransfer\(rsp\.data\)/);
  assert.ok((downloader.match(/setTransfer\(undefined\)/g) ?? []).length >= 4);
  assert.match(downloader, /formatDownloadTransferSummary\(/);
  assert.match(downloader, /t\('download\.remaining'/);
});

test('image download offers a real cancel action after the start request is accepted', () => {
  assert.match(storageApi, /http\.post\('\/api\/storage\/download\/image\/cancel'\)/);
  assert.match(deviceMocks, /http\.post\(`\$\{origin\}\/api\/storage\/download\/image\/cancel`/);
  assert.match(
    downloader,
    /function cancelDownload\(\) \{\s*if \(!canCancel \|\| isCancelling \|\| isChangingPause\) return;/
  );
  assert.match(
    downloader,
    /stopStatusPolling\(\);\s*invalidateStatusRequests\(\);\s*cancelImageDownload\(\)/
  );
  assert.match(
    downloader,
    /cancelImageDownload\(\)[\s\S]*?stopStatusPolling\(\);[\s\S]*?setStatus\('idle'\);/
  );
  assert.match(
    downloader,
    /downloadActive \?[\s\S]*?<Button[\s\S]*?danger[\s\S]*?disabled=\{!canCancel \|\| isChangingPause\}[\s\S]*?loading=\{isCancelling\}[\s\S]*?download\.cancel/
  );
  assert.match(
    downloader,
    /rsp\.data\.status === 'idle' && startRequestPending\.current[\s\S]*?return;/
  );
  assert.match(
    downloader,
    /setActionError\(t\('download\.cancelFailed'\)\);\s*getDownloadStatus\(\);\s*startStatusPolling\(\);/
  );
});

test('image download can pause and resume without losing cancel support', () => {
  assert.match(storageApi, /http\.post\('\/api\/storage\/download\/image\/pause'\)/);
  assert.match(storageApi, /http\.post\('\/api\/storage\/download\/image\/resume'\)/);
  assert.match(deviceMocks, /http\.post\(`\$\{origin\}\/api\/storage\/download\/image\/pause`/);
  assert.match(deviceMocks, /http\.post\(`\$\{origin\}\/api\/storage\/download\/image\/resume`/);
  assert.match(
    downloader,
    /rsp\.data\.status === 'in_progress' \|\| rsp\.data\.status === 'paused'/
  );
  assert.match(
    downloader,
    /const downloadActive = status === 'in_progress' \|\| status === 'paused';/
  );
  assert.match(
    downloader,
    /function changeDownloadPaused\(paused: boolean\) \{\s*if \(!canCancel \|\| isCancelling \|\| isChangingPause\) return;/
  );
  assert.match(
    downloader,
    /const request = paused \? pauseImageDownload\(\) : resumeImageDownload\(\);[\s\S]*?setStatus\(rsp\.data\.status\);/
  );
  assert.match(
    downloader,
    /onClick=\{\(\) => changeDownloadPaused\(status !== 'paused'\)\}[\s\S]*?t\(status === 'paused' \? 'download\.resume' : 'download\.pause'\)/
  );
  assert.match(
    downloader,
    /setActionError\(t\(paused \? 'download\.pauseFailed' : 'download\.resumeFailed'\)\);/
  );
});

test('paused downloads stop animating and hide stale speed and remaining time', () => {
  assert.match(
    downloader,
    /transferDetails && status === 'paused'[\s\S]*?speed: undefined, remaining: undefined/
  );
  assert.match(downloader, /status=\{status === 'paused' \? 'normal' : 'active'\}/);
});
