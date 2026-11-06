import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Button, Slider, Spin } from 'antd';
import clsx from 'clsx';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/ai.ts';
import type { ScreenTimelapseRecordingConfig } from '@/api/ai.ts';
import { feedback } from '@/lib/feedback.tsx';

import {
  screenTimelapseQPToQuality,
  screenTimelapseQualityDefault,
  screenTimelapseQualityMax,
  screenTimelapseQualityMin,
  screenTimelapseQualitySteps,
  screenTimelapseQualityToQP
} from './screen-timelapse-recording-model.ts';

type ScreenTimelapseRecordingPanelProps = {
  disabled?: boolean;
};

const translationRoot = 'settings.ai.screenTimelapseRecording';

const screenTimelapseSliderRailStyle = {
  backgroundColor: 'rgba(255, 255, 255, 0.2)'
};

export const ScreenTimelapseRecordingPanel = ({
  disabled = false
}: ScreenTimelapseRecordingPanelProps) => {
  const { t } = useTranslation();
  const [config, setConfig] = useState<ScreenTimelapseRecordingConfig>();
  const [draftQuality, setDraftQuality] = useState<number>();
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const mountedRef = useRef(false);
  const loadingRef = useRef(false);
  const savingRef = useRef(false);
  const configRef = useRef<ScreenTimelapseRecordingConfig>();

  const text = useCallback(
    (key: string, fallback: string) => t(`${translationRoot}.${key}`, { defaultValue: fallback }),
    [t]
  );

  const applyConfig = useCallback((nextConfig: ScreenTimelapseRecordingConfig) => {
    configRef.current = nextConfig;
    setDraftQuality(screenTimelapseQPToQuality(nextConfig.qp));
    setConfig(nextConfig);
  }, []);

  const loadConfig = useCallback(async () => {
    if (loadingRef.current || savingRef.current) return;
    loadingRef.current = true;
    if (mountedRef.current) setIsLoading(true);

    try {
      const response = await api.getScreenTimelapseRecording();
      if (response.code !== 0 || !response.data) {
        if (mountedRef.current) {
          feedback.error(text('loadFailed', 'Failed to load recording settings.'));
        }
        return;
      }
      if (mountedRef.current) applyConfig(response.data);
    } catch {
      if (mountedRef.current) {
        feedback.error(text('loadFailed', 'Failed to load recording settings.'));
      }
    } finally {
      loadingRef.current = false;
      if (mountedRef.current) setIsLoading(false);
    }
  }, [applyConfig, text]);

  useEffect(() => {
    mountedRef.current = true;
    void loadConfig();
    return () => {
      mountedRef.current = false;
    };
  }, [loadConfig]);

  const save = useCallback(async () => {
    if (
      !configRef.current ||
      draftQuality === undefined ||
      loadingRef.current ||
      savingRef.current ||
      disabled
    ) {
      return;
    }

    const qp = screenTimelapseQualityToQP(draftQuality);
    if (qp === null) return;

    savingRef.current = true;
    if (mountedRef.current) setIsSaving(true);
    try {
      const response = await api.setScreenTimelapseRecording(qp);
      if (response.code !== 0 || !response.data) {
        if (mountedRef.current) {
          feedback.error(text('saveFailed', 'Failed to save recording settings.'));
        }
        return;
      }
      if (!mountedRef.current) return;

      applyConfig({ qp: response.data.qp });
      if (!response.data.changed) {
        feedback.success(text('noChanges', 'No changes to save.'));
      } else if (response.data.restarted) {
        feedback.success(
          text('savedAndRestarted', 'Recording quality saved and the video service restarted.')
        );
      } else {
        feedback.success(text('saved', 'Recording quality saved.'));
      }
    } catch {
      if (mountedRef.current) {
        feedback.error(text('saveFailed', 'Failed to save recording settings.'));
      }
    } finally {
      savingRef.current = false;
      if (mountedRef.current) setIsSaving(false);
    }
  }, [applyConfig, disabled, draftQuality, text]);

  const isBusy = isLoading || isSaving || disabled;
  const qualityMarks = useMemo(
    () =>
      Object.fromEntries(
        screenTimelapseQualitySteps.map(({ quality: value }) => {
          if (value === screenTimelapseQualityMin) {
            return [
              value,
              <span className="whitespace-nowrap" key={value}>
                {text('low', 'Low')}
              </span>
            ];
          }
          if (value === screenTimelapseQualityMax) {
            return [
              value,
              <span className="whitespace-nowrap" key={value}>
                {text('high', 'High')}
              </span>
            ];
          }
          if (value === screenTimelapseQualityDefault) {
            return [
              value,
              <span className="whitespace-nowrap" key={value}>
                {text('default', 'Default')}
              </span>
            ];
          }
          return [value, { label: value, style: { display: 'none' } }];
        })
      ),
    [text]
  );

  return (
    <div className="border-t border-neutral-700/50 pt-6">
      <div className="mb-4 text-sm font-medium">{text('title', 'Recording quality')}</div>

      {isLoading && !config ? (
        <div className="flex min-h-24 items-center justify-center">
          <Spin size="small" />
        </div>
      ) : (
        <>
          <div className={clsx('px-1 pb-3', isSaving && 'pointer-events-none')}>
            <Slider
              value={draftQuality}
              min={screenTimelapseQualityMin}
              max={screenTimelapseQualityMax}
              step={null}
              marks={qualityMarks}
              tooltip={{ open: false }}
              styles={{ rail: screenTimelapseSliderRailStyle }}
              className="screen-timelapse-quality-slider"
              disabled={isLoading || disabled || !config}
              ariaLabelForHandle={text('title', 'Recording quality')}
              ariaValueTextFormatterForHandle={(value) => `${value}%`}
              onChange={setDraftQuality}
            />
          </div>

          <div className="mt-2 text-xs text-neutral-500">
            {text(
              'activeHint',
              'Higher recording quality uses more storage. Saving changes briefly restarts the video service.'
            )}
          </div>

          <div className="mt-4 flex justify-end">
            <Button
              type="primary"
              disabled={isBusy || !config}
              loading={isSaving}
              onClick={() => void save()}
            >
              {text('save', 'Save')}
            </Button>
          </div>
        </>
      )}
    </div>
  );
};
