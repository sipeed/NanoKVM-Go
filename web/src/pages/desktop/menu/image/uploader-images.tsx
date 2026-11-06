import { useRef, useState, type ChangeEvent, type DragEvent } from 'react';
import { Modal, Progress } from 'antd';
import clsx from 'clsx';
import { HardDriveUploadIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { cancelImageUpload, imageEnabled, uploadImage } from '@/api/storage';
import { getIntlLocale } from '@/i18n/locale.ts';

import {
  formatDownloadBytes,
  formatDownloadTransferSummary,
  getDownloadTransferDetails
} from './download-progress.ts';
import { ModalHeader } from '@/components/modal-header.tsx';
import { getUploadTransferMetrics, type UploadTransferMetrics } from './upload-progress.ts';

const acceptedImageFiles = '.img,.iso';
type UploadState = 'idle' | 'uploading' | 'cancelling';

const insufficientStorageResponseMessage = 'insufficient storage';
const imageSizeLimitResponseMessage = 'image exceeds size limit';

class ImageUploadCancelledError extends Error {
  constructor() {
    super('image upload cancelled');
    this.name = 'ImageUploadCancelledError';
  }
}

class ImageUploadRequestError extends Error {
  readonly i18nKey?: string;

  constructor(i18nKey?: string) {
    super(i18nKey ?? 'image upload failed');
    this.name = 'ImageUploadRequestError';
    this.i18nKey = i18nKey;
  }
}

// The server reports capacity rejections with stable response messages; map
// exactly those to localized copy and keep every other message off the UI.
function imageUploadErrorKey(msg: string): string | undefined {
  if (msg === insufficientStorageResponseMessage) return 'image.uploadInsufficientStorage';
  if (msg === imageSizeLimitResponseMessage) return 'image.uploadTooLarge';
  return undefined;
}

function createImageUploadID() {
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');
}

export const UploaderImages = () => {
  const { t, i18n } = useTranslation();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [storageWritable, setStorageWritable] = useState(false);
  const [uploadState, setUploadState] = useState<UploadState>('idle');
  const [isDragging, setIsDragging] = useState(false);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [progress, setProgress] = useState(0);
  const [transfer, setTransfer] = useState<UploadTransferMetrics | undefined>(undefined);
  const [uploadError, setUploadError] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);
  const uploadingRef = useRef(false);
  const uploadStateRef = useRef<UploadState>('idle');
  const cancelRequestedRef = useRef(false);
  const abortUploadRef = useRef<AbortController | null>(null);
  const currentUploadIDRef = useRef<string | null>(null);
  const uploadTaskRef = useRef<Promise<void> | null>(null);

  function updateUploadState(state: UploadState) {
    uploadStateRef.current = state;
    setUploadState(state);
  }

  function uploadSelectedFile() {
    const file = selectedFile;
    if (!file || uploadingRef.current || !storageWritable) return;

    const task = performImageUpload(file);
    uploadTaskRef.current = task;
    void task.finally(() => {
      if (uploadTaskRef.current === task) uploadTaskRef.current = null;
    });
  }

  async function performImageUpload(file: File) {
    const uploadID = createImageUploadID();
    const uploadStartedAt = performance.now();

    const chunkSize = 8 * 1024 * 1024;
    uploadingRef.current = true;
    currentUploadIDRef.current = uploadID;
    cancelRequestedRef.current = false;
    updateUploadState('uploading');
    setProgress(0);
    setTransfer(getUploadTransferMetrics(0, file.size, 0));
    setUploadError('');
    let uploadSucceeded = false;
    let chunkIndex = 0;

    function updateTransferProgress(uploadedBytes: number) {
      const nextTransfer = getUploadTransferMetrics(
        uploadedBytes,
        file.size,
        performance.now() - uploadStartedAt
      );
      setTransfer(nextTransfer);
      setProgress(file.size > 0 ? (nextTransfer.uploadedBytes / file.size) * 100 : 0);
    }

    try {
      const totalChunks = Math.ceil(file.size / chunkSize);

      if (totalChunks === 0) {
        throw new Error(t('image.uploadError'));
      }

      while (chunkIndex < totalChunks) {
        if (cancelRequestedRef.current) throw new ImageUploadCancelledError();

        const start = chunkIndex * chunkSize;
        const end = Math.min(start + chunkSize, file.size);
        const blob = file.slice(start, end, file.type || 'application/octet-stream');
        const controller = new AbortController();
        abortUploadRef.current = controller;

        let response;
        try {
          response = await uploadImage(
            blob,
            {
              uploadId: uploadID,
              filename: file.name,
              chunkSize,
              chunkIndex,
              totalChunks
            },
            {
              signal: controller.signal,
              onProgress: (uploadedChunkBytes) => {
                updateTransferProgress(start + uploadedChunkBytes);
              }
            }
          );
        } finally {
          if (abortUploadRef.current === controller) abortUploadRef.current = null;
        }

        if (cancelRequestedRef.current) throw new ImageUploadCancelledError();
        const { code, msg } = response;

        if (code !== 0) {
          console.log(`image upload chunk failed: ${msg}`);
          throw new ImageUploadRequestError(imageUploadErrorKey(msg));
        }

        chunkIndex++;
        updateTransferProgress(end);
      }
      uploadSucceeded = true;
    } catch (error) {
      if (!cancelRequestedRef.current) {
        try {
          await cancelImageUpload(uploadID);
        } catch {
          // The original upload error remains the most useful message. The
          // unfinished file stays hidden and can be cleaned up by cancellation.
        }
        setUploadError(
          error instanceof ImageUploadRequestError && error.i18nKey
            ? t(error.i18nKey)
            : t('image.uploadError')
        );
      }
    } finally {
      uploadingRef.current = false;
      abortUploadRef.current = null;
      if (currentUploadIDRef.current === uploadID) currentUploadIDRef.current = null;
      if (!cancelRequestedRef.current) {
        updateUploadState('idle');
        setProgress(0);
        setTransfer(undefined);
      }
      if (uploadSucceeded) {
        setSelectedFile(null);
        if (inputRef.current) inputRef.current.value = '';
      }
    }
  }

  function chooseFile(file?: File) {
    if (!file || uploadingRef.current || uploadStateRef.current !== 'idle') return;
    setSelectedFile(file);
    setProgress(0);
    setTransfer(undefined);
    setUploadError('');
  }

  function selectFile(): void {
    if (uploadingRef.current || uploadStateRef.current !== 'idle') return;
    if (inputRef.current) inputRef.current.value = '';
    inputRef.current?.click();
  }

  function handleFileInput(event: ChangeEvent<HTMLInputElement>) {
    chooseFile(event.target.files?.[0]);
  }

  function handleDrop(event: DragEvent<HTMLDivElement>) {
    event.preventDefault();
    setIsDragging(false);
    chooseFile(event.dataTransfer.files?.[0]);
  }

  function closeModal() {
    if (uploadingRef.current || uploadStateRef.current !== 'idle') return;
    setIsDragging(false);
    setSelectedFile(null);
    setProgress(0);
    setTransfer(undefined);
    setUploadError('');
    if (inputRef.current) inputRef.current.value = '';
    setIsModalOpen(false);
  }

  async function cancelUpload() {
    const uploadID = currentUploadIDRef.current;
    if (!uploadingRef.current || !uploadID || uploadStateRef.current === 'cancelling') return;

    cancelRequestedRef.current = true;
    updateUploadState('cancelling');
    setUploadError('');
    abortUploadRef.current?.abort();

    let cancelSucceeded = false;
    try {
      const { code, msg } = await cancelImageUpload(uploadID);
      if (code !== 0) {
        console.log(`cancel image upload failed: ${msg}`);
        throw new ImageUploadRequestError(imageUploadErrorKey(msg));
      }
      await uploadTaskRef.current;
      cancelSucceeded = true;
    } catch {
      await uploadTaskRef.current;
      setUploadError(t('image.cancelUploadError'));
    } finally {
      cancelRequestedRef.current = false;
      uploadingRef.current = false;
      currentUploadIDRef.current = null;
      updateUploadState('idle');
      setProgress(0);
      setTransfer(undefined);
    }

    if (cancelSucceeded) closeModal();
  }

  function handleCancelAction() {
    if (uploadStateRef.current === 'idle') closeModal();
    else void cancelUpload();
  }

  const intlLocale = getIntlLocale(i18n.resolvedLanguage || i18n.language);
  const isBusy = uploadState !== 'idle';
  const transferDetails = transfer
    ? getDownloadTransferDetails(
        {
          downloadedBytes: transfer.uploadedBytes,
          totalBytes: transfer.totalBytes,
          bytesPerSecond: transfer.bytesPerSecond,
          remainingSeconds: transfer.remainingSeconds
        },
        intlLocale
      )
    : undefined;
  const transferSummary = transferDetails
    ? formatDownloadTransferSummary(
        transferDetails,
        intlLocale,
        transferDetails.remaining
          ? t('download.remaining', { time: transferDetails.remaining })
          : undefined
      )
    : '';

  return (
    <>
      <div
        onClick={() => {
          setIsModalOpen(true);
          setStorageWritable(false);
          imageEnabled()
            .then((rsp) => setStorageWritable(rsp.code === 0 && !!rsp.data?.enabled))
            .catch(() => setStorageWritable(false));
        }}
        className="flex h-[30px] cursor-pointer select-none items-center space-x-2 rounded pl-3 pr-6 text-neutral-300 hover:bg-neutral-700/70"
      >
        <HardDriveUploadIcon size={18} />
        <span className="text-sm">{t('image.upload')}</span>
      </div>

      <Modal
        open={isModalOpen}
        title={<ModalHeader icon={<HardDriveUploadIcon size={18} />} title={t('image.upload')} />}
        footer={storageWritable ? undefined : null}
        okText={t('image.uploadAction')}
        cancelText={t('image.cancelAction')}
        okButtonProps={{
          disabled: !selectedFile || isBusy || !storageWritable,
          loading: uploadState === 'uploading'
        }}
        cancelButtonProps={{
          disabled: uploadState === 'cancelling',
          loading: uploadState === 'cancelling'
        }}
        closable={!isBusy}
        keyboard={!isBusy}
        maskClosable={!isBusy}
        onOk={uploadSelectedFile}
        onCancel={handleCancelAction}
      >
        <div className="flex w-full flex-col gap-4 pt-2">
          {!storageWritable && (
            <div className="text-sm text-red-500">{t('image.storageReadOnly')}</div>
          )}
          {storageWritable && (
            <>
              <input
                ref={inputRef}
                type="file"
                accept={acceptedImageFiles}
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
                  isBusy
                    ? 'cursor-not-allowed opacity-50'
                    : 'cursor-pointer hover:border-blue-500/70'
                )}
                onClick={selectFile}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' || event.key === ' ') {
                    event.preventDefault();
                    selectFile();
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
                <HardDriveUploadIcon size={30} className="mb-3 text-blue-400" />
                {selectedFile ? (
                  <>
                    <div className="max-w-full truncate text-sm text-neutral-100">
                      {selectedFile.name}
                    </div>
                    <div className="mt-1 text-xs text-neutral-500">
                      {formatDownloadBytes(selectedFile.size, intlLocale)}
                    </div>
                  </>
                ) : (
                  <>
                    <div className="text-sm">{t('image.dropFile')}</div>
                    <div className="mt-1 text-xs text-neutral-500">.iso, .img</div>
                  </>
                )}
              </div>
              {uploadError && (
                <div className="w-full break-words text-sm text-red-500">{uploadError}</div>
              )}
              {isBusy && (
                <div className="flex w-full flex-col gap-2">
                  <Progress
                    className="m-0 w-full"
                    percent={Math.round(progress)}
                    showInfo={false}
                    size="small"
                    status="active"
                  />
                  {transferSummary && (
                    <div className="w-full text-sm tabular-nums text-neutral-400">
                      {transferSummary}
                    </div>
                  )}
                </div>
              )}
            </>
          )}
        </div>
      </Modal>
    </>
  );
};
