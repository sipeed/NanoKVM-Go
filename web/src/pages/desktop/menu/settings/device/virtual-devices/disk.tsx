import { useState } from 'react';
import { Button, InputNumber, Modal, Switch } from 'antd';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/virtual-device.ts';

interface DiskProps {
  isDisabled: boolean;
  isStorageDisableDisabled: boolean;
  isDiskEnabled: boolean;
  isSdCardExist: boolean;
  diskMinSizeMiB: number;
  diskMaxSizeMiB: number;
  getVirtualDevices: () => void;
}

export const Disk = ({
  isDisabled,
  isStorageDisableDisabled,
  isDiskEnabled,
  isSdCardExist,
  diskMinSizeMiB,
  diskMaxSizeMiB,
  getVirtualDevices
}: DiskProps) => {
  const { t } = useTranslation();

  const [isInitializeModalOpen, setIsInitializeModalOpen] = useState(false);
  const [isDisableModalOpen, setIsDisableModalOpen] = useState(false);
  const [isUpdating, setIsUpdating] = useState(false);
  const [isDisabling, setIsDisabling] = useState(false);
  const [diskSizeGiB, setDiskSizeGiB] = useState<number | null>(null);
  const [operationFailed, setOperationFailed] = useState(false);
  const [storageBusy, setStorageBusy] = useState(false);

  const minSizeGiB = Math.ceil(diskMinSizeMiB / 1024);
  const maxSizeGiB = Math.floor(diskMaxSizeMiB / 1024);
  const hasValidSizeRange = minSizeGiB > 0 && maxSizeGiB >= minSizeGiB;
  const hasValidSelectedSize =
    diskSizeGiB !== null && diskSizeGiB >= minSizeGiB && diskSizeGiB <= maxSizeGiB;

  async function update() {
    if (!isDiskEnabled && !isSdCardExist) {
      setDiskSizeGiB(hasValidSizeRange ? maxSizeGiB : null);
      setOperationFailed(false);
      setIsInitializeModalOpen(true);
      return;
    }

    if (await updateVirtualDevice()) {
      await getVirtualDevices();
    }
  }

  async function initialize() {
    if (!hasValidSelectedSize) return;

    if (await updateVirtualDevice(diskSizeGiB * 1024)) {
      setIsInitializeModalOpen(false);
      await getVirtualDevices();
    }
  }

  async function updateVirtualDevice(sizeMiB?: number) {
    if (isUpdating) return false;

    setIsUpdating(true);
    setStorageBusy(false);

    try {
      const rsp = await api.updateVirtualDevice('disk', sizeMiB);
      if (rsp.code !== 0) {
        setStorageBusy(rsp.code === -7);
        setOperationFailed(true);
        return false;
      }
      setOperationFailed(false);
      return true;
    } catch (err) {
      console.log(err);
      setOperationFailed(true);
      return false;
    } finally {
      setIsUpdating(false);
    }
  }

  async function disable() {
    if (isDisabling) return;

    setIsDisabling(true);
    try {
      const rsp = await api.disableVirtualDisk();
      if (rsp.code !== 0) {
        setOperationFailed(true);
        return;
      }

      setOperationFailed(false);
      setIsDisableModalOpen(false);
      await getVirtualDevices();
    } catch (err) {
      console.log(err);
      setOperationFailed(true);
    } finally {
      setIsDisabling(false);
    }
  }

  return (
    <>
      <div className="flex items-center justify-between">
        <div className="flex flex-col space-y-1">
          <span>{t('settings.device.disk')}</span>
          <span className="text-xs text-neutral-500">{t('settings.device.diskDesc')}</span>
        </div>

        <div className="flex items-center gap-3">
          {isSdCardExist && !isDiskEnabled && (
            <Button
              danger
              size="small"
              disabled={isStorageDisableDisabled || isUpdating}
              loading={isDisabling}
              onClick={() => {
                setOperationFailed(false);
                setIsDisableModalOpen(true);
              }}
            >
              {t('settings.device.sdCard.disable')}
            </Button>
          )}

          <Switch
            disabled={isDisabled || isDisabling}
            checked={isDiskEnabled}
            loading={isUpdating}
            onChange={() => update()}
          />
        </div>
      </div>

      {storageBusy && <p className="text-xs text-red-500">{t('image.storageBusy')}</p>}

      <Modal
        title={t('settings.device.sdCard.warning')}
        open={isInitializeModalOpen}
        centered={true}
        okText={t('settings.device.okBtn')}
        cancelText={t('settings.device.cancelBtn')}
        confirmLoading={isUpdating}
        okButtonProps={{ disabled: !hasValidSelectedSize }}
        onOk={initialize}
        onCancel={() => setIsInitializeModalOpen(false)}
      >
        <div className="mb-4 flex items-center justify-between">
          <span>{t('settings.device.sdCard.capacity')}</span>
          <InputNumber
            className="w-40"
            min={minSizeGiB}
            max={maxSizeGiB}
            precision={0}
            addonAfter="GiB"
            value={diskSizeGiB}
            disabled={!hasValidSizeRange}
            onChange={setDiskSizeGiB}
          />
        </div>
        {!hasValidSizeRange && (
          <p className="text-xs text-red-500">{t('download.insufficientStorage')}</p>
        )}
        <p>{t('settings.device.sdCard.tip1')}</p>
        <p className="text-red-500">{t('settings.device.sdCard.tip2')}</p>
        {operationFailed && (
          <p className="text-red-500">{t('settings.device.diskInquiry.failed')}</p>
        )}
      </Modal>

      <Modal
        title={t('settings.device.sdCard.disableWarning')}
        open={isDisableModalOpen}
        centered={true}
        okText={t('settings.device.okBtn')}
        cancelText={t('settings.device.cancelBtn')}
        confirmLoading={isDisabling}
        onOk={disable}
        onCancel={() => setIsDisableModalOpen(false)}
      >
        <p>{t('settings.device.sdCard.disableTip')}</p>
        {operationFailed && (
          <p className="text-red-500">{t('settings.device.diskInquiry.failed')}</p>
        )}
      </Modal>
    </>
  );
};
