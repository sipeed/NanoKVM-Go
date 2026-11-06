import { useAtom } from 'jotai';
import { useTranslation } from 'react-i18next';

import * as storage from '@/lib/localstorage.ts';
import { videoParametersAtom } from '@/jotai/screen.ts';
import { MarkedSlider } from './marked-slider.tsx';
import { SettingRow } from './setting-row.tsx';

export const Scale = () => {
  const { t } = useTranslation();
  const [videoParameters, setVideoParameters] = useAtom(videoParametersAtom);

  async function update(scale: number) {
    const parameters = { ...videoParameters, scale };

    setVideoParameters(parameters);
    storage.setVideoParameters(JSON.stringify(parameters));
  }

  return (
    <SettingRow
      label={<span>{t('settings.screen.scale.title')}</span>}
      description={t('settings.screen.scale.description')}
      control={
        <MarkedSlider
          value={videoParameters.scale}
          included={false}
          min={0.5}
          max={2}
          step={0.1}
          minLabel="0.5"
          maxLabel="2"
          onChange={update}
        />
      }
    />
  );
};
