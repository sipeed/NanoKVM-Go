import { useCallback, useEffect, useLayoutEffect, useRef } from 'react';
import { useAtomValue, useSetAtom } from 'jotai';

import { setBaseInputEnabled } from '@/lib/input-suspension.ts';
import * as ls from '@/lib/localstorage.ts';
import {
  fetchPointerModeStatus,
  POINTER_MODE_SYNC_EVENT,
  pointerModeStatusFromError,
  resolveAuthoritativeMouseMode
} from '@/lib/pointer-mode.ts';
import { client } from '@/lib/websocket.ts';
import {
  mouseModeAtom,
  normalizeMouseMode,
  pointerModeDeviceStatusAtom,
  pointerModeOperationAtom,
  type MouseMode,
  type PointerModeDeviceStatus
} from '@/jotai/mouse.ts';

const POINTER_MODE_SYNC_INTERVAL_MS = 15_000;

export const PointerModeController = () => {
  const mouseMode = useAtomValue(mouseModeAtom);
  const deviceStatus = useAtomValue(pointerModeDeviceStatusAtom);
  const localOperation = useAtomValue(pointerModeOperationAtom);
  const setMouseMode = useSetAtom(mouseModeAtom);
  const setDeviceStatus = useSetAtom(pointerModeDeviceStatusAtom);
  const mouseModeRef = useRef<MouseMode>(mouseMode);
  const deviceStatusRef = useRef(deviceStatus);
  const localOperationRef = useRef(localOperation);
  const syncingRef = useRef<Promise<PointerModeDeviceStatus | null> | null>(null);
  const disposedRef = useRef(false);

  mouseModeRef.current = mouseMode;
  deviceStatusRef.current = deviceStatus;
  localOperationRef.current = localOperation;

  const applyDeviceStatus = useCallback(
    (status: PointerModeDeviceStatus) => {
      if (disposedRef.current || localOperationRef.current) return;

      const activeLocalMode = mouseModeRef.current;
      const nextLocalMode = resolveAuthoritativeMouseMode(activeLocalMode, status);
      const profileChanged = activeLocalMode !== nextLocalMode;
      if (profileChanged) setBaseInputEnabled(false);

      deviceStatusRef.current = status;
      setDeviceStatus(status);

      // With a single endpoint, every viewer must follow the device profile.
      if (profileChanged) {
        mouseModeRef.current = nextLocalMode;
        setMouseMode(nextLocalMode);
        ls.setMouseMode(nextLocalMode);
      }
    },
    [setDeviceStatus, setMouseMode]
  );

  const syncFromDevice = useCallback(async () => {
    if (disposedRef.current || localOperation) return null;
    if (syncingRef.current) return syncingRef.current;

    const promise = fetchPointerModeStatus()
      .then((status) => {
        applyDeviceStatus(status);
        return status;
      })
      .catch((error: unknown) => {
        applyDeviceStatus(pointerModeStatusFromError(error));
        return null;
      })
      .finally(() => {
        syncingRef.current = null;
      });
    syncingRef.current = promise;
    return promise;
  }, [applyDeviceStatus, localOperation]);

  useLayoutEffect(() => {
    // No input is sent using a browser-local default before the initial GET.
    setBaseInputEnabled(false);
    let storedValue: string | null = null;
    try {
      storedValue = ls.getMouseMode();
    } catch (error) {
      console.warn('[PointerMode] Unable to read local mouse preference', error);
    }
    const storedMode = normalizeMouseMode(storedValue);
    mouseModeRef.current = storedMode;
    setMouseMode(storedMode);
    if (storedValue === 'absolute-android') ls.setMouseMode(storedMode);

    return () => setBaseInputEnabled(true);
  }, [setMouseMode]);

  useEffect(() => {
    // A remote layout change can replace the local pointer while the device
    // stays ready and keeps reporting the same HID2 mode.
    setBaseInputEnabled(
      deviceStatus.ready && !deviceStatus.switching && !deviceStatus.degraded && !localOperation &&
      mouseMode === resolveAuthoritativeMouseMode(mouseMode, deviceStatus)
    );
  }, [deviceStatus, localOperation, mouseMode]);

  useEffect(() => {
    disposedRef.current = false;
    void syncFromDevice();

    const handleSyncRequest = () => void syncFromDevice();
    const handleFocus = () => void syncFromDevice();
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') void syncFromDevice();
    };
    const handleStorage = (event: StorageEvent) => {
      if (event.key !== ls.MOUSE_MODE_KEY) return;
      const localMode = normalizeMouseMode(event.newValue);
      if (localMode === 'relative' && !deviceStatusRef.current.singleEndpoint) {
        mouseModeRef.current = localMode;
        setMouseMode(localMode);
        return;
      }
      void syncFromDevice();
    };
    const unsubscribeConnection = client.onConnectionState((state) => {
      if (state === 'open') void syncFromDevice();
    });
    const interval = window.setInterval(() => void syncFromDevice(), POINTER_MODE_SYNC_INTERVAL_MS);

    window.addEventListener(POINTER_MODE_SYNC_EVENT, handleSyncRequest);
    window.addEventListener('focus', handleFocus);
    document.addEventListener('visibilitychange', handleVisibilityChange);
    window.addEventListener('storage', handleStorage);
    return () => {
      disposedRef.current = true;
      window.clearInterval(interval);
      unsubscribeConnection();
      window.removeEventListener(POINTER_MODE_SYNC_EVENT, handleSyncRequest);
      window.removeEventListener('focus', handleFocus);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      window.removeEventListener('storage', handleStorage);
    };
  }, [setMouseMode, syncFromDevice]);

  return null;
};
