import { useEffect, useState } from 'react';

import { getStatus, type CloudflareStatus } from '@/api/extensions/cloudflare.ts';

import { RemoteNetworkStatusBadge } from '../../status-badge.tsx';
import type { RemoteNetworkStatus } from '../../types.ts';
import { toRemoteNetworkStatus } from './model.ts';

export const CloudflareStatusBadge = () => {
  const [status, setStatus] = useState<RemoteNetworkStatus>();
  useEffect(() => {
    const controller = new AbortController();
    void getStatus(controller.signal)
      .then((rsp) => {
        if (!controller.signal.aborted) {
          setStatus(rsp.code === 0 ? toRemoteNetworkStatus(rsp.data as CloudflareStatus) : 'bad');
        }
      })
      .catch(() => {
        if (!controller.signal.aborted) setStatus('bad');
      });
    return () => controller.abort();
  }, []);
  return status ? <RemoteNetworkStatusBadge status={status} /> : null;
};
