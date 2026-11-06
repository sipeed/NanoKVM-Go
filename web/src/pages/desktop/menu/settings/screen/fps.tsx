import { useEffect, useState } from 'react';
import { Popover } from 'antd';
import { useAtom } from 'jotai';
import { CircleHelpIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/stream.ts';
import * as storage from '@/lib/localstorage.ts';
import { videoParametersAtom } from '@/jotai/screen.ts';
import { MarkedSlider } from './marked-slider.tsx';
import { SettingRow } from './setting-row.tsx';

export const Fps = () => {
  const { t } = useTranslation();
  const [videoParameters, setVideoParameters] = useAtom(videoParametersAtom);
  const [draftFps, setDraftFps] = useState(videoParameters.fps);

  useEffect(() => {
    setDraftFps(videoParameters.fps);
  }, [videoParameters.fps]);

  async function update(value: number) {
    if (value === videoParameters.fps) {
      return;
    }

    try {
      const rsp = await api.setFps(value);
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        setDraftFps(videoParameters.fps);
        return;
      }

      const parameters = { ...videoParameters, fps: value };
      setVideoParameters(parameters);
      storage.setVideoParameters(JSON.stringify(parameters));
    } catch (err) {
      console.log(err);
      setDraftFps(videoParameters.fps);
    }
  }

  const help = (
    <ul className="max-w-[550px] list-outside">
      <li>{t('settings.screen.fps.tip1')}</li>
      <li>{t('settings.screen.fps.tip2')}</li>
    </ul>
  );

  return (
    <SettingRow
      label={
        <div className="flex items-center space-x-2">
          <span>FPS</span>
          <Popover className="cursor-pointer text-neutral-500" content={help}>
            <CircleHelpIcon size={15} />
          </Popover>
        </div>
      }
      description={t('settings.screen.fps.description')}
      control={
        <MarkedSlider
          value={draftFps}
          min={0}
          max={120}
          minLabel={draftFps === 0 ? t('settings.screen.fps.auto') : '0'}
          maxLabel="120"
          onChange={setDraftFps}
          onChangeComplete={update}
        />
      }
    />
  );
};
