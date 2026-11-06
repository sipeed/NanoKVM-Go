import { ChangeEvent, useEffect, useRef, useState } from 'react';
import { Button, Input, InputRef, Modal, Progress } from 'antd';
import { useSetAtom } from 'jotai';
import { HardDriveDownloadIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import {
  cancelImageDownload,
  downloadImage,
  imageEnabled,
  pauseImageDownload,
  resumeImageDownload,
  statusImage
} from '@/api/storage';
import type { ImageDownloadStatus } from '@/api/storage';
import { getIntlLocale } from '@/i18n/locale.ts';
import { isKeyboardEnableAtom } from '@/jotai/keyboard';

import {
  formatDownloadTransferSummary,
  getDownloadTransferDetails,
  parseDownloadProgress
} from './download-progress.ts';
import { ModalHeader } from '@/components/modal-header.tsx';

const insufficientStorageResponseMessage = 'insufficient storage';

export const DownloadImage = () => {
  const [modal, setModal] = useState(false);
  const { t, i18n } = useTranslation();
  const setIsKeyboardEnable = useSetAtom(isKeyboardEnableAtom);

  const [input, setInput] = useState('');
  const [status, setStatus] = useState('');
  const [log, setLog] = useState('');
  const [progress, setProgress] = useState<number | undefined>(undefined);
  const [transfer, setTransfer] = useState<ImageDownloadStatus | undefined>(undefined);
  const [canCancel, setCanCancel] = useState(false);
  const [isCancelling, setIsCancelling] = useState(false);
  const [isChangingPause, setIsChangingPause] = useState(false);
  const [actionError, setActionError] = useState('');
  const [diskEnabled, setDiskEnabled] = useState(false);
  const [popoverKey, setPopoverKey] = useState(0);

  const inputRef = useRef<InputRef>(null);

  const intervalId = useRef<ReturnType<typeof setInterval> | undefined>(undefined);
  const startRequestPending = useRef(false);
  const statusRequestGeneration = useRef(0);

  useEffect(() => {
    checkDiskEnabled();
    return () => {
      stopStatusPolling();
      invalidateStatusRequests();
      setIsKeyboardEnable(true);
    };
  }, [setIsKeyboardEnable]);

  function checkDiskEnabled() {
    imageEnabled()
      .then((res) => {
        setDiskEnabled(res.data.enabled);
      })
      .catch(() => {
        setDiskEnabled(false);
      });
  }

  function stopStatusPolling() {
    clearInterval(intervalId.current);
    intervalId.current = undefined;
  }

  function startStatusPolling() {
    stopStatusPolling();
    intervalId.current = setInterval(getDownloadStatus, 2500);
  }

  function invalidateStatusRequests() {
    statusRequestGeneration.current += 1;
  }

  function handleOpenChange(open: boolean) {
    if (open) {
      stopStatusPolling();
      checkDiskEnabled();
      getDownloadStatus();
      startStatusPolling();
      setIsKeyboardEnable(false);
      setPopoverKey((prevKey) => prevKey + 1); // Force re-render
    } else {
      invalidateStatusRequests();
      setInput('');
      setStatus('');
      setLog('');
      setProgress(undefined);
      setTransfer(undefined);
      setCanCancel(false);
      setActionError('');

      setIsKeyboardEnable(true);
      stopStatusPolling();
    }
  }

  function handleChange(e: ChangeEvent<HTMLInputElement>) {
    setInput(e.target.value);
  }

  function getDownloadStatus() {
    const requestGeneration = statusRequestGeneration.current;
    statusImage().then((rsp) => {
      if (requestGeneration !== statusRequestGeneration.current) {
        return;
      }
      if (rsp.data.status === 'idle' && startRequestPending.current) {
        return;
      }
      if (rsp.data.status) {
        setStatus(rsp.data.status);
        if (rsp.data.status === 'in_progress' || rsp.data.status === 'paused') {
          setProgress(parseDownloadProgress(rsp.data.percentage));
          setTransfer(rsp.data);
          if (!startRequestPending.current) {
            setCanCancel(true);
          }
          // Check if rsp has a percentage value
          if (rsp.data.percentage) {
            setLog(
              t('download.downloadingProgress', {
                percentage: rsp.data.percentage,
                file: rsp.data.file
              })
            );
          } else {
            setLog(t('download.downloadingFile', { file: rsp.data.file }));
          }
          setInput(rsp.data.file);
        }
        if (rsp.data.status === 'failed') {
          invalidateStatusRequests();
          setProgress(undefined);
          setTransfer(undefined);
          setCanCancel(false);
          setLog(t('download.failed'));
          stopStatusPolling();
        }
        if (rsp.data.status === 'idle') {
          invalidateStatusRequests();
          setProgress(undefined);
          setTransfer(undefined);
          setCanCancel(false);
          setActionError('');
          setLog(''); // Clear the log
          stopStatusPolling();
        }
      }
    });
  }

  function download(url?: string) {
    if (!url) return;

    setStatus('in_progress');
    setLog(t('download.downloadingFile', { file: url }));
    setProgress(undefined);
    setTransfer(undefined);
    setCanCancel(false);
    setActionError('');
    invalidateStatusRequests();
    startRequestPending.current = true;

    downloadImage(url)
      .then((rsp) => {
        startRequestPending.current = false;
        if (rsp.code !== 0) {
          invalidateStatusRequests();
          setStatus('failed');
          setProgress(undefined);
          setTransfer(undefined);
          setCanCancel(false);
          console.log(rsp.msg);
          setLog(
            t(
              rsp.msg === insufficientStorageResponseMessage
                ? 'download.insufficientStorage'
                : 'download.failed'
            )
          );
          stopStatusPolling();
          return;
        }

        setCanCancel(true);
        getDownloadStatus();
        startStatusPolling();
      })
      .catch(() => {
        startRequestPending.current = false;
        invalidateStatusRequests();
        stopStatusPolling();
        setStatus('failed');
        setProgress(undefined);
        setTransfer(undefined);
        setCanCancel(false);
        setLog(t('download.failed'));
      });
  }

  function cancelDownload() {
    if (!canCancel || isCancelling || isChangingPause) return;

    setIsCancelling(true);
    setActionError('');
    stopStatusPolling();
    invalidateStatusRequests();
    cancelImageDownload()
      .then((rsp) => {
        if (rsp.code !== 0) {
          throw new Error('failed to cancel image download');
        }

        stopStatusPolling();
        setStatus('idle');
        setLog('');
        setProgress(undefined);
        setTransfer(undefined);
        setCanCancel(false);
      })
      .catch(() => {
        setActionError(t('download.cancelFailed'));
        getDownloadStatus();
        startStatusPolling();
      })
      .finally(() => {
        setIsCancelling(false);
      });
  }

  function changeDownloadPaused(paused: boolean) {
    if (!canCancel || isCancelling || isChangingPause) return;

    setIsChangingPause(true);
    setActionError('');
    stopStatusPolling();
    invalidateStatusRequests();
    const request = paused ? pauseImageDownload() : resumeImageDownload();
    request
      .then((rsp) => {
        if (rsp.code !== 0) {
          throw new Error('failed to change image download pause state');
        }

        setStatus(rsp.data.status);
        if (rsp.data.status === 'idle') {
          setCanCancel(false);
        }
        getDownloadStatus();
        startStatusPolling();
      })
      .catch(() => {
        setActionError(t(paused ? 'download.pauseFailed' : 'download.resumeFailed'));
        getDownloadStatus();
        startStatusPolling();
      })
      .finally(() => {
        setIsChangingPause(false);
      });
  }

  const intlLocale = getIntlLocale(i18n.resolvedLanguage || i18n.language);
  const transferDetails = transfer ? getDownloadTransferDetails(transfer, intlLocale) : undefined;
  const summaryDetails =
    transferDetails && status === 'paused'
      ? { ...transferDetails, speed: undefined, remaining: undefined }
      : transferDetails;
  const transferSummary = summaryDetails
    ? formatDownloadTransferSummary(
        summaryDetails,
        intlLocale,
        summaryDetails.remaining
          ? t('download.remaining', { time: summaryDetails.remaining })
          : undefined
      )
    : '';
  const downloadActive = status === 'in_progress' || status === 'paused';

  return (
    <>
      <div
        onClick={() => setModal(true)}
        className="flex h-[30px] cursor-pointer select-none items-center space-x-2 rounded pl-3 pr-6 text-neutral-300 hover:bg-neutral-700/70"
      >
        <HardDriveDownloadIcon size={18} />
        <span className="select-none text-sm">{t('image.download')}</span>
      </div>

      <Modal
        open={modal}
        afterOpenChange={handleOpenChange}
        onCancel={() => setModal(false)}
        title={
          <ModalHeader icon={<HardDriveDownloadIcon size={18} />} title={t('download.title')} />
        }
        footer={null}
      >
        <div key={popoverKey} className="flex w-full flex-col gap-4 pt-2">
          {!diskEnabled ? (
            <div className="w-full break-words text-sm text-red-500">
              {t('image.storageReadOnly')}
            </div>
          ) : (
            <div className="flex w-full flex-col gap-2">
              <div className="text-xs text-neutral-500">{t('download.input')}</div>
              <div className="flex w-full items-center gap-2">
                <Input
                  ref={inputRef}
                  className="min-w-0 flex-1"
                  value={input}
                  onChange={handleChange}
                  disabled={downloadActive}
                />
                {downloadActive ? (
                  <div className="flex shrink-0 items-center gap-2">
                    <Button
                      disabled={!canCancel || isCancelling}
                      loading={isChangingPause}
                      onClick={() => changeDownloadPaused(status !== 'paused')}
                    >
                      {t(status === 'paused' ? 'download.resume' : 'download.pause')}
                    </Button>
                    <Button
                      danger
                      disabled={!canCancel || isChangingPause}
                      loading={isCancelling}
                      onClick={cancelDownload}
                    >
                      {t('download.cancel')}
                    </Button>
                  </div>
                ) : (
                  <Button className="shrink-0" type="primary" onClick={() => download(input)}>
                    {t('download.ok')}
                  </Button>
                )}
              </div>
            </div>
          )}
          {actionError && (
            <div className="w-full break-words text-sm text-red-500">{actionError}</div>
          )}
          {status === 'failed' && (
            <div className="w-full break-words text-sm text-red-500">{log}</div>
          )}
          {downloadActive && (progress !== undefined || transferSummary) && (
            <div className="flex w-full flex-col gap-2">
              {progress !== undefined && (
                <Progress
                  className="m-0 w-full"
                  percent={progress}
                  showInfo={false}
                  size="small"
                  status={status === 'paused' ? 'normal' : 'active'}
                />
              )}
              {transferSummary && (
                <div className="w-full text-sm tabular-nums text-neutral-400">
                  {transferSummary}
                </div>
              )}
            </div>
          )}
        </div>
      </Modal>
    </>
  );
};
