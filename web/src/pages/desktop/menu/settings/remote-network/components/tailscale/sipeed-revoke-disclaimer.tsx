import { useRef, useState } from 'react';
import { Modal } from 'antd';

import * as api from '@/api/extensions/tailscale.ts';
import { useProtectionChallengeGate } from '@/hooks/useProtectionChallengeGate.tsx';
import { ButtonBase } from '@/components/ui/button-base.tsx';

import { useRemoteNetworkMessages } from '../../messages.ts';

type SipeedRevokeDisclaimerProps = {
  disabled?: boolean;
  onSuccess: () => void;
};

export const SipeedRevokeDisclaimer = ({ disabled, onSuccess }: SipeedRevokeDisclaimerProps) => {
  const t = useRemoteNetworkMessages();
  const { runWithChallenge, modal, isGateBusy } = useProtectionChallengeGate();
  const [isOpen, setIsOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [failed, setFailed] = useState(false);
  const inFlightRef = useRef(false);
  const busy = isLoading || isGateBusy;

  function revoke() {
    if (disabled || busy || inFlightRef.current) return;
    inFlightRef.current = true;
    void runWithChallenge('tailscale_logout', 'tailscale', async (challenge) => {
      setIsLoading(true);
      setFailed(false);
      try {
        const rsp = await api.revokeSipeedDisclaimer(challenge);
        if (rsp.code === -3) return rsp;
        if (rsp.code !== 0) {
          setFailed(true);
          return rsp;
        }
        setIsOpen(false);
        onSuccess();
        return rsp;
      } catch {
        setFailed(true);
        return { code: -1 };
      } finally {
        setIsLoading(false);
      }
    }).finally(() => {
      inFlightRef.current = false;
    });
  }

  return (
    <>
      <ButtonBase
        className="flex min-h-[30px] items-center rounded px-3 py-0.5 text-left text-neutral-300 enabled:hover:bg-neutral-700/70"
        disabled={disabled || busy}
        onClick={() => {
          setFailed(false);
          setIsOpen(true);
        }}
      >
        {t('settings.tailscale.sipeed.revokeDisclaimer')}
      </ButtonBase>
      <Modal
        title={t('settings.tailscale.sipeed.revokeDisclaimer')}
        open={isOpen}
        okType="danger"
        okText={t('settings.tailscale.sipeed.revokeDisclaimer')}
        cancelText={t('settings.tailscale.cancelBtn')}
        confirmLoading={busy}
        okButtonProps={{ disabled }}
        onOk={revoke}
        onCancel={() => {
          if (!busy) setIsOpen(false);
        }}
      >
        <p className="py-5">{t('settings.tailscale.sipeed.revokeDisclaimerDesc')}</p>
        {failed && (
          <p className="text-red-400" role="alert">
            {t('settings.tailscale.sipeed.revokeDisclaimerFailed')}
          </p>
        )}
      </Modal>
      {modal}
    </>
  );
};
