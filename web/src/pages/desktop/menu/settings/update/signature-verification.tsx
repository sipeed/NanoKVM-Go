import { useEffect, useState } from 'react';
import { Input, Switch } from 'antd';
import { useTranslation } from 'react-i18next';

import * as applicationApi from '@/api/application.ts';
import * as authApi from '@/api/auth.ts';
import { useAuthRateLimit } from '@/hooks/useAuthRateLimit.ts';

import { SensitiveActionModal } from '../account/sensitive-action-modal.tsx';

type SignatureVerificationProps = {
  disabled: boolean;
  onBusyChange: (busy: boolean) => void;
};

function isTwoFactorStatus(value: unknown): value is authApi.TwoFactorStatus {
  return (
    typeof value === 'object' &&
    value !== null &&
    typeof (value as authApi.TwoFactorStatus).enabled === 'boolean'
  );
}

function isSignatureVerificationStatus(
  value: unknown
): value is applicationApi.SignatureVerificationStatus {
  return (
    typeof value === 'object' &&
    value !== null &&
    typeof (value as applicationApi.SignatureVerificationStatus).enabled === 'boolean'
  );
}

export const SignatureVerification = ({ disabled, onBusyChange }: SignatureVerificationProps) => {
  const { t } = useTranslation();
  const [isEnabled, setIsEnabled] = useState(true);
  const [isStatusAvailable, setIsStatusAvailable] = useState(false);
  const [isStatusLoading, setIsStatusLoading] = useState(true);
  const [isGateLoading, setIsGateLoading] = useState(false);
  const [pendingEnabled, setPendingEnabled] = useState<boolean | null>(null);
  const [isTwoFactorEnabled, setIsTwoFactorEnabled] = useState(true);
  const [password, setPassword] = useState('');
  const [code, setCode] = useState('');
  const [verificationError, setVerificationError] = useState('');
  const [controlError, setControlError] = useState('');
  const [isVerifying, setIsVerifying] = useState(false);
  const rateLimit = useAuthRateLimit();

  const isBusy = isStatusLoading || isGateLoading || isVerifying || pendingEnabled !== null;

  useEffect(() => {
    let active = true;
    setIsStatusLoading(true);

    applicationApi
      .getSignatureVerification()
      .then((rsp) => {
        if (!active) return;
        if (rsp.code !== 0 || !isSignatureVerificationStatus(rsp.data)) {
          throw new Error('invalid signature verification status');
        }
        setIsEnabled(rsp.data.enabled);
        setIsStatusAvailable(true);
        setControlError('');
      })
      .catch(() => {
        if (!active) return;
        // Fail closed: an unknown server state must never be presented as
        // permission to install unsigned updates.
        setIsEnabled(true);
        setIsStatusAvailable(false);
        setControlError(t('settings.update.signatureVerification.loadFailed'));
      })
      .finally(() => {
        if (active) setIsStatusLoading(false);
      });

    return () => {
      active = false;
    };
  }, [t]);

  useEffect(() => {
    onBusyChange(isBusy);
  }, [isBusy, onBusyChange]);

  useEffect(() => () => onBusyChange(false), [onBusyChange]);

  useEffect(() => {
    if (!rateLimit.isLocked) return;
    setVerificationError(t('settings.account.authRateLimited', { seconds: rateLimit.retryAfter }));
  }, [rateLimit.isLocked, rateLimit.retryAfter, t]);

  function clearVerification() {
    setPendingEnabled(null);
    setPassword('');
    setCode('');
    setVerificationError('');
    rateLimit.clear();
  }

  function requestChange(nextEnabled: boolean) {
    if (disabled || isBusy || !isStatusAvailable || nextEnabled === isEnabled) return;

    setIsGateLoading(true);
    setControlError('');
    authApi
      .getTwoFactorStatus()
      .then((rsp) => {
        const status = isTwoFactorStatus(rsp.data) ? rsp.data : null;
        if (rsp.code !== 0 || !status) throw new Error('invalid two-factor status');
        setIsTwoFactorEnabled(status.enabled);
        setPendingEnabled(nextEnabled);
        setPassword('');
        setCode('');
        setVerificationError('');
        rateLimit.clear();
      })
      .catch(() => {
        // The backend accepts a different credential shape depending on 2FA
        // state. If that state is unknown, do not guess or open an unusable
        // authentication prompt.
        setControlError(t('settings.update.signatureVerification.changeFailed'));
      })
      .finally(() => setIsGateLoading(false));
  }

  function closeVerification() {
    if (isVerifying) return;
    clearVerification();
  }

  function submitVerification() {
    const nextEnabled = pendingEnabled;
    const trimmedCode = code.trim();
    if (
      nextEnabled === null ||
      isVerifying ||
      isGateLoading ||
      rateLimit.isLocked ||
      (isTwoFactorEnabled ? !trimmedCode : !password)
    ) {
      return;
    }

    setIsVerifying(true);
    setVerificationError('');

    authApi
      .verifySensitiveAction(
        'update_signature_verification',
        isTwoFactorEnabled ? undefined : password,
        isTwoFactorEnabled ? trimmedCode : ''
      )
      .then(async (rsp) => {
        const data = rsp.data as authApi.SensitiveActionData | undefined;
        if (rsp.code !== 0 || !data?.challenge) {
          const limit = authApi.getAuthRateLimitData(rsp);
          if (limit && rateLimit.lock(limit)) {
            setPassword('');
            setCode('');
            setVerificationError(
              t('settings.account.authRateLimited', { seconds: limit.retryAfter })
            );
            return;
          }
          setPassword('');
          setCode('');
          setVerificationError(t('auth.verificationFailed'));
          return;
        }

        const updateRsp = await applicationApi.setSignatureVerification(
          nextEnabled,
          data.challenge
        );
        if (
          updateRsp.code !== 0 ||
          !isSignatureVerificationStatus(updateRsp.data) ||
          updateRsp.data.enabled !== nextEnabled
        ) {
          setPassword('');
          setCode('');
          setVerificationError(t('settings.update.signatureVerification.changeFailed'));
          return;
        }

        setIsEnabled(updateRsp.data.enabled);
        setIsStatusAvailable(true);
        setControlError('');
        clearVerification();
      })
      .catch(() => {
        setPassword('');
        setCode('');
        setVerificationError(t('settings.update.signatureVerification.changeFailed'));
      })
      .finally(() => setIsVerifying(false));
  }

  const isDisabling = pendingEnabled === false;

  return (
    <>
      <div className="flex items-center justify-between gap-4 py-3">
        <div className="flex min-w-0 flex-col space-y-1">
          <span>{t('settings.update.signatureVerification.title')}</span>
          {!isEnabled && (
            <span className="text-xs text-red-400">
              {t('settings.update.signatureVerification.disabledDesc')}
            </span>
          )}
          {controlError && (
            <span role="alert" className="text-xs text-red-400">
              {controlError}
            </span>
          )}
        </div>

        <Switch
          aria-label={t('settings.update.signatureVerification.title')}
          checked={isEnabled}
          loading={isStatusLoading || isGateLoading}
          disabled={disabled || isBusy || !isStatusAvailable}
          onChange={requestChange}
        />
      </div>

      <SensitiveActionModal
        open={pendingEnabled !== null}
        title={t('settings.update.signatureVerification.title')}
        description={t(
          isDisabling
            ? 'settings.update.signatureVerification.disableAuthDesc'
            : 'settings.update.signatureVerification.enableAuthDesc'
        )}
        confirmText={t(
          isDisabling
            ? 'settings.update.signatureVerification.disableConfirm'
            : 'settings.update.signatureVerification.enableConfirm'
        )}
        danger={isDisabling}
        requirePassword={!isTwoFactorEnabled}
        password={password}
        onPasswordChange={setPassword}
        error={verificationError}
        isLoading={isVerifying || isGateLoading}
        isLocked={rateLimit.isLocked}
        onSubmit={submitVerification}
        onCancel={closeVerification}
        submitDisabled={isTwoFactorEnabled && !code.trim()}
        extra={
          isTwoFactorEnabled ? (
            <Input
              autoFocus
              value={code}
              autoComplete="one-time-code"
              placeholder={t('settings.account.authCodeOrRecoveryCode')}
              disabled={isVerifying || isGateLoading || rateLimit.isLocked}
              onChange={(event) => setCode(event.target.value)}
              onPressEnter={submitVerification}
            />
          ) : undefined
        }
      />
    </>
  );
};
