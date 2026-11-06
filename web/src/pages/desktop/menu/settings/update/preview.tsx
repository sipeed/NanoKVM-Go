import { useCallback, useEffect, useRef, useState } from 'react';
import { Switch, Tooltip } from 'antd';
import { CircleAlertIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/application.ts';

type PreviewProps = {
  checkForUpdates: () => void;
  disabled: boolean;
  onBusyChange: (busy: boolean) => void;
};

export const Preview = ({ checkForUpdates, disabled, onBusyChange }: PreviewProps) => {
  const { t } = useTranslation();

  const [isLoading, setIsLoading] = useState(false);
  const [isEnabled, setIsEnabled] = useState(false);
  const requestIdRef = useRef(0);

  const getPreviewUpdates = useCallback(() => {
    const requestId = ++requestIdRef.current;
    setIsLoading(true);
    onBusyChange(true);

    api
      .getPreviewUpdates()
      .then((rsp) => {
        if (requestId !== requestIdRef.current) return;
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          return;
        }

        setIsEnabled(rsp.data.enabled);
      })
      .finally(() => {
        if (requestId !== requestIdRef.current) return;
        setIsLoading(false);
        onBusyChange(false);
      });
  }, [onBusyChange]);

  useEffect(() => {
    getPreviewUpdates();
    return () => {
      requestIdRef.current += 1;
      onBusyChange(false);
    };
  }, [getPreviewUpdates, onBusyChange]);

  function setPreviewUpdates() {
    if (disabled || isLoading) return;
    const requestId = ++requestIdRef.current;
    let changed = false;
    setIsLoading(true);
    onBusyChange(true);

    const enable = !isEnabled;

    api
      .setPreviewUpdates(enable)
      .then((rsp) => {
        if (requestId !== requestIdRef.current) return;
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          return;
        }

        setIsEnabled(enable);
        changed = true;
      })
      .finally(() => {
        if (requestId !== requestIdRef.current) return;
        setIsLoading(false);
        onBusyChange(false);
        if (changed) {
          checkForUpdates();
        }
      });
  }

  return (
    <div className="flex items-center justify-between py-3">
      <div className="flex flex-col">
        <div className="flex items-center space-x-2">
          <span>{t('settings.update.preview')}</span>

          <Tooltip
            title={t('settings.update.previewTip')}
            className="cursor-pointer text-neutral-500/60"
            placement="bottom"
            styles={{ root: { maxWidth: '350px' } }}
          >
            <CircleAlertIcon size={15} />
          </Tooltip>
        </div>

        <span className="text-xs text-neutral-500">{t('settings.update.previewDesc')}</span>
      </div>

      <Switch
        checked={isEnabled}
        disabled={disabled}
        loading={isLoading}
        onChange={setPreviewUpdates}
      />
    </div>
  );
};
