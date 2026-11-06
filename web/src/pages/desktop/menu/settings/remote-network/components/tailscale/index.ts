import { Tailscale as TailscaleIcon } from '@/components/icons/tailscale';

import type { RemoteNetworkComponent } from '../../types.ts';
import { tailscaleMessages } from './messages.ts';
import { TailscaleStatus } from './status.tsx';
import { TailscaleView } from './view.tsx';

/** Tailscale as a remote network component: one registry entry, nothing else. */
export const tailscaleComponent: RemoteNetworkComponent = {
  id: 'tailscale',
  name: 'Tailscale',
  Icon: TailscaleIcon,
  Status: TailscaleStatus,
  messages: tailscaleMessages,
  View: TailscaleView
};
