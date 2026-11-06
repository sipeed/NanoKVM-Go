import { useEffect, useState } from 'react';

import * as api from '@/api/extensions/tailscale.ts';

import { RemoteNetworkStatusBadge } from '../../status-badge.tsx';
import type { RemoteNetworkStatus } from '../../types.ts';
import { toRemoteNetworkStatus } from './status-model.ts';

/** Home card badge: Tailscale owns how its own state maps to a card status. */
export const TailscaleStatus = () => {
  const [status, setStatus] = useState<RemoteNetworkStatus>();

  useEffect(() => {
    let isCancelled = false;

    api
      .getStatus()
      .then((rsp) => {
        if (isCancelled) return;
        setStatus(rsp.code === 0 && rsp.data ? toRemoteNetworkStatus(rsp.data) : 'bad');
      })
      .catch(() => {
        if (!isCancelled) {
          setStatus('bad');
        }
      });

    return () => {
      isCancelled = true;
    };
  }, []);

  return status ? <RemoteNetworkStatusBadge status={status} /> : null;
};
