import { Fragment, useState } from 'react';
import { Divider } from 'antd';
import clsx from 'clsx';
import { useAtomValue } from 'jotai';
import { CheckIcon, LoaderCircleIcon, TvMinimalPlayIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/stream.ts';
import { setVideoMode as setCookie } from '@/lib/localstorage.ts';
import { getSupportedVideoModes } from '@/lib/video.ts';
import { videoModeAtom } from '@/jotai/screen.ts';
import { MenuSubmenu } from '@/components/menu-item.tsx';

const videoGroups = [
  {
    key: 'h265',
    name: 'H.265',
    modes: [
      { key: 'h265-webrtc', name: 'H.265 WebRTC' },
      { key: 'h265-direct', name: 'H.265 Direct ' }
    ]
  },
  {
    key: 'h264',
    name: 'H.264',
    modes: [
      { key: 'h264-webrtc', name: 'H.264 WebRTC' },
      { key: 'h264-direct', name: 'H.264 Direct ' }
    ]
  }
];

export const VideoMode = () => {
  const { t } = useTranslation();
  const videoMode = useAtomValue(videoModeAtom);
  const [loadingMode, setLoadingMode] = useState('');

  const supportedVideoModes = getSupportedVideoModes();

  async function update(mode: string) {
    if (loadingMode || mode === videoMode || !supportedVideoModes.includes(mode)) return;

    setLoadingMode(mode);

    try {
      const rsp = await api.setMode(mode);
      if (rsp.code !== 0) {
        setLoadingMode('');
        return;
      }

      setCookie(mode);
      window.location.reload();
    } catch (err) {
      console.log(err);
      setLoadingMode('');
    }
  }

  const content = (
    <>
      {videoGroups.map((group, index) => (
        <Fragment key={group.key}>
          {group.modes.map((mode) => (
            <div
              key={mode.key}
              className={clsx(
                'flex select-none items-center rounded py-1.5 pl-1 pr-5 hover:bg-neutral-700/70',
                loadingMode && 'pointer-events-none',
                supportedVideoModes.includes(mode.key) && !loadingMode
                  ? 'cursor-pointer'
                  : 'cursor-not-allowed text-neutral-500'
              )}
              onClick={() => update(mode.key)}
            >
              <div className="flex h-[14px] w-[20px] items-end">
                {loadingMode === mode.key ? (
                  <LoaderCircleIcon className="animate-spin text-neutral-400" size={15} />
                ) : (
                  !loadingMode &&
                  mode.key === videoMode && <CheckIcon className="text-blue-500" size={15} />
                )}
              </div>
              <span>{mode.name}</span>
            </div>
          ))}
          {index < videoGroups.length - 1 && <Divider style={{ margin: '5px 0', opacity: 0.5 }} />}
        </Fragment>
      ))}
    </>
  );

  return (
    <MenuSubmenu
      title={t('screen.video')}
      content={content}
      popoverProps={{ placement: 'rightTop', arrow: false, align: { offset: [14, 0] } }}
    >
      <div className="flex h-[30px] cursor-pointer items-center space-x-2 rounded pl-3 pr-6 text-neutral-300 hover:bg-neutral-700/70">
        {loadingMode ? (
          <LoaderCircleIcon className="animate-spin text-neutral-400" size={18} />
        ) : (
          <TvMinimalPlayIcon size={18} />
        )}
        <span className="select-none text-sm">{t('screen.video')}</span>
      </div>
    </MenuSubmenu>
  );
};
