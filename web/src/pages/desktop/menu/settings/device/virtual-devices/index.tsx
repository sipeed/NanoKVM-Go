import { useEffect, useState } from 'react';
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

  return (
    <>
      {isHidOnlyMode ? (
        <span className="text-neutral-400">{t('settings.device.hidOnly')}</span>
      ) : (
        isImageMounted && (
          <span className="text-neutral-400">{t('settings.device.imageMounted')}</span>
        )
      )}

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
