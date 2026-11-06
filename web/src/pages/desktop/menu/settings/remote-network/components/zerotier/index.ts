import { createElement } from 'react';
import { NetworkIcon } from 'lucide-react';

import type { RemoteNetworkComponent } from '../../types.ts';
import { zerotierMessages } from './messages.ts';
import { ZeroTierStatusBadge } from './status.tsx';
import { ZeroTierView } from './view.tsx';

export const zerotierComponent: RemoteNetworkComponent = {
  id: 'zerotier',
  name: 'ZeroTier',
  Icon: ({ size }) => createElement(NetworkIcon, { size }),
  Status: ZeroTierStatusBadge,
  messages: zerotierMessages,
  View: ZeroTierView
};
