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

function loadModel() {
  const file = path.join(webRoot, 'src/pages/desktop/menu/settings/update/model.ts');
  const output = ts.transpileModule(source('src/pages/desktop/menu/settings/update/model.ts'), {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
    fileName: file
  }).outputText;
  const exports = {};
  const context = {
    exports,
    module: { exports },
    require() {
      return {};
    }
  };
  vm.runInNewContext(output, context, { filename: file });
  return context.module.exports;
}

const page = source('src/pages/desktop/menu/settings/update/index.tsx');
const api = source('src/api/application.ts');
const updating = source('src/pages/desktop/menu/settings/update/updating.tsx');
const updateMessages = source('src/i18n/update.ts');
const preview = source('src/pages/desktop/menu/settings/update/preview.tsx');
const signatureVerification = source(
  'src/pages/desktop/menu/settings/update/signature-verification.tsx'
);
const authApi = source('src/api/auth.ts');
const model = loadModel();

function deferred() {
  let resolve;
  let reject;
  const promise = new Promise((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
}

function fakeTimers() {
  let nextId = 0;
  const pending = new Map();
  return {
    setTimer(callback, delayMs) {
      const id = ++nextId;
      pending.set(id, { callback, delayMs });
      return id;
    },
    clearTimer(id) {
      pending.delete(id);
    },
    get size() {
      return pending.size;
    },
    runNext() {
      const entry = pending.entries().next().value;
      assert.ok(entry, 'expected a pending timer');
      const [id, timer] = entry;
      pending.delete(id);
      timer.callback();
      return timer.delayMs;
    }
  };
}

async function flushPromises() {
  await Promise.resolve();
  await Promise.resolve();
}

test('offline API sends one FormData file with browser-owned boundary and cancellation', () => {
  assert.match(api, /data\.append\('file', file\)/);
  assert.match(api, /http\.post\('\/api\/application\/update\/offline'/);
  assert.match(api, /signal,/);
  assert.match(api, /onUploadProgress:/);
  assert.doesNotMatch(api, /multipart\/form-data/);
});

test('terminal update results are acknowledged before returning to version checks', () => {
  assert.match(
    api,
    /http\.post\('\/api\/application\/update\/status\/acknowledge', \{ operationId \}\)/
  );
  assert.match(
    page,
    /clearPollTimer\(\);\s*let requestGeneration = beginGeneration\(\);[\s\S]*?await api\.acknowledgeUpdateStatus\(terminal\.operationId\)/
  );
  assert.match(
    page,
    /rsp\.data\.phase === initialUpdateStatus\.phase[\s\S]*?requestGeneration = beginGeneration\(\);\s*await recoverInitialStatus\(requestGeneration\)/
  );
  assert.match(
    page,
    /const shouldAcknowledge =\s*!!terminal\.operationId && \(terminal\.phase === 'succeeded' \|\| terminal\.phase === 'failed'\)/
  );
  assert.match(page, /if \(acknowledgingTerminalRef\.current\) return/);
  assert.match(page, /terminalActionError \|\| updateStatus\.version/);
  assert.match(
    page,
    /terminalActionError \|\| \(pageStatus === 'query-failed' \? undefined : errMsg\)/
  );
});

test('successful updates acknowledge idle before refreshing the page once', () => {
  assert.match(page, /const autoRefreshOperationRef = useRef\(''\)/);
  assert.match(
    page,
    /pageStatus !== 'succeeded'[\s\S]*?updateStatus\.phase !== 'succeeded'[\s\S]*?autoRefreshOperationRef\.current === operationId/
  );
  assert.match(
    page,
    /autoRefreshOperationRef\.current = operationId;\s*void resetTerminalStatus\(true\)/
  );
  assert.match(
    page,
    /const acknowledgedIdle =[\s\S]*?rsp\.data\.phase === initialUpdateStatus\.phase[\s\S]*?if \(acknowledgedIdle\)[\s\S]*?reloadAfterSuccess && terminal\.phase === 'succeeded'[\s\S]*?window\.location\.reload\(\)/
  );
  assert.ok(
    page.indexOf('window.location.reload()') >
      page.indexOf('await api.acknowledgeUpdateStatus(terminal.operationId)'),
    'the page must not reload before the terminal status is acknowledged'
  );
  assert.match(page, /onClick=\{\(\) => void resetTerminalStatus\(\)\}/);
});

test('page restores authoritative status before requesting version metadata', () => {
  assert.match(page, /await recoverInitialStatus\(requestGeneration\)/);
  assert.match(
    page,
    /if \(next\.phase === 'idle'\)[\s\S]*?operationIdRef\.current = '';[\s\S]*?checkForUpdates\(idleGeneration\)/
  );
  assert.match(page, /STATUS_RETRY_DELAYS_MS/);
  assert.match(page, /operationIdRef/);
  assert.match(page, /generationRef/);
  assert.match(page, /const versionRequest = \+\+versionRequestRef\.current/);
  assert.match(page, /versionRequest !== versionRequestRef\.current/);
  assert.match(
    page,
    /requestGeneration !== generationRef\.current \|\|\s*busyRef\.current \|\|\s*operationIdRef\.current \|\|\s*versionCheckingRef\.current/
  );
  assert.match(
    page,
    /catch \{[\s\S]*?versionRequest !== versionRequestRef\.current \|\|[\s\S]*?operationIdRef\.current[\s\S]*?return;/
  );
});

test('online, offline, preview, and navigation share one busy boundary', () => {
  assert.match(
    page,
    /isUpdating \|\|[\s\S]*?isPreviewBusy \|\|[\s\S]*?isSignatureVerificationBusy \|\|[\s\S]*?isAcknowledgingTerminal \|\|[\s\S]*?pageStatus === 'loading'/
  );
  assert.match(page, /<Preview[\s\S]*?disabled=\{isBusy \|\| isUpdateServerPending\}/);
  assert.match(page, /<SignatureVerification[\s\S]*?disabled=\{isBusy\}/);
  assert.match(page, /okButtonProps=\{\{ disabled: !selectedFile \|\| isBusy \}\}/);
  assert.match(page, /settingsBusyLock\.setApplicationBusy\(true\)/);
  assert.match(preview, /if \(disabled \|\| isLoading\) return/);
});

test('application, Preview, and signature settings independently own the navigation lock', () => {
  const changes = [];
  const lock = model.createSettingsBusyLock((locked) => changes.push(locked));

  lock.setPreviewBusy(true);
  assert.equal(lock.isLocked(), true, 'Preview busy locks close and tab navigation');
  assert.deepEqual(changes, [true]);

  lock.setApplicationBusy(true);
  lock.setPreviewBusy(false);
  assert.equal(lock.isLocked(), true, 'Preview finally must not unlock an active update');
  assert.deepEqual(changes, [true]);

  lock.setPreviewBusy(true);
  lock.setApplicationBusy(false);
  assert.equal(lock.isLocked(), true, 'an update terminal state must not unlock busy Preview');
  assert.deepEqual(changes, [true]);

  lock.setPreviewBusy(false);
  assert.equal(lock.isLocked(), false);

  lock.setSignatureVerificationBusy(true);
  assert.equal(lock.isLocked(), true, 'signature re-auth locks Settings navigation');
  lock.setApplicationBusy(true);
  lock.setSignatureVerificationBusy(false);
  assert.equal(lock.isLocked(), true, 'signature completion must not unlock an active update');
  lock.setApplicationBusy(false);
  assert.equal(lock.isLocked(), false);

  assert.deepEqual(changes, [true, false, true, false]);
});

test('idle unmount releases Preview ownership but active update ownership is retained', () => {
  const changes = [];
  const lock = model.createSettingsBusyLock((locked) => changes.push(locked));
  lock.setPreviewBusy(true);
  lock.releaseIfApplicationIdle();
  assert.equal(lock.isLocked(), false);

  lock.setApplicationBusy(true);
  lock.setPreviewBusy(true);
  lock.releaseIfApplicationIdle();
  assert.equal(lock.isLocked(), true);
  lock.setPreviewBusy(false);
  assert.equal(lock.isLocked(), true);
  assert.deepEqual(changes, [true, false, true]);
});

test('server settings keep navigation locked until concurrent settings and updates finish', () => {
  const changes = [];
  const lock = model.createSettingsBusyLock((locked) => changes.push(locked));
  lock.setUpdateServerBusy(true);
  lock.setPreviewBusy(true);
  lock.setSignatureVerificationBusy(true);
  lock.setPreviewBusy(false);
  lock.setSignatureVerificationBusy(false);
  assert.equal(lock.isLocked(), true, 'loading other settings must not unlock the server save');
  lock.setApplicationBusy(true);
  lock.setUpdateServerBusy(false);
  assert.equal(lock.isLocked(), true, 'completing the server save must not unlock an update');
  lock.releaseIfApplicationIdle();
  assert.equal(lock.isLocked(), true);
  lock.setApplicationBusy(false);
  assert.equal(lock.isLocked(), false);
  lock.setUpdateServerBusy(true);
  lock.releaseIfApplicationIdle();
  assert.equal(lock.isLocked(), false, 'idle unmount releases the server setting lock');
  assert.deepEqual(changes, [true, false, true, false]);
});

test('Preview cleanup releases only its owner and invalidates late requests', () => {
  assert.match(preview, /const requestId = \+\+requestIdRef\.current/);
  assert.match(preview, /if \(requestId !== requestIdRef\.current\) return/);
  assert.doesNotMatch(preview, /generationRef|requestGeneration/);
  assert.match(
    preview,
    /return \(\) => \{\s*requestIdRef\.current \+= 1;\s*onBusyChange\(false\);\s*\}/
  );
  assert.match(
    page,
    /createSettingsBusyLock\(\(locked\) =>\s*setIsLockedRef\.current\(locked\)\s*\)/
  );
});

test('upload UI requires confirmation and presentation component owns no transport', () => {
  const previewEntry = page.indexOf('<Preview');
  const signatureEntry = page.indexOf('<SignatureVerification', previewEntry);
  const offlineEntry = page.indexOf('onClick={openOfflineUpdate}', signatureEntry);
  const resultArea = page.indexOf('className="flex min-h-[400px]', offlineEntry);

  assert.ok(previewEntry >= 0);
  assert.ok(signatureEntry > previewEntry, 'signature control must appear below Preview');
  assert.ok(
    offlineEntry > signatureEntry,
    'offline update entry must appear below signature control'
  );
  assert.ok(resultArea > offlineEntry, 'offline update entry must appear above update results');
  assert.match(
    page,
    /<Preview[\s\S]*?\/>\s*<UpdateServer[\s\S]*?\/>\s*<Divider className="opacity-50" \/>\s*<SignatureVerification[\s\S]*?\/>\s*<Divider className="opacity-50" \/>\s*<ButtonBase/
  );
  assert.match(page, /<ButtonBase[\s\S]*?onClick=\{openOfflineUpdate\}/);
  assert.match(page, /const \[isOfflineModalOpen, setIsOfflineModalOpen\] = useState\(false\)/);
  assert.match(page, /open=\{isOfflineModalOpen\}/);
  assert.match(page, /onOk=\{confirmOfflineUpdate\}/);
  assert.match(page, /okButtonProps=\{\{ disabled: !selectedFile \|\| isBusy \}\}/);
  assert.doesNotMatch(page, /t\('settings\.update\.fileHint'\)/);
  assert.match(page, /const \[modal, contextHolder\] = Modal\.useModal\(\)/);
  assert.match(page, /modal\.confirm\(\{/);
  assert.match(page, /\{contextHolder\}/);
  assert.doesNotMatch(page, /Modal\.confirm/);
  assert.match(page, /onDrop=\{handleDrop\}/);
  assert.match(page, /api\.updateOffline/);
  assert.doesNotMatch(page, /UploadOutlined|settings\.update\.selectFile/);
  assert.match(updating, /import \{ ButtonBase \} from '@\/components\/ui\/button-base\.tsx'/);
  assert.match(updating, /<ButtonBase[\s\S]*?bg-neutral-800\/80[\s\S]*?onClick=\{onCancelUpload\}/);
  assert.doesNotMatch(updating, /<button/);
  assert.doesNotMatch(updating, /settings\.update\.source(?:Offline|Online)|status\.version/);
  assert.match(
    updating,
    /verifying: 1,\s*staging: 1,\s*installing: 2,\s*restarting: 3,\s*succeeded: 3/
  );
  assert.doesNotMatch(updating, /settings\.update\.prepare/);
  assert.doesNotMatch(updating, /websocket|client\.on|setInterval|api\./i);
  assert.match(updating, /settings\.update\.upload/);
  assert.match(updating, /settings\.update\.download/);
  assert.match(updating, /settings\.update\.phases\.\$\{status\.phase\}/);
  assert.match(updating, /<Alert[\s\S]*?settings\.update\.powerWarning/);
  assert.match(updating, /<Alert\s+className="w-fit max-w-full"/);
  assert.doesNotMatch(updating, /<Progress/);
  assert.doesNotMatch(updating, /import \{[^}]*\bProgress\b[^}]*\} from 'antd'/);
  assert.match(
    updating,
    /const isTransferPhase = status\.phase === 'downloading' \|\| status\.phase === 'uploading'/
  );
  assert.match(
    updating,
    /status\.phase === 'uploading' && uploadProgress !== undefined[\s\S]*?uploadProgress[\s\S]*?: status\.progress/
  );
  assert.match(
    updating,
    /icon: isTransferPhase \? \([\s\S]*?rounded-full[\s\S]*?\{displayedTransferProgress\}/
  );
  assert.doesNotMatch(updating, /\{displayedTransferProgress\}%/);
  assert.match(updating, /flex size-6 items-center justify-center/);
  assert.doesNotMatch(updating, /update-transfer-progress/);
  const transferStatusLabels = [
    ...updateMessages.matchAll(/downloading: '([^']+)',\s*uploading: '([^']+)'/g)
  ];
  assert.equal(transferStatusLabels.length, 20);
  for (const [, downloading, uploading] of transferStatusLabels) {
    assert.match(downloading, /…$/);
    assert.match(uploading, /…$/);
  }
  assert.equal([...updateMessages.matchAll(/^  upload: '([^']+)'/gm)].length, 20);
  assert.equal([...updateMessages.matchAll(/^  powerWarning: '([^']+)'/gm)].length, 20);
});

test('signature verification is fail-closed and every change requires re-authentication', () => {
  assert.match(api, /http\.get\('\/api\/application\/signature-verification'\)/);
  assert.match(api, /http\.post\('\/api\/application\/signature-verification', data\)/);
  assert.match(api, /challenge \? \{ enabled, challenge \} : \{ enabled \}/);
  assert.match(authApi, /'update_signature_verification'/);

  assert.match(signatureVerification, /const \[isEnabled, setIsEnabled\] = useState\(true\)/);
  assert.match(signatureVerification, /setIsEnabled\(true\);[\s\S]*?setIsStatusAvailable\(false\)/);
  assert.match(signatureVerification, /disabled=\{disabled \|\| isBusy \|\| !isStatusAvailable\}/);
  assert.match(signatureVerification, /authApi\s*\.getTwoFactorStatus\(\)/);
  assert.match(
    signatureVerification,
    /verifySensitiveAction\([\s\S]*?'update_signature_verification'/
  );
  assert.match(
    signatureVerification,
    /isTwoFactorEnabled \? undefined : password,[\s\S]*?isTwoFactorEnabled \? trimmedCode : ''/
  );
  assert.match(signatureVerification, /data\.challenge/);
  assert.match(signatureVerification, /danger=\{isDisabling\}/);
  assert.match(signatureVerification, /requirePassword=\{!isTwoFactorEnabled\}/);
});

test('status model validates phases and maps stable errors', () => {
  const good = {
    operationId: 'op-1',
    source: 'offline',
    version: '0.0.25',
    phase: 'verifying',
    progress: 100,
    errorCode: '',
    startedAt: '2026-08-12T00:00:00Z',
    updatedAt: '2026-08-12T00:00:01Z'
  };
  assert.equal(model.isApplicationUpdateStatus(good), true);
  assert.equal(model.isApplicationUpdateStatus({ ...good, phase: 'unknown' }), false);
  assert.equal(model.isUpdateBusyPhase('restarting'), true);
  assert.equal(model.isUpdateBusyPhase('failed'), false);
  assert.equal(
    model.updateErrorTranslationKey('signature_invalid'),
    'settings.update.errors.signature_invalid'
  );
  assert.equal(
    model.updateErrorTranslationKey('internal-secret'),
    'settings.update.errors.generic'
  );
});

test('bundle filename check is only a local UX filter', () => {
  assert.equal(model.isCandidateApplicationBundle({ name: 'nanokvm-go_0.0.25.tar.gz' }), true);
  assert.equal(model.isCandidateApplicationBundle({ name: 'firmware.tar.gz' }), false);
});

test('status response gate rejects delayed responses and superseded generations', () => {
  let generation = 7;
  const gate = model.createStatusResponseGate(() => generation);
  const older = gate.begin();
  const newer = gate.begin();

  assert.equal(gate.accept(newer), true);
  assert.equal(gate.accept(older), false);

  const superseded = gate.begin();
  generation += 1;
  assert.equal(gate.accept(superseded), false);
  assert.equal(gate.accept(gate.begin()), true);
});

test('status response gate rejects an older response as soon as a newer request begins', () => {
  const gate = model.createStatusResponseGate(() => 4);
  const first = gate.begin();
  const second = gate.begin();

  assert.equal(
    gate.accept(first),
    false,
    'the first response must not apply while the later-started request is unresolved'
  );
  assert.equal(gate.accept(second), true);
});

test('a failed WebSocket status request cannot strand the superseded active poll', async () => {
  const timers = fakeTimers();
  const gate = model.createStatusResponseGate(() => 9);
  const pollRequest = gate.begin();
  const refresh = model.createAuthoritativeStatusRefresh({
    debounceMs: 25,
    getGeneration: () => 9,
    request: async () => {
      gate.begin();
      throw new Error('server restart');
    },
    accept: () => assert.fail('a failed WebSocket request must not be accepted'),
    setTimer: timers.setTimer,
    clearTimer: timers.clearTimer
  });

  refresh.trigger();
  assert.equal(timers.runNext(), 25);
  await flushPromises();
  assert.equal(gate.accept(pollRequest), false, 'the older poll remains superseded');

  const recovered = model.recoverRejectedStatusPoll(true, () => {
    timers.setTimer(() => {}, 1000);
  });
  assert.equal(recovered, true);
  assert.equal(timers.size, 1, 'the active operation must retain an authoritative poll timer');
  assert.equal(timers.runNext(), 1000);

  assert.equal(
    model.recoverRejectedStatusPoll(false, () => {
      timers.setTimer(() => {}, 1000);
    }),
    false,
    'a terminal operation must not restart polling'
  );
  assert.equal(timers.size, 0);
  refresh.dispose();
});

test('operation status binding permits authoritative takeover only after the local operation ends', () => {
  const oldStatus = { operationId: 'old-operation' };
  const newStatus = { operationId: 'new-operation' };

  assert.equal(
    model.isStaleApplicationUpdateStatus(
      {
        updateActive: true,
        currentOperationId: 'old-operation',
        previousOperationId: ''
      },
      newStatus
    ),
    true,
    'a locally active operation remains bound to its operation ID'
  );
  assert.equal(
    model.isStaleApplicationUpdateStatus(
      {
        updateActive: false,
        currentOperationId: 'old-operation',
        previousOperationId: ''
      },
      newStatus
    ),
    false,
    'a terminal page must accept a newer operation started elsewhere'
  );
  assert.equal(
    model.isStaleApplicationUpdateStatus(
      {
        updateActive: true,
        currentOperationId: '',
        previousOperationId: 'old-operation'
      },
      oldStatus
    ),
    true,
    'a new local submission ignores the previous operation while awaiting its new ID'
  );
  assert.equal(
    model.isStaleApplicationUpdateStatus(
      {
        updateActive: true,
        currentOperationId: '',
        previousOperationId: 'old-operation'
      },
      { operationId: '' }
    ),
    true,
    'idle status cannot release a new local submission before admission returns its ID'
  );
  assert.equal(
    model.isStaleApplicationUpdateStatus(
      {
        updateActive: true,
        currentOperationId: '',
        previousOperationId: 'old-operation'
      },
      newStatus
    ),
    false
  );
});

test('WebSocket hint refresh coalesces bursts and queues one follow-up GET', async () => {
  const timers = fakeTimers();
  const first = deferred();
  const second = deferred();
  const requests = [first, second];
  const accepted = [];
  let requestCount = 0;
  let generation = 3;
  const refresh = model.createAuthoritativeStatusRefresh({
    debounceMs: 25,
    getGeneration: () => generation,
    request: () => requests[requestCount++].promise,
    accept: (value, requestGeneration) => accepted.push([value, requestGeneration]),
    setTimer: timers.setTimer,
    clearTimer: timers.clearTimer
  });

  refresh.trigger();
  refresh.trigger();
  refresh.trigger();
  assert.equal(timers.size, 1);
  assert.equal(timers.runNext(), 25);
  assert.equal(requestCount, 1);

  refresh.trigger();
  refresh.trigger();
  assert.equal(timers.size, 0, 'an in-flight request must not start another GET');

  first.resolve('first-authoritative-status');
  await flushPromises();
  assert.deepEqual(accepted, [['first-authoritative-status', 3]]);
  assert.equal(timers.size, 1, 'hints received in flight queue one follow-up refresh');

  timers.runNext();
  assert.equal(requestCount, 2);
  second.resolve('second-authoritative-status');
  await flushPromises();
  assert.deepEqual(accepted, [
    ['first-authoritative-status', 3],
    ['second-authoritative-status', 3]
  ]);
  assert.equal(timers.size, 0);
});

test('WebSocket hint refresh drops late generations and disposes timers', async () => {
  const timers = fakeTimers();
  const pending = deferred();
  const accepted = [];
  let generation = 1;
  const refresh = model.createAuthoritativeStatusRefresh({
    debounceMs: 10,
    getGeneration: () => generation,
    request: () => pending.promise,
    accept: (value) => accepted.push(value),
    setTimer: timers.setTimer,
    clearTimer: timers.clearTimer
  });

  refresh.trigger();
  timers.runNext();
  generation = 2;
  pending.resolve('stale');
  await flushPromises();
  assert.deepEqual(accepted, []);

  refresh.trigger();
  assert.equal(timers.size, 1);
  refresh.dispose();
  assert.equal(timers.size, 0);
});

test('settings lock is released on idle unmount but retained for a background update', () => {
  assert.equal(model.shouldReleaseSettingsLockOnUnmount(false), true);
  assert.equal(model.shouldReleaseSettingsLockOnUnmount(true), false);
});

test('WebSocket payload is ignored, each event triggers a refresh, and unsubscribe detaches it', () => {
  let subscribedType = '';
  let handler = null;
  let detached = false;
  let refreshCount = 0;
  const unsubscribe = model.subscribeAuthoritativeUpdateHints(
    {
      on(type, nextHandler) {
        subscribedType = type;
        handler = nextHandler;
        return () => {
          detached = true;
          handler = null;
        };
      }
    },
    { trigger: () => refreshCount++ }
  );

  assert.equal(subscribedType, 'update');
  handler({ data: '{"phase":"succeeded","untrusted":true}' });
  handler({ arbitrary: 'payload' });
  assert.equal(refreshCount, 2);

  unsubscribe();
  assert.equal(detached, true);
  assert.equal(handler, null);
});

test('WebSocket events fetch authoritative status and keyboard activation is safe', () => {
  assert.match(page, /subscribeAuthoritativeUpdateHints\(client, statusRefresh\)/);
  assert.match(page, /request:\s*async \(\) =>[\s\S]*?api\.getUpdateStatus\(\)/);
  assert.match(
    page,
    /const unsubscribe = subscribeAuthoritativeUpdateHints[\s\S]*?unsubscribe\(\)/
  );
  assert.match(page, /event\.preventDefault\(\);\s*browseFile\(\)/);
  assert.match(
    page,
    /if \(operationIdRef\.current\) \{\s*previousOperationIdRef\.current = operationIdRef\.current;/
  );
});
