import clsx from 'clsx';
import type { TFunction } from 'i18next';
import { useTranslation } from 'react-i18next';

import type { RemoteNetworkStatus } from './types.ts';

const toneClass: Record<RemoteNetworkStatus, string> = {
  off: 'text-neutral-400',
  busy: 'text-amber-400',
  on: 'text-green-500',
  bad: 'text-red-500'
};

function statusLabel(status: RemoteNetworkStatus, t: TFunction): string {
  switch (status) {
    case 'busy':
      return t('settings.remoteNetwork.status.busy');
    case 'on':
      return t('settings.remoteNetwork.status.on');
    case 'bad':
      return t('settings.remoteNetwork.status.bad');
    default:
      return t('settings.remoteNetwork.status.off');
  }
}

type RemoteNetworkStatusBadgeProps = {
  status: RemoteNetworkStatus;
};

/** Shared status chip for home cards: shape and text carry the state, not colour alone. */
export const RemoteNetworkStatusBadge = ({ status }: RemoteNetworkStatusBadgeProps) => {
  const { t } = useTranslation();

  return (
    <span className={clsx('inline-flex items-center gap-1.5 text-xs', toneClass[status])}>
      <span aria-hidden="true" className="size-1.5 shrink-0 rounded-full bg-current" />
      {statusLabel(status, t)}
    </span>
  );
};
