import { useEffect, useState } from 'react';

import { getStatus, type ZeroTierStatus } from '@/api/extensions/zerotier.ts';

import { RemoteNetworkStatusBadge } from '../../status-badge.tsx';
import type { RemoteNetworkStatus } from '../../types.ts';
import { toRemoteNetworkStatus } from './model.ts';

export const ZeroTierStatusBadge = () => {
  const [status, setStatus] = useState<RemoteNetworkStatus>();
  useEffect(() => {
    const controller = new AbortController();
    void getStatus(controller.signal)
      .then((rsp) => {
        if (!controller.signal.aborted) {
          setStatus(rsp.code === 0 ? toRemoteNetworkStatus(rsp.data as ZeroTierStatus) : 'bad');
        }
      })
      .catch(() => {
        if (!controller.signal.aborted) setStatus('bad');
      });
    return () => controller.abort();
  }, []);
  return status ? <RemoteNetworkStatusBadge status={status} /> : null;
};
