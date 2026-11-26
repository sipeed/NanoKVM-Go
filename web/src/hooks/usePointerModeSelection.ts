import { useCallback, useEffect, useRef } from 'react';
import { useAtomValue, useSetAtom } from 'jotai';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/hid.ts';
import { feedback } from '@/lib/feedback.tsx';
import { setBaseInputEnabled } from '@/lib/input-suspension.ts';
import * as ls from '@/lib/localstorage.ts';
import {
  performPointerModeSelection,
  pollPointerModeStatus,
  requestPointerModeSync
} from '@/lib/pointer-mode.ts';
import { client } from '@/lib/websocket.ts';
import {
  mouseModeAtom,
  pointerModeDeviceStatusAtom,
  pointerModeOperationAtom,
  type MouseMode
} from '@/jotai/mouse.ts';

export function usePointerModeSelection() {
  const { t } = useTranslation();
  const mouseMode = useAtomValue(mouseModeAtom);
  const deviceStatus = useAtomValue(pointerModeDeviceStatusAtom);
  const setMouseMode = useSetAtom(mouseModeAtom);
  const setDeviceStatus = useSetAtom(pointerModeDeviceStatusAtom);
  const setOperation = useSetAtom(pointerModeOperationAtom);
  const operationRef = useRef(false);
  const relativeReconnectTimerRef = useRef<number | null>(null);
  const mouseModeRef = useRef(mouseMode);
  const deviceStatusRef = useRef(deviceStatus);
  mouseModeRef.current = mouseMode;
  deviceStatusRef.current = deviceStatus;

  useEffect(
    () => () => {
      if (relativeReconnectTimerRef.current !== null) {
        window.clearTimeout(relativeReconnectTimerRef.current);
      }
    },
    []
  );

  const selectMode = useCallback(
    async (mode: MouseMode, singleEndpoint = !!deviceStatusRef.current.singleEndpoint): Promise<boolean> => {
      if (operationRef.current || deviceStatusRef.current.switching) return false;

      return performPointerModeSelection(mouseModeRef.current, deviceStatusRef.current, mode, {
        applyMouseMode(nextMode) {
          mouseModeRef.current = nextMode;
          setMouseMode(nextMode);
          ls.setMouseMode(nextMode);
        },
        publishStatus(status) {
          deviceStatusRef.current = status;
          setDeviceStatus(status);
        },
        beginOperation() {
          operationRef.current = true;
          if (relativeReconnectTimerRef.current !== null) {
            window.clearTimeout(relativeReconnectTimerRef.current);
            relativeReconnectTimerRef.current = null;
          }
          setOperation(true);
        },
        endOperation() {
          operationRef.current = false;
          setOperation(false);
        },
        setInputEnabled: setBaseInputEnabled,
        ensureConnected() {
          if (!client.isConnected) client.connect();
        },
        reconnectRelative() {
          if (relativeReconnectTimerRef.current !== null) {
            window.clearTimeout(relativeReconnectTimerRef.current);
          }
          client.close();
          relativeReconnectTimerRef.current = window.setTimeout(() => {
            relativeReconnectTimerRef.current = null;
            client.connect();
          }, 500);
        },
        postPointerMode: api.setPointerMode,
        pollStatus: (publish) => pollPointerModeStatus(publish),
        requestSync: requestPointerModeSync,
        reportError: (error) => feedback.error(error || t('error.title')),
        fallbackError: t('error.title')
      }, singleEndpoint);
    },
    [setDeviceStatus, setMouseMode, setOperation, t]
  );

  return { selectMode, deviceStatus, switching: deviceStatus.switching };
}
