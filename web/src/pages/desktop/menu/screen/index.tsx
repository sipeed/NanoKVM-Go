import { useEffect } from 'react';
import { Button, ColorPicker, Modal, Tooltip } from 'antd';
import clsx from 'clsx';
import { useAtom, useAtomValue } from 'jotai';
import { FrameIcon, MonitorIcon, RotateCcwIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/stream.ts';
import { VideoParameters } from '@/types';
import * as storage from '@/lib/localstorage.ts';
import {
  portalBorderColorAtom,
  portalBorderEnabledAtom,
  videoModeAtom,
  videoParametersAtom
} from '@/jotai/screen.ts';
import { MenuItem } from '@/components/menu-item.tsx';

import { Advanced } from './advanced.tsx';
import { Bitrate } from './bitrate.tsx';
import { Edid } from './edid.tsx';
import { Scale } from './scale.tsx';
import { VideoMode } from './video-mode.tsx';

type EncoderParameters = Pick<VideoParameters, 'rateControlMode' | 'bitrate' | 'gop' | 'fps'>;

const defaultPortalBorderColor = '#00e5ff';

function isRateControlMode(value: unknown): value is api.EncoderSettings['rate_control'] {
  return value === 'cbr' || value === 'vbr';
}

function isIntegerInRange(value: unknown, min: number, max: number): value is number {
  return typeof value === 'number' && Number.isInteger(value) && value >= min && value <= max;
}

function getScale(value: unknown, fallback: number) {
  if (typeof value === 'number' && Number.isFinite(value) && value >= 0.5 && value <= 2) {
    return value;
  }

  if (Number.isFinite(fallback) && fallback >= 0.5 && fallback <= 2) {
    return fallback;
  }
  return 1;
}

function getCachedVideoParameters(fallback: VideoParameters) {
  try {
    const cached = storage.getVideoParameters();
    if (!cached) {
      return fallback;
    }

    const value: unknown = JSON.parse(cached);
    if (!value || typeof value !== 'object' || Array.isArray(value)) {
      return fallback;
    }

    const parameters = value as Partial<VideoParameters>;
    return {
      rateControlMode: isRateControlMode(parameters.rateControlMode)
        ? parameters.rateControlMode
        : fallback.rateControlMode,
      bitrate: isIntegerInRange(parameters.bitrate, 1000, 20000)
        ? parameters.bitrate
        : fallback.bitrate,
      gop: isIntegerInRange(parameters.gop, 1, 300) ? parameters.gop : fallback.gop,
      fps: isIntegerInRange(parameters.fps, 0, 120) ? parameters.fps : fallback.fps,
      scale: getScale(parameters.scale, fallback.scale)
    };
  } catch {
    return fallback;
  }
}

function cacheVideoParameters(parameters: VideoParameters) {
  try {
    storage.setVideoParameters(JSON.stringify(parameters));
  } catch (err) {
    console.log(err);
  }
}

function getEncoderParameters(parameters: VideoParameters): EncoderParameters {
  return {
    rateControlMode: parameters.rateControlMode,
    bitrate: parameters.bitrate,
    gop: parameters.gop,
    fps: parameters.fps
  };
}

function hasSameEncoderParameters(parameters: VideoParameters, baseline: EncoderParameters) {
  return (
    parameters.rateControlMode === baseline.rateControlMode &&
    parameters.bitrate === baseline.bitrate &&
    parameters.gop === baseline.gop &&
    parameters.fps === baseline.fps
  );
}

function isEncoderSettings(data: unknown): data is api.EncoderSettings {
  if (!data || typeof data !== 'object') {
    return false;
  }

  const settings = data as Partial<api.EncoderSettings>;
  return (
    isRateControlMode(settings.rate_control) &&
    isIntegerInRange(settings.bitrate, 1000, 20000) &&
    isIntegerInRange(settings.gop, 1, 300) &&
    isIntegerInRange(settings.fps, 0, 120)
  );
}

export const Screen = () => {
  const { t } = useTranslation();

  const videoMode = useAtomValue(videoModeAtom);
  const [, setVideoParameters] = useAtom(videoParametersAtom);
  const [borderEnabled, setBorderEnabled] = useAtom(portalBorderEnabledAtom);
  const [borderColor, setBorderColor] = useAtom(portalBorderColorAtom);
  const [modal, modalContextHolder] = Modal.useModal();

  function toggleBorder() {
    const enabled = !borderEnabled;
    setBorderEnabled(enabled);
    storage.setPortalBorderEnabled(enabled);
  }

  function confirmResetBorderColor() {
    modal.confirm({
      title: t('portal.resetBorderColorConfirmTitle'),
      content: t('portal.resetBorderColorConfirm'),
      okText: t('portal.confirm'),
      cancelText: t('portal.cancel'),
      onOk: () => {
        setBorderColor(defaultPortalBorderColor);
        storage.setPortalBorderColor(defaultPortalBorderColor);
      }
    });
  }

  useEffect(() => {
    if (!videoMode) return;

    api
      .setMode(videoMode)
      .then((rsp) => {
        if (rsp.code !== 0) {
          console.log(rsp.msg || 'failed to set video mode');
        }
      })
      .catch((err) => console.log(err));
  }, [videoMode]);

  useEffect(() => {
    let cancelled = false;
    let encoderBaseline: EncoderParameters | undefined;

    setVideoParameters((current) => {
      const parameters = getCachedVideoParameters(current);
      encoderBaseline = getEncoderParameters(parameters);
      return parameters;
    });

    async function initScreen() {
      try {
        const rsp = await api.getEncoderSettings();
        if (rsp.code !== 0) {
          console.log(rsp.msg || 'failed to get encoder settings');
          return;
        }
        if (!isEncoderSettings(rsp.data)) {
          console.log('invalid encoder settings response');
          return;
        }
        if (cancelled) return;

        setVideoParameters((current) => {
          if (!encoderBaseline || !hasSameEncoderParameters(current, encoderBaseline)) {
            return current;
          }

          const parameters: VideoParameters = {
            rateControlMode: rsp.data.rate_control,
            bitrate: rsp.data.bitrate,
            gop: rsp.data.gop,
            fps: rsp.data.fps,
            scale: getScale(current.scale, 1)
          };

          cacheVideoParameters(parameters);
          return parameters;
        });
      } catch (err) {
        console.log(err);
      }
    }

    initScreen();
    return () => {
      cancelled = true;
    };
  }, [setVideoParameters]);

  const content = (
    <div className="flex flex-col space-y-1">
      <VideoMode />
      <Edid />
      <Bitrate />
      <Scale />
      <Advanced />
      <div
        className={clsx(
          'flex h-[30px] items-center justify-between rounded pr-2 transition-colors hover:bg-neutral-700/70',
          borderEnabled ? 'text-blue-500' : 'text-neutral-300'
        )}
      >
        <button
          type="button"
          className="flex h-full min-w-0 flex-1 cursor-pointer select-none items-center space-x-2 border-0 bg-transparent pl-3 text-left text-inherit"
          aria-pressed={borderEnabled}
          onClick={toggleBorder}
        >
          <FrameIcon size={18} />
          <span className="text-sm">{t('portal.border')}</span>
        </button>
        <ColorPicker
          aria-label={`${t('portal.borderColor')}: ${borderColor.toUpperCase()}`}
          size="small"
          value={borderColor}
          onChange={(color) => {
            const nextColor = color.toHexString();
            setBorderColor(nextColor);
            storage.setPortalBorderColor(nextColor);
          }}
          panelRender={(panel) => (
            <div>
              {panel}
              <div className="flex justify-end border-t border-neutral-700 px-1 pt-1">
                <Tooltip title={t('portal.resetBorderColor')} placement="bottom">
                  <Button
                    type="text"
                    size="small"
                    aria-label={t('portal.resetBorderColor')}
                    icon={<RotateCcwIcon size={14} />}
                    onClick={confirmResetBorderColor}
                  />
                </Tooltip>
              </div>
            </div>
          )}
        />
      </div>
    </div>
  );

  return (
    <>
      {modalContextHolder}
      <MenuItem title={t('screen.title')} icon={<MonitorIcon size={18} />} content={content} />
    </>
  );
};
