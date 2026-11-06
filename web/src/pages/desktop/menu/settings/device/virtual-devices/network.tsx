import { useEffect, useState } from 'react';
import { Switch, Tooltip } from 'antd';
import clsx from 'clsx';
import { ChevronRightIcon, CircleAlertIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/virtual-device.ts';
import { ButtonBase } from '@/components/ui/button-base.tsx';

import { UsbNetworkConfig } from './usb-network.tsx';

interface NetworkProps {
  isDisabled: boolean;
  isNetworkEnabled: boolean;
  getVirtualDevices: () => Promise<void>;
}

export const Network = ({ isDisabled, isNetworkEnabled, getVirtualDevices }: NetworkProps) => {
  const { t } = useTranslation();

  const [isToggleLoading, setIsToggleLoading] = useState(false);
  const [isConfigSaving, setIsConfigSaving] = useState(false);
  const [isConfigExpanded, setIsConfigExpanded] = useState(false);
  const [hasOpenedConfig, setHasOpenedConfig] = useState(false);

  useEffect(() => {
    if (!isNetworkEnabled) {
      setIsConfigExpanded(false);
      setHasOpenedConfig(false);
    }
  }, [isNetworkEnabled]);

  async function update() {
    if (isToggleLoading || isConfigSaving) return;
    setIsToggleLoading(true);

    try {
      const rsp = await api.updateVirtualDevice('network');
      if (rsp.code === 0) {
        await getVirtualDevices();
      }
    } catch (err) {
      console.log(err);
    } finally {
      setIsToggleLoading(false);
    }
  }

  function toggleConfig() {
    if (!isNetworkEnabled || isToggleLoading || isConfigSaving) return;

    setHasOpenedConfig(true);
    setIsConfigExpanded((expanded) => !expanded);
  }

  return (
    <div className="flex flex-col">
      <div className="flex items-center justify-between gap-3">
        <div className="flex min-w-0 flex-1 flex-col space-y-1 pr-3">
          <div className="flex items-center gap-1.5">
            <span>{t('settings.device.network')}</span>
            <Tooltip
              title={t('settings.device.usbNetwork.forwardingTip')}
              placement="top"
              styles={{ root: { maxWidth: '400px' } }}
            >
              <span
                className="inline-flex cursor-pointer text-amber-400"
                tabIndex={0}
                aria-label={t('settings.device.usbNetwork.forwardingTip')}
              >
                <CircleAlertIcon size={14} aria-hidden="true" />
              </span>
            </Tooltip>
          </div>
          <span className="text-xs text-neutral-500">{t('settings.device.networkDesc')}</span>
        </div>

        <div className="flex shrink-0 items-center justify-end">
          <Switch
            className="shrink-0"
            disabled={isDisabled || isConfigSaving}
            checked={isNetworkEnabled}
            loading={isToggleLoading}
            onChange={() => void update()}
          />

          <div
            aria-hidden={!isNetworkEnabled}
            className={clsx(
              'overflow-hidden transition-[width,margin,opacity] duration-200',
              isNetworkEnabled ? 'ml-2 w-8 opacity-100' : 'ml-0 w-0 opacity-0'
            )}
          >
            <ButtonBase
              className="flex size-8 items-center justify-center rounded-md border border-neutral-700/70 bg-neutral-800/40 text-neutral-300 transition-colors enabled:hover:bg-neutral-700/70 enabled:hover:text-white enabled:active:bg-neutral-700"
              aria-label={t('settings.device.usbNetwork.title')}
              aria-expanded={isConfigExpanded}
              aria-controls={isNetworkEnabled ? 'usb-network-config' : undefined}
              disabled={!isNetworkEnabled || isToggleLoading || isConfigSaving}
              tabIndex={isNetworkEnabled ? 0 : -1}
              onClick={toggleConfig}
            >
              <ChevronRightIcon
                size={18}
                aria-hidden="true"
                className={clsx(
                  'transition-transform duration-200',
                  isConfigExpanded && 'rotate-90'
                )}
              />
            </ButtonBase>
          </div>
        </div>
      </div>

      {isNetworkEnabled && hasOpenedConfig && (
        <div
          id="usb-network-config"
          className="mt-3 rounded-lg border border-neutral-700/60 bg-neutral-800/30 p-3 sm:p-4"
          hidden={!isConfigExpanded}
        >
          <UsbNetworkConfig
            disabled={isDisabled || isToggleLoading}
            getVirtualDevices={getVirtualDevices}
            onSavingChange={setIsConfigSaving}
          />
        </div>
      )}
    </div>
  );
};
