import { useCallback, useEffect, useRef, useState, type ChangeEvent, type DragEvent } from 'react';
import {
  CheckCircleOutlined,
  LoadingOutlined,
  RocketOutlined,
  SmileOutlined
} from '@ant-design/icons';
import { Button, Divider, Modal, Result, Spin } from 'antd';
import clsx from 'clsx';
import { ChevronRightIcon, FileArchiveIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/application.ts';
import type { ApplicationUpdateStatus } from '@/api/application.ts';
import { projectUrl, releaseName } from '@/lib/branding.ts';
import { isNewerVersion, normalizeVersion } from '@/lib/version.ts';
import { client } from '@/lib/websocket.ts';
import { ButtonBase } from '@/components/ui/button-base.tsx';

import {
  createAuthoritativeStatusRefresh,
  createSettingsBusyLock,
  createStatusResponseGate,
  formatUpdateFileSize,
  isApplicationUpdateStatus,
  isCandidateApplicationBundle,
  isStaleApplicationUpdateStatus,
  isUpdateBusyPhase,
  recoverRejectedStatusPoll,
  shouldReleaseSettingsLockOnUnmount,
  subscribeAuthoritativeUpdateHints,
  updateErrorTranslationKey
} from './model.ts';
import { Preview } from './preview.tsx';
import { UpdateServer } from './server.tsx';
import { SignatureVerification } from './signature-verification.tsx';
import { Updating } from './updating.tsx';

type UpdateProps = {
  setIsLocked: (isLocked: boolean) => void;
};

type PageStatus =
  | ''
  | 'loading'
  | 'updating'
  | 'outdated'
  | 'latest'
  | 'query-failed'
  | 'failed'
  | 'succeeded';

const POLL_INTERVAL_MS = 1000;
const WS_STATUS_REFRESH_DEBOUNCE_MS = 100;
const STATUS_RETRY_DELAYS_MS = [1000, 2000, 3000, 5000, 8000, 10000, 15000];
const ACKNOWLEDGE_RETRY_DELAYS_MS = [250, 500, 1000, 2000];
const initialUpdateStatus: ApplicationUpdateStatus = {
  operationId: '',
  source: '',
  version: '',
  phase: 'idle',
  progress: 0,
  errorCode: '',
  startedAt: '',
  updatedAt: ''
};

export const Update = ({ setIsLocked }: UpdateProps) => {
  const { t } = useTranslation();
  const [modal, contextHolder] = Modal.useModal();

  const [pageStatus, setPageStatus] = useState<PageStatus>('');
  const [updateStatus, setUpdateStatus] = useState<ApplicationUpdateStatus>(initialUpdateStatus);
  const [currentVersion, setCurrentVersion] = useState('');
  const [latestVersion, setLatestVersion] = useState('');
  const [errMsg, setErrMsg] = useState('');
  const [tipMsg, setTipMsg] = useState('');
  const [offlineFileError, setOfflineFileError] = useState('');
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [uploadProgress, setUploadProgress] = useState(0);
  const [isDragging, setIsDragging] = useState(false);
  const [isOfflineModalOpen, setIsOfflineModalOpen] = useState(false);
  const [isPreviewBusy, setIsPreviewBusy] = useState(false);
  const [isUpdateServerBusy, setIsUpdateServerBusy] = useState(false);
  const [isUpdateServerPending, setIsUpdateServerPending] = useState(false);
  const [isSignatureVerificationBusy, setIsSignatureVerificationBusy] = useState(false);
  const [isAcknowledgingTerminal, setIsAcknowledgingTerminal] = useState(false);
  const [terminalActionError, setTerminalActionError] = useState('');

  const mountedRef = useRef(false);
  const generationRef = useRef(0);
  const setIsLockedRef = useRef(setIsLocked);
  setIsLockedRef.current = setIsLocked;
  const settingsBusyLockRef = useRef<ReturnType<typeof createSettingsBusyLock> | null>(null);
  if (settingsBusyLockRef.current === null) {
    settingsBusyLockRef.current = createSettingsBusyLock((locked) =>
      setIsLockedRef.current(locked)
    );
  }
  const settingsBusyLock = settingsBusyLockRef.current;
  const statusResponseGateRef = useRef<ReturnType<typeof createStatusResponseGate> | null>(null);
  if (statusResponseGateRef.current === null) {
    statusResponseGateRef.current = createStatusResponseGate(() => generationRef.current);
  }
  const statusResponseGate = statusResponseGateRef.current;
  const operationIdRef = useRef('');
  const previousOperationIdRef = useRef('');
  const busyRef = useRef(false);
  const pollTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pollFailuresRef = useRef(0);
  const reconnectObservedRef = useRef(false);
  const versionCheckingRef = useRef(false);
  const versionRequestRef = useRef(0);
  const previewBusyRef = useRef(false);
  const acknowledgingTerminalRef = useRef(false);
  const autoRefreshOperationRef = useRef('');
  const abortUploadRef = useRef<AbortController | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const isUpdating = pageStatus === 'updating';
  const isBusy =
    isUpdating ||
    isUpdateServerBusy ||
    isPreviewBusy ||
    isSignatureVerificationBusy ||
    isAcknowledgingTerminal ||
    pageStatus === 'loading';

  const beginGeneration = useCallback(() => {
    generationRef.current += 1;
    versionRequestRef.current += 1;
    versionCheckingRef.current = false;
    return generationRef.current;
  }, []);

  const clearPollTimer = useCallback(() => {
    if (!pollTimerRef.current) return;
    clearTimeout(pollTimerRef.current);
    pollTimerRef.current = null;
  }, []);

  const handlePreviewBusyChange = useCallback(
    (busy: boolean) => {
      previewBusyRef.current = busy;
      setIsPreviewBusy(busy);
      settingsBusyLock.setPreviewBusy(busy);
    },
    [settingsBusyLock]
  );

  const handleSignatureVerificationBusyChange = useCallback(
    (busy: boolean) => {
      setIsSignatureVerificationBusy(busy);
      settingsBusyLock.setSignatureVerificationBusy(busy);
    },
    [settingsBusyLock]
  );

  const handleUpdateServerBusyChange = useCallback(
    (busy: boolean) => {
      setIsUpdateServerBusy(busy);
      settingsBusyLock.setUpdateServerBusy(busy);
    },
    [settingsBusyLock]
  );

  const isStaleOperationStatus = useCallback((next: ApplicationUpdateStatus) => {
    return isStaleApplicationUpdateStatus(
      {
        updateActive: busyRef.current,
        currentOperationId: operationIdRef.current,
        previousOperationId: previousOperationIdRef.current
      },
      next
    );
  }, []);

  const checkForUpdates = useCallback(
    async (requestGeneration = generationRef.current) => {
      if (
        requestGeneration !== generationRef.current ||
        busyRef.current ||
        operationIdRef.current ||
        versionCheckingRef.current
      ) {
        return;
      }
      versionCheckingRef.current = true;
      const versionRequest = ++versionRequestRef.current;
      setPageStatus('loading');

      try {
        const rsp = await api.getVersion();
        if (
          !mountedRef.current ||
          requestGeneration !== generationRef.current ||
          versionRequest !== versionRequestRef.current ||
          operationIdRef.current
        ) {
          return;
        }
        if (rsp.code !== 0 || !rsp.data) throw new Error('invalid version response');

        const current = normalizeVersion(rsp.data.current);
        const latest = normalizeVersion(rsp.data.latest);
        setCurrentVersion(current);
        setLatestVersion(latest);

        const hasUpdate = isNewerVersion(latest, current);
        setTipMsg(hasUpdate ? t('settings.update.available') : '');
        setPageStatus(hasUpdate ? 'outdated' : 'latest');
      } catch {
        if (
          !mountedRef.current ||
          requestGeneration !== generationRef.current ||
          versionRequest !== versionRequestRef.current ||
          operationIdRef.current
        ) {
          return;
        }
        setPageStatus('query-failed');
        setErrMsg(t('settings.update.queryFailed'));
      } finally {
        if (
          requestGeneration === generationRef.current &&
          versionRequest === versionRequestRef.current
        ) {
          versionCheckingRef.current = false;
        }
      }
    },
    [t]
  );

  const applyStatus = useCallback(
    (next: ApplicationUpdateStatus, requestGeneration: number) => {
      if (!mountedRef.current || requestGeneration !== generationRef.current) return false;

      if (isStaleOperationStatus(next)) return busyRef.current;
      if (next.phase === 'idle') {
        clearPollTimer();
        const idleGeneration = beginGeneration();
        operationIdRef.current = '';
        previousOperationIdRef.current = '';
        busyRef.current = false;
        setSelectedFile(null);
        setOfflineFileError('');
        setUpdateStatus(initialUpdateStatus);
        setErrMsg('');
        setTerminalActionError('');
        setPageStatus('loading');
        settingsBusyLock.setApplicationBusy(false);
        void checkForUpdates(idleGeneration);
        return false;
      }

      if (next.operationId) {
        if (next.operationId !== operationIdRef.current) setTerminalActionError('');
        operationIdRef.current = next.operationId;
        previousOperationIdRef.current = '';
      }

      setUpdateStatus(next);
      if (isUpdateBusyPhase(next.phase)) {
        setIsOfflineModalOpen(false);
        setIsDragging(false);
        busyRef.current = true;
        setPageStatus('updating');
        settingsBusyLock.setApplicationBusy(true);
        return true;
      }

      if (next.phase === 'failed') {
        busyRef.current = false;
        setPageStatus('failed');
        setErrMsg(t(updateErrorTranslationKey(next.errorCode)));
        settingsBusyLock.setApplicationBusy(false);
        return false;
      }

      if (next.phase === 'succeeded') {
        busyRef.current = false;
        setPageStatus('succeeded');
        settingsBusyLock.setApplicationBusy(false);
        return false;
      }

      busyRef.current = false;
      settingsBusyLock.setApplicationBusy(false);
      return false;
    },
    [beginGeneration, checkForUpdates, clearPollTimer, isStaleOperationStatus, settingsBusyLock, t]
  );

  const pollStatus = useCallback(
    async (
      requestGeneration: number,
      scheduleRetry = true
    ): Promise<ApplicationUpdateStatus | null> => {
      clearPollTimer();
      if (!mountedRef.current || requestGeneration !== generationRef.current) return null;
      const responseToken = statusResponseGate.begin(requestGeneration);

      try {
        const rsp = await api.getUpdateStatus();
        if (!mountedRef.current || requestGeneration !== generationRef.current) return null;
        if (rsp.code !== 0 || !isApplicationUpdateStatus(rsp.data)) {
          throw new Error('invalid update status response');
        }
        if (!statusResponseGate.accept(responseToken)) {
          recoverRejectedStatusPoll(busyRef.current, () => {
            clearPollTimer();
            pollTimerRef.current = setTimeout(
              () => void pollStatus(requestGeneration),
              POLL_INTERVAL_MS
            );
          });
          return null;
        }

        if (isStaleOperationStatus(rsp.data)) {
          const retryIndex = pollFailuresRef.current;
          if (retryIndex < STATUS_RETRY_DELAYS_MS.length) {
            pollFailuresRef.current += 1;
            pollTimerRef.current = setTimeout(
              () => void pollStatus(requestGeneration),
              STATUS_RETRY_DELAYS_MS[retryIndex]
            );
          } else {
            busyRef.current = false;
            setPageStatus('failed');
            setErrMsg(t('settings.update.statusUnavailable'));
            settingsBusyLock.setApplicationBusy(false);
          }
          return null;
        }

        reconnectObservedRef.current = true;
        pollFailuresRef.current = 0;
        const keepPolling = applyStatus(rsp.data, requestGeneration);
        if (keepPolling) {
          pollTimerRef.current = setTimeout(
            () => void pollStatus(requestGeneration),
            POLL_INTERVAL_MS
          );
        }
        return rsp.data;
      } catch {
        if (!mountedRef.current || requestGeneration !== generationRef.current) return null;
        if (!busyRef.current || !scheduleRetry) return null;

        if (reconnectObservedRef.current) {
          // A request succeeded after submission, then the server disappeared.
          // The application update replaces the server itself, so restart the
          // bounded retry window for this expected hand-off.
          reconnectObservedRef.current = false;
          pollFailuresRef.current = 0;
        }

        // The application server is replaced during installation. Keep the
        // operation locked while a bounded sequence of status requests fails.
        const retryIndex = pollFailuresRef.current;
        if (retryIndex < STATUS_RETRY_DELAYS_MS.length) {
          pollFailuresRef.current += 1;
          pollTimerRef.current = setTimeout(
            () => void pollStatus(requestGeneration),
            STATUS_RETRY_DELAYS_MS[retryIndex]
          );
          return null;
        }

        busyRef.current = false;
        setPageStatus('failed');
        setErrMsg(t('settings.update.statusUnavailable'));
        settingsBusyLock.setApplicationBusy(false);
        return null;
      }
    },
    [applyStatus, clearPollTimer, isStaleOperationStatus, settingsBusyLock, statusResponseGate, t]
  );

  const recoverInitialStatus = useCallback(
    async (requestGeneration: number): Promise<ApplicationUpdateStatus | null> => {
      settingsBusyLock.setApplicationBusy(true);

      for (let attempt = 0; attempt <= STATUS_RETRY_DELAYS_MS.length; attempt += 1) {
        if (!mountedRef.current || requestGeneration !== generationRef.current) return null;
        try {
          const responseToken = statusResponseGate.begin(requestGeneration);
          const rsp = await api.getUpdateStatus();
          if (!mountedRef.current || requestGeneration !== generationRef.current) return null;
          if (rsp.code !== 0 || !isApplicationUpdateStatus(rsp.data)) {
            throw new Error('invalid update status response');
          }
          if (!statusResponseGate.accept(responseToken)) continue;
          pollFailuresRef.current = 0;
          const keepPolling = applyStatus(rsp.data, requestGeneration);
          if (keepPolling) {
            pollTimerRef.current = setTimeout(
              () => void pollStatus(requestGeneration),
              POLL_INTERVAL_MS
            );
          }
          return rsp.data;
        } catch {
          if (attempt === STATUS_RETRY_DELAYS_MS.length) break;
          await new Promise((resolve) => setTimeout(resolve, STATUS_RETRY_DELAYS_MS[attempt]));
        }
      }

      if (!mountedRef.current || requestGeneration !== generationRef.current) return null;
      setPageStatus('failed');
      setErrMsg(t('settings.update.statusUnavailable'));
      settingsBusyLock.setApplicationBusy(false);
      return null;
    },
    [applyStatus, pollStatus, settingsBusyLock, statusResponseGate, t]
  );

  useEffect(() => {
    mountedRef.current = true;
    const requestGeneration = beginGeneration();
    setPageStatus('loading');

    void (async () => {
      await recoverInitialStatus(requestGeneration);
    })();

    return () => {
      mountedRef.current = false;
      clearPollTimer();
      abortUploadRef.current?.abort();
      if (shouldReleaseSettingsLockOnUnmount(busyRef.current)) {
        settingsBusyLock.setApplicationBusy(false);
        settingsBusyLock.releaseIfApplicationIdle();
      }
    };
  }, [beginGeneration, checkForUpdates, clearPollTimer, recoverInitialStatus, settingsBusyLock]);

  useEffect(() => {
    const statusRefresh = createAuthoritativeStatusRefresh({
      debounceMs: WS_STATUS_REFRESH_DEBOUNCE_MS,
      getGeneration: () => generationRef.current,
      request: async () => {
        const responseToken = statusResponseGate.begin();
        return { rsp: await api.getUpdateStatus(), responseToken };
      },
      accept: ({ rsp, responseToken }, requestGeneration) => {
        if (rsp.code !== 0 || !isApplicationUpdateStatus(rsp.data)) return;
        if (!statusResponseGate.accept(responseToken)) return;

        clearPollTimer();
        reconnectObservedRef.current = true;
        pollFailuresRef.current = 0;
        if (applyStatus(rsp.data, requestGeneration)) {
          pollTimerRef.current = setTimeout(
            () => void pollStatus(requestGeneration),
            POLL_INTERVAL_MS
          );
        }
      }
    });
    const unsubscribe = subscribeAuthoritativeUpdateHints(client, statusRefresh);

    return () => {
      unsubscribe();
      statusRefresh.dispose();
    };
  }, [applyStatus, clearPollTimer, pollStatus, statusResponseGate]);

  function beginOperation(source: 'online' | 'offline', phase: 'downloading' | 'uploading') {
    const requestGeneration = beginGeneration();
    if (operationIdRef.current) {
      previousOperationIdRef.current = operationIdRef.current;
    }
    operationIdRef.current = '';
    pollFailuresRef.current = 0;
    reconnectObservedRef.current = false;
    busyRef.current = true;
    setErrMsg('');
    setTerminalActionError('');
    setUploadProgress(0);
    setUpdateStatus({ ...initialUpdateStatus, source, phase });
    setPageStatus('updating');
    settingsBusyLock.setApplicationBusy(true);
    return requestGeneration;
  }

  async function reconcileSubmittedOperation(
    requestGeneration: number,
    responseStatus?: unknown
  ): Promise<ApplicationUpdateStatus | null> {
    let recovered: ApplicationUpdateStatus | null = null;
    if (isApplicationUpdateStatus(responseStatus) && !isStaleOperationStatus(responseStatus)) {
      applyStatus(responseStatus, requestGeneration);
      recovered = responseStatus;
    }
    return (await pollStatus(requestGeneration, true)) ?? recovered;
  }

  function statusOwnsOutcome(status: ApplicationUpdateStatus | null): boolean {
    return status !== null && status.phase !== 'idle';
  }

  function responseStatusFromError(error: unknown): unknown {
    if (!error || typeof error !== 'object') return null;
    return (error as { response?: { data?: { data?: unknown } } }).response?.data?.data;
  }

  async function updateOnline() {
    if (
      pageStatus !== 'outdated' ||
      busyRef.current ||
      previewBusyRef.current ||
      isBusy ||
      isUpdateServerPending
    ) {
      return;
    }
    const requestGeneration = beginOperation('online', 'downloading');

    try {
      const rsp = await api.update();
      if (requestGeneration !== generationRef.current) return;
      if (rsp.code !== 0) {
        const recovered = await reconcileSubmittedOperation(requestGeneration, rsp.data);
        if (busyRef.current || statusOwnsOutcome(recovered)) return;
        throw new Error(rsp.msg);
      }
      await reconcileSubmittedOperation(requestGeneration, rsp.data);
    } catch (error) {
      if (requestGeneration !== generationRef.current) return;
      const recovered = await reconcileSubmittedOperation(
        requestGeneration,
        responseStatusFromError(error)
      );
      if (!busyRef.current && !statusOwnsOutcome(recovered)) {
        busyRef.current = false;
        setPageStatus('failed');
        setErrMsg(t('settings.update.updateFailed'));
        settingsBusyLock.setApplicationBusy(false);
      }
    }
  }

  function chooseFile(file?: File) {
    if (!file || busyRef.current || previewBusyRef.current || isBusy) return;
    if (!isCandidateApplicationBundle(file)) {
      setSelectedFile(null);
      setOfflineFileError(t('settings.update.invalidFileName'));
      return;
    }
    setOfflineFileError('');
    setSelectedFile(file);
  }

  function browseFile() {
    if (busyRef.current || previewBusyRef.current || isBusy) return;
    if (fileInputRef.current) fileInputRef.current.value = '';
    fileInputRef.current?.click();
  }

  function handleFileInput(event: ChangeEvent<HTMLInputElement>) {
    chooseFile(event.target.files?.[0]);
  }

  function handleDrop(event: DragEvent<HTMLDivElement>) {
    event.preventDefault();
    setIsDragging(false);
    chooseFile(event.dataTransfer.files?.[0]);
  }

  function openOfflineUpdate() {
    if (busyRef.current || previewBusyRef.current || isBusy) return;
    setOfflineFileError('');
    setIsOfflineModalOpen(true);
  }

  function closeOfflineUpdate() {
    if (isBusy) return;
    setIsDragging(false);
    setIsOfflineModalOpen(false);
  }

  function confirmOfflineUpdate() {
    if (!selectedFile || busyRef.current || previewBusyRef.current || isBusy) return;
    modal.confirm({
      title: t('settings.update.offlineConfirmTitle'),
      content: t('settings.update.offlineConfirmDesc', { name: selectedFile.name }),
      okText: t('settings.update.confirm'),
      cancelText: t('settings.update.cancel'),
      centered: true,
      onOk: () => {
        setIsOfflineModalOpen(false);
        setIsDragging(false);
        void uploadOffline(selectedFile);
      }
    });
  }

  async function uploadOffline(file: File) {
    if (busyRef.current || previewBusyRef.current) return;
    const requestGeneration = beginOperation('offline', 'uploading');
    const controller = new AbortController();
    abortUploadRef.current = controller;

    try {
      const rsp = await api.updateOffline(file, {
        signal: controller.signal,
        onProgress: (progress) => {
          if (mountedRef.current && requestGeneration === generationRef.current) {
            setUploadProgress(progress);
          }
        }
      });
      if (!mountedRef.current || requestGeneration !== generationRef.current) return;
      if (rsp.code !== 0) {
        const recovered = await reconcileSubmittedOperation(requestGeneration, rsp.data);
        if (busyRef.current || statusOwnsOutcome(recovered)) return;
        throw new Error(rsp.msg);
      }
      await reconcileSubmittedOperation(requestGeneration, rsp.data);
    } catch (error) {
      if (!mountedRef.current || requestGeneration !== generationRef.current) return;
      const recovered = await reconcileSubmittedOperation(
        requestGeneration,
        responseStatusFromError(error)
      );
      if (!busyRef.current && !statusOwnsOutcome(recovered)) {
        busyRef.current = false;
        setPageStatus('failed');
        setErrMsg(
          controller.signal.aborted
            ? t('settings.update.errors.upload_interrupted')
            : t('settings.update.updateFailed')
        );
        settingsBusyLock.setApplicationBusy(false);
      }
    } finally {
      if (abortUploadRef.current === controller) abortUploadRef.current = null;
    }
  }

  function cancelUpload() {
    abortUploadRef.current?.abort();
  }

  const resetTerminalStatus = useCallback(
    async (reloadAfterSuccess = false) => {
      if (acknowledgingTerminalRef.current) return;
      acknowledgingTerminalRef.current = true;
      setIsAcknowledgingTerminal(true);
      clearPollTimer();
      let requestGeneration = beginGeneration();
      const terminal = updateStatus;
      const shouldAcknowledge =
        !!terminal.operationId && (terminal.phase === 'succeeded' || terminal.phase === 'failed');
      setTerminalActionError('');
      settingsBusyLock.setApplicationBusy(true);

      try {
        if (shouldAcknowledge) {
          let acknowledged = false;
          for (let attempt = 0; attempt <= ACKNOWLEDGE_RETRY_DELAYS_MS.length; attempt += 1) {
            if (
              !mountedRef.current ||
              (!reloadAfterSuccess && requestGeneration !== generationRef.current)
            ) {
              return;
            }
            try {
              const rsp = await api.acknowledgeUpdateStatus(terminal.operationId);
              if (!mountedRef.current) return;
              const acknowledgedIdle =
                rsp.code === 0 &&
                isApplicationUpdateStatus(rsp.data) &&
                rsp.data.phase === initialUpdateStatus.phase &&
                rsp.data.operationId === initialUpdateStatus.operationId &&
                rsp.data.source === initialUpdateStatus.source &&
                rsp.data.version === initialUpdateStatus.version &&
                rsp.data.progress === initialUpdateStatus.progress &&
                rsp.data.errorCode === initialUpdateStatus.errorCode &&
                rsp.data.startedAt === initialUpdateStatus.startedAt &&
                rsp.data.updatedAt === initialUpdateStatus.updatedAt;
              if (acknowledgedIdle) {
                acknowledged = true;
                if (reloadAfterSuccess && terminal.phase === 'succeeded') {
                  window.location.reload();
                  return;
                }
                break;
              }
              if (requestGeneration !== generationRef.current) return;
              if (
                isApplicationUpdateStatus(rsp.data) &&
                (rsp.data.operationId !== terminal.operationId ||
                  (rsp.data.phase !== 'succeeded' && rsp.data.phase !== 'failed'))
              ) {
                const keepPolling = applyStatus(rsp.data, requestGeneration);
                if (keepPolling) {
                  pollTimerRef.current = setTimeout(
                    () => void pollStatus(requestGeneration),
                    POLL_INTERVAL_MS
                  );
                }
                return;
              }
            } catch {
              // A lost response is safe to retry because acknowledgement is idempotent.
            }
            if (attempt < ACKNOWLEDGE_RETRY_DELAYS_MS.length) {
              await new Promise((resolve) =>
                setTimeout(resolve, ACKNOWLEDGE_RETRY_DELAYS_MS[attempt])
              );
            }
          }
          if (!acknowledged) {
            throw new Error('failed to acknowledge terminal update status');
          }
        }

        if (!mountedRef.current || requestGeneration !== generationRef.current) return;
        // Invalidate every status request started while acknowledgement was in
        // flight, then recover the authoritative state. Another browser may have
        // started a new update immediately after the terminal state was cleared.
        requestGeneration = beginGeneration();
        await recoverInitialStatus(requestGeneration);
      } catch {
        if (!mountedRef.current || requestGeneration !== generationRef.current) return;
        busyRef.current = false;
        setTerminalActionError(t('settings.update.statusUnavailable'));
        settingsBusyLock.setApplicationBusy(false);
      } finally {
        acknowledgingTerminalRef.current = false;
        if (mountedRef.current) setIsAcknowledgingTerminal(false);
      }
    },
    [
      applyStatus,
      beginGeneration,
      clearPollTimer,
      pollStatus,
      recoverInitialStatus,
      settingsBusyLock,
      t,
      updateStatus
    ]
  );

  useEffect(() => {
    const operationId = updateStatus.operationId;
    if (
      pageStatus !== 'succeeded' ||
      updateStatus.phase !== 'succeeded' ||
      !operationId ||
      isAcknowledgingTerminal ||
      autoRefreshOperationRef.current === operationId
    ) {
      return;
    }

    autoRefreshOperationRef.current = operationId;
    void resetTerminalStatus(true);
  }, [
    isAcknowledgingTerminal,
    pageStatus,
    resetTerminalStatus,
    updateStatus.operationId,
    updateStatus.phase
  ]);

  return (
    <>
      {contextHolder}
      <div className="text-base font-bold">{t('settings.update.title')}</div>
      <div className="mt-1 text-sm text-neutral-400">{releaseName}</div>
      <Divider className="opacity-50" />

      <Preview
        checkForUpdates={() => void checkForUpdates()}
        disabled={isBusy || isUpdateServerPending}
        onBusyChange={handlePreviewBusyChange}
      />
      <UpdateServer
        checkForUpdates={() => {
          versionRequestRef.current += 1;
          versionCheckingRef.current = false;
          void checkForUpdates();
        }}
        disabled={isBusy}
        onBusyChange={handleUpdateServerBusyChange}
        onPendingChange={setIsUpdateServerPending}
      />
      <Divider className="opacity-50" />
      <SignatureVerification
        disabled={isBusy}
        onBusyChange={handleSignatureVerificationBusyChange}
      />
      <Divider className="opacity-50" />
      <ButtonBase
        type="button"
        className={clsx(
          'flex w-full items-center justify-between py-3 text-left transition-colors',
          isBusy ? 'cursor-not-allowed opacity-50' : 'cursor-pointer hover:text-blue-400'
        )}
        disabled={isBusy}
        onClick={openOfflineUpdate}
      >
        <span>{t('settings.update.offlineTitle')}</span>
        <ChevronRightIcon size={18} aria-hidden="true" className="shrink-0 text-neutral-500" />
      </ButtonBase>
      <Divider className="opacity-50" />

      <div className="flex min-h-[400px] flex-col justify-between">
        {pageStatus === 'loading' && (
          <div className="flex justify-center pt-24">
            <Spin indicator={<LoadingOutlined spin />} size="large" />
          </div>
        )}

        {pageStatus === 'updating' && (
          <Updating
            status={updateStatus}
            fileName={updateStatus.source === 'offline' ? selectedFile?.name : undefined}
            fileSize={selectedFile ? formatUpdateFileSize(selectedFile.size) : undefined}
            uploadProgress={uploadProgress}
            onCancelUpload={abortUploadRef.current ? cancelUpload : undefined}
          />
        )}

        {pageStatus === 'latest' && (
          <Result
            status="success"
            icon={<SmileOutlined />}
            title={currentVersion}
            subTitle={t('settings.update.isLatest')}
            extra={[
              <Button key="confirm" disabled={isBusy} onClick={() => void checkForUpdates()}>
                {t('settings.update.title')}
              </Button>
            ]}
          />
        )}

        {pageStatus === 'outdated' && (
          <Result
            status="warning"
            icon={<RocketOutlined />}
            title={`${currentVersion} -> ${latestVersion}`}
            subTitle={tipMsg}
            extra={[
              <Button
                key="confirm"
                type="primary"
                disabled={isBusy || isUpdateServerPending}
                onClick={() => void updateOnline()}
              >
                {t('settings.update.confirm')}
              </Button>
            ]}
          />
        )}

        {pageStatus === 'succeeded' && (
          <Result
            status="success"
            icon={<CheckCircleOutlined />}
            title={t('settings.update.updateSucceeded')}
            subTitle={terminalActionError || updateStatus.version}
            extra={[
              <Button
                key="continue"
                loading={isAcknowledgingTerminal}
                disabled={isBusy}
                onClick={() => void resetTerminalStatus()}
              >
                {t('settings.update.continue')}
              </Button>
            ]}
          />
        )}

        {(pageStatus === 'query-failed' || pageStatus === 'failed') && (
          <Result
            status="error"
            title={t(
              pageStatus === 'query-failed'
                ? 'settings.update.queryFailed'
                : 'settings.update.updateFailedTitle'
            )}
            subTitle={
              terminalActionError || (pageStatus === 'query-failed' ? undefined : errMsg)
            }
            extra={[
              <Button
                key="retry"
                loading={isAcknowledgingTerminal}
                disabled={isBusy}
                onClick={() => void resetTerminalStatus()}
              >
                {t('settings.update.retry')}
              </Button>
            ]}
          />
        )}

        <div className="flex justify-center">
          <Button
            type="link"
            size="small"
            href={`${projectUrl}/blob/main/CHANGELOG.md`}
            target="_blank"
            disabled={isBusy}
          >
            {t('settings.update.changelog')}
          </Button>
        </div>
      </div>

      <Modal
        title={t('settings.update.offlineTitle')}
        open={isOfflineModalOpen}
        centered
        okText={t('settings.update.offlineConfirm')}
        cancelText={t('settings.update.cancel')}
        okButtonProps={{ disabled: !selectedFile || isBusy }}
        cancelButtonProps={{ disabled: isBusy }}
        closable={!isBusy}
        maskClosable={!isBusy}
        onOk={confirmOfflineUpdate}
        onCancel={closeOfflineUpdate}
      >
        <div className="space-y-4 pt-2">
          <input
            ref={fileInputRef}
            type="file"
            accept=".tar.gz,application/gzip"
            className="hidden"
            disabled={isBusy}
            onChange={handleFileInput}
          />
          <div
            role="button"
            tabIndex={isBusy ? -1 : 0}
            aria-disabled={isBusy}
            className={clsx(
              'flex min-h-36 flex-col items-center justify-center rounded-lg border border-dashed px-4 py-6 text-center transition',
              isDragging
                ? 'border-blue-400 bg-blue-500/10'
                : 'border-neutral-600 bg-neutral-800/30',
              isBusy ? 'cursor-not-allowed opacity-50' : 'cursor-pointer hover:border-blue-500/70'
            )}
            onClick={browseFile}
            onKeyDown={(event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault();
                browseFile();
              }
            }}
            onDragEnter={(event) => {
              event.preventDefault();
              if (!isBusy) setIsDragging(true);
            }}
            onDragOver={(event) => event.preventDefault()}
            onDragLeave={() => setIsDragging(false)}
            onDrop={handleDrop}
          >
            <FileArchiveIcon size={30} className="mb-3 text-blue-400" />
            {selectedFile ? (
              <>
                <div className="max-w-full truncate text-sm text-neutral-100">
                  {selectedFile.name}
                </div>
                <div className="mt-1 text-xs text-neutral-500">
                  {formatUpdateFileSize(selectedFile.size)}
                </div>
              </>
            ) : (
              <div className="text-sm">{t('settings.update.dropFile')}</div>
            )}
          </div>
          {offlineFileError && (
            <div role="alert" className="text-center text-xs text-red-400">
              {offlineFileError}
            </div>
          )}
        </div>
      </Modal>
    </>
  );
};
