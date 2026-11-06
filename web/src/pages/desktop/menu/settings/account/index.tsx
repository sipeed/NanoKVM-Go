import { useEffect, useRef, useState } from 'react';
import { Button, Divider, Input, Modal, Tooltip } from 'antd';
import { CircleAlertIcon, CopyIcon, DownloadIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import * as api from '@/api/auth.ts';
import { expireAuthSession } from '@/lib/auth-session.ts';
import { feedback } from '@/lib/feedback.tsx';
import { useAuthRateLimit } from '@/hooks/useAuthRateLimit.ts';

import { Logout } from './logout.tsx';
import { SensitiveActionModal } from './sensitive-action-modal.tsx';
import { AccountSettingRow } from './setting-row.tsx';
import { TwoFactorGuide } from './two-factor-guide.tsx';
import { TwoFactorProtection } from './two-factor-protection.tsx';
import { TwoFactorSetup } from './two-factor-setup.tsx';

type RecoveryCodesAction = 'enable' | 'regenerate';

export const Account = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();

  const [username, setUsername] = useState('');
  const [twoFactorStatus, setTwoFactorStatus] = useState<api.TwoFactorStatus | null>(null);
  const [isTwoFactorStatusLoading, setIsTwoFactorStatusLoading] = useState(false);
  const [isSetupPasswordModalOpen, setIsSetupPasswordModalOpen] = useState(false);
  const [setupPassword, setSetupPassword] = useState('');
  const [setupError, setSetupError] = useState('');
  const [isSetupLoading, setIsSetupLoading] = useState(false);
  const [pendingSetup, setPendingSetup] = useState<api.TwoFactorSetupData | null>(null);
  const [setupCode, setSetupCode] = useState('');
  const [setupVerifyError, setSetupVerifyError] = useState('');
  const [isSetupVerifyLoading, setIsSetupVerifyLoading] = useState(false);
  const [isSetupCancelLoading, setIsSetupCancelLoading] = useState(false);
  const pendingSetupRef = useRef<api.TwoFactorSetupData | null>(null);
  const isPasswordVerificationMountedRef = useRef(true);
  const [isDisableModalOpen, setIsDisableModalOpen] = useState(false);
  const [disablePassword, setDisablePassword] = useState('');
  const [disableCode, setDisableCode] = useState('');
  const [disableError, setDisableError] = useState('');
  const [isDisableLoading, setIsDisableLoading] = useState(false);
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([]);
  const [recoveryCodesAction, setRecoveryCodesAction] = useState<RecoveryCodesAction | null>(null);
  const [recoveryPassword, setRecoveryPassword] = useState('');
  const [recoveryCode, setRecoveryCode] = useState('');
  const [recoveryError, setRecoveryError] = useState('');
  const [isRecoveryLoading, setIsRecoveryLoading] = useState(false);
  const [isPasswordVerificationModalOpen, setIsPasswordVerificationModalOpen] = useState(false);
  const [passwordVerificationPassword, setPasswordVerificationPassword] = useState('');
  const [passwordVerificationCode, setPasswordVerificationCode] = useState('');
  const [passwordVerificationError, setPasswordVerificationError] = useState('');
  const [isPasswordVerificationLoading, setIsPasswordVerificationLoading] = useState(false);
  const passwordVerificationRateLimit = useAuthRateLimit();
  const setupRateLimit = useAuthRateLimit();
  const setupVerifyRateLimit = useAuthRateLimit();
  const disableRateLimit = useAuthRateLimit();
  const recoveryRateLimit = useAuthRateLimit();

  useEffect(() => {
    api.getAccount().then((rsp) => {
      if (rsp.code === 0) {
        setUsername(rsp.data.username);
      }
    });

    loadTwoFactorStatus();
  }, []);

  useEffect(() => {
    return () => {
      isPasswordVerificationMountedRef.current = false;
      if (!pendingSetupRef.current) return;

      pendingSetupRef.current = null;
      api.cancelTwoFactorSetup().catch(() => undefined);
    };
  }, []);

  useEffect(() => {
    if (!passwordVerificationRateLimit.isLocked) return;
    setPasswordVerificationError(
      t('settings.account.authRateLimited', { seconds: passwordVerificationRateLimit.retryAfter })
    );
  }, [passwordVerificationRateLimit.isLocked, passwordVerificationRateLimit.retryAfter, t]);

  useEffect(() => {
    if (!setupRateLimit.isLocked) return;
    setSetupError(t('settings.account.authRateLimited', { seconds: setupRateLimit.retryAfter }));
  }, [setupRateLimit.isLocked, setupRateLimit.retryAfter, t]);

  useEffect(() => {
    if (!setupVerifyRateLimit.isLocked) return;
    setSetupVerifyError(
      t('settings.account.authRateLimited', { seconds: setupVerifyRateLimit.retryAfter })
    );
  }, [setupVerifyRateLimit.isLocked, setupVerifyRateLimit.retryAfter, t]);

  useEffect(() => {
    if (!disableRateLimit.isLocked) return;
    setDisableError(
      t('settings.account.authRateLimited', { seconds: disableRateLimit.retryAfter })
    );
  }, [disableRateLimit.isLocked, disableRateLimit.retryAfter, t]);

  useEffect(() => {
    if (!recoveryRateLimit.isLocked) return;
    setRecoveryError(
      t('settings.account.authRateLimited', { seconds: recoveryRateLimit.retryAfter })
    );
  }, [recoveryRateLimit.isLocked, recoveryRateLimit.retryAfter, t]);

  function applyRateLimit(
    rsp: { data?: unknown },
    limiter: ReturnType<typeof useAuthRateLimit>,
    setError: (message: string) => void
  ) {
    const rateLimit = api.getAuthRateLimitData(rsp);
    if (!rateLimit || !limiter.lock(rateLimit)) return false;
    setError(t('settings.account.authRateLimited', { seconds: rateLimit.retryAfter }));
    return true;
  }

  function loadTwoFactorStatus() {
    setIsTwoFactorStatusLoading(true);

    api
      .getTwoFactorStatus()
      .then((rsp) => {
        if (rsp.code !== 0 || !rsp.data) {
          setTwoFactorStatus(null);
          return;
        }

        const status = rsp.data as api.TwoFactorStatus;
        setTwoFactorStatus(status);
        if (!status.enabled) {
          setRecoveryCodes([]);
        }
      })
      .catch(() => {
        setTwoFactorStatus(null);
      })
      .finally(() => {
        setIsTwoFactorStatusLoading(false);
      });
  }

  function changePassword() {
    if (twoFactorStatus?.enabled) {
      openPasswordVerificationModal();
      return;
    }

    navigate('/auth/password');
  }

  function clearPasswordVerificationState() {
    setPasswordVerificationPassword('');
    setPasswordVerificationCode('');
    setPasswordVerificationError('');
    setIsPasswordVerificationLoading(false);
    passwordVerificationRateLimit.clear();
  }

  function openPasswordVerificationModal() {
    clearPasswordVerificationState();
    setIsPasswordVerificationModalOpen(true);
  }

  function closePasswordVerificationModal() {
    if (isPasswordVerificationLoading) return;

    setIsPasswordVerificationModalOpen(false);
    clearPasswordVerificationState();
  }

  function submitPasswordVerification() {
    const password = passwordVerificationPassword;
    const code = passwordVerificationCode.trim();
    if (
      isPasswordVerificationLoading ||
      passwordVerificationRateLimit.isLocked ||
      !password ||
      !code
    )
      return;
    setIsPasswordVerificationLoading(true);
    setPasswordVerificationError('');

    api
      .verifySensitiveAction('change_password', password, code)
      .then((rsp) => {
        if (!isPasswordVerificationMountedRef.current) return;

        const data = rsp.data as api.SensitiveActionData | undefined;
        if (rsp.code !== 0 || !data?.challenge || typeof data.expiresIn !== 'number') {
          if (applyRateLimit(rsp, passwordVerificationRateLimit, setPasswordVerificationError)) {
            setPasswordVerificationCode('');
            return;
          }

          setPasswordVerificationCode('');
          setPasswordVerificationError(t('settings.account.verifyPasswordChangeFailed'));
          return;
        }

        setIsPasswordVerificationModalOpen(false);
        clearPasswordVerificationState();
        navigate('/auth/password', {
          state: {
            sensitiveAction: 'change_password',
            challenge: data.challenge,
            expiresAt: Date.now() + data.expiresIn * 1000
          }
        });
      })
      .catch(() => {
        if (!isPasswordVerificationMountedRef.current) return;

        setPasswordVerificationCode('');
        setPasswordVerificationError(t('settings.account.verifyPasswordChangeFailed'));
      })
      .finally(() => {
        if (!isPasswordVerificationMountedRef.current) return;

        setIsPasswordVerificationLoading(false);
      });
  }

  function setPendingTwoFactorSetup(setup: api.TwoFactorSetupData | null) {
    pendingSetupRef.current = setup;
    setPendingSetup(setup);
  }

  function clearPendingSetupState() {
    setPendingTwoFactorSetup(null);
    setSetupCode('');
    setSetupVerifyError('');
    setIsSetupVerifyLoading(false);
    setIsSetupCancelLoading(false);
    setupVerifyRateLimit.clear();
  }

  function openSetupPasswordModal() {
    setSetupPassword('');
    setSetupError('');
    setupRateLimit.clear();
    setIsSetupPasswordModalOpen(true);
  }

  function closeSetupPasswordModal() {
    if (isSetupLoading) return;

    setIsSetupPasswordModalOpen(false);
    setSetupPassword('');
    setSetupError('');
    setupRateLimit.clear();
  }

  function submitTwoFactorSetup() {
    if (isSetupLoading || setupRateLimit.isLocked || !setupPassword) return;
    setIsSetupLoading(true);
    setSetupError('');

    api
      .setupTwoFactor(setupPassword)
      .then((rsp) => {
        if (rsp.code !== 0 || !rsp.data) {
          if (applyRateLimit(rsp, setupRateLimit, setSetupError)) return;

          setSetupError(t('settings.account.twoFactorSetupFailed'));
          return;
        }

        setPendingTwoFactorSetup(rsp.data as api.TwoFactorSetupData);
        setSetupCode('');
        setSetupVerifyError('');
        setIsSetupPasswordModalOpen(false);
        setSetupPassword('');
      })
      .catch(() => {
        setSetupError(t('settings.account.twoFactorSetupFailed'));
      })
      .finally(() => {
        setIsSetupLoading(false);
      });
  }

  function verifyAndEnableTwoFactor() {
    const code = setupCode.trim();
    if (isSetupVerifyLoading || setupVerifyRateLimit.isLocked || !pendingSetup || !code) return;
    setIsSetupVerifyLoading(true);
    setSetupVerifyError('');

    api
      .enableTwoFactor(code)
      .then((rsp) => {
        if (rsp.code !== 0) {
          if (applyRateLimit(rsp, setupVerifyRateLimit, setSetupVerifyError)) {
            setSetupCode('');
            return;
          }

          setSetupCode('');
          setSetupVerifyError(t('settings.account.twoFactorVerifyFailed'));
          return;
        }

        clearPendingSetupState();
        expireAuthSession('twoFactorEnabled');
      })
      .catch(() => {
        setSetupCode('');
        setSetupVerifyError(t('settings.account.twoFactorVerifyFailed'));
      })
      .finally(() => {
        setIsSetupVerifyLoading(false);
      });
  }

  function cancelPendingTwoFactorSetup() {
    if (isSetupCancelLoading || !pendingSetup) return;
    setIsSetupCancelLoading(true);

    api
      .cancelTwoFactorSetup()
      .catch(() => undefined)
      .finally(() => {
        clearPendingSetupState();
        loadTwoFactorStatus();
      });
  }

  function openDisableModal() {
    setDisablePassword('');
    setDisableCode('');
    setDisableError('');
    disableRateLimit.clear();
    setIsDisableModalOpen(true);
  }

  function closeDisableModal() {
    if (isDisableLoading) return;

    setIsDisableModalOpen(false);
    setDisablePassword('');
    setDisableCode('');
    setDisableError('');
    disableRateLimit.clear();
  }

  function submitDisableTwoFactor() {
    const password = disablePassword;
    const code = disableCode.trim();
    if (isDisableLoading || disableRateLimit.isLocked || !password || !code) return;
    setIsDisableLoading(true);
    setDisableError('');

    api
      .disableTwoFactor(password, code)
      .then((rsp) => {
        if (rsp.code !== 0) {
          if (applyRateLimit(rsp, disableRateLimit, setDisableError)) {
            setDisableCode('');
            return;
          }

          setDisableCode('');
          setDisableError(t('settings.account.twoFactorDisableFailed'));
          return;
        }

        setIsDisableModalOpen(false);
        setDisablePassword('');
        setDisableCode('');
        setDisableError('');
        setRecoveryCodes([]);
        loadTwoFactorStatus();
      })
      .catch(() => {
        setDisableCode('');
        setDisableError(t('settings.account.twoFactorDisableFailed'));
      })
      .finally(() => {
        setIsDisableLoading(false);
      });
  }

  function openRecoveryCodesModal(action: RecoveryCodesAction) {
    setRecoveryPassword('');
    setRecoveryCode('');
    setRecoveryError('');
    recoveryRateLimit.clear();
    setRecoveryCodesAction(action);
  }

  function closeRecoveryCodesModal() {
    if (isRecoveryLoading) return;

    setRecoveryCodesAction(null);
    setRecoveryPassword('');
    setRecoveryCode('');
    setRecoveryError('');
    recoveryRateLimit.clear();
  }

  function submitRecoveryCodesAction() {
    const password = recoveryPassword;
    const code = recoveryCode.trim();
    if (
      isRecoveryLoading ||
      recoveryRateLimit.isLocked ||
      !recoveryCodesAction ||
      !password ||
      !code
    )
      return;
    setIsRecoveryLoading(true);
    setRecoveryError('');

    const request =
      recoveryCodesAction === 'enable' ? api.enableRecoveryCodes : api.regenerateRecoveryCodes;

    request(password, code)
      .then((rsp) => {
        const data = rsp.data as api.RecoveryCodesData | undefined;
        if (rsp.code !== 0 || !data || !Array.isArray(data.recoveryCodes)) {
          if (applyRateLimit(rsp, recoveryRateLimit, setRecoveryError)) {
            setRecoveryCode('');
            return;
          }

          setRecoveryCode('');
          setRecoveryError(t('settings.account.recoveryCodesVerifyFailed'));
          return;
        }

        setRecoveryCodes(data.recoveryCodes);
        setRecoveryCodesAction(null);
        setRecoveryPassword('');
        setRecoveryCode('');
        setRecoveryError('');
        loadTwoFactorStatus();
      })
      .catch(() => {
        setRecoveryCode('');
        setRecoveryError(t('settings.account.recoveryCodesVerifyFailed'));
      })
      .finally(() => {
        setIsRecoveryLoading(false);
      });
  }

  async function copyRecoveryCodes() {
    if (!recoveryCodes.length) return;

    try {
      await navigator.clipboard.writeText(recoveryCodes.join('\n'));
    } catch (err) {
      console.log(err);
      feedback.error(t('settings.account.recoveryCodesCopyFailed'));
    }
  }

  function downloadRecoveryCodes() {
    if (!recoveryCodes.length) return;

    const blob = new Blob([`${recoveryCodes.join('\n')}\n`], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = 'nanokvm-recovery-codes.txt';
    link.click();
    URL.revokeObjectURL(url);
  }

  function clearRecoveryCodes() {
    setRecoveryCodes([]);
  }

  function renderTwoFactorAction() {
    if (pendingSetup) {
      return null;
    }

    if (!twoFactorStatus) {
      return (
        <Button disabled={true} loading={isTwoFactorStatusLoading}>
          -
        </Button>
      );
    }

    if (twoFactorStatus.enabled) {
      return (
        <Button danger={true} onClick={openDisableModal}>
          {t('settings.account.disableTwoFactor')}
        </Button>
      );
    }

    return (
      <Button type="primary" onClick={openSetupPasswordModal}>
        {t('settings.account.enableTwoFactor')}
      </Button>
    );
  }

  function renderRecoveryCodesLabel() {
    const isTwoFactorEnabled = twoFactorStatus?.enabled === true;
    const tooltip = isTwoFactorEnabled
      ? t('settings.account.recoveryCodesTooltipEnabled')
      : t('settings.account.recoveryCodesTooltipDisabled');

    return (
      <div className="min-w-0 space-y-1">
        <div className="flex min-w-0 items-center space-x-2">
          <span className="truncate">{t('settings.account.recoveryCodes')}</span>

          <Tooltip
            title={tooltip}
            className="cursor-pointer text-neutral-500/60"
            placement="bottom"
            styles={{ root: { maxWidth: '350px' } }}
          >
            <CircleAlertIcon size={15} />
          </Tooltip>
        </div>
      </div>
    );
  }

  function renderRecoveryCodesAction() {
    if (!twoFactorStatus) {
      return (
        <Button disabled={true} loading={isTwoFactorStatusLoading}>
          -
        </Button>
      );
    }

    if (!twoFactorStatus.enabled) {
      return <Button disabled={true}>{t('settings.account.enableRecoveryCodes')}</Button>;
    }

    if (twoFactorStatus.recoveryEnabled) {
      return (
        <Button onClick={() => openRecoveryCodesModal('regenerate')}>
          {t('settings.account.updateRecoveryCodes')}
        </Button>
      );
    }

    return (
      <Button type="primary" onClick={() => openRecoveryCodesModal('enable')}>
        {t('settings.account.enableRecoveryCodes')}
      </Button>
    );
  }

  function renderRecoveryCodesResult() {
    if (!recoveryCodes.length) return null;

    return (
      <div className="space-y-4 border border-neutral-700 bg-neutral-800/40 p-4">
        <div className="space-y-1">
          <div className="text-sm font-medium text-neutral-100">
            {t('settings.account.recoveryCodesSavedTitle')}
          </div>
          <div className="text-xs text-neutral-500">
            {t('settings.account.recoveryCodesSavedDesc')}
          </div>
        </div>

        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          {recoveryCodes.map((code) => (
            <div
              key={code}
              className="select-all border border-neutral-700 bg-neutral-900/70 px-3 py-2 font-mono text-sm text-neutral-100"
            >
              {code}
            </div>
          ))}
        </div>

        <div className="flex flex-wrap justify-end gap-2">
          <Button icon={<CopyIcon size={15} />} onClick={copyRecoveryCodes}>
            {t('settings.account.copyRecoveryCodes')}
          </Button>
          <Button icon={<DownloadIcon size={15} />} onClick={downloadRecoveryCodes}>
            {t('settings.account.downloadRecoveryCodes')}
          </Button>
          <Button type="primary" onClick={clearRecoveryCodes}>
            {t('settings.account.done')}
          </Button>
        </div>
      </div>
    );
  }

  return (
    <>
      <div className="text-base font-bold">{t('settings.account.title')}</div>
      <Divider className="opacity-50" />

      <div className="flex flex-col space-y-5">
        <AccountSettingRow label={t('settings.account.webAccount')}>
          <Button type="text">{username ? username : '-'}</Button>
        </AccountSettingRow>

        <AccountSettingRow label={t('settings.account.password')}>
          <Button type="primary" onClick={changePassword}>
            {t('settings.account.updateBtn')}
          </Button>
        </AccountSettingRow>

        <AccountSettingRow
          label={
            <div className="flex min-w-0 items-center space-x-2">
              <span>{t('settings.account.twoFactor')}</span>
              <TwoFactorGuide />
            </div>
          }
        >
          {renderTwoFactorAction()}
        </AccountSettingRow>

        {twoFactorStatus?.enabled && (
          <AccountSettingRow label={renderRecoveryCodesLabel()}>
            {renderRecoveryCodesAction()}
          </AccountSettingRow>
        )}

        {renderRecoveryCodesResult()}

        {twoFactorStatus?.enabled && <TwoFactorProtection />}

        {pendingSetup && (
          <TwoFactorSetup
            setup={pendingSetup}
            code={setupCode}
            error={setupVerifyError}
            isVerifying={isSetupVerifyLoading}
            isCancelling={isSetupCancelLoading}
            isLocked={setupVerifyRateLimit.isLocked}
            onCodeChange={setSetupCode}
            onVerify={verifyAndEnableTwoFactor}
            onCancel={cancelPendingTwoFactorSetup}
          />
        )}
      </div>

      <Divider className="opacity-50" style={{ margin: '40px 0' }} />

      <Logout />

      <SensitiveActionModal
        open={isSetupPasswordModalOpen}
        title={t('settings.account.enableTwoFactor')}
        description={t('settings.account.enableTwoFactorDesc')}
        confirmText={t('settings.account.enableTwoFactor')}
        password={setupPassword}
        onPasswordChange={setSetupPassword}
        error={setupError}
        isLoading={isSetupLoading}
        isLocked={setupRateLimit.isLocked}
        onSubmit={submitTwoFactorSetup}
        onCancel={closeSetupPasswordModal}
      />

      <SensitiveActionModal
        open={recoveryCodesAction !== null}
        title={
          recoveryCodesAction === 'regenerate'
            ? t('settings.account.updateRecoveryCodes')
            : t('settings.account.enableRecoveryCodes')
        }
        description={
          recoveryCodesAction === 'regenerate'
            ? t('settings.account.recoveryCodesRegenerateDesc')
            : t('settings.account.enableRecoveryCodesDesc')
        }
        confirmText={
          recoveryCodesAction === 'regenerate'
            ? t('settings.account.updateRecoveryCodes')
            : t('settings.account.enableRecoveryCodes')
        }
        danger={recoveryCodesAction === 'regenerate'}
        password={recoveryPassword}
        onPasswordChange={setRecoveryPassword}
        error={recoveryError}
        isLoading={isRecoveryLoading}
        isLocked={recoveryRateLimit.isLocked}
        onSubmit={submitRecoveryCodesAction}
        onCancel={closeRecoveryCodesModal}
        submitDisabled={!recoveryCode.trim()}
        extra={
          <Input
            value={recoveryCode}
            autoComplete="one-time-code"
            placeholder={t(
              recoveryCodesAction === 'regenerate'
                ? 'settings.account.authCodeOrRecoveryCode'
                : 'settings.account.authCode'
            )}
            disabled={isRecoveryLoading || recoveryRateLimit.isLocked}
            onChange={(event) => setRecoveryCode(event.target.value)}
            onPressEnter={submitRecoveryCodesAction}
          />
        }
      />

      <Modal
        open={isPasswordVerificationModalOpen}
        title={t('settings.account.verifyPasswordChange')}
        okText={t('settings.account.verifyPasswordChange')}
        cancelText={t('settings.account.cancelBtn')}
        okButtonProps={{
          disabled:
            passwordVerificationRateLimit.isLocked ||
            !passwordVerificationPassword ||
            !passwordVerificationCode.trim()
        }}
        confirmLoading={isPasswordVerificationLoading}
        onOk={submitPasswordVerification}
        onCancel={closePasswordVerificationModal}
        destroyOnHidden={true}
      >
        <div className="space-y-3 pt-2">
          <div className="text-sm text-neutral-300">
            {t('settings.account.verifyPasswordChangeDesc')}
          </div>

          <Input.Password
            value={passwordVerificationPassword}
            autoComplete="current-password"
            placeholder={t('settings.account.currentPassword')}
            disabled={isPasswordVerificationLoading || passwordVerificationRateLimit.isLocked}
            onChange={(event) => setPasswordVerificationPassword(event.target.value)}
            onPressEnter={submitPasswordVerification}
          />

          <Input
            value={passwordVerificationCode}
            autoComplete="one-time-code"
            placeholder={t('settings.account.authCodeOrRecoveryCode')}
            disabled={isPasswordVerificationLoading || passwordVerificationRateLimit.isLocked}
            onChange={(event) => setPasswordVerificationCode(event.target.value)}
            onPressEnter={submitPasswordVerification}
          />

          {passwordVerificationError && (
            <div className="text-sm text-red-500">{passwordVerificationError}</div>
          )}
        </div>
      </Modal>

      <SensitiveActionModal
        open={isDisableModalOpen}
        title={t('settings.account.disableTwoFactor')}
        description={t('settings.account.disableTwoFactorDesc')}
        confirmText={t('settings.account.disableTwoFactor')}
        danger={true}
        password={disablePassword}
        onPasswordChange={setDisablePassword}
        error={disableError}
        isLoading={isDisableLoading}
        isLocked={disableRateLimit.isLocked}
        onSubmit={submitDisableTwoFactor}
        onCancel={closeDisableModal}
        submitDisabled={!disableCode.trim()}
        extra={
          <Input
            value={disableCode}
            autoComplete="one-time-code"
            placeholder={t('settings.account.authCodeOrRecoveryCode')}
            disabled={isDisableLoading || disableRateLimit.isLocked}
            onChange={(event) => setDisableCode(event.target.value)}
            onPressEnter={submitDisableTwoFactor}
          />
        }
      />
    </>
  );
};
