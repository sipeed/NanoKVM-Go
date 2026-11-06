import { createElement } from 'react';
import { CloudIcon } from 'lucide-react';

import type { RemoteNetworkComponent } from '../../types.ts';
import { cloudflareMessages } from './messages.ts';
import { CloudflareStatusBadge } from './status.tsx';
import { CloudflareView } from './view.tsx';

export const cloudflareComponent: RemoteNetworkComponent = {
  id: 'cloudflare',
  name: 'Cloudflare Tunnel',
  Icon: ({ size }) => createElement(CloudIcon, { size }),
  Status: CloudflareStatusBadge,
  messages: cloudflareMessages,
  View: CloudflareView
};
