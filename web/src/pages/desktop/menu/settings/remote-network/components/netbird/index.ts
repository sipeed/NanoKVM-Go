import { NetBird as NetBirdIcon } from '@/components/icons/netbird';

import type { RemoteNetworkComponent } from '../../types.ts';
import { netbirdMessages } from './messages.ts';
import { NetBirdStatusBadge } from './status.tsx';
import { NetBirdView } from './view.tsx';

export const netbirdComponent: RemoteNetworkComponent = {
  id: 'netbird',
  name: 'NetBird',
  Icon: NetBirdIcon,
  Status: NetBirdStatusBadge,
  messages: netbirdMessages,
  View: NetBirdView
};
