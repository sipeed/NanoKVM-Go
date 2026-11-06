import { useEffect, useRef, useState } from 'react';

import * as api from '@/api/extensions/tailscale.ts';
import { useProtectionChallengeGate } from '@/hooks/useProtectionChallengeGate.tsx';

import { useRemoteNetworkMessages } from '../../messages.ts';
import { TunnelConnection } from '../../tunnel-controls.tsx';
import { toRemoteNetworkStatus } from './status-model.ts';
import type { Status, TailscaleModeId } from './types.ts';

type Action = 'start' | 'stop' | 'restart';

type ConnectionProps = {
  mode: TailscaleModeId;
  status: Status;
  onBusyChange: (busy: boolean) => void;
  onStatus: (status: Status) => void;
  onSuccess: (action: Action) => void | Promise<void>;
};

export const Connection = ({
  mode,
  status,
  onBusyChange,
  onStatus,
  onSuccess
}: ConnectionProps) => {
  const t = useRemoteNetworkMessages();
  const { runWithChallenge, modal, isGateBusy } = useProtectionChallengeGate();
  const [action, setAction] = useState<Action>();
  const [error, setError] = useState(false);
  const inFlight = useRef(false);
  const starting = action === 'start' || action === 'restart';
  const transitioning =
    status.managedState === 'enrolling' || status.managedState === 'enabledConnecting';
  const busy = action !== undefined || isGateBusy || transitioning;
  const active = status.state !== 'notRunning';
  const tone = starting ? 'busy' : active ? toRemoteNetworkStatus(status) : 'off';
  const state =
    starting || transitioning
      ? 'connecting'
      : !active
        ? 'stopped'
        : tone === 'bad'
          ? 'error'
          : status.state === 'running'
            ? 'connected'
            : status.state === 'stopped'
              ? 'offline'
              : 'notLogin';

  useEffect(() => {
    onBusyChange(action !== undefined || isGateBusy);
    return () => onBusyChange(false);
  }, [action, isGateBusy, onBusyChange]);

  function perform(next: Action) {
    if (busy || inFlight.current) return;
    inFlight.current = true;
    setError(false);
    const enableSipeed = next === 'start' && mode === 'sipeed';
    void runWithChallenge(
      enableSipeed ? 'tailscale_login' : `tailscale_${next}`,
      'tailscale',
      async (challenge) => {
        inFlight.current = true;
        setAction(next);
        try {
          const rsp = enableSipeed ? await api.enableSipeed(challenge) : await api[next](challenge);
          if (rsp.code === -3 || rsp.code === -4) return rsp;
          if (enableSipeed && rsp.data) onStatus(rsp.data as Status);
          if (rsp.code === 0) await onSuccess(next);
          else setError(true);
          return rsp;
        } catch {
          setError(true);
          return { code: -1 };
        } finally {
          inFlight.current = false;
          setAction(undefined);
        }
      }
    )
      .catch(() => setError(true))
      .finally(() => {
        inFlight.current = false;
        setAction(undefined);
      });
  }

  return (
    <>
      <TunnelConnection
        provider="tailscale"
        state={state}
        tone={tone}
        active={active}
        busy={busy}
        canStart={!transitioning}
        action={action}
        restartDescription={t('settings.tailscale.restart')}
        onAction={perform}
      />
      {error && (
        <div role="alert" className="text-red-500">
          {t('settings.tailscale.failed')}
        </div>
      )}
      {modal}
    </>
  );
};
