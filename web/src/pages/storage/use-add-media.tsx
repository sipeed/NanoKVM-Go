import { useEffect, useRef, useState, type DragEvent } from 'react';
import { Form, Input, Modal } from 'antd';
import clsx from 'clsx';
import { UploadIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { getImageTransferStatus } from '@/api/media.ts';
import * as downloadApi from '@/api/storage.ts';
import { formatSize } from '@/lib/format.ts';

type Props = {
  active: boolean;
  // Confirmed by the backend after recovering any host-side USB eject.
  localWritable: boolean;
  onChanged: () => void;
  onError: (message: string, description?: string) => void;
};

// An image arriving from the browser or from a URL, reported by the image
// downloader's status endpoint. The page renders it at the top, in the same
// place and shape as a media job, instead of a separate hint inside the
// image group.
export type Transfer = {
  // Each transfer has its own id, so the timer that clears a finished one
  // cannot clear the next one started in the meantime.
  id: number;
  kind: 'upload' | 'download';
  name: string;
  percentage: string;
  // Bytes transferred so far and the expected total; total is 0 while the
  // server does not know the size yet.
  loaded: number;
  total: number;
  state: 'running' | 'success' | 'failed';
  message: string;
  cancellable: boolean;
  paused?: boolean;
};

const sha256Pattern = /^[a-fA-F0-9]{64}$/;

// How long a finished transfer stays on screen before it goes away.
const successLinger = 3000;

// A URL names its file in the last path segment; the whole URL is too long to
// be a useful label.
function urlBaseName(url: string): string {
  try {
    const name = new URL(url).pathname.split('/').filter(Boolean).pop();
    return name ? decodeURIComponent(name) : url;
  } catch {
    return url;
  }
}

export function useAddMedia({ active, localWritable, onChanged, onError }: Props) {
  const { t } = useTranslation();

  const [isUrlOpen, setIsUrlOpen] = useState(false);
  const [transfer, setTransfer] = useState<Transfer | null>(null);
  // The server verifies the checksum of an upload when the client sends one;
  // a browser cannot hash gigabytes before sending them, so the expected sum is
  // typed in instead.
  const [checksum, setChecksum] = useState('');
  const [isUploadOpen, setIsUploadOpen] = useState(false);
  const [pendingFile, setPendingFile] = useState<File | null>(null);
  const [isDragging, setIsDragging] = useState(false);

  const [urlForm] = Form.useForm();
  const imageInputRef = useRef<HTMLInputElement>(null);

  // The transfer being followed. The status endpoint keeps the last result of
  // the previous transfer until a new one starts on the server, so a result is
  // only believed once this transfer has been seen running there.
  const task = useRef<{
    id: number;
    polling: boolean;
    seenRunning: boolean;
    finished: boolean;
    controller?: AbortController;
    uploadId?: string;
  } | null>(null);
  const taskId = useRef(0);
  const followingDownload = transfer?.kind === 'download' && transfer.state === 'running';

  useEffect(
    () => () => {
      const current = task.current;
      if (!current || current.finished || !current.controller) return;
      current.controller.abort();
      if (current.uploadId) void downloadApi.cancelImageUpload(current.uploadId).catch(() => {});
    },
    []
  );

  // The page hands a new callback on every render; reading it through a ref
  // keeps the poll interval below from being torn down each time.
  const onChangedRef = useRef(onChanged);
  useEffect(() => {
    onChangedRef.current = onChanged;
  }, [onChanged]);

  function start(
    next: Omit<Transfer, 'id' | 'state' | 'message' | 'loaded' | 'total' | 'percentage'>
  ) {
    const id = ++taskId.current;
    task.current = { id, polling: false, seenRunning: false, finished: false };
    setTransfer({
      ...next,
      id,
      state: 'running',
      message: '',
      loaded: 0,
      total: 0,
      percentage: ''
    });
    return task.current;
  }

  // Ends a transfer once, however many of the request, the poll and a cancel
  // report it: a success refreshes the list a single time and lingers briefly.
  function finish(id: number, result: 'success' | 'failed' | 'cancelled', message = '') {
    const current = task.current;
    if (!current || current.id !== id || current.finished) return;
    current.finished = true;
    current.polling = false;

    if (result === 'cancelled') {
      setTransfer((value) => (value?.id === id ? null : value));
      return;
    }

    setTransfer((value) =>
      value?.id === id ? { ...value, state: result, message, cancellable: false } : value
    );
    if (result === 'success') {
      onChangedRef.current();
      setTimeout(() => setTransfer((value) => (value?.id === id ? null : value)), successLinger);
    }
  }

  // The image downloader reports progress through its own status endpoint;
  // this polls it for both the browser upload and the remote download.
  useEffect(() => {
    if (!active && !followingDownload) return;
    let disposed = false;

    async function poll(initial: boolean) {
      const current = task.current;
      if (!initial && !current?.polling) return;

      let rsp;
      try {
        rsp = await getImageTransferStatus();
      } catch {
        return;
      }
      if (disposed || rsp.code !== 0) return;

      const { status, file, percentage, downloadedBytes: loaded, totalBytes: total } = rsp.data;

      // A transfer that was already running when the page opened - started
      // from another tab, or before a reload - is picked up and followed.
      if (initial) {
        if (!['in_progress', 'paused'].includes(status) || task.current) return;
        const picked = start({
          kind: 'download',
          name: urlBaseName(file ?? ''),
          cancellable: true
        });
        picked.polling = true;
      }

      const followed = task.current;
      if (!followed || !followed.polling) return;

      if (status === 'in_progress' || status === 'paused') {
        followed.seenRunning = true;
        setTransfer((value) =>
          value?.id === followed.id
            ? {
                ...value,
                name: value.name || urlBaseName(file ?? ''),
                percentage: percentage ?? '',
                paused: status === 'paused',
                loaded: loaded ?? 0,
                total: total ?? 0
              }
            : value
        );
        return;
      }

      // A browser upload ends with its own request, which carries the server's
      // answer; the status endpoint only reports its progress (and goes back
      // to idle when an upload succeeds).
      if (!followed.seenRunning || followed.controller) return;
      if (status === 'success') {
        finish(followed.id, 'success');
      } else if (status === 'checksum_failed') {
        finish(followed.id, 'failed', t('storage.add.checksumFailed'));
      } else if (status === 'failed') {
        finish(followed.id, 'failed');
      } else if (status === 'idle') {
        // Cancelled on the server, possibly from another tab.
        finish(followed.id, 'cancelled');
      }
    }

    void poll(active);
    const timer = setInterval(() => void poll(false), 2500);
    return () => {
      disposed = true;
      clearInterval(timer);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [active, followingDownload]);

  async function cancel() {
    const current = task.current;
    if (!current || current.finished) return;

    // A browser upload ends with its request; aborting it is the cancel.
    if (current.controller) {
      current.controller.abort();
      try {
        if (current.uploadId) await downloadApi.cancelImageUpload(current.uploadId);
      } catch {
        onError(t('download.cancelFailed'));
        return;
      }
      finish(current.id, 'cancelled');
      return;
    }

    try {
      const rsp = await downloadApi.cancelImageDownload();
      if (rsp.code !== 0) {
        onError(rsp.msg || t('download.cancelFailed'));
        return;
      }
    } catch {
      onError(t('download.cancelFailed'));
      return;
    }

    finish(current.id, 'cancelled');
  }

  async function togglePause() {
    if (!transfer || transfer.kind !== 'download' || transfer.state !== 'running') return;
    try {
      const rsp = await (transfer.paused
        ? downloadApi.resumeImageDownload()
        : downloadApi.pauseImageDownload());
      if (rsp.code !== 0) {
        onError(t('storage.failed'));
        return;
      }
      setTransfer((value) => (value ? { ...value, paused: !value.paused } : value));
    } catch {
      onError(t('storage.failed'));
    }
  }

  // A failed transfer stays on screen until the user closes it.
  function dismiss() {
    setTransfer((value) => (value?.state === 'failed' ? null : value));
  }

  async function uploadImage(file: File | null) {
    if (!file || !localWritable || (task.current && !task.current.finished)) return;

    const name = file.name.toLowerCase();
    if (!name.endsWith('.iso') && !name.endsWith('.img')) {
      onError(t('storage.add.onlyImages'));
      return;
    }

    const sum = checksum.trim();
    if (sum && !sha256Pattern.test(sum)) {
      onError(t('download.invalidSHA256'));
      return;
    }

    // Close the picker before starting the transfer so progress and errors are
    // visible in the media list. Invalid files and checksums stay in the dialog.
    closeUpload();

    const controller = new AbortController();
    const current = start({ kind: 'upload', name: file.name, cancellable: true });
    current.controller = controller;
    const uploadId = Array.from(crypto.getRandomValues(new Uint8Array(16)), (byte) =>
      byte.toString(16).padStart(2, '0')
    ).join('');
    current.uploadId = uploadId;
    const chunkSize = 8 * 1024 * 1024;
    try {
      const totalChunks = Math.ceil(file.size / chunkSize);
      if (totalChunks === 0) throw new Error(t('storage.add.uploadFailed'));
      for (let chunkIndex = 0; chunkIndex < totalChunks; chunkIndex++) {
        if (controller.signal.aborted) throw new Error('cancelled');
        const offset = chunkIndex * chunkSize;
        const chunk = file.slice(offset, Math.min(offset + chunkSize, file.size));
        const rsp = await downloadApi.uploadImage(
          chunk,
          {
            uploadId,
            filename: file.name,
            chunkSize,
            chunkIndex,
            totalChunks,
            sha256: sum
          },
          {
            signal: controller.signal,
            onProgress: (bytes) =>
              setTransfer((value) =>
                value?.id === current.id
                  ? {
                      ...value,
                      loaded: offset + bytes,
                      total: file.size,
                      percentage: `${Math.min(100, ((offset + bytes) / file.size) * 100).toFixed(1)}%`
                    }
                  : value
              )
          }
        );
        if (rsp.code !== 0)
          throw new Error(
            rsp.msg === 'sha256 mismatch'
              ? t('storage.add.checksumFailed')
              : rsp.msg === 'insufficient storage'
                ? t('download.insufficientStorage')
                : t('storage.add.uploadFailed')
          );
      }
      finish(current.id, 'success');
    } catch (error) {
      try {
        await downloadApi.cancelImageUpload(uploadId);
      } catch {
        /* the session expires if the device is unreachable */
      }
      finish(
        current.id,
        controller.signal.aborted ? 'cancelled' : 'failed',
        error instanceof Error ? error.message : t('storage.add.uploadFailed')
      );
    }
  }

  async function downloadFromUrl() {
    if (!localWritable || (task.current && !task.current.finished)) return;
    let values: { url: string; sha256?: string };
    try {
      values = await urlForm.validateFields();
    } catch {
      // The form shows what is wrong next to the field.
      return;
    }

    const current = start({ kind: 'download', name: urlBaseName(values.url), cancellable: false });

    let rsp;
    try {
      rsp = await downloadApi.downloadImage(values.url, values.sha256 ?? '');
    } catch {
      finish(current.id, 'cancelled');
      onError(t('storage.add.downloadFailed'));
      return;
    }
    if (rsp.code !== 0) {
      finish(current.id, 'cancelled');
      onError(rsp.msg || t('storage.add.downloadFailed'));
      return;
    }

    // The server answers once the transfer is registered, so from here on the
    // status endpoint reports this one.
    current.seenRunning = true;
    current.polling = true;
    setTransfer((value) => (value?.id === current.id ? { ...value, cancellable: true } : value));

    setIsUrlOpen(false);
    urlForm.resetFields();
  }

  function closeUpload() {
    setIsUploadOpen(false);
    setPendingFile(null);
    setChecksum('');
    setIsDragging(false);
  }

  // The dialog's area takes a click (choose a file) and a drop, so an image can
  // come from a file manager window as well.
  const dropProps = {
    onDragOver: (event: DragEvent) => {
      event.preventDefault();
      if (localWritable) setIsDragging(true);
    },
    onDragLeave: () => setIsDragging(false),
    onDrop: (event: DragEvent) => {
      event.preventDefault();
      setIsDragging(false);
      if (!localWritable) return;
      setPendingFile(event.dataTransfer?.files?.[0] ?? null);
    }
  };

  // Wait for local write access and for the previous transfer to finish.
  const addDisabled = !localWritable || transfer?.state === 'running';

  const overlays = (
    <>
      <input
        ref={imageInputRef}
        type="file"
        accept=".iso,.img"
        className="hidden"
        onChange={(event) => {
          setPendingFile(event.target.files?.[0] ?? null);
          if (imageInputRef.current) imageInputRef.current.value = '';
        }}
      />

      <Modal
        title={t('storage.add.title')}
        open={isUploadOpen}
        okText={t('storage.yes')}
        cancelText={t('storage.no')}
        okButtonProps={{
          disabled: !pendingFile || !localWritable || transfer?.state === 'running'
        }}
        onOk={() => void uploadImage(pendingFile)}
        onCancel={closeUpload}
      >
        <div className="flex flex-col space-y-3">
          {/* Clicking chooses a file, dropping one works the same way. */}
          <div
            {...dropProps}
            onClick={() => imageInputRef.current?.click()}
            className={clsx(
              'flex min-h-28 cursor-pointer flex-col items-center justify-center gap-2 rounded-xl border border-dashed px-4 py-5 text-center transition-colors',
              isDragging
                ? 'border-blue-400 bg-blue-500/10'
                : 'border-neutral-700 bg-neutral-800/30 hover:border-blue-500/60'
            )}
          >
            <UploadIcon size={24} className={isDragging ? 'text-blue-400' : 'text-neutral-500'} />
            {pendingFile ? (
              <span
                className="max-w-full truncate text-sm text-neutral-200"
                title={pendingFile.name}
              >
                {pendingFile.name} · {formatSize(pendingFile.size)}
              </span>
            ) : (
              <span className="text-sm text-neutral-400">{t('storage.add.dropHint')}</span>
            )}
          </div>

          <div className="flex flex-col space-y-1">
            <span className="text-xs text-neutral-500">{t('download.sha256')}</span>
            <Input
              allowClear
              placeholder={t('download.sha256Placeholder')}
              value={checksum}
              onChange={(event) => setChecksum(event.target.value)}
            />
          </div>
        </div>
      </Modal>

      <Modal
        title={t('storage.add.fromUrl')}
        open={isUrlOpen}
        okText={t('storage.yes')}
        cancelText={t('storage.no')}
        onOk={() => void downloadFromUrl()}
        onCancel={() => setIsUrlOpen(false)}
      >
        <Form form={urlForm} layout="vertical">
          <Form.Item
            name="url"
            label={t('storage.add.url')}
            rules={[{ required: true, message: t('storage.add.urlRequired') }]}
          >
            <Input placeholder="https://example.com/image.iso" />
          </Form.Item>
          <Form.Item
            name="sha256"
            label={t('download.sha256')}
            rules={[
              {
                validator: (_, value) =>
                  !value || sha256Pattern.test(value)
                    ? Promise.resolve()
                    : Promise.reject(new Error(t('download.invalidSHA256')))
              }
            ]}
          >
            <Input placeholder={t('download.sha256Placeholder')} />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );

  return {
    openUpload: () => setIsUploadOpen(true),
    openUrl: () => setIsUrlOpen(true),
    addDisabled,
    transfer,
    cancelTransfer: cancel,
    togglePause,
    dismissTransfer: dismiss,
    overlays
  };
}
