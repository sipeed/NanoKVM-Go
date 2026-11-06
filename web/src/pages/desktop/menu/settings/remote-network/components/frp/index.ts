import { createElement } from 'react';
import { CableIcon } from 'lucide-react';

import type { RemoteNetworkComponent } from '../../types.ts';
import { frpMessages } from './messages.ts';
import { FRPStatusBadge } from './status.tsx';
import { FRPView } from './view.tsx';

export const frpComponent: RemoteNetworkComponent = {
  id: 'frp',
  name: 'FRP',
  Icon: ({ size }) => createElement(CableIcon, { size }),
  Status: FRPStatusBadge,
  messages: frpMessages,
  View: FRPView
};
