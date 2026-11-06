import { Divider } from 'antd';
import { useTranslation } from 'react-i18next';

import { Bitrate } from './bitrate.tsx';
import { Edid } from './edid.tsx';
import { Fps } from './fps.tsx';
import { Gop } from './gop.tsx';
import { RateControl } from './rate-control.tsx';
import { Scale } from './scale.tsx';
import { VideoMode } from './video-mode.tsx';

export const Screen = () => {
  const { t } = useTranslation();

  return (
    <>
      <div className="text-base font-bold">{t('settings.screen.title')}</div>
      <Divider className="opacity-50" />

      <div className="flex flex-col space-y-8">
        <VideoMode />
        <RateControl />
        <Bitrate />
        <Gop />
        <Fps />
        <Scale />
        <Edid />
      </div>
    </>
  );
};
