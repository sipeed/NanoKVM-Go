import { useCallback, useEffect, useRef, useState } from 'react';
import { Input, Modal } from 'antd';
import { useTranslation } from 'react-i18next';

import * as authApi from '@/api/auth.ts';
import {
  resolveProtectionRequiresChallenge,
  type ProtectionGateKey,
  type ProtectionSensitiveAction
} from '@/lib/protection-challenge.ts';
import { useAuthRateLimit } from '@/hooks/useAuthRateLimit.ts';

/** Optional API result so the gate can reopen the modal on a sensitive-action response. */
export type ProtectionGateExecuteResult = { code: number } | void | undefined;

type PendingGate = {
  action: ProtectionSensitiveAction;
  execute: (challenge?: string) => Promise<ProtectionGateExecuteResult>;
};

function isVerificationRequired(result: ProtectionGateExecuteResult): boolean {
  return (
    typeof result === 'object' && result !== null && (result.code === -3 || result.code === -4)
  );
}

/**
 * Code-only challenge gate for advanced 2FA protection ops (SSH, Tailscale).
 * When the matching policy switch is off (or 2FA is off), execute runs without a modal.
 * If the backend still returns a sensitive-action error (gate probe mismatch),
 * the modal opens and retries.
 */
export function useProtectionChallengeGate() {
  const { t } = useTranslation();
  const [pending, setPending] = useState<PendingGate | null>(null);
  const [code, setCode] = useState('');
  const [error, setError] = useState('');
  const [isVerifying, setIsVerifying] = useState(false);
  const [isGateLoading, setIsGateLoading] = useState(false);
  const rateLimit = useAuthRateLimit();
  const isMountedRef = useRef(true);

  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  useEffect(() => {
    if (!rateLimit.isLocked) return;
    setError(t('settings.account.authRateLimited', { seconds: rateLimit.retryAfter }));
  }, [rateLimit.isLocked, rateLimit.retryAfter, t]);

  const clearForm = useCallback(() => {
    setCode('');
    setError('');
    rateLimit.clear();
  }, [rateLimit]);

  const close = useCallback(() => {
    if (isVerifying) return;
    setPending(null);
    clearForm();
  }, [clearForm, isVerifying]);

  const openModal = useCallback(
    (action: ProtectionSensitiveAction, execute: PendingGate['execute']) => {
      clearForm();
      setPending({ action, execute });
    },
    [clearForm]
  );

  const runWithChallenge = useCallback(
    async (
      action: ProtectionSensitiveAction,
      policyKey: ProtectionGateKey,
      execute: (challenge?: string) => Promise<ProtectionGateExecuteResult>
    ) => {
      if (isVerifying || isGateLoading || pending) return;

      setIsGateLoading(true);
      try {
        const { requireChallenge } = await resolveProtectionRequiresChallenge(policyKey);
        if (requireChallenge) {
          if (!isMountedRef.current) return;
          openModal(action, execute);
          return;
        }

        const result = await execute();
        // Backend may still require a challenge when the frontend probe is stale/wrong.
        if (isVerificationRequired(result) && isMountedRef.current) {
          openModal(action, execute);
        }
      } finally {
        if (isMountedRef.current) setIsGateLoading(false);
      }
    },
    [isGateLoading, isVerifying, openModal, pending]
  );

  const submit = useCallback(() => {
    const trimmed = code.trim();
    const current = pending;
    if (isVerifying || rateLimit.isLocked || !current || !trimmed) return;

    setIsVerifying(true);
    setError('');

    authApi
      .verifySensitiveAction(current.action, undefined, trimmed)
      .then(async (rsp) => {
        if (!isMountedRef.current) return;
        const data = rsp.data as authApi.SensitiveActionData | undefined;
        if (rsp.code !== 0 || !data?.challenge) {
          const limit = authApi.getAuthRateLimitData(rsp);
          if (limit && rateLimit.lock(limit)) {
            setCode('');
            setError(t('settings.account.authRateLimited', { seconds: limit.retryAfter }));
            return;
          }
          setCode('');
          setError(t('auth.verificationFailed'));
          return;
        }

        const challenge = data.challenge;
        setPending(null);
        clearForm();
        try {
          const result = await current.execute(challenge);
          if (isVerificationRequired(result) && isMountedRef.current) {
            setError(t('auth.verificationFailed'));
            openModal(current.action, current.execute);
          }
        } catch {
          // Callers surface their own API errors.
        }
      })
      .catch(() => {
        if (!isMountedRef.current) return;
        setCode('');
        setError(t('auth.verificationFailed'));
      })
      .finally(() => {
        if (isMountedRef.current) setIsVerifying(false);
      });
  }, [clearForm, code, openModal, pending, rateLimit, t, isVerifying]);

  const modal = (
    <Modal
      open={pending !== null}
      title={t('auth.twoFactorTitle')}
      okText={t('auth.verify')}
      cancelText={t('auth.cancel')}
      confirmLoading={isVerifying || isGateLoading}
      // Every caller renders this modal inside the settings Modal or a menu
      // Popover subtree, so Ant Design's zindexContext stacking keeps it above
      // the owning surface without an explicit zIndex.
      okButtonProps={{
        disabled: isGateLoading || rateLimit.isLocked || !code.trim()
      }}
      onOk={submit}
      onCancel={close}
      destroyOnHidden={true}
    >
      <div className="space-y-3 pt-2">
        <div className="text-sm text-neutral-300">{t('auth.sensitiveActionDesc')}</div>
        <Input
          autoFocus
          value={code}
          autoComplete="one-time-code"
          placeholder={t('settings.account.authCodeOrRecoveryCode')}
          disabled={isVerifying || isGateLoading || rateLimit.isLocked}
          onChange={(event) => setCode(event.target.value)}
          onPressEnter={submit}
        />
        {error && <div className="text-sm text-red-500">{error}</div>}
      </div>
    </Modal>
  );

  return {
    runWithChallenge,
    modal,
    isGateBusy: isVerifying || isGateLoading || pending !== null
  };
}
