import { useEffect, useState } from 'react';
import { Modal, Switch, Tooltip } from 'antd';
import clsx from 'clsx';
import { CheckIcon, DiscIcon, HardDriveIcon, LoaderCircleIcon, RefreshCwIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/storage';
import { getVirtualDevice, refreshVirtualDevice } from '@/api/virtual-device.ts';
import { ModalHeader } from '@/components/modal-header.tsx';
import { ButtonBase } from '@/components/ui/button-base.tsx';

import { Images } from './images';

export function MounteImage() {
  const { t } = useTranslation();

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isMounted, setIsMounted] = useState(false);
  const [mode, setMode] = useState('mass-storage');
  const [readOnly, setReadOnly] = useState(true);
  const [isDiskEnabled, setIsDiskEnabled] = useState(false);
  const [isSdCardExist, setIsSdCardExist] = useState(false);
  const [refreshVersion, setRefreshVersion] = useState(0);
  const [isRefresh, setIsRefresh] = useState(false);

  const modes = [
    { value: 'mass-storage', icon: HardDriveIcon, label: t('image.massStorage') },
    { value: 'cd-rom', icon: DiscIcon, label: t('image.cdRom') }
  ];

  useEffect(() => {
    getMountedImage();
    getVirtualDisk();
  }, []);

  async function getMountedImage() {
    const rsp = await api.getMountedImage();
    if (rsp.code !== 0) {
      console.log(rsp.msg);
      return;
    }

    if (!rsp?.data?.file) {
      return;
    }

    setIsMounted(true);
    setMode(rsp.data?.cdrom ? 'cd-rom' : 'mass-storage');
    setReadOnly(!!rsp.data?.readOnly);
  }

  function updateMode(value: string) {
    setMode(value);
    if (value === 'cd-rom' && !readOnly) {
      setReadOnly(true);
    }
  }

  function updateReadOnly(value: boolean) {
    if (mode === 'cd-rom' && !value) {
      return;
    }
    setReadOnly(value);
  }

  async function getVirtualDisk() {
    const rsp = await getVirtualDevice();
    if (rsp.code !== 0 || !rsp.data) {
      return;
    }

    setIsDiskEnabled(rsp.data.isDiskEnabled);
    setIsSdCardExist(rsp.data.isSdCardExist);
  }

  async function refresh() {
    if (isRefresh) return;
    setIsRefresh(true);

    try {
      const rsp = await refreshVirtualDevice('disk');
      if (rsp.code === 0) setRefreshVersion((version) => version + 1);
    } catch (err) {
      console.log(err);
    } finally {
      setIsRefresh(false);
    }
  }

  const showRefresh = isSdCardExist && !isDiskEnabled && !isMounted;

  return (
    <>
      <div
        onClick={() => {
          setIsModalOpen(true);
          getVirtualDisk();
        }}
        className="flex h-[30px] cursor-pointer select-none items-center space-x-2 rounded pl-3 pr-6 text-neutral-300 hover:bg-neutral-700/70"
      >
        <HardDriveIcon size={18} />
        <span
          className={clsx(
            'select-none text-sm',
            isMounted ? 'text-blue-500' : 'text-neutral-300 hover:text-white'
          )}
        >
          {t('image.mount')}
        </span>
      </div>

      <Modal
        open={isModalOpen}
        forceRender
        width={560}
        onCancel={() => setIsModalOpen(false)}
        title={
          <ModalHeader
            icon={<HardDriveIcon size={18} />}
            title={t('image.mount')}
            status={
              isMounted ? (
                <span className="mt-1 flex items-center gap-1.5 text-xs font-normal text-blue-400">
                  <span className="h-1.5 w-1.5 rounded-full bg-blue-400" />
                  {t('image.mounted')}
                </span>
              ) : undefined
            }
            extra={
              showRefresh ? (
                <Tooltip title={t('image.refresh')}>
                  {isRefresh ? (
                    <span className="flex h-8 w-8 items-center justify-center">
                      <LoaderCircleIcon className="animate-spin text-blue-500" size={16} />
                    </span>
                  ) : (
                    <ButtonBase
                      className="flex h-8 w-8 items-center justify-center rounded-md text-neutral-500 transition-colors hover:bg-neutral-700/70 hover:text-neutral-200"
                      onClick={refresh}
                    >
                      <RefreshCwIcon size={15} />
                    </ButtonBase>
                  )}
                </Tooltip>
              ) : undefined
            }
          />
        }
        footer={false}
      >
        <div className="flex flex-col gap-5 pb-1 pt-2">
          {/* mount mode cards */}
          <div className="flex flex-col gap-2">
            <span className="text-xs font-medium uppercase tracking-wider text-neutral-500">
              {t('image.mountMode')}
            </span>
            <div className="grid grid-cols-2 gap-3">
              {modes.map((item) => {
                const selected = mode === item.value;
                return (
                  <ButtonBase
                    key={item.value}
                    disabled={isMounted}
                    onClick={() => updateMode(item.value)}
                    className={clsx(
                      'relative flex flex-col items-center gap-2.5 rounded-xl border py-5 transition-colors',
                      selected
                        ? 'border-blue-400/70 bg-blue-500/15 text-blue-300'
                        : 'border-white/10 bg-white/[0.04] text-neutral-400 hover:border-white/20 hover:bg-white/[0.08] hover:text-neutral-200',
                      isMounted && 'opacity-60'
                    )}
                  >
                    {selected && (
                      <span className="absolute right-2 top-2 flex h-4 w-4 items-center justify-center rounded-full bg-blue-500">
                        <CheckIcon size={10} strokeWidth={3.5} className="text-white" />
                      </span>
                    )}
                    <item.icon size={24} strokeWidth={1.8} />
                    <span className="text-sm">{item.label}</span>
                  </ButtonBase>
                );
              })}
            </div>
          </div>

          {/* read only */}
          <div
            className={clsx(
              'flex items-center justify-between rounded-xl border border-white/10 bg-white/[0.04] px-4 py-3',
              (isMounted || mode === 'cd-rom') && 'opacity-60'
            )}
          >
            <span className="text-sm text-neutral-300">{t('image.readOnly')}</span>
            <Switch
              value={readOnly}
              disabled={isMounted || mode === 'cd-rom'}
              onChange={updateReadOnly}
            />
          </div>

          {/* image list */}
          <div className="flex flex-col gap-2">
            <span className="text-xs font-medium uppercase tracking-wider text-neutral-500">
              {t('image.title')}
            </span>
            {!isRefresh && (
              <Images
                refreshVersion={refreshVersion}
                isOpen={isModalOpen}
                cdrom={mode === 'cd-rom'}
                readOnly={readOnly}
                setIsMounted={setIsMounted}
              />
            )}
          </div>
        </div>
      </Modal>
    </>
  );
}
