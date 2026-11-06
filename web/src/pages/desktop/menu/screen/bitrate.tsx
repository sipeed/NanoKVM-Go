import { useEffect, useState } from 'react';
import { Divider } from 'antd';
import { useAtom } from 'jotai';
import { CheckIcon, SquareActivityIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/stream.ts';
import * as storage from '@/lib/localstorage.ts';
import { videoParametersAtom } from '@/jotai/screen.ts';
import { MenuSubmenu } from '@/components/menu-item.tsx';

export const Bitrate = () => {
  const { t } = useTranslation();
  const [videoParameters, setVideoParameters] = useAtom(videoParametersAtom);

  const [customBitrate, setCustomBitrate] = useState(0);
  const [isLoading, setIsLoading] = useState(false);

  const qualityPresets = [
    { id: 'auto', mode: 'vbr', bitrate: 10240, gop: 300, label: t('screen.qualityAuto') },
    { id: 'lossless', mode: 'cbr', bitrate: 10240, gop: 120, label: t('screen.qualityLossless') },
    { id: 'high', mode: 'cbr', bitrate: 8192, gop: 120, label: t('screen.qualityHigh') },
    { id: 'medium', mode: 'cbr', bitrate: 4096, gop: 240, label: t('screen.qualityMedium') },
    { id: 'low', mode: 'cbr', bitrate: 1024, gop: 240, label: t('screen.qualityLow') }
  ];

  useEffect(() => {
    if (videoParameters.rateControlMode === 'vbr') {
      setCustomBitrate(0);
      return;
    }

    const isExist = qualityPresets.some(
      (preset) => preset.mode === 'cbr' && preset.bitrate === videoParameters.bitrate
    );
    if (isExist) {
      setCustomBitrate(0);
      return;
    }

    setCustomBitrate(Math.floor(videoParameters.bitrate / 100));
  }, [videoParameters]);

  async function update(preset: (typeof qualityPresets)[number]) {
    if (isLoading) {
      return;
    }
    setIsLoading(true);

    try {
      let rsp = await api.setRateControlMode(preset.mode);
      if (rsp.code !== 0) {
        return;
      }

      rsp = await api.setQuality(preset.bitrate);
      if (rsp.code !== 0) {
        return;
      }

      rsp = await api.setGop(preset.gop);
      if (rsp.code !== 0) {
        return;
      }

      const parameters = {
        ...videoParameters,
        rateControlMode: preset.mode,
        bitrate: preset.bitrate,
        gop: preset.gop
      };
      setVideoParameters(parameters);
      storage.setVideoParameters(JSON.stringify(parameters));
    } catch (err) {
      console.log(err);
    } finally {
      setIsLoading(false);
    }
  }

  function isChecked(preset: (typeof qualityPresets)[number]) {
    if (videoParameters.rateControlMode === 'vbr') {
      return preset.mode === 'vbr';
    }
    return preset.mode === 'cbr' && preset.bitrate === videoParameters.bitrate;
  }

  const content = (
    <>
      {qualityPresets.map((preset) => (
        <div
          key={preset.id}
          className="flex h-[30px] cursor-pointer select-none items-center rounded pl-1 pr-5 hover:bg-neutral-700/70"
          onClick={() => update(preset)}
        >
          <div className="flex h-[14px] w-[20px] items-end text-blue-500">
            {isChecked(preset) && <CheckIcon size={14} />}
          </div>
          <span>{preset.label}</span>
        </div>
      ))}

      {customBitrate > 0 && (
        <>
          <Divider style={{ margin: '5px 0' }} />
          <div className="flex h-[30px] cursor-pointer select-none items-center rounded pl-1 pr-5 hover:bg-neutral-700/70">
            <div className="flex h-[14px] w-[20px] items-end text-blue-500">
              <CheckIcon size={14} />
            </div>
            <span>{customBitrate}%</span>
          </div>
        </>
      )}
    </>
  );

  return (
    <MenuSubmenu
      title={t('screen.quality')}
      content={content}
      popoverProps={{ placement: 'rightTop', arrow: false, align: { offset: [14, 0] } }}
    >
      <div className="flex h-[30px] cursor-pointer items-center space-x-2 rounded pl-3 pr-6 text-neutral-300 hover:bg-neutral-700/70">
        <SquareActivityIcon size={18} />
        <span className="select-none text-sm">{t('screen.quality')}</span>
      </div>
    </MenuSubmenu>
  );
};
