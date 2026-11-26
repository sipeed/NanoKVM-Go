import { useEffect, useState } from 'react';

import { wheelStats } from '@/lib/wheel.ts';

// Diagnostic readout for wheel scrolling, shown when the URL hash contains
// "wheel-debug" (https://<kvm>/#wheel-debug). Rates are per second over the last
// half second; it never touches input, it only reads the wheel counters.
const SAMPLE_MS = 500;

function enabled(): boolean {
  return window.location.hash.includes('wheel-debug');
}

export const WheelDebug = () => {
  const [visible, setVisible] = useState(enabled);
  const [line, setLine] = useState('waiting for wheel events');

  useEffect(() => {
    const onHash = () => setVisible(enabled());
    window.addEventListener('hashchange', onHash);
    return () => window.removeEventListener('hashchange', onHash);
  }, []);

  useEffect(() => {
    if (!visible) return;
    let last = { ...wheelStats };
    const timer = window.setInterval(() => {
      const now = { ...wheelStats };
      const rate = (key: 'events' | 'notches' | 'sent') =>
        Math.round(((now[key] - last[key]) * 1000) / SAMPLE_MS);
      setLine(
        `events ${rate('events')}/s · notches ${rate('notches')}/s · sent ${rate('sent')}/s · ` +
          `step ${now.pixelsPerNotch} px · gain ${now.gain.toFixed(2)} · direction ${now.direction === 1 ? 'Natural (1)' : now.direction === -1 ? 'Conventional (-1)' : '—'} · total sent ${now.sent}`
      );
      last = now;
    }, SAMPLE_MS);
    return () => window.clearInterval(timer);
  }, [visible]);

  if (!visible) return null;
  return (
    <div className="pointer-events-none fixed bottom-2 left-2 z-[1000] rounded bg-black/75 px-3 py-1.5 font-mono text-xs text-green-300">
      {line}
    </div>
  );
};
