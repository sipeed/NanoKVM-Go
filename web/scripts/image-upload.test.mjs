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

const api = source('src/api/storage.ts');
const uploader = source('src/pages/desktop/menu/image/uploader-images.tsx');
const uploadProgressFile = path.join(webRoot, 'src/pages/desktop/menu/image/upload-progress.ts');

function loadUploadProgressModel() {
  const output = ts.transpileModule(fs.readFileSync(uploadProgressFile, 'utf8'), {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022
    },
    fileName: uploadProgressFile
  }).outputText;
  const module = { exports: {} };

  vm.runInNewContext(output, { exports: module.exports, module }, { filename: uploadProgressFile });
  return module.exports;
}

const uploadProgress = loadUploadProgressModel();

test('image chunks use a raw cancellable request with server-owned metadata', () => {
  assert.match(api, /export type ImageUploadChunk = \{[\s\S]*?uploadId: string;/);
  assert.match(
    api,
    /export type ImageUploadOptions = \{[\s\S]*?signal\?: AbortSignal;[\s\S]*?onProgress\?: \(uploadedBytes: number\) => void;/
  );
  assert.match(api, /\{ signal, onProgress \}: ImageUploadOptions = \{\}/);
  assert.match(api, /url: '\/api\/storage\/image\/upload'/);
  assert.match(api, /params: metadata/);
  assert.match(api, /signal,/);
  assert.match(api, /onUploadProgress:[\s\S]*?onProgress\?\./);
  assert.match(api, /'Content-Type': 'application\/octet-stream'/);
  assert.match(api, /data: chunk/);
  assert.doesNotMatch(api, /multipart\/form-data|FormData/);
});

test('image upload has a dedicated incomplete-upload cancellation API', () => {
  assert.match(api, /export function cancelImageUpload\(/);
  assert.match(api, /http\.post\('\/api\/storage\/image\/upload\/cancel', \{ uploadId \}\)/);
  assert.doesNotMatch(
    api.slice(api.indexOf('export function cancelImageUpload'), api.indexOf('// delete image')),
    /deleteImage|image\/delete/
  );
});

test('image uploader keeps fixed sequential 8 MiB chunks', () => {
  assert.match(uploader, /const chunkSize = 8 \* 1024 \* 1024/);
  assert.match(uploader, /while \(chunkIndex < totalChunks\)/);
  assert.match(
    uploader,
    /if \(cancelRequestedRef\.current\)[\s\S]*?await uploadImage\([\s\S]*?uploadId: uploadID,[\s\S]*?filename: file\.name,[\s\S]*?chunkSize,[\s\S]*?chunkIndex,[\s\S]*?totalChunks[\s\S]*?signal: controller\.signal,[\s\S]*?onProgress:/
  );
  assert.match(uploader, /chunkIndex\+\+;[\s\S]*?setProgress/);
  assert.doesNotMatch(uploader, /new FormData|formData\.append|Promise\.all/);
});

test('image upload cleans its temporary session and releases state after an API error', () => {
  assert.match(uploader, /if \(code !== 0\) \{[\s\S]*?throw new ImageUploadRequestError/);
  assert.match(
    uploader,
    /catch \(error\) \{[\s\S]*?if \(!cancelRequestedRef\.current\) \{[\s\S]*?await cancelImageUpload\(uploadID\)/
  );
  assert.match(
    uploader,
    /finally \{[\s\S]*?uploadingRef\.current = false;[\s\S]*?updateUploadState\('idle'\);[\s\S]*?setProgress\(0\);/
  );
});

test('image uploader maps capacity rejections to localized copy without raw server messages', () => {
  assert.match(uploader, /const insufficientStorageResponseMessage = 'insufficient storage';/);
  assert.match(uploader, /const imageSizeLimitResponseMessage = 'image exceeds size limit';/);
  assert.match(
    uploader,
    /function imageUploadErrorKey\(msg: string\)[\s\S]*?'image\.uploadInsufficientStorage'[\s\S]*?'image\.uploadTooLarge'/
  );
  assert.match(uploader, /if \(code !== 0\) \{\s*console\.log\(/);
  assert.doesNotMatch(uploader, /setUploadError\(\s*(msg|error\.message)/);
  assert.doesNotMatch(uploader, /t\(\s*(msg|error\.message)\s*\)/);
  assert.doesNotMatch(uploader, /throw new Error\(msg/);
});

test('image upload reports failures inline in the modal like the download modal', () => {
  assert.match(uploader, /const \[uploadError, setUploadError\] = useState\(''\)/);
  assert.match(
    uploader,
    /setUploadError\(\s*error instanceof ImageUploadRequestError && error\.i18nKey\s*\? t\(error\.i18nKey\)\s*: t\('image\.uploadError'\)\s*\)/
  );
  assert.doesNotMatch(uploader, /setUploadError\(\s*(msg|error\.message)/);
  assert.match(
    uploader,
    /\{uploadError && \(\s*<div className="w-full break-words text-sm text-red-500">\{uploadError\}<\/div>\s*\)\}/
  );
  const uploadBody = uploader.slice(
    uploader.indexOf('async function performImageUpload'),
    uploader.indexOf('function chooseFile')
  );
  assert.match(uploadBody, /updateUploadState\('uploading'\);[\s\S]*?setUploadError\(''\);/);
  const chooseFileBody = uploader.slice(
    uploader.indexOf('function chooseFile'),
    uploader.indexOf('function selectFile')
  );
  assert.match(chooseFileBody, /setUploadError\(''\);/);
  const closeModalBody = uploader.slice(
    uploader.indexOf('function closeModal'),
    uploader.indexOf('async function cancelUpload')
  );
  assert.match(closeModalBody, /setUploadError\(''\);/);
});

test('image uploader omits the uploaded image list', () => {
  assert.doesNotMatch(uploader, /getImages/);
  assert.doesNotMatch(uploader, /t\('image\.uploaded'\)/);
  assert.doesNotMatch(uploader, /images\.map/);
});

test('image uploader uses the offline-update drag and drop layout with upload and cancel', () => {
  assert.match(uploader, /type UploadState = 'idle' \| 'uploading' \| 'cancelling'/);
  assert.match(uploader, /title=\{t\('image\.upload'\)\}/);
  assert.match(uploader, /okText=\{t\('image\.uploadAction'\)\}/);
  assert.match(uploader, /cancelText=\{t\('image\.cancelAction'\)\}/);
  assert.match(
    uploader,
    /okButtonProps=\{\{[\s\S]*?disabled: !selectedFile \|\| isBusy \|\| !storageWritable,[\s\S]*?loading: uploadState === 'uploading'[\s\S]*?\}\}/
  );
  assert.doesNotMatch(uploader, /confirmLoading=/);
  assert.match(uploader, /onOk=\{uploadSelectedFile\}/);
  assert.match(uploader, /onCancel=\{handleCancelAction\}/);
  assert.match(uploader, /loading: uploadState === 'cancelling'/);
  assert.doesNotMatch(uploader, /pauseAction|resumeAction|pauseUpload|resumeUpload|paused/);
  assert.doesNotMatch(uploader, /footer=\{null\}/);
  assert.match(uploader, /className="flex w-full flex-col gap-4 pt-2"/);
  assert.match(
    uploader,
    /flex min-h-36 flex-col items-center justify-center rounded-lg border border-dashed/
  );
  assert.match(uploader, /onClick=\{selectFile\}/);
  assert.match(uploader, /onDrop=\{handleDrop\}/);
  assert.match(uploader, /const acceptedImageFiles = '\.img,\.iso';/);
  assert.match(uploader, /accept=\{acceptedImageFiles\}/);
  assert.match(uploader, /\{t\('image\.dropFile'\)\}/);
  assert.match(uploader, /formatDownloadBytes\(selectedFile\.size, intlLocale\)/);
  assert.match(
    uploader,
    /<Progress[\s\S]*?percent=\{Math\.round\(progress\)\}[\s\S]*?status="active"/
  );
  assert.match(uploader, /tabular-nums text-neutral-400[\s\S]*?\{transferSummary\}/);
  assert.doesNotMatch(uploader, /notification|contextHolder/);
  assert.doesNotMatch(uploader, /UploadOutlined|<Button|width=\{600\}|ant-modal-title/);
});

test('image upload reports uploaded bytes, average speed, and remaining time', () => {
  const mebibyte = 1024 * 1024;
  const uploadedBytes = 34.3 * mebibyte;
  const totalBytes = 186.9 * mebibyte;
  const elapsedMilliseconds = (34.3 / 4.8) * 1000;
  const metrics = uploadProgress.getUploadTransferMetrics(
    uploadedBytes,
    totalBytes,
    elapsedMilliseconds
  );

  assert.equal(Math.round(metrics.uploadedBytes), Math.round(uploadedBytes));
  assert.equal(Math.round(metrics.totalBytes), Math.round(totalBytes));
  assert.equal(Math.round(metrics.bytesPerSecond), Math.round(4.8 * mebibyte));
  assert.equal(Math.ceil(metrics.remainingSeconds), 32);
  assert.equal(
    uploadProgress.getUploadTransferMetrics(mebibyte, totalBytes, 100).bytesPerSecond,
    undefined
  );
  assert.match(
    uploader,
    /getUploadTransferMetrics\([\s\S]*?performance\.now\(\) - uploadStartedAt/
  );
  assert.match(uploader, /formatDownloadTransferSummary\(/);
  assert.match(uploader, /t\('download\.remaining'/);
});

test('selecting or dropping an image waits for the upload action', () => {
  const selectionHandlers = uploader.slice(
    uploader.indexOf('function handleFileInput'),
    uploader.indexOf('function closeModal')
  );

  assert.match(selectionHandlers, /chooseFile\(event\.target\.files\?\.\[0\]\)/);
  assert.match(selectionHandlers, /chooseFile\(event\.dataTransfer\.files\?\.\[0\]\)/);
  assert.doesNotMatch(selectionHandlers, /uploadSelectedFile|uploadImage/);
  assert.match(
    uploader,
    /function uploadSelectedFile\(\) \{[\s\S]*?if \(!file \|\| uploadingRef\.current \|\| !storageWritable\) return;[\s\S]*?performImageUpload\(file\)/
  );
});

test('cancel works while uploading and cleans up after aborting the request', () => {
  assert.match(
    uploader,
    /async function cancelUpload\(\)[\s\S]*?updateUploadState\('cancelling'\);[\s\S]*?abortUploadRef\.current\?\.abort\(\)/
  );
  assert.match(
    uploader,
    /abortUploadRef\.current\?\.abort\(\);[\s\S]*?await cancelImageUpload\(uploadID\)/
  );
  assert.match(uploader, /await uploadTaskRef\.current;/);
  assert.match(uploader, /if \(cancelSucceeded\) closeModal\(\)/);
  assert.match(uploader, /updateUploadState\('cancelling'\);\s*setUploadError\(''\);/);
  assert.match(uploader, /setUploadError\(t\('image\.cancelUploadError'\)\)/);
});
