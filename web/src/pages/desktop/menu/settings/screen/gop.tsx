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

export const Gop = () => {
  const { t } = useTranslation();
  const [videoParameters, setVideoParameters] = useAtom(videoParametersAtom);
  const [draftGop, setDraftGop] = useState(videoParameters.gop);

  useEffect(() => {
    setDraftGop(videoParameters.gop);
  }, [videoParameters.gop]);

  async function update(value: number) {
    if (value === videoParameters.gop) {
      return;
    }

    try {
      const rsp = await api.setGop(value);
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        setDraftGop(videoParameters.gop);
        return;
      }

      const parameters = { ...videoParameters, gop: value };
      setVideoParameters(parameters);
      storage.setVideoParameters(JSON.stringify(parameters));
    } catch (err) {
      console.log(err);
      setDraftGop(videoParameters.gop);
    }
  }

  const help = (
    <ul className="max-w-[550px] list-outside">
      <li>{t('settings.screen.gop.lower')}</li>
      <li>{t('settings.screen.gop.higher')}</li>
    </ul>
  );

  return (
    <SettingRow
      label={
        <div className="flex items-center space-x-2">
          <span>GOP</span>
          <Popover className="cursor-pointer text-neutral-500" content={help}>
            <CircleHelpIcon size={15} />
          </Popover>
        </div>
      }
      description={t('settings.screen.gop.description')}
      control={
        <MarkedSlider
          value={draftGop}
          min={1}
          max={300}
          minLabel="1"
          maxLabel="300"
          onChange={setDraftGop}
          onChangeComplete={update}
        />
      }
    />
  );
};
