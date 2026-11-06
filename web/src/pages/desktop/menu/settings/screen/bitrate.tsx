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

export const Bitrate = () => {
  const { t } = useTranslation();

  const [videoParameters, setVideoParameters] = useAtom(videoParametersAtom);
  const [draftBitrate, setDraftBitrate] = useState(videoParameters.bitrate);

  useEffect(() => {
    setDraftBitrate(videoParameters.bitrate);
  }, [videoParameters.bitrate]);

  async function update(value: number) {
    if (value === videoParameters.bitrate) {
      return;
    }

    try {
      const rsp = await api.setQuality(value);
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        setDraftBitrate(videoParameters.bitrate);
        return;
      }

      const parameters = { ...videoParameters, bitrate: value };
      setVideoParameters(parameters);
      storage.setVideoParameters(JSON.stringify(parameters));
    } catch (err) {
      console.log(err);
      setDraftBitrate(videoParameters.bitrate);
    }
  }

  const help = (
    <ul className="max-w-[550px] list-outside">
      <li>{t('settings.screen.bitrate.lower')}</li>
      <li>{t('settings.screen.bitrate.higher')}</li>
    </ul>
  );

  return (
    <SettingRow
      label={
        <div className="flex items-center space-x-2">
          <span>{t('settings.screen.bitrate.title')}</span>
          <Popover className="cursor-pointer text-neutral-500" content={help}>
            <CircleHelpIcon size={15} />
          </Popover>
        </div>
      }
      description={t('settings.screen.bitrate.description')}
      control={
        <MarkedSlider
          value={draftBitrate}
          min={1000}
          max={20000}
          minLabel="1000"
          maxLabel="20000"
          onChange={setDraftBitrate}
          onChangeComplete={update}
        />
      }
    />
  );
};
