import { ReactNode, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Alert,
  App,
  Button,
  ConfigProvider,
  Modal,
  notification,
  Switch,
  theme,
  Tooltip
} from 'antd';
import clsx from 'clsx';
import {
  CheckIcon,
  DiscIcon,
  DownloadIcon,
  HardDriveIcon,
  LinkIcon,
  LoaderCircleIcon,
  PlugIcon,
  Trash2Icon,
  UnplugIcon,
  UploadIcon,
  XIcon
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/media.ts';
import { formatSize } from '@/lib/format.ts';
import { MediaIcon } from '@/lib/media-icons.tsx';
import { useMediaMount } from '@/hooks/useMediaMount.ts';
import { useProtectionChallengeGate } from '@/hooks/useProtectionChallengeGate.tsx';
import { ButtonBase } from '@/components/ui/button-base.tsx';

import { useAddMedia } from './use-add-media.tsx';
import { useFileBrowser, type FileUpload } from './use-file-browser.tsx';

const statePollInterval = 5000;

// The failed media job the user closed. The server keeps a failed job's state
// until the next job starts, so without this it would come back on every visit.
const dismissedJobKey = 'nanokvm-storage-dismissed-job';

function readDismissedJob(): string {
  try {
    return localStorage.getItem(dismissedJobKey) ?? '';
  } catch {
    return '';
  }
}

function jobKey(job: api.MediaStatus): string {
  return `${job.name}\n${job.message}`;
}

function baseName(name: string): string {
  return name.split('/').filter(Boolean).pop() ?? name;
}

// A titled block in the medium list: the emulated disk is one thing, the image
// library is another, and mixing them in one list made both harder to read.
const MediaGroup = ({
  title,
  action,
  children
}: {
  title: string;
  action?: ReactNode;
  children: ReactNode;
}) => (
  <div className="flex flex-col space-y-1">
    <div className="flex h-6 items-center justify-between px-2">
      <span className="text-xs font-medium uppercase tracking-wide text-neutral-300">{title}</span>
      {action}
    </div>
    {children}
  </div>
);

type StorageViewProps = {
  // 'mount' is the disk-mount manager; 'files' is the shared-files browser.
  // The desktop menu opens each inside its own modal.
  view: 'mount' | 'files';
  active: boolean;
  // Extra slot on the header's right side (the modals put their close button
  // there).
  headerExtra?: ReactNode;
};

const StorageContent = ({ view, active, headerExtra }: StorageViewProps) => {
  const { t } = useTranslation();
  const { token } = theme.useToken();
  const [notificationApi, contextHolder] = notification.useNotification();
  const { modal } = App.useApp();
  const { runWithChallenge, modal: protectionModal } = useProtectionChallengeGate();

  const [media, setMedia] = useState<api.MediaItem[]>([]);
  const [state, setState] = useState<api.MediaStateInfo>({
    sharedDiskInitialized: false,
    state: 'none',
    hidOnly: false,
    open: [],
    cdrom: false,
    readOnly: false,
    localWritable: false
  });
  // The mode the next mount uses. It follows what is attached, so the control
  // shows the mode the target machine is looking at rather than a stale choice.
  const [mountAsCdrom, setMountAsCdrom] = useState(false);
  const [readOnly, setReadOnly] = useState(true);
  const [isLoading, setIsLoading] = useState(true);
  const [storageSignal, setStorageSignal] = useState(0);
  const [job, setJob] = useState<api.MediaStatus | null>(null);
  const [browsePath, setBrowsePath] = useState('/');
  const [confirm, setConfirm] = useState<{
    title: string;
    description: string;
    ok: () => void | Promise<void>;
  } | null>(null);
  const lastStatus = useRef<api.MediaStatus['status']>('idle');
  const [dismissedJob, setDismissedJob] = useState(readDismissedJob);
  // Names of file uploads the user cancelled: the server records the aborted
  // request as a failed job, which is not news to the user who stopped it.
  const cancelledUploads = useRef(new Set<string>());
  const activeRef = useRef(active);
  activeRef.current = active;

  // Both callbacks are handed to children that poll with them in their
  // dependency lists: a new identity on every render would tear the intervals
  // down and re-list the medium every few seconds.
  const notifyError = useCallback(
    (message: string, description?: string) => {
      notificationApi.error({
        message: message || t('storage.failed'),
        description,
        duration: 10
      });
    },
    [notificationApi, t]
  );

  // This runs on a timer, so a request that fails (the device rebooting, the
  // network dropping) is left to the next round instead of raising a
  // notification every few seconds.
  const refresh = useCallback(async () => {
    if (!activeRef.current) return;
    try {
      // State recovery recreates /data, so list files from the fresh mount.
      const stateRsp = await api.getMediaState();
      if (!activeRef.current) return;
      if (stateRsp.code === 0) {
        setState(stateRsp.data);
      } else {
        setState((previous) => ({ ...previous, localWritable: false }));
      }

      const rsp = await api.getMedia();
      if (!activeRef.current) return;
      if (rsp.code === 0) {
        setMedia(rsp.data?.media ?? []);
      }
    } catch {
      if (activeRef.current) {
        setState((previous) => ({ ...previous, localWritable: false }));
      }
      // Retried by the poll.
    }
  }, []);

  const refreshMedia = useCallback(async () => {
    await refresh();
  }, [refresh]);

  useEffect(() => {
    if (!active) return;
    void refresh().finally(() => setIsLoading(false));

    const timer = setInterval(() => void refresh(), statePollInterval);
    return () => clearInterval(timer);
  }, [active, refresh]);

  useEffect(() => {
    if (state.state === 'image' || state.state === 'partition') {
      setMountAsCdrom(state.cdrom);
    }
  }, [state.state, state.cdrom]);

  useEffect(() => {
    if (state.state === 'image') setReadOnly(state.readOnly || state.cdrom);
  }, [state.state, state.readOnly, state.cdrom]);

  // The Go storage API refreshes its read-only loop mount on demand.
  // Poll only while this page is visible; no extra input WebSocket is needed.
  useEffect(() => {
    if (!active || state.state !== 'partition' || view !== 'files') return;
    const timer = setInterval(() => {
      if (!document.hidden) setStorageSignal((value) => value + 1);
    }, 3000);
    return () => clearInterval(timer);
  }, [active, state.state, view]);

  // Poll the media job while one is running: creating a medium writes
  // gigabytes to the card, so the page reports it like any other transfer.
  useEffect(() => {
    if (!active) return;
    let cancelled = false;

    async function poll() {
      let rsp;
      try {
        rsp = await api.getMediaStatus();
      } catch {
        return;
      }
      if (cancelled || rsp.code !== 0) return;

      const status: api.MediaStatus = rsp.data;

      if (status.status === 'success' && lastStatus.current === 'in_progress') {
        await refresh();
      }
      lastStatus.current = status.status;
      if (status.status === 'in_progress') {
        cancelledUploads.current.delete(baseName(status.name));
      }

      // A finished job leaves its last state behind for the next caller to
      // read, so only a running or failed one belongs on screen.
      setJob(status.status === 'in_progress' || status.status === 'failed' ? status : null);
    }

    void poll();
    const timer = setInterval(() => void poll(), 2500);
    return () => {
      cancelled = true;
      clearInterval(timer);
    };
  }, [active, refresh]);

  // Operations the target machine can refuse are confirmed here rather than in
  // the menu panel: this page is a full window, so a dialog cannot be
  // swallowed by a popover closing.
  const { mount, eject, busy } = useMediaMount({
    onError: notifyError,
    onNeedEnableDisk: () => notifyError(t('storage.enableDisk.menuHint')),
    onNeedForce: (retry) =>
      setConfirm({
        title: t('storage.force.title'),
        description: t('storage.force.desc'),
        ok: retry
      }),
    onChanged: refreshMedia
  });

  // Attachment controls the USB hint; actual local access controls writes.
  // Host-side ejects may need a fresh writable mount before writes resume.
  const dataOwned = state.state === 'partition';

  const addMedia = useAddMedia({
    active: active && view === 'mount',
    localWritable: state.localWritable,
    onChanged: refreshMedia,
    onError: notifyError
  });

  const volumeMedia = useMemo(() => media.filter((item) => item.type === 'partition'), [media]);
  const images = useMemo(() => media.filter((item) => item.type === 'image'), [media]);
  const sharedPartition = volumeMedia[0] ?? null;

  // A download the browser starts on its own saves whatever comes back, so a
  // refused request ends up as a file named after the URL. The status is asked
  // for first - one byte of the image, aborted as soon as the headers arrive -
  // and a refusal is reported the way the rest of the page reports errors.
  async function downloadImage(item: api.MediaItem) {
    const url = api.mediaImageDownloadUrl(item.rel);
    const controller = new AbortController();

    let response: Response;
    try {
      response = await fetch(url, {
        headers: { Range: 'bytes=0-0' },
        credentials: api.fetchCredentials(),
        signal: controller.signal
      });
    } catch {
      notifyError(t('storage.failed'));
      return;
    }

    if (!response.ok || response.headers.get('content-type')?.includes('application/json')) {
      const data = await response.json().catch(() => null);
      controller.abort();
      notifyError(data?.msg ?? `${response.status} ${response.statusText}`);
      return;
    }

    controller.abort();

    const link = document.createElement('a');
    link.href = url;
    link.download = '';
    document.body.appendChild(link);
    link.click();
    link.remove();
  }

  function confirmDelete(item: api.MediaItem) {
    modal.confirm({
      title: t('storage.deleteTitle'),
      content: t('storage.deleteConfirm', { name: item.name }),
      okText: t('storage.yes'),
      cancelText: t('storage.no'),
      okButtonProps: { danger: true },
      onOk: () =>
        runWithChallenge('delete_image', 'deleteImage', async (challenge) => {
          let rsp;
          try {
            rsp = await api.deleteMedia({ type: item.type, rel: item.rel, challenge });
          } catch {
            notifyError(t('storage.failed'));
            return;
          }
          if (rsp.code !== 0) {
            if (rsp.code !== -3) notifyError(rsp.msg);
            return rsp;
          }
          await refresh();
          return rsp;
        })
    });
  }

  const browser = useFileBrowser({
    active: active && view === 'files',
    // The file browser always shows the shared partition; the mount view never
    // browses, and passing null skips its listing poll.
    media: view === 'files' && state.sharedDiskInitialized ? sharedPartition : null,
    reason: state.sharedDiskInitialized ? '' : 'notInitialized',
    localWritable: state.localWritable,
    path: browsePath,
    onPathChange: setBrowsePath,
    changeSignal: storageSignal,
    onMediaChanged: refreshMedia,
    onError: notifyError
  });

  useEffect(() => {
    if (state.localStorageError) notifyError(state.localStorageError);
  }, [state.localStorageError, notifyError]);

  // Every transfer the page runs is shown at the top in one shape: the media
  // job (a file written into a medium) and an image arriving from the browser
  // or a URL. They used to be two differently styled hints in two places.
  function renderTransfer(transfer: {
    key: string;
    label: string;
    percentage: string;
    loaded: number;
    total: number;
    state: 'running' | 'success' | 'failed';
    message?: string;
    onCancel?: () => void;
    onPause?: () => void;
    paused?: boolean;
    onClose?: () => void;
  }) {
    const failed = transfer.state === 'failed';
    return (
      <Alert
        key={transfer.key}
        className="mb-3"
        type={failed ? 'error' : transfer.state === 'success' ? 'success' : 'info'}
        showIcon
        icon={
          failed ? undefined : transfer.state === 'success' ? (
            <CheckIcon size={14} />
          ) : (
            <LoaderCircleIcon size={14} className={transfer.paused ? '' : 'animate-spin'} />
          )
        }
        closable={failed && !!transfer.onClose}
        onClose={transfer.onClose}
        message={
          failed ? (
            transfer.message || t('storage.failed')
          ) : (
            <span className="flex items-center space-x-2">
              <span className="min-w-0 truncate" title={transfer.label}>
                {transfer.label}
              </span>
              {transfer.state === 'running' && transfer.total > 0 && (
                <span className="shrink-0 text-neutral-400">
                  {transfer.percentage} · {formatSize(transfer.loaded)} /{' '}
                  {formatSize(transfer.total)}
                </span>
              )}
            </span>
          )
        }
        action={
          <span className="flex items-center gap-1">
            {transfer.onPause && (
              <Button type="text" size="small" onClick={transfer.onPause}>
                {transfer.paused ? t('download.resume') : t('download.pause')}
              </Button>
            )}
            {transfer.onCancel && (
              <Button
                type="text"
                size="small"
                icon={<XIcon size={12} />}
                title={t('download.cancel')}
                onClick={transfer.onCancel}
              />
            )}
          </span>
        }
      />
    );
  }

  function dismissJob(key: string) {
    setDismissedJob(key);
    try {
      localStorage.setItem(dismissedJobKey, key);
    } catch {
      // Remembered for this visit only.
    }
  }

  // A file upload from the browser is the media job the server reports, so
  // the job carries that upload's cancel button. Uploads still waiting for
  // the job slot are shown on their own.
  const jobUpload =
    job?.status === 'in_progress'
      ? browser.uploads.find((upload) => upload.name === baseName(job.name))
      : undefined;
  const waitingUploads = browser.uploads.filter((upload) => upload !== jobUpload);
  const showJob =
    !!job &&
    !(
      job.status === 'failed' &&
      (dismissedJob === jobKey(job) || cancelledUploads.current.has(baseName(job.name)))
    );

  function cancelUpload(upload: FileUpload) {
    cancelledUploads.current.add(upload.name);
    upload.cancel();
  }

  const addTransfer = addMedia.transfer;
  const addTransferLabel = !addTransfer
    ? ''
    : addTransfer.state === 'success'
      ? t('storage.add.done', { name: addTransfer.name })
      : addTransfer.kind === 'upload'
        ? t('storage.add.uploading', { name: addTransfer.name })
        : t('storage.add.downloading', { name: addTransfer.name });

  // The read-only switch sits on the images group's title row because it only
  // affects image mounts.
  const mediaGroups = (
    <MediaGroup
      title={t('storage.groups.images')}
      action={
        <div className="flex items-center gap-2">
          <span className="text-xs text-neutral-300">{t('image.readOnly')}</span>
          <Switch
            size="small"
            aria-label={t('image.readOnly')}
            checked={mountAsCdrom || readOnly}
            disabled={
              mountAsCdrom ||
              state.state === 'image' ||
              state.state === 'unsupported' ||
              busy !== ''
            }
            onChange={setReadOnly}
          />
        </div>
      }
    >
      {images.length === 0 ? (
        <div className="rounded-lg border border-dashed border-neutral-600 px-3 py-3 text-center text-xs text-neutral-200">
          {t('storage.empty')}
        </div>
      ) : (
        renderRows(images)
      )}
    </MediaGroup>
  );

  return (
    <div
      className={clsx(
        'box-border w-full text-neutral-200',
        view === 'files'
          ? 'flex h-full min-h-0 flex-col overflow-hidden p-4 sm:p-5'
          : 'overflow-auto p-4 sm:p-5'
      )}
      style={{ backgroundColor: token.colorBgElevated }}
    >
      {contextHolder}
      {protectionModal}
      {addMedia.overlays}
      {browser.overlays}

      <div
        className={clsx(
          'mx-auto flex min-h-0 flex-col',
          view === 'files' ? 'h-full w-full max-w-6xl' : 'max-w-xl'
        )}
      >
        <div className="flex shrink-0 flex-wrap items-center justify-between gap-3 pb-5">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center text-neutral-300">
              <HardDriveIcon size={20} />
            </div>
            <span className="text-lg font-semibold text-neutral-100">
              {view === 'mount' ? t('storage.cards.mount') : t('storage.cards.share')}
            </span>
          </div>

          <div className="flex items-center gap-2 text-sm">
            {state.state === 'unsupported' && (
              <span className="rounded-full border border-solid border-neutral-600 px-3 py-1 text-xs text-neutral-300">
                {state.hidOnly ? t('storage.hidOnly') : t('storage.disabled')}
              </span>
            )}
            {headerExtra}
          </div>
        </div>

        {job &&
          showJob &&
          renderTransfer({
            key: 'job',
            label: t('storage.job.running', { name: job.name }),
            percentage: job.percentage,
            loaded: job.loaded,
            total: job.total,
            state: job.status === 'failed' ? 'failed' : 'running',
            message: job.message,
            onCancel: jobUpload ? () => cancelUpload(jobUpload) : undefined,
            onClose: () => dismissJob(jobKey(job))
          })}
        {waitingUploads.map((upload) =>
          renderTransfer({
            key: `upload-${upload.id}`,
            label: t('storage.add.uploading', { name: upload.name }),
            percentage:
              upload.total > 0 ? `${((upload.loaded / upload.total) * 100).toFixed(1)}%` : '',
            loaded: upload.loaded,
            total: upload.total,
            state: 'running',
            onCancel: () => cancelUpload(upload)
          })
        )}
        {addTransfer &&
          renderTransfer({
            key: `add-${addTransfer.id}`,
            label: addTransferLabel,
            percentage: addTransfer.percentage,
            loaded: addTransfer.loaded,
            total: addTransfer.total,
            state: addTransfer.state,
            message: addTransfer.message,
            // Cancelling a finished transfer only earns an error from the
            // server, so the button goes away once it ends.
            onCancel:
              addTransfer.cancellable && addTransfer.state === 'running'
                ? () => void addMedia.cancelTransfer()
                : undefined,
            onPause:
              addTransfer.kind === 'download' && addTransfer.state === 'running'
                ? () => void addMedia.togglePause()
                : undefined,
            paused: addTransfer.paused,
            onClose: addMedia.dismissTransfer
          })}

        <Modal
          title={confirm?.title}
          open={!!confirm}
          okText={t('storage.yes')}
          cancelText={t('storage.no')}
          onOk={() => {
            const pending = confirm;
            setConfirm(null);
            void pending?.ok();
          }}
          onCancel={() => setConfirm(null)}
        >
          <p>{confirm?.description}</p>
        </Modal>

        {view === 'mount' ? (
          <div className="flex flex-col gap-4">
            <div className="shrink-0">
              <div className="flex flex-col gap-3">
                <div className="h-px bg-neutral-700" />

                <div className="flex flex-col gap-2">
                  <span className="text-xs font-medium text-neutral-300">
                    {t('storage.mountMode')}
                  </span>
                  <div className="grid grid-cols-2 gap-1">
                    {[
                      { value: 'disk', icon: HardDriveIcon, label: t('storage.mountModeDisk') },
                      { value: 'cdrom', icon: DiscIcon, label: t('storage.mountModeCdrom') }
                    ].map((modeOption) => {
                      const cdrom = modeOption.value === 'cdrom';
                      const isSelected = mountAsCdrom === cdrom;
                      const disabled =
                        busy !== '' || state.state === 'image' || state.state === 'unsupported';
                      return (
                        <ButtonBase
                          key={modeOption.value}
                          disabled={disabled}
                          onClick={() => {
                            setMountAsCdrom(cdrom);
                            if (cdrom) setReadOnly(true);
                          }}
                          className={clsx(
                            'relative flex min-h-10 min-w-0 items-center justify-center gap-2 rounded-md px-2 py-2 text-sm transition-colors',
                            isSelected
                              ? 'bg-blue-600 text-white'
                              : 'bg-neutral-900 text-neutral-300 hover:bg-neutral-800 hover:text-white'
                          )}
                        >
                          <modeOption.icon size={15} className="shrink-0" />
                          <span className="min-w-0 whitespace-normal break-words text-center leading-tight">
                            {modeOption.label}
                          </span>
                        </ButtonBase>
                      );
                    })}
                  </div>
                </div>

                <div className="h-px bg-neutral-700" />

                <div className="flex flex-col gap-2">
                  <span className="text-xs font-medium text-neutral-300">
                    {t('storage.add.title')}
                  </span>
                  <div className="grid grid-cols-2 gap-1">
                    {[
                      {
                        icon: UploadIcon,
                        label: t('storage.add.upload'),
                        open: addMedia.openUpload
                      },
                      { icon: LinkIcon, label: t('storage.add.fromUrl'), open: addMedia.openUrl }
                    ].map((action) => (
                      <Tooltip
                        key={action.label}
                        title={dataOwned ? t('storage.add.blocked') : undefined}
                      >
                        <span className="block min-w-0">
                          <ButtonBase
                            disabled={addMedia.addDisabled}
                            onClick={action.open}
                            className="flex min-h-11 w-full min-w-0 items-center justify-center gap-2 rounded-md bg-neutral-900 px-2 py-2 text-center text-sm text-neutral-300 transition-colors enabled:hover:bg-neutral-800 enabled:hover:text-white disabled:text-neutral-400 disabled:opacity-100"
                          >
                            <action.icon size={16} className="shrink-0" />
                            <span className="min-w-0 whitespace-normal break-words leading-snug">
                              {action.label}
                            </span>
                          </ButtonBase>
                        </span>
                      </Tooltip>
                    ))}
                  </div>
                </div>

                <div className="h-px bg-neutral-700" />
              </div>
            </div>

            <div className="flex max-h-[55vh] flex-col gap-4 overflow-auto">
              {isLoading ? (
                <div className="flex items-center justify-center py-6 text-neutral-400">
                  <LoaderCircleIcon size={18} className="animate-spin" />
                </div>
              ) : (
                mediaGroups
              )}
            </div>
          </div>
        ) : (
          <div className="flex min-h-0 min-w-0 flex-1 flex-col">
            {isLoading ? (
              <div className="flex flex-1 items-center justify-center text-neutral-400">
                <LoaderCircleIcon size={18} className="animate-spin" />
              </div>
            ) : (
              browser.content
            )}
          </div>
        )}
      </div>
    </div>
  );

  function renderRows(items: api.MediaItem[]) {
    return (
      <div className="space-y-1.5">
        {items.map((item) => {
          const key = `${item.type}:${item.rel}`;
          const isBusy = busy === key;

          return (
            <div
              key={key}
              className="group flex items-center gap-3 rounded-lg border border-solid border-neutral-700 px-3 py-2.5 text-sm transition-colors"
            >
              <div
                className={clsx(
                  'flex h-9 w-9 shrink-0 items-center justify-center',
                  item.attached ? 'text-blue-400' : 'text-neutral-300'
                )}
              >
                <MediaIcon item={item} size={17} />
              </div>

              <div className="flex min-w-0 flex-1 flex-col">
                <div className="flex items-center gap-1.5">
                  <span className="truncate text-neutral-100" title={item.name}>
                    {item.type === 'partition' ? t('storage.typePartition') : item.name}
                  </span>
                  {item.attached && (
                    <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-blue-400" />
                  )}
                </div>
                <span className="text-xs text-neutral-300">
                  {item.type === 'partition' ? t('storage.volumeLabel') : formatSize(item.size)}
                  {item.fs && item.fs !== 'unknown' ? ` · ${item.fs}` : ''}
                </span>
              </div>

              <div
                className="flex shrink-0 items-center space-x-1"
                onClick={(event) => event.stopPropagation()}
              >
                {item.attached ? (
                  <Tooltip title={t('storage.eject')}>
                    <Button
                      type="default"
                      size="small"
                      icon={<UnplugIcon size={14} />}
                      onClick={() => void eject()}
                    />
                  </Tooltip>
                ) : (
                  <Tooltip
                    title={
                      state.state === 'image'
                        ? t('storage.add.blocked')
                        : mountAsCdrom && !api.isOpticalImage(item)
                          ? t('storage.add.cdromOnlyIso')
                          : t('storage.mount')
                    }
                  >
                    <Button
                      type="default"
                      size="small"
                      loading={isBusy}
                      disabled={
                        busy !== '' ||
                        state.state === 'image' ||
                        (mountAsCdrom && !api.isOpticalImage(item))
                      }
                      icon={<PlugIcon size={14} />}
                      onClick={() => void mount(item, mountAsCdrom, { readOnly })}
                    />
                  </Tooltip>
                )}

                {item.type === 'image' && (
                  <>
                    <Tooltip
                      title={item.attached ? t('storage.add.blocked') : t('storage.downloadImage')}
                    >
                      <Button
                        type="default"
                        size="small"
                        disabled={item.attached}
                        icon={<DownloadIcon size={14} />}
                        onClick={() => void downloadImage(item)}
                      />
                    </Tooltip>
                    <Tooltip
                      title={
                        dataOwned || item.attached ? t('storage.add.blocked') : t('storage.delete')
                      }
                    >
                      <Button
                        type="default"
                        size="small"
                        danger
                        disabled={!state.localWritable || item.attached}
                        icon={<Trash2Icon size={14} />}
                        onClick={() => confirmDelete(item)}
                      />
                    </Tooltip>
                  </>
                )}
              </div>
            </div>
          );
        })}
      </div>
    );
  }
};

// One surface, two faces: the desktop menu mounts this inside the disk-mount
// modal ('mount') and the file-manager modal ('files').
export const StorageView = ({ view, active, headerExtra }: StorageViewProps) => {
  const { token } = theme.useToken();

  return (
    <ConfigProvider theme={{ token: { colorBgContainer: token.colorBgElevated } }}>
      <App className={view === 'files' ? 'h-full min-h-0' : undefined}>
        <StorageContent view={view} active={active} headerExtra={headerExtra} />
      </App>
    </ConfigProvider>
  );
};
