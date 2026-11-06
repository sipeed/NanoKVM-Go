import type { ApplicationUpdatePhase, ApplicationUpdateStatus } from '@/api/application.ts';

const busyPhases = new Set<ApplicationUpdatePhase>([
  'downloading',
  'uploading',
  'verifying',
  'staging',
  'installing',
  'restarting'
]);

const knownErrorCodes = new Set([
  'busy',
  'upload_interrupted',
  'no_space',
  'invalid_archive',
  'signature_invalid',
  'unsupported_package',
  'same_version',
  'install_start_failed',
  'install_failed',
  'interrupted'
]);

export function isUpdateBusyPhase(phase?: ApplicationUpdatePhase): boolean {
  return phase !== undefined && busyPhases.has(phase);
}

export function isApplicationUpdateStatus(value: unknown): value is ApplicationUpdateStatus {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return false;
  const status = value as Record<string, unknown>;
  return (
    typeof status.operationId === 'string' &&
    (status.source === '' || status.source === 'online' || status.source === 'offline') &&
    typeof status.version === 'string' &&
    typeof status.phase === 'string' &&
    [
      'idle',
      'downloading',
      'uploading',
      'verifying',
      'staging',
      'installing',
      'restarting',
      'succeeded',
      'failed'
    ].includes(status.phase) &&
    typeof status.progress === 'number' &&
    Number.isFinite(status.progress) &&
    status.progress >= 0 &&
    status.progress <= 100 &&
    typeof status.errorCode === 'string' &&
    typeof status.startedAt === 'string' &&
    typeof status.updatedAt === 'string'
  );
}

export function updateErrorTranslationKey(errorCode?: string): string {
  return knownErrorCodes.has(errorCode ?? '')
    ? `settings.update.errors.${errorCode}`
    : 'settings.update.errors.generic';
}

export function isCandidateApplicationBundle(file: File): boolean {
  return /^nanokvm-go_[0-9A-Za-z][0-9A-Za-z.+~-]*\.tar\.gz$/i.test(file.name);
}

export function formatUpdateFileSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return '';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KiB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MiB`;
}

export function shouldReleaseSettingsLockOnUnmount(updateActive: boolean): boolean {
  return !updateActive;
}

export type SettingsBusyLock = {
  setApplicationBusy: (busy: boolean) => void;
  setPreviewBusy: (busy: boolean) => void;
  setUpdateServerBusy: (busy: boolean) => void;
  setSignatureVerificationBusy: (busy: boolean) => void;
  setRegionBusy: (busy: boolean) => void;
  releaseIfApplicationIdle: () => void;
  isLocked: () => boolean;
};

// Settings has one outer navigation lock but independent update, Preview,
// server-setting, signature-setting, and region-switch owners. Keeping the OR in one
// coordinator prevents one completion callback from unlocking the modal while
// another operation is still active.
export function createSettingsBusyLock(onChange: (locked: boolean) => void): SettingsBusyLock {
  let applicationBusy = false;
  let previewBusy = false;
  let updateServerBusy = false;
  let signatureVerificationBusy = false;
  let regionBusy = false;
  let locked = false;

  const sync = () => {
    const next =
      applicationBusy || previewBusy || updateServerBusy || signatureVerificationBusy || regionBusy;
    if (next === locked) return;
    locked = next;
    onChange(locked);
  };

  return {
    setApplicationBusy(busy) {
      applicationBusy = busy;
      sync();
    },
    setPreviewBusy(busy) {
      previewBusy = busy;
      sync();
    },
    setUpdateServerBusy(busy) {
      updateServerBusy = busy;
      sync();
    },
    setSignatureVerificationBusy(busy) {
      signatureVerificationBusy = busy;
      sync();
    },
    setRegionBusy(busy) {
      regionBusy = busy;
      sync();
    },
    releaseIfApplicationIdle() {
      if (applicationBusy) return;
      previewBusy = false;
      updateServerBusy = false;
      signatureVerificationBusy = false;
      sync();
    },
    isLocked() {
      return locked;
    }
  };
}

export function recoverRejectedStatusPoll(
  updateActive: boolean,
  scheduleAuthoritativePoll: () => void
): boolean {
  if (!updateActive) return false;
  scheduleAuthoritativePoll();
  return true;
}

type OperationStatusGateState = {
  updateActive: boolean;
  currentOperationId: string;
  previousOperationId: string;
};

// Bind status to one operation only while this page owns an active submission
// or has recovered an active operation. Once that operation reaches a terminal
// state, a status GET may legitimately reveal a newer operation started by a
// different browser, and that authoritative operation must be allowed to take
// over the UI.
export function isStaleApplicationUpdateStatus(
  state: OperationStatusGateState,
  next: Pick<ApplicationUpdateStatus, 'operationId'>
): boolean {
  if (!state.updateActive) return false;
  if (state.currentOperationId) {
    return !next.operationId || next.operationId !== state.currentOperationId;
  }
  return (
    !next.operationId ||
    (!!state.previousOperationId && next.operationId === state.previousOperationId)
  );
}

type RefreshTimer = ReturnType<typeof setTimeout>;

type AuthoritativeStatusRefreshOptions<T> = {
  debounceMs: number;
  getGeneration: () => number;
  request: () => Promise<T>;
  accept: (response: T, generation: number) => void;
  setTimer?: (callback: () => void, delayMs: number) => RefreshTimer;
  clearTimer?: (timer: RefreshTimer) => void;
};

export type AuthoritativeStatusRefresh = {
  trigger: () => void;
  dispose: () => void;
};

type UpdateHintSource = {
  on: (type: string, handler: (payload: unknown) => void) => () => void;
};

export function subscribeAuthoritativeUpdateHints(
  source: UpdateHintSource,
  refresh: Pick<AuthoritativeStatusRefresh, 'trigger'>
): () => void {
  return source.on('update', () => refresh.trigger());
}

export type StatusResponseToken = {
  generation: number;
  sequence: number;
};

export type StatusResponseGate = {
  begin: (generation?: number) => StatusResponseToken;
  accept: (token: StatusResponseToken) => boolean;
};

// Polling and WebSocket-triggered GETs can overlap. A later-started accepted
// response supersedes an older request so delayed network responses cannot
// roll the UI back within the same operation generation.
export function createStatusResponseGate(getGeneration: () => number): StatusResponseGate {
  let nextSequence = 0;
  let acceptedSequence = 0;

  return {
    begin(generation = getGeneration()) {
      nextSequence += 1;
      return { generation, sequence: nextSequence };
    },
    accept(token) {
      if (
        token.generation !== getGeneration() ||
        token.sequence !== nextSequence ||
        token.sequence <= acceptedSequence
      ) {
        return false;
      }
      acceptedSequence = token.sequence;
      return true;
    }
  };
}

// WebSocket messages are only hints that authoritative state may have changed.
// This coordinator coalesces bursts, keeps at most one status request in flight,
// and drops a response when another update generation superseded its request.
export function createAuthoritativeStatusRefresh<T>({
  debounceMs,
  getGeneration,
  request,
  accept,
  setTimer = setTimeout,
  clearTimer = clearTimeout
}: AuthoritativeStatusRefreshOptions<T>): AuthoritativeStatusRefresh {
  let disposed = false;
  let queued = false;
  let inFlight = false;
  let timer: RefreshTimer | null = null;

  const schedule = () => {
    if (disposed || inFlight || timer !== null) return;
    timer = setTimer(() => {
      timer = null;
      void refresh();
    }, debounceMs);
  };

  const refresh = async () => {
    if (disposed || inFlight || !queued) return;

    queued = false;
    inFlight = true;
    const requestGeneration = getGeneration();

    try {
      const response = await request();
      if (!disposed && requestGeneration === getGeneration()) {
        accept(response, requestGeneration);
      }
    } catch {
      // Polling remains the recovery path. A WebSocket hint must never turn a
      // transient status request failure into a terminal UI state.
    } finally {
      inFlight = false;
      if (!disposed && queued) schedule();
    }
  };

  return {
    trigger() {
      if (disposed) return;
      queued = true;
      schedule();
    },
    dispose() {
      disposed = true;
      queued = false;
      if (timer !== null) {
        clearTimer(timer);
        timer = null;
      }
    }
  };
}
