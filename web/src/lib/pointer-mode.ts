import * as api from '@/api/hid.ts';
import type { MouseMode, PointerMode, PointerModeDeviceStatus } from '@/jotai/mouse.ts';

export const POINTER_MODES = api.POINTER_MODES;
export const POINTER_MODE_SYNC_EVENT = 'nanokvm:pointer-mode-sync';

export const POINTER_MODE_POLL_INTERVAL_MS = 500;
// The backend transaction has a bounded 30-second lifetime; allow the browser
// to observe a slow re-enumeration for the same window before reporting an
// unconfirmed result.
export const POINTER_MODE_POLL_ATTEMPTS = 60;

export function isPointerMode(value: unknown): value is PointerMode {
  return typeof value === 'string' && (POINTER_MODES as readonly string[]).includes(value);
}

export function requestPointerModeSync() {
  if (typeof window === 'undefined') return;
  window.dispatchEvent(new Event(POINTER_MODE_SYNC_EVENT));
}

export function normalizePointerModeStatus(
  response: api.PointerModeStatusResponse | null | undefined
): PointerModeDeviceStatus {
  const mode = isPointerMode(response?.mode) ? response.mode : 'unknown';
  const switching = Boolean(response?.switching);
  const degraded = Boolean(response?.degraded) || mode === 'unknown';
  const error = typeof response?.error === 'string' ? response.error : '';
  return {
    mode,
    switching,
    degraded,
    ready: isPointerMode(mode) && !switching && !degraded,
    error
  };
}

export function pointerModeStatusFromError(error: unknown): PointerModeDeviceStatus {
  const messageText = error instanceof Error ? error.message : String(error || '');
  return {
    mode: null,
    switching: false,
    degraded: false,
    ready: false,
    error: messageText
  };
}

export function resolveAuthoritativeMouseMode(
  browserMode: MouseMode,
  deviceStatus: PointerModeDeviceStatus
): MouseMode {
  if (browserMode === 'relative') return browserMode;
  return deviceStatus.ready && isPointerMode(deviceStatus.mode) ? deviceStatus.mode : browserMode;
}

export function canRepairIphoneDrag(mode: MouseMode, inputEnabled: boolean): boolean {
  return mode === 'absolute' && inputEnabled;
}

export async function fetchPointerModeStatus(): Promise<PointerModeDeviceStatus> {
  const response = await api.getPointerMode();
  if (response.code !== 0) {
    throw new Error(response.msg || 'pointer mode status request failed');
  }
  return normalizePointerModeStatus(response.data);
}

export async function pollPointerModeStatus(
  publish: (status: PointerModeDeviceStatus) => void,
  options: {
    attempts?: number;
    intervalMs?: number;
    getStatus?: () => Promise<PointerModeDeviceStatus>;
    sleep?: (milliseconds: number) => Promise<void>;
  } = {}
): Promise<PointerModeDeviceStatus | null> {
  const attempts = options.attempts ?? POINTER_MODE_POLL_ATTEMPTS;
  const intervalMs = options.intervalMs ?? POINTER_MODE_POLL_INTERVAL_MS;
  const getStatus = options.getStatus ?? fetchPointerModeStatus;
  const sleep = options.sleep ?? wait;

  for (let attempt = 0; attempt < attempts; attempt += 1) {
    try {
      const status = await getStatus();
      publish({ ...status, ready: false });
      if (!status.switching) return status;
    } catch {
      // USB re-enumeration can make the HTTP endpoint unavailable temporarily.
    }
    if (attempt + 1 < attempts) await sleep(intervalMs);
  }
  return null;
}

function wait(milliseconds: number) {
  return new Promise<void>((resolve) => setTimeout(resolve, milliseconds));
}

export type PointerModeSelectionActions = {
  applyMouseMode: (mode: MouseMode) => void;
  publishStatus: (status: PointerModeDeviceStatus) => void;
  beginOperation: () => void;
  endOperation: () => void;
  setInputEnabled: (enabled: boolean) => void;
  ensureConnected: () => void;
  reconnectRelative: () => void;
  postPointerMode: (mode: PointerMode) => Promise<{ code: number; msg: string }>;
  pollStatus: (
    publish: (status: PointerModeDeviceStatus) => void
  ) => Promise<PointerModeDeviceStatus | null>;
  requestSync: () => void;
  reportError: (message: string) => void;
  fallbackError: string;
};

export async function performPointerModeSelection(
  currentMode: MouseMode,
  currentStatus: PointerModeDeviceStatus,
  targetMode: MouseMode,
  actions: PointerModeSelectionActions
): Promise<boolean> {
  if (currentStatus.switching) return false;

  if (targetMode === 'relative') {
    actions.applyMouseMode('relative');
    actions.reconnectRelative();
    return true;
  }

  if (!currentStatus.ready && !currentStatus.degraded) {
    actions.reportError(currentStatus.error || actions.fallbackError);
    actions.requestSync();
    return false;
  }

  if (currentStatus.mode === targetMode && !currentStatus.degraded) {
    actions.applyMouseMode(targetMode);
    return true;
  }

  actions.beginOperation();
  actions.setInputEnabled(false);
  actions.publishStatus({
    ...currentStatus,
    switching: true,
    ready: false,
    error: ''
  });

  try {
    let postError = '';
    try {
      const response = await actions.postPointerMode(targetMode);
      if (response.code !== 0) postError = response.msg;
    } catch (error) {
      // The response may be lost with USB network re-enumeration.
      postError = error instanceof Error ? error.message : '';
    }

    const finalStatus = await actions.pollStatus((status) => {
      actions.publishStatus({ ...status, ready: false });
    });

    if (finalStatus && finalStatus.mode === targetMode && finalStatus.ready) {
      actions.publishStatus(finalStatus);
      actions.applyMouseMode(targetMode);
      return true;
    }

    if (finalStatus && isPointerMode(finalStatus.mode) && !finalStatus.degraded) {
      actions.publishStatus(finalStatus);
      actions.applyMouseMode(currentMode === 'relative' ? 'relative' : finalStatus.mode);
      actions.reportError(finalStatus.error || postError || actions.fallbackError);
      return false;
    }

    const degraded = Boolean(finalStatus?.degraded || finalStatus?.mode === 'unknown');
    const error = degraded
      ? finalStatus?.error || postError || actions.fallbackError
      : postError || actions.fallbackError;
    actions.publishStatus({
      mode: degraded ? 'unknown' : currentStatus.mode,
      switching: false,
      degraded,
      ready: false,
      error
    });
    actions.reportError(error);
    return false;
  } finally {
    try {
      actions.ensureConnected();
    } finally {
      actions.endOperation();
    }
  }
}
