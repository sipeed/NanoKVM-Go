import { CheckOutlined, CopyOutlined, ReloadOutlined } from '@ant-design/icons';
import { Button, Card } from 'antd';
import clsx from 'clsx';

import { useRemoteNetworkMessages } from '../../messages.ts';

function formatRemaining(seconds: number): string {
  const minutes = Math.floor(seconds / 60);
  const rest = seconds % 60;

  return `${String(minutes).padStart(2, '0')}:${String(rest).padStart(2, '0')}`;
}

type SipeedClaimCodeProps = {
  code: string;
  secondsLeft: number;
  isCopied: boolean;
  isLoading?: boolean;
  isRenewDisabled?: boolean;
  onCopy: () => void;
  onRenew: () => void;
};

/** Renders the server-issued one-time code and its client-side expiry clock. */
export const SipeedClaimCode = ({
  code,
  secondsLeft,
  isCopied,
  isLoading = false,
  isRenewDisabled = false,
  onCopy,
  onRenew
}: SipeedClaimCodeProps) => {
  const t = useRemoteNetworkMessages();
  const hasCode = code.length > 0;
  const isExpired = secondsLeft === 0;

  return (
    <div className="flex flex-col items-center gap-6">
      <h3 className="self-start text-sm font-semibold text-neutral-200">
        {t('settings.tailscale.sipeed.addDevice')}
      </h3>

      <Card aria-busy={isLoading} className="w-full">
        <div className="flex flex-col items-center gap-3">
          <span className="h-10 select-all font-mono text-4xl font-semibold tracking-[0.18em]">
            {hasCode ? `${code.slice(0, 3)} ${code.slice(3)}` : '--- ---'}
          </span>

          <span
            className={clsx('min-h-4 text-xs', isExpired ? 'text-red-400' : 'text-neutral-500')}
            role="status"
          >
            {hasCode
              ? isExpired
                ? t('settings.tailscale.sipeed.claimExpired')
                : t('settings.tailscale.sipeed.claimExpiry', {
                    time: formatRemaining(secondsLeft)
                  })
              : '\u00a0'}
          </span>

          <div className="flex flex-wrap items-center justify-center gap-2 pt-1">
            <Button
              disabled={!hasCode || isExpired || isLoading}
              icon={isCopied ? <CheckOutlined className="text-green-500" /> : <CopyOutlined />}
              onClick={onCopy}
              shape="round"
            >
              {t('settings.tailscale.sipeed.claimCopy')}
            </Button>

            <Button
              disabled={isRenewDisabled || isLoading}
              icon={<ReloadOutlined />}
              loading={isLoading}
              onClick={onRenew}
              shape="round"
              type={isExpired ? 'primary' : 'default'}
            >
              {t('settings.tailscale.sipeed.claimRenew')}
            </Button>
          </div>
        </div>
      </Card>
    </div>
  );
};
