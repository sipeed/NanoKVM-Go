import { Button } from 'antd';

import { useRemoteNetworkMessages } from '../../messages.ts';
import { RemoteNetworkModePicker } from '../../mode-picker.tsx';
import type { TailscaleMode, TailscaleModeId } from './modes.ts';

type ChooseModeProps = {
  modes: TailscaleMode[];
  value: TailscaleModeId;
  onChange: (id: TailscaleModeId) => void;
  onConfirm: () => void;
};

/**
 * "Choose a connection network" step. The mode list is component data, so this
 * screen only renders what the current build is allowed to offer.
 */
export const ChooseMode = ({ modes, value, onChange, onConfirm }: ChooseModeProps) => {
  const t = useRemoteNetworkMessages();

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-3">
        <h3 className="text-sm font-semibold text-neutral-200">
          {t('settings.remoteNetwork.mode.heading')}
        </h3>

        <RemoteNetworkModePicker
          legend={t('settings.remoteNetwork.mode.heading')}
          name="tailscale-mode"
          onChange={(id) => onChange(id as TailscaleModeId)}
          options={modes.map((mode) => ({
            id: mode.id,
            label: t(mode.nameKey),
            description: t(mode.descriptionKey),
            badge: mode.badgeKey ? t(mode.badgeKey) : undefined,
            Icon: mode.Icon
          }))}
          value={value}
        />
      </div>

      <div className="flex justify-center">
        <Button onClick={onConfirm} shape="round" size="large" type="primary">
          {t('tailscale.continue')}
        </Button>
      </div>
    </div>
  );
};
