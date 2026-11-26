import { useEffect, useState } from 'react';
import { Switch } from 'antd';
import { useTranslation } from 'react-i18next';

import { getHidMode } from '@/api/hid.ts';
import { getMountedImage } from '@/api/storage.ts';
import * as api from '@/api/virtual-device.ts';
import type { VirtualAudioMode } from '@/api/virtual-device.ts';

import { Disk } from './disk.tsx';
import { Microphone } from './microphone.tsx';
import { Network } from './network.tsx';

export const VirtualDevices = () => {
  const { t } = useTranslation();

  const [isHidOnlyMode, setIsHidOnlyMode] = useState(false);
  const [isImageMounted, setIsImageMounted] = useState(false);

  const [endpoints, setEndpoints] = useState<{ used: number; total: number } | null>(null);
  const [usbDisplayEnabled, setUsbDisplayEnabled] = useState<boolean | null>(null);
  const [webcamEnabled, setWebcamEnabled] = useState(false);
  const [webcamBusy, setWebcamBusy] = useState(false);
  const [webcamError, setWebcamError] = useState('');
  const [isNetworkEnabled, setIsNetworkEnabled] = useState(false);
  const [isDiskEnabled, setIsDiskEnabled] = useState(false);
  const [isSdCardExist, setIsSdCardExist] = useState(false);
  const [diskMinSizeMiB, setDiskMinSizeMiB] = useState(0);
  const [diskMaxSizeMiB, setDiskMaxSizeMiB] = useState(0);
  const [audioMode, setAudioMode] = useState<VirtualAudioMode>('off');
  const [isAudioRuntimeReady, setIsAudioRuntimeReady] = useState(true);

  useEffect(() => {
    getHidMode().then((rsp) => {
      if (rsp.code === 0) {
        setIsHidOnlyMode(rsp.data.mode === 'hid-only');
      }
    });

    getMountedImage().then((rsp) => {
      if (rsp.code === 0) {
        setIsImageMounted(!!rsp.data?.mounted);
      }
    });

    getVirtualDevices();

    const handleVirtualAudioModeChanged = () => {
      void getVirtualDevices();
    };
    window.addEventListener(api.VIRTUAL_AUDIO_MODE_CHANGED_EVENT, handleVirtualAudioModeChanged);

    return () => {
      window.removeEventListener(
        api.VIRTUAL_AUDIO_MODE_CHANGED_EVENT,
        handleVirtualAudioModeChanged
      );
    };
  }, []);

  async function getVirtualDevices() {
    const rsp = await api.getVirtualDevice();
    if (rsp.code !== 0 || !rsp.data) {
      return;
    }

    const { usbInEndpointsUsed: used, usbInEndpointsTotal: total } = rsp.data;
    setEndpoints(Number.isInteger(used) && Number.isInteger(total) ? { used, total } : null);
    setUsbDisplayEnabled(!!rsp.data.isUsbDisplayEnabled);
    setWebcamEnabled(!!rsp.data.isWebcamEnabled);
    setIsNetworkEnabled(rsp.data.isNetworkEnabled);
    setIsDiskEnabled(rsp.data.isDiskEnabled);
    setIsSdCardExist(rsp.data.isSdCardExist);
    setDiskMinSizeMiB(Number(rsp.data.diskMinSizeMiB) || 0);
    setDiskMaxSizeMiB(Number(rsp.data.diskMaxSizeMiB) || 0);
    if (api.isVirtualAudioMode(rsp.data.audioMode)) {
      setAudioMode(rsp.data.audioMode);
      setIsAudioRuntimeReady(
        api.isVirtualAudioRuntimeReady(
          rsp.data.audioMode,
          rsp.data.isMicEnabled,
          rsp.data.isSpeakerEnabled
        )
      );
    }
  }

  const webcamBlockReason = usbDisplayEnabled
    ? t('settings.device.webcamUsbDisplay')
    : endpoints && endpoints.total - endpoints.used < 2
      ? t('settings.device.webcamEndpoints')
      : '';

  return (
    <>
      {isHidOnlyMode ? (
        <span className="text-neutral-400">{t('settings.device.hidOnly')}</span>
      ) : (
        isImageMounted && (
          <span className="text-neutral-400">{t('settings.device.imageMounted')}</span>
        )
      )}

      <div className="text-sm text-neutral-400" aria-live="polite">
        {endpoints
          ? t('settings.device.usbEndpointUsage', {
              defaultValue: 'USB IN endpoints: {{used}} of {{total}} used · {{free}} free',
              ...endpoints,
              free: Math.max(0, endpoints.total - endpoints.used)
            })
          : t('settings.device.usbEndpointUnknown', 'USB IN endpoint count unavailable')}
      </div>

      <div className="flex items-center justify-between gap-3">
        <div className="flex flex-col gap-1">
          <span>{t('webcam.title', 'Webcam')}</span>
          <span className="text-xs text-neutral-500">
            {(!webcamEnabled && webcamBlockReason) || 'Expose a USB camera to the connected computer.'}
          </span>
          {webcamError && (
            <span role="alert" className="text-xs text-red-400">
              {webcamError}
            </span>
          )}
        </div>
        <Switch
          aria-label="Webcam"
          checked={webcamEnabled}
          loading={webcamBusy}
          disabled={isHidOnlyMode || isImageMounted || usbDisplayEnabled === null || (!webcamEnabled && !!webcamBlockReason)}
          onChange={async (enabled) => {
            setWebcamBusy(true);
            setWebcamError('');
            try {
              const rsp = await api.setWebcamEnabled(enabled);
              if (rsp.code !== 0) setWebcamError(rsp.msg || 'Could not change the USB webcam.');
              await getVirtualDevices();
            } catch {
              setWebcamError('Could not reach the webcam service.');
            } finally {
              setWebcamBusy(false);
            }
          }}
        />
      </div>

      <Disk
        isDisabled={isHidOnlyMode || isImageMounted}
        isStorageDisableDisabled={isImageMounted}
        isDiskEnabled={isDiskEnabled}
        isSdCardExist={isSdCardExist}
        diskMinSizeMiB={diskMinSizeMiB}
        diskMaxSizeMiB={diskMaxSizeMiB}
        getVirtualDevices={getVirtualDevices}
      />

      <Network
        isDisabled={isHidOnlyMode || isImageMounted}
        isNetworkEnabled={isNetworkEnabled}
        getVirtualDevices={getVirtualDevices}
      />

      <Microphone
        isDisabled={isHidOnlyMode || isImageMounted}
        audioMode={audioMode}
        isAudioRuntimeReady={isAudioRuntimeReady}
        getVirtualDevices={getVirtualDevices}
      />
    </>
  );
};
