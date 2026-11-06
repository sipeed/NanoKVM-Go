import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

const updateSource = fs.readFileSync(
  path.resolve(import.meta.dirname, '../src/pages/desktop/menu/settings/update/index.tsx'),
  'utf8'
);
const previewSource = fs.readFileSync(
  path.resolve(import.meta.dirname, '../src/pages/desktop/menu/settings/update/preview.tsx'),
  'utf8'
);
const signatureVerificationSource = fs.readFileSync(
  path.resolve(
    import.meta.dirname,
    '../src/pages/desktop/menu/settings/update/signature-verification.tsx'
  ),
  'utf8'
);

test('every update-page action uses the shared busy boundary', () => {
  assert.match(
    updateSource,
    /isUpdating \|\|\s*isUpdateServerBusy \|\|\s*isPreviewBusy \|\|\s*isSignatureVerificationBusy \|\|\s*isAcknowledgingTerminal \|\|\s*pageStatus === 'loading'/
  );
  assert.match(updateSource, /<Preview[\s\S]*?disabled=\{isBusy \|\| isUpdateServerPending\}/);
  assert.match(updateSource, /<UpdateServer[\s\S]*?disabled=\{isBusy\}/);
  assert.match(updateSource, /<SignatureVerification[\s\S]*?disabled=\{isBusy\}/);
  assert.match(updateSource, /<ButtonBase[\s\S]*?disabled=\{isBusy\}/);
  assert.match(updateSource, /disabled=\{isBusy\}\s+onClick=\{openOfflineUpdate\}/);
  assert.match(updateSource, /target="_blank"\s+disabled=\{isBusy\}/);
  assert.match(updateSource, /okButtonProps=\{\{ disabled: !selectedFile \|\| isBusy \}\}/);
  assert.match(updateSource, /cancelButtonProps=\{\{ disabled: isBusy \}\}/);
  assert.match(updateSource, /settingsBusyLock\.setPreviewBusy\(busy\)/);
  assert.match(updateSource, /settingsBusyLock\.setSignatureVerificationBusy\(busy\)/);
  assert.match(updateSource, /settingsBusyLock\.setApplicationBusy/);
  assert.match(previewSource, /if \(disabled \|\| isLoading\) return/);
  assert.match(previewSource, /<Switch\s+[\s\S]*?disabled=\{disabled\}/);
  assert.match(
    signatureVerificationSource,
    /disabled=\{disabled \|\| isBusy \|\| !isStatusAvailable\}/
  );
});

test('update status and Preview requests use independent stale-response guards', () => {
  assert.match(updateSource, /const generationRef = useRef\(0\)/);
  assert.match(updateSource, /const operationIdRef = useRef\(''\)/);
  assert.match(updateSource, /requestGeneration !== generationRef\.current/);
  assert.match(updateSource, /isStaleApplicationUpdateStatus/);
  assert.match(previewSource, /if \(requestId !== requestIdRef\.current\) return/);
  assert.doesNotMatch(previewSource, /generationRef|requestGeneration/);
});

test('request errors reconcile status before releasing the settings lock', () => {
  assert.match(updateSource, /responseStatusFromError/);
  assert.match(updateSource, /await reconcileSubmittedOperation/);
  assert.match(updateSource, /if \(!busyRef\.current && !statusOwnsOutcome\(recovered\)\) \{/);
  assert.match(
    updateSource,
    /busyRef\.current = false;[\s\S]*?settingsBusyLock\.setApplicationBusy\(false\)/
  );
});
