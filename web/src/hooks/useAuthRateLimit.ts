import { useEffect, useMemo, useState } from 'react';

import { type AuthRateLimitData } from '@/api/auth.ts';

export function useAuthRateLimit() {
  const [lockedUntilMs, setLockedUntilMs] = useState(0);
  const [nowMs, setNowMs] = useState(Date.now());

  useEffect(() => {
    if (lockedUntilMs <= Date.now()) return;

    const timer = window.setInterval(() => {
      const now = Date.now();
      setNowMs(now);
      if (now >= lockedUntilMs) {
        window.clearInterval(timer);
      }
    }, 1000);
    return () => window.clearInterval(timer);
  }, [lockedUntilMs]);

  const retryAfter = Math.max(0, Math.ceil((lockedUntilMs - nowMs) / 1000));
  const isLocked = retryAfter > 0;

  return useMemo(
    () => ({
      isLocked,
      retryAfter,
      lock(data: AuthRateLimitData | null | undefined) {
        if (!data?.rateLimited || typeof data.retryAfter !== 'number') return false;
        const seconds = Math.max(1, Math.ceil(data.retryAfter));
        const now = Date.now();
        setNowMs(now);
        setLockedUntilMs(now + seconds * 1000);
        return true;
      },
      clear() {
        setLockedUntilMs(0);
        setNowMs(Date.now());
      }
    }),
    [isLocked, retryAfter]
  );
}
