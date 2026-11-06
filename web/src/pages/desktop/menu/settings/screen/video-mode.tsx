import { useState } from 'react';
import { Select } from 'antd';
import { useAtomValue } from 'jotai';
import { LoaderCircleIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/stream.ts';
import type { VideoMode as TVideoMode } from '@/types';
import * as storage from '@/lib/localstorage.ts';
import { getSupportedVideoModes } from '@/lib/video.ts';
import { videoModeAtom } from '@/jotai/screen.ts';
import { SettingRow } from './setting-row.tsx';

export const VideoMode = () => {
  const { t } = useTranslation();

  const videoMode = useAtomValue(videoModeAtom);
  const [loadingMode, setLoadingMode] = useState<TVideoMode | null>(null);
  const supportedVideoModes = getSupportedVideoModes();

  const videoModes = [
    { value: 'h265-webrtc', label: 'H.265 WebRTC' },
    { value: 'h265-direct', label: 'H.265 Direct' },
    { value: 'h264-webrtc', label: 'H.264 WebRTC' },
    { value: 'h264-direct', label: 'H.264 Direct' }
  ];

  async function update(value: string) {
    if (loadingMode || value === videoMode || !supportedVideoModes.includes(value)) {
      return;
    }

    const mode = value as TVideoMode;
    setLoadingMode(mode);

    try {
      const rsp = await api.setMode(mode);
      if (rsp.code !== 0) {
        setLoadingMode(null);
        return;
      }

      storage.setVideoMode(mode);
      window.location.reload();
    } catch (err) {
      console.log(err);
      setLoadingMode(null);
    }
  }

  return (
    <SettingRow
      label={<span>{t('settings.screen.videoMode.title')}</span>}
      description={t('settings.screen.videoMode.description')}
      control={
        <Select
          value={loadingMode ?? videoMode}
          style={{ width: '100%' }}
          options={videoModes.map((mode) => ({
            ...mode,
            disabled: !supportedVideoModes.includes(mode.value)
          }))}
          disabled={!!loadingMode}
          suffixIcon={
            loadingMode ? (
              <LoaderCircleIcon className="animate-spin text-neutral-400" size={15} />
            ) : undefined
          }
          onSelect={update}
        />
      }
    />
  );
};
