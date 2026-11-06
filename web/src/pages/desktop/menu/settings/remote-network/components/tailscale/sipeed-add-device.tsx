import { useCallback, useEffect, useRef, useState } from 'react';
import { CloudOutlined } from '@ant-design/icons';
import { Button, Card } from 'antd';

import * as api from '@/api/extensions/tailscale.ts';
import { useProtectionChallengeGate } from '@/hooks/useProtectionChallengeGate.tsx';

import { useRemoteNetworkMessages } from '../../messages.ts';
import type { Status } from './types.ts';

type SipeedAddDeviceProps = {
  status: Status;
  onFailure?: () => void;
  onStatus: (status: Status) => void;
};

function managedErrorKey(error: string): string | undefined {
  switch (error) {
    case 'officialIdentityActive':
      return 'settings.tailscale.sipeed.officialActive';
    case 'invalidToken':
      return 'settings.tailscale.sipeed.invalidIdentity';
    case 'identityMissing':
      return 'settings.tailscale.sipeed.identityMissing';
    case 'rateLimited':
      return 'settings.tailscale.sipeed.rateLimited';
    case 'pairingUnavailable':
      return 'settings.tailscale.sipeed.pairingUnavailable';
    case 'pairingUpstreamError':
      return 'settings.tailscale.sipeed.pairingUpstreamError';
    case 'pairingRejected':
      return 'settings.tailscale.sipeed.pairingRejected';
    case 'nodeNotFound':
      return 'settings.tailscale.sipeed.nodeNotFound';
    case 'protocolError':
      return 'settings.tailscale.sipeed.protocolError';
    case 'localAPIUnavailable':
      return 'settings.tailscale.sipeed.localAPIUnavailable';
    case 'stateFileInvalid':
      return 'settings.tailscale.sipeed.stateFileInvalid';
    case 'stateWriteFailed':
      return 'settings.tailscale.sipeed.stateWriteFailed';
    case 'installFailed':
      return 'settings.tailscale.sipeed.installFailed';
    case 'profileReadFailed':
      return 'settings.tailscale.sipeed.profileReadFailed';
    case 'hostnameUnavailable':
      return 'settings.tailscale.sipeed.hostnameUnavailable';
    case 'profileSetupFailed':
      return 'settings.tailscale.sipeed.profileSetupFailed';
    case 'profileStartFailed':
      return 'settings.tailscale.sipeed.profileStartFailed';
    case 'connectionFailed':
      return 'settings.tailscale.sipeed.connectionFailed';
    case 'profileVerificationFailed':
      return 'settings.tailscale.sipeed.profileVerificationFailed';
    case 'rollbackFailed':
      return 'settings.tailscale.sipeed.rollbackFailed';
    case 'interrupted':
      return 'settings.tailscale.sipeed.interrupted';
    case 'identityGenerationFailed':
      return 'settings.tailscale.sipeed.identityGenerationFailed';
    default:
      return undefined;
  }
}

/** First-identity action. The backend owns installation, enrollment and retry. */
export const SipeedAddDevice = ({ status, onFailure, onStatus }: SipeedAddDeviceProps) => {
  const t = useRemoteNetworkMessages();
  const { runWithChallenge, modal, isGateBusy } = useProtectionChallengeGate();
  const [isRequesting, setIsRequesting] = useState(false);
  const [requestFailed, setRequestFailed] = useState(false);
  const [autoStartPending, setAutoStartPending] = useState(
    () =>
      status.sipeedDisclaimerAccepted &&
      status.activeMode !== 'official' &&
      status.managedState === 'disabled' &&
      status.managedError.trim() === '' &&
      status.retryAfterSeconds <= 0
  );
  const startAttemptedRef = useRef(false);
  const clickGuardRef = useRef(false);
  const requestInFlightRef = useRef(false);

  const isBackgroundRunning =
    status.managedState === 'enrolling' || status.managedState === 'enabledConnecting';
  const isBusy = isRequesting || isGateBusy || isBackgroundRunning;
  const isStarting =
    isBackgroundRunning ||
    isRequesting ||
    (status.sipeedDisclaimerAccepted && (autoStartPending || isGateBusy));
  const hasManagedError = status.managedError.trim() !== '';
  const errorKey = managedErrorKey(status.managedError);

  const enable = useCallback(() => {
    if (isBusy || status.retryAfterSeconds > 0 || clickGuardRef.current) return;
    startAttemptedRef.current = true;
    clickGuardRef.current = true;

    void runWithChallenge('tailscale_login', 'tailscale', async (challenge) => {
      if (requestInFlightRef.current) return { code: -1 };
      requestInFlightRef.current = true;
      setIsRequesting(true);
      setRequestFailed(false);
      try {
        const rsp = await api.enableSipeed(challenge, !status.sipeedDisclaimerAccepted);
        if (rsp.code === -3) return rsp;
        if (rsp.data) {
          const nextStatus = rsp.data as Status;
          onStatus(nextStatus);
          // A structured managedError is authoritative, even when the envelope
          // itself carries a non-zero code.
          const failedWithoutManagedState = rsp.code !== 0 && !nextStatus.managedError;
          setRequestFailed(failedWithoutManagedState);
          if (failedWithoutManagedState) onFailure?.();
        } else if (rsp.code !== 0) {
          setRequestFailed(true);
          onFailure?.();
        }
        return rsp;
      } catch {
        setRequestFailed(true);
        onFailure?.();
        return { code: -1 };
      } finally {
        requestInFlightRef.current = false;
        setIsRequesting(false);
      }
    })
      .catch(() => {
        setRequestFailed(true);
        onFailure?.();
      })
      .finally(() => {
        setAutoStartPending(false);
        clickGuardRef.current = false;
      });
  }, [
    isBusy,
    onFailure,
    onStatus,
    runWithChallenge,
    status.retryAfterSeconds,
    status.sipeedDisclaimerAccepted
  ]);

  useEffect(() => {
    // Start once when entering this page with saved consent. Status refreshes,
    // failed attempts and canceled verification must not start another request.
    if (
      !autoStartPending ||
      startAttemptedRef.current ||
      !status.sipeedDisclaimerAccepted ||
      status.activeMode === 'official'
    )
      return;
    enable();
  }, [autoStartPending, enable, status.activeMode, status.sipeedDisclaimerAccepted]);

  return (
    <>
      <div className="flex flex-col items-center justify-center space-y-4">
        {!status.sipeedDisclaimerAccepted && (
          <Card
            className="w-full max-w-lg text-center"
            styles={{ body: { padding: 0 } }}
            title={t('settings.tailscale.sipeed.disclaimerTitle')}
          >
            <div
              aria-label={t('settings.tailscale.sipeed.disclaimerTitle')}
              className="max-h-[clamp(3rem,calc(80dvh-28rem),15rem)] space-y-3 overflow-y-auto overscroll-contain p-4 text-left text-sm leading-6"
              role="region"
              tabIndex={0}
            >
              {t('settings.tailscale.sipeed.enableDesc')
                .split('\n\n')
                .map((paragraph) => (
                  <p key={paragraph}>{paragraph}</p>
                ))}
            </div>
          </Card>
        )}

        {isStarting && (
          <div aria-live="polite" className="text-sm text-amber-300">
            {t('settings.tailscale.sipeed.enabling')}
          </div>
        )}

        {(hasManagedError ||
          status.managedState === 'degraded' ||
          status.managedState === 'invalidIdentity') && (
          <div className="w-full max-w-lg text-center text-sm text-red-400" role="alert">
            {status.managedError === 'rateLimited'
              ? t('settings.tailscale.sipeed.rateLimited', {
                  seconds: Math.max(0, status.retryAfterSeconds)
                })
              : t(errorKey ?? 'settings.tailscale.sipeed.enableFailed')}
          </div>
        )}

        {requestFailed && status.managedState === 'disabled' && (
          <div className="w-full max-w-lg text-center text-sm text-red-400" role="alert">
            {t('settings.tailscale.sipeed.enableFailed')}
          </div>
        )}

        {!isStarting && (
          <div className="flex flex-wrap items-center justify-center gap-3">
            <Button
              disabled={isBusy || status.retryAfterSeconds > 0}
              icon={<CloudOutlined />}
              loading={isRequesting}
              onClick={enable}
              shape="round"
              size="large"
              type="primary"
            >
              {!status.sipeedDisclaimerAccepted
                ? t('settings.tailscale.sipeed.acceptAndEnable')
                : t('settings.tailscale.sipeed.retry')}
            </Button>
          </div>
        )}
      </div>
      {modal}
    </>
  );
};
