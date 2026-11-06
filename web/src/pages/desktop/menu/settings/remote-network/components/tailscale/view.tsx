import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Divider } from 'antd';
import { LoaderCircleIcon } from 'lucide-react';

import * as api from '@/api/extensions/tailscale.ts';

import { useRemoteNetworkMessages } from '../../messages.ts';
import type { RemoteNetworkViewProps } from '../../types.ts';
import { ChooseMode } from './choose-mode.tsx';
import { Connection } from './connection.tsx';
import { Device } from './device.tsx';
import { Header } from './header.tsx';
import { Install } from './install.tsx';
import { Login } from './login.tsx';
import { getDefaultTailscaleModeId, getTailscaleModes, type TailscaleModeId } from './modes.ts';
import { SipeedAddDevice } from './sipeed-add-device.tsx';
import { SipeedOverview } from './sipeed-overview.tsx';
import type { Status } from './types.ts';

const STATUS_POLL_INTERVAL_MS = 1500;
const STARTUP_TIMEOUT_MS = 30_000;

function canChooseMode(state: Status['state'] | undefined): boolean {
  return state === 'notInstall' || state === 'notRunning' || state === 'notLogin';
}

function isManagedTransition(status: Status | undefined): boolean {
  return status?.managedState === 'enrolling' || status?.managedState === 'enabledConnecting';
}

function hasManagedRecord(status: Status | undefined): boolean {
  if (!status) return false;
  return (
    status.managedState !== 'disabled' ||
    status.managedError.trim() !== '' ||
    status.retryAfterSeconds > 0
  );
}

function shouldChooseModeAfterManagedFailure(status: Status | undefined): boolean {
  if (
    !status ||
    status.state === 'notRunning' ||
    !canChooseMode(status.state) ||
    status.activeMode === 'official'
  ) {
    return false;
  }
  return status.managedState === 'degraded' || status.managedState === 'invalidIdentity';
}

export const TailscaleView = ({ onBack, setLocked }: RemoteNetworkViewProps) => {
  const t = useRemoteNetworkMessages();
  const [isLoading, setIsLoading] = useState(false);
  const [isInstallRequestRunning, setIsInstallRequestRunning] = useState(false);
  const [isConnectionBusy, setIsConnectionBusy] = useState(false);
  const [status, setStatus] = useState<Status>();
  const statusRef = useRef<Status>();
  const [errMsg, setErrMsg] = useState('');
  const [selectedModeId, setSelectedModeId] = useState<TailscaleModeId>('official');
  const [isModeConfirmed, setIsModeConfirmed] = useState(false);
  const [isSipeedAddPageOpen, setIsSipeedAddPageOpen] = useState(false);
  const isMountedRef = useRef(true);
  const modesInitializedRef = useRef(false);
  const previousActiveModeRef = useRef<Status['activeMode']>(null);
  const statusRequestInFlightRef = useRef(false);
  const statusRefreshQueuedRef = useRef(false);

  const availableModes = useMemo(
    () => getTailscaleModes(status?.availableModes),
    [status?.availableModes]
  );
  const hasModeChoice = availableModes.length > 1;

  const applyStatus = useCallback((next: Status) => {
    if (!isMountedRef.current) return;
    statusRef.current = next;
    setErrMsg('');
    setStatus(next);
  }, []);

  const returnToModeChoice = useCallback(() => {
    setIsModeConfirmed(false);
  }, []);

  const getStatus = useCallback(
    async (showLoading = true) => {
      if (statusRequestInFlightRef.current) {
        statusRefreshQueuedRef.current = true;
        return;
      }
      statusRequestInFlightRef.current = true;
      if (showLoading && isMountedRef.current) setIsLoading(true);

      try {
        do {
          statusRefreshQueuedRef.current = false;
          try {
            const rsp = await api.getStatus();
            if (!isMountedRef.current) return;
            if (rsp.code !== 0) {
              setErrMsg(t('settings.tailscale.failed'));
            } else {
              applyStatus(rsp.data as Status);
            }
          } catch {
            if (isMountedRef.current) setErrMsg(t('settings.tailscale.failed'));
          }
        } while (isMountedRef.current && statusRefreshQueuedRef.current);
      } finally {
        statusRequestInFlightRef.current = false;
        if (showLoading && isMountedRef.current) setIsLoading(false);
      }
    },
    [applyStatus, t]
  );

  const refreshConnectionStatus = useCallback(
    async (action: 'start' | 'stop' | 'restart') => {
      await getStatus(false);
      if (action === 'stop' && statusRef.current?.activeMode === null) {
        modesInitializedRef.current = false;
        previousActiveModeRef.current = null;
        setIsModeConfirmed(false);
      }
      if (action !== 'start' && action !== 'restart') return;

      // systemd can finish starting the unit before tailscaled restores its
      // profile and exposes LocalAPI. Keep the action busy until it is ready.
      const deadline = Date.now() + STARTUP_TIMEOUT_MS;
      while (
        isMountedRef.current &&
        statusRef.current?.state === 'notRunning' &&
        Date.now() < deadline
      ) {
        await new Promise((resolve) => window.setTimeout(resolve, STATUS_POLL_INTERVAL_MS));
        if (!isMountedRef.current) return;
        await getStatus(false);
      }
      if (isMountedRef.current && statusRef.current?.state === 'notRunning') {
        setErrMsg(t('tailscale.error'));
      }
    },
    [getStatus, t]
  );

  const returnToModeChoiceAfterSipeedLogout = useCallback(async () => {
    modesInitializedRef.current = false;
    previousActiveModeRef.current = null;
    setIsModeConfirmed(false);
    await getStatus(false);
  }, [getStatus]);

  const handleSipeedIdentityExpired = useCallback(() => {
    modesInitializedRef.current = false;
    previousActiveModeRef.current = null;
    setIsModeConfirmed(false);
    void (async () => {
      await getStatus(false);
      if (isMountedRef.current) {
        setErrMsg(t('settings.tailscale.sipeed.identityExpired'));
      }
    })();
  }, [getStatus, t]);

  useEffect(() => {
    isMountedRef.current = true;
    void getStatus();
    return () => {
      isMountedRef.current = false;
    };
  }, [getStatus]);

  useEffect(() => {
    if (
      status?.state !== 'installing' &&
      !isManagedTransition(status) &&
      !((status?.retryAfterSeconds ?? 0) > 0)
    )
      return;

    let cancelled = false;
    let timer: number | undefined;
    const poll = async () => {
      await getStatus(false);
      if (!cancelled) timer = window.setTimeout(poll, STATUS_POLL_INTERVAL_MS);
    };
    timer = window.setTimeout(poll, STATUS_POLL_INTERVAL_MS);
    return () => {
      cancelled = true;
      if (timer !== undefined) window.clearTimeout(timer);
    };
  }, [getStatus, status, status?.managedState, status?.state]);

  useEffect(() => {
    if (!status || availableModes.length === 0) return;
    const activeMode = shouldChooseModeAfterManagedFailure(status) ? null : status.activeMode;
    const activeIsAvailable =
      activeMode !== null && availableModes.some((mode) => mode.id === activeMode);

    if (activeIsAvailable) {
      setSelectedModeId(activeMode);
      setIsModeConfirmed(true);
      modesInitializedRef.current = true;
    } else if (!modesInitializedRef.current) {
      setSelectedModeId(getDefaultTailscaleModeId(availableModes));
      setIsModeConfirmed(!hasModeChoice);
      modesInitializedRef.current = true;
    } else if (
      previousActiveModeRef.current !== null &&
      activeMode === null &&
      !(previousActiveModeRef.current === 'official' && isConnectionBusy)
    ) {
      // Starting or restarting can temporarily hide the official profile.
      // Explicit stops reset the selection in refreshConnectionStatus instead.
      setSelectedModeId(getDefaultTailscaleModeId(availableModes));
      setIsModeConfirmed(!hasModeChoice);
    }

    previousActiveModeRef.current = activeMode;
  }, [availableModes, hasModeChoice, isConnectionBusy, status]);

  const isInstalling = isInstallRequestRunning || status?.state === 'installing';
  const needsInstall = status?.state === 'notInstall' || status?.state === 'installing';
  const isDeviceReady = status?.state === 'stopped' || status?.state === 'running';
  const sipeedModeAvailable = availableModes.some((mode) => mode.id === 'sipeed');
  const managedRecord = hasManagedRecord(status);
  const managedFailureNeedsModeChoice = shouldChooseModeAfterManagedFailure(status);
  const activeMode = managedFailureNeedsModeChoice ? null : status?.activeMode;
  const isSipeedMode =
    sipeedModeAvailable &&
    (activeMode === 'sipeed' ||
      (activeMode === null && managedRecord && !managedFailureNeedsModeChoice) ||
      (isModeConfirmed && selectedModeId === 'sipeed'));
  const showModePicker =
    !isLoading &&
    status !== undefined &&
    hasModeChoice &&
    canChooseMode(status.state) &&
    activeMode === null &&
    (!managedRecord || managedFailureNeedsModeChoice) &&
    !isModeConfirmed;
  const showSipeedSetup =
    isSipeedMode &&
    status?.managedState !== 'enabledConnected' &&
    (!status?.sipeedDisclaimerAccepted || isManagedTransition(status));

  useEffect(() => {
    setLocked(isInstalling || isConnectionBusy);
  }, [isInstalling, isConnectionBusy, setLocked]);

  useEffect(() => {
    if (!isSipeedMode || status?.managedState !== 'enabledConnected') {
      setIsSipeedAddPageOpen(false);
    }
  }, [isSipeedMode, status?.managedState]);

  useEffect(() => () => setLocked(false), [setLocked]);

  return (
    <>
      <Header
        wikiMode={
          needsInstall
            ? 'official'
            : isSipeedMode
              ? 'sipeed'
              : showModePicker
                ? selectedModeId
                : 'official'
        }
        status={
          isSipeedAddPageOpen ||
          status?.state === 'installing' ||
          (isSipeedMode && status?.managedState !== 'enabledConnected')
            ? undefined
            : status
        }
        isBackDisabled={isInstalling || isConnectionBusy}
        onBack={isSipeedAddPageOpen ? () => setIsSipeedAddPageOpen(false) : onBack}
        onSuccess={getStatus}
        onSipeedLoggedOut={returnToModeChoiceAfterSipeedLogout}
        title={isSipeedAddPageOpen ? t('settings.tailscale.sipeed.addDevice') : undefined}
      />
      <Divider className="opacity-50" />
      <div className="h-10" />

      {isLoading ? (
        <div className="flex items-center justify-center space-x-2 pt-5 text-neutral-500">
          <LoaderCircleIcon className="animate-spin" size={18} />
          <span>{t('settings.tailscale.loading')}</span>
        </div>
      ) : needsInstall && status ? (
        <Install
          isInstalling={status.state === 'installing'}
          onBusyChange={setIsInstallRequestRunning}
          onSuccess={getStatus}
        />
      ) : showModePicker ? (
        <ChooseMode
          modes={availableModes}
          onChange={setSelectedModeId}
          onConfirm={() => setIsModeConfirmed(true)}
          value={selectedModeId}
        />
      ) : status ? (
        <div className="space-y-4">
          {!isSipeedAddPageOpen && !showSipeedSetup && (
            <Connection
              mode={isSipeedMode ? 'sipeed' : 'official'}
              status={status}
              onBusyChange={setIsConnectionBusy}
              onStatus={applyStatus}
              onSuccess={refreshConnectionStatus}
            />
          )}
          {(status.state !== 'notRunning' || showSipeedSetup) &&
            (isSipeedMode ? (
              status.managedState === 'enabledConnected' ? (
                <SipeedOverview
                  isAddPageOpen={isSipeedAddPageOpen}
                  status={status}
                  onAddPageOpenChange={setIsSipeedAddPageOpen}
                  onIdentityExpired={handleSipeedIdentityExpired}
                />
              ) : (
                <SipeedAddDevice
                  status={status}
                  onFailure={
                    hasModeChoice && canChooseMode(status.state) ? returnToModeChoice : undefined
                  }
                  onStatus={applyStatus}
                />
              )
            ) : (
              <>
                {status?.state === 'notLogin' && (
                  <Login onSuccess={() => refreshConnectionStatus('start')} />
                )}
                {isDeviceReady && status && (
                  <Device status={status} onStatusRefresh={() => getStatus(false)} />
                )}
              </>
            ))}
        </div>
      ) : null}
      {errMsg && (
        <div className="pt-5 text-red-500" role="alert">
          {errMsg}
        </div>
      )}
    </>
  );
};
