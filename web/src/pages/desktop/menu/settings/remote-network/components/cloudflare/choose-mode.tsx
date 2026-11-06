import { Button, Popconfirm } from 'antd';
import { KeyRoundIcon, ZapIcon } from 'lucide-react';

import type { CloudflareMode } from '@/api/extensions/cloudflare.ts';

import { useRemoteNetworkMessages } from '../../messages.ts';
import { RemoteNetworkModePicker } from '../../mode-picker.tsx';

const NamedTunnelIcon = ({ size }: { size?: number }) => <KeyRoundIcon size={size} />;
const QuickTunnelIcon = ({ size }: { size?: number }) => <ZapIcon size={size} />;

export const ChooseMode = ({
  value,
  busy,
  confirmDisconnect,
  onChange,
  onConfirm
}: {
  value: CloudflareMode;
  busy: boolean;
  confirmDisconnect: boolean;
  onChange: (mode: CloudflareMode) => void;
  onConfirm: () => void;
}) => {
  const t = useRemoteNetworkMessages();
  return (
    <div className="flex flex-col gap-6 py-4">
      <fieldset disabled={busy} className="flex min-w-0 flex-col gap-3 border-0 p-0">
        <h3 className="text-sm font-semibold text-neutral-200">{t('cloudflare.modeHeading')}</h3>
        <RemoteNetworkModePicker
          legend={t('cloudflare.modeHeading')}
          name="cloudflare-mode"
          value={value}
          onChange={(mode) => onChange(mode as CloudflareMode)}
          options={[
            {
              id: 'quick',
              label: 'Quick Tunnel',
              description: t('cloudflare.quickModeDesc'),
              Icon: QuickTunnelIcon
            },
            {
              id: 'named',
              label: t('cloudflare.namedMode'),
              description: t('cloudflare.namedModeDesc'),
              Icon: NamedTunnelIcon
            }
          ]}
        />
      </fieldset>
      <div className="flex justify-center">
        <Popconfirm
          title={t('cloudflare.modeHeading')}
          description={t('cloudflare.disconnect')}
          styles={{ root: { maxWidth: 'min(360px, calc(100vw - 32px))' } }}
          okText={t('cloudflare.confirm')}
          cancelText={t('cloudflare.cancel')}
          disabled={busy || !confirmDisconnect}
          onConfirm={onConfirm}
        >
          <Button
            type="primary"
            shape="round"
            size="large"
            loading={busy}
            disabled={busy}
            onClick={() => {
              if (!confirmDisconnect) onConfirm();
            }}
          >
            {t('cloudflare.continue')}
          </Button>
        </Popconfirm>
      </div>
    </div>
  );
};
