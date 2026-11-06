import { Popover } from 'antd';
import { ArrowUpRightIcon, BookOpenIcon, ChevronLeftIcon, EllipsisIcon } from 'lucide-react';

import { Tailscale as TailscaleIcon } from '@/components/icons/tailscale';
import { ButtonBase } from '@/components/ui/button-base.tsx';

import { useRemoteNetworkMessages } from '../../messages.ts';
import type { TailscaleModeId } from './modes.ts';
import { SipeedRevokeDisclaimer } from './sipeed-revoke-disclaimer.tsx';
import type { Status } from './types.ts';
import { Uninstall } from './uninstall.tsx';

type HeaderProps = {
  status: Status | undefined;
  title?: string;
  wikiMode: TailscaleModeId;
  /** True while the component holds the shell lock; back would be ignored anyway. */
  isBackDisabled: boolean;
  onBack: () => void;
  onSuccess: () => void;
  onSipeedLoggedOut?: () => void;
};

export const Header = ({
  status,
  title,
  wikiMode,
  isBackDisabled,
  onBack,
  onSuccess,
  onSipeedLoggedOut
}: HeaderProps) => {
  const t = useRemoteNetworkMessages();
  const wikiTitle = `${t(
    wikiMode === 'sipeed' ? 'settings.tailscale.mode.sipeed' : 'settings.tailscale.mode.official'
  )} Wiki`;

  return (
    <>
      <div className="flex items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2.5">
          <ButtonBase
            aria-label={t('settings.back')}
            className="flex size-6 shrink-0 items-center justify-center rounded-lg text-neutral-400 enabled:hover:bg-neutral-800 enabled:hover:text-neutral-100"
            disabled={isBackDisabled}
            onClick={onBack}
          >
            <ChevronLeftIcon size={18} aria-hidden="true" />
          </ButtonBase>
          <span
            className="hidden size-6 shrink-0 items-center justify-center rounded-lg border border-neutral-700/70 bg-neutral-800/60 text-neutral-200 min-[400px]:flex"
            aria-hidden="true"
          >
            <TailscaleIcon size={16} />
          </span>
          <span className="min-w-0 truncate text-base font-bold text-neutral-100">
            {title ?? t('settings.tailscale.title')}
          </span>
        </div>

        <div className="ml-auto flex shrink-0 items-center space-x-2">
          <a
            href={
              wikiMode === 'sipeed'
                ? 'https://wiki.sipeed.com/hardware/en/kvm/NanoKVM_Go/network/sipeed_remote_network.html'
                : 'https://wiki.sipeed.com/hardware/en/kvm/NanoKVM_Go/network/tailscale.html'
            }
            target="_blank"
            rel="noopener noreferrer"
            aria-label={wikiTitle}
            title={wikiTitle}
            className="inline-flex h-6 items-center gap-1.5 rounded-lg px-2 text-xs text-neutral-400 transition-colors hover:bg-neutral-800 hover:text-neutral-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
          >
            <BookOpenIcon size={14} aria-hidden="true" />
            Wiki
            <ArrowUpRightIcon size={12} aria-hidden="true" />
          </a>
          {/* more button */}
          {status?.state && status.state !== 'notInstall' && (
            <Popover
              content={
                <div className="flex min-w-[150px] flex-col">
                  <Uninstall onSuccess={onSuccess} />
                  {status?.activeMode === 'sipeed' &&
                    status.managedState === 'enabledConnected' && (
                      <SipeedRevokeDisclaimer
                        disabled={isBackDisabled}
                        onSuccess={onSipeedLoggedOut ?? onSuccess}
                      />
                    )}
                </div>
              }
              placement="bottom"
              trigger={isBackDisabled ? [] : ['click']}
            >
              <div className="flex cursor-pointer rounded p-0.5 text-white hover:bg-neutral-600">
                <EllipsisIcon size={18} />
              </div>
            </Popover>
          )}
        </div>
      </div>
    </>
  );
};
