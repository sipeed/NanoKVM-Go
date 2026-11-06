import { useCallback, useEffect, useRef, useState } from 'react';
import { Button, Collapse, Input, Modal, Switch, Tooltip } from 'antd';
import { CircleAlertIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/auth.ts';
import { useAuthRateLimit } from '@/hooks/useAuthRateLimit.ts';
import { ButtonBase } from '@/components/ui/button-base.tsx';

import { SensitiveActionModal } from './sensitive-action-modal.tsx';

type ProtectionKey = keyof api.TwoFactorProtectionPolicy;

type VisibleProtectionItem = {
  key: ProtectionKey;
  label: string;
  scope: string;
};

const policyKeys: ProtectionKey[] = [
  'login',
  'webTerminal',
  'ssh',
  'tailscale',
  'zerotier',
  'netbird',
  'frp',
  'cloudflare',
  'mcp',
  'targetPower',
  'nanoKVMRestart',
  'scripts',
  'screenTimelapse',
  'deleteImage'
];

const visibleProtectionKeys: ProtectionKey[] = [
  'login',
  'webTerminal',
  'ssh',
  'tailscale',
  'zerotier',
  'netbird',
  'frp',
  'cloudflare',
  'mcp',
  'targetPower',
  'nanoKVMRestart',
  'scripts',
  'screenTimelapse',
  'deleteImage'
];

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function isTwoFactorProtectionPolicy(value: unknown): value is api.TwoFactorProtectionPolicy {
  if (!isRecord(value)) return false;
  return policyKeys.every((key) => typeof value[key] === 'boolean');
}

function parseTwoFactorProtectionStatus(value: unknown): api.TwoFactorProtectionStatus | null {
  if (!isRecord(value) || !isTwoFactorProtectionPolicy(value.policy)) return null;
  if (typeof value.authorizationExpiresIn !== 'number' || value.authorizationExpiresIn < 0)
    return null;
  const mcpAuthorizationExpiresIn =
    typeof value.mcpAuthorizationExpiresIn === 'number' && value.mcpAuthorizationExpiresIn >= 0
      ? value.mcpAuthorizationExpiresIn
      : 0;
  const screenTimelapseAuthorizationExpiresIn =
    typeof value.screenTimelapseAuthorizationExpiresIn === 'number' &&
    value.screenTimelapseAuthorizationExpiresIn >= 0
      ? value.screenTimelapseAuthorizationExpiresIn
      : 0;

  return {
    policy: value.policy,
    authorizationExpiresIn: value.authorizationExpiresIn,
    mcpAuthorizationExpiresIn,
    screenTimelapseAuthorizationExpiresIn
  };
}

function parseTwoFactorProtectionAuthorization(
  value: unknown
): api.TwoFactorProtectionAuthorization | null {
  if (!isRecord(value) || typeof value.expiresIn !== 'number' || value.expiresIn <= 0) return null;
  return { expiresIn: value.expiresIn };
}

export const TwoFactorProtection = () => {
  const { t } = useTranslation();
  const [status, setStatus] = useState<api.TwoFactorProtectionStatus | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const [saveError, setSaveError] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [authorizationExpiresAt, setAuthorizationExpiresAt] = useState(0);
  const [pendingPolicy, setPendingPolicy] = useState<api.TwoFactorProtectionPolicy | null>(null);
  const [pendingScriptDisablePolicy, setPendingScriptDisablePolicy] =
    useState<api.TwoFactorProtectionPolicy | null>(null);
  const [scriptDisableCountdown, setScriptDisableCountdown] = useState(0);
  const [isAuthorizationOpen, setIsAuthorizationOpen] = useState(false);
  const [authorizationCode, setAuthorizationCode] = useState('');
  const [authorizationError, setAuthorizationError] = useState('');
  const [isAuthorizing, setIsAuthorizing] = useState(false);
  const authorizationRateLimit = useAuthRateLimit();
  const isMountedRef = useRef(true);
  const scriptDisableCountdownTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const visibleItems: VisibleProtectionItem[] = visibleProtectionKeys.map((key) => ({
    key,
    label: t(`settings.account.twoFactorProtection.${key}`),
    scope: t(`settings.account.twoFactorProtection.${key}Scope`)
  }));

  const loadProtection = useCallback(async () => {
    setIsLoading(true);
    setLoadError('');

    try {
      const rsp = await api.getTwoFactorProtection();
      const nextStatus = rsp.code === 0 ? parseTwoFactorProtectionStatus(rsp.data) : null;
      if (!nextStatus) {
        if (isMountedRef.current) {
          setStatus(null);
          setAuthorizationExpiresAt(0);
          setLoadError(t('settings.account.twoFactorProtection.loadFailed'));
        }
        return;
      }

      if (!isMountedRef.current) return;
      setStatus(nextStatus);
      setAuthorizationExpiresAt(Date.now() + nextStatus.authorizationExpiresIn * 1000);
    } catch {
      if (isMountedRef.current) {
        setStatus(null);
        setAuthorizationExpiresAt(0);
        setLoadError(t('settings.account.twoFactorProtection.loadFailed'));
      }
    } finally {
      if (isMountedRef.current) setIsLoading(false);
    }
  }, [t]);

  useEffect(() => {
    isMountedRef.current = true;
    void loadProtection();

    return () => {
      isMountedRef.current = false;
      clearScriptDisableCountdown();
    };
  }, [loadProtection]);

  useEffect(() => {
    if (!authorizationRateLimit.isLocked) return;
    setAuthorizationError(
      t('settings.account.authRateLimited', { seconds: authorizationRateLimit.retryAfter })
    );
  }, [authorizationRateLimit.isLocked, authorizationRateLimit.retryAfter, t]);

  function clearAuthorizationForm() {
    setAuthorizationCode('');
    setAuthorizationError('');
    authorizationRateLimit.clear();
  }

  function openAuthorization(policy: api.TwoFactorProtectionPolicy) {
    setPendingPolicy(policy);
    clearAuthorizationForm();
    setIsAuthorizationOpen(true);
  }

  function closeAuthorization() {
    if (isAuthorizing) return;
    setIsAuthorizationOpen(false);
    setPendingPolicy(null);
    clearAuthorizationForm();
  }

  function clearScriptDisableCountdown() {
    if (!scriptDisableCountdownTimerRef.current) return;

    clearInterval(scriptDisableCountdownTimerRef.current);
    scriptDisableCountdownTimerRef.current = null;
  }

  function closeScriptDisableWarning() {
    clearScriptDisableCountdown();
    setPendingScriptDisablePolicy(null);
    setScriptDisableCountdown(0);
  }

  function openScriptDisableWarning(policy: api.TwoFactorProtectionPolicy) {
    clearScriptDisableCountdown();
    setPendingScriptDisablePolicy(policy);
    setScriptDisableCountdown(5);

    let remainingSeconds = 5;
    scriptDisableCountdownTimerRef.current = setInterval(() => {
      remainingSeconds -= 1;
      setScriptDisableCountdown(Math.max(remainingSeconds, 0));

      if (remainingSeconds <= 0) {
        clearScriptDisableCountdown();
      }
    }, 1000);
  }

  function applyAuthorizationRateLimit(rsp: { data?: unknown }) {
    const rateLimit = api.getAuthRateLimitData(rsp);
    if (!rateLimit || !authorizationRateLimit.lock(rateLimit)) return false;
    setAuthorizationError(t('settings.account.authRateLimited', { seconds: rateLimit.retryAfter }));
    return true;
  }

  async function savePolicy(policy: api.TwoFactorProtectionPolicy) {
    setIsSaving(true);
    setSaveError('');

    try {
      const rsp = await api.setTwoFactorProtection(policy);
      const nextStatus = rsp.code === 0 ? parseTwoFactorProtectionStatus(rsp.data) : null;
      if (nextStatus) {
        if (isMountedRef.current) {
          setStatus(nextStatus);
          setAuthorizationExpiresAt(Date.now() + nextStatus.authorizationExpiresIn * 1000);
        }
        return;
      }

      if (!isMountedRef.current) return;
      if (rsp.code === -4) {
        setAuthorizationExpiresAt(0);
        openAuthorization(policy);
        return;
      }
      setSaveError(t('settings.account.twoFactorProtection.updateFailed'));
    } catch {
      if (isMountedRef.current) {
        setSaveError(t('settings.account.twoFactorProtection.updateFailed'));
      }
    } finally {
      if (isMountedRef.current) setIsSaving(false);
    }
  }

  function applyPolicyChange(policy: api.TwoFactorProtectionPolicy) {
    if (authorizationExpiresAt > Date.now()) {
      void savePolicy(policy);
      return;
    }

    openAuthorization(policy);
  }

  function requestPolicyChange(key: ProtectionKey, enabled: boolean) {
    if (!status || isSaving || isAuthorizing || pendingScriptDisablePolicy) return;
    if (!visibleProtectionKeys.includes(key)) return;

    const nextPolicy = { ...status.policy, [key]: enabled };
    if (key === 'scripts' && !enabled) {
      openScriptDisableWarning(nextPolicy);
      return;
    }

    applyPolicyChange(nextPolicy);
  }

  function confirmScriptDisable() {
    const policy = pendingScriptDisablePolicy;
    if (!policy || scriptDisableCountdown > 0 || isSaving || isAuthorizing) return;

    closeScriptDisableWarning();
    applyPolicyChange(policy);
  }

  function submitAuthorization() {
    const code = authorizationCode.trim();
    const policy = pendingPolicy;
    if (isAuthorizing || authorizationRateLimit.isLocked || !policy || !code) {
      return;
    }

    setIsAuthorizing(true);
    setAuthorizationError('');

    api
      .authorizeTwoFactorProtection(code)
      .then((rsp) => {
        if (!isMountedRef.current) return;
        const authorization =
          rsp.code === 0 ? parseTwoFactorProtectionAuthorization(rsp.data) : null;
        if (!authorization) {
          if (applyAuthorizationRateLimit(rsp)) {
            setAuthorizationCode('');
            return;
          }

          setAuthorizationCode('');
          setAuthorizationError(t('settings.account.twoFactorProtection.authorizeFailed'));
          return;
        }

        setAuthorizationExpiresAt(Date.now() + authorization.expiresIn * 1000);
        setIsAuthorizationOpen(false);
        setPendingPolicy(null);
        clearAuthorizationForm();
        void savePolicy(policy);
      })
      .catch(() => {
        if (!isMountedRef.current) return;
        setAuthorizationCode('');
        setAuthorizationError(t('settings.account.twoFactorProtection.authorizeFailed'));
      })
      .finally(() => {
        if (isMountedRef.current) setIsAuthorizing(false);
      });
  }

  const panelHeader = (
    <div className="max-w-2xl space-y-1 pr-2 text-left">
      <div id="two-factor-protection-title" className="text-sm font-semibold text-neutral-100">
        {t('settings.account.twoFactorProtection.title')}
      </div>
      <p className="text-xs font-normal leading-5 text-neutral-400">
        {t('settings.account.twoFactorProtection.description')}
      </p>
    </div>
  );

  const panelBody = (
    <>
      {isLoading && (
        <div aria-busy="true" className="mt-2 space-y-3">
          <div className="h-3 w-24 animate-pulse bg-neutral-800" />
          <div className="h-14 animate-pulse border border-neutral-800 bg-neutral-950/30" />
          <div className="h-14 animate-pulse border border-neutral-800 bg-neutral-950/30" />
        </div>
      )}

      {!isLoading && loadError && (
        <div
          role="alert"
          className="mt-2 flex flex-col gap-3 border border-red-900/70 bg-red-950/20 p-3 text-sm text-red-300 sm:flex-row sm:items-center sm:justify-between"
        >
          <span>{loadError}</span>
          <Button type="text" size="small" onClick={() => void loadProtection()}>
            {t('settings.account.twoFactorProtection.retry')}
          </Button>
        </div>
      )}

      {!isLoading && status && (
        <div className="mt-2 space-y-3">
          {saveError && (
            <div
              role="alert"
              className="border border-red-900/70 bg-red-950/20 p-3 text-sm text-red-300"
            >
              {saveError}
            </div>
          )}

          <div className="overflow-hidden border border-neutral-800 bg-neutral-950/20">
            {visibleItems.map((item, index) => {
              const labelID = `two-factor-protection-${item.key}-label`;
              return (
                <div
                  key={item.key}
                  className={
                    index === 0
                      ? 'flex flex-col gap-3 px-3 py-3 sm:flex-row sm:items-center sm:justify-between'
                      : 'flex flex-col gap-3 border-t border-neutral-800 px-3 py-3 sm:flex-row sm:items-center sm:justify-between'
                  }
                >
                  <div
                    id={labelID}
                    className="flex min-w-0 items-center gap-1 text-sm font-medium text-neutral-200"
                  >
                    <span>{item.label}</span>
                    <Tooltip
                      title={item.scope}
                      placement="top"
                      trigger={['hover', 'focus']}
                      styles={{ root: { maxWidth: '360px' } }}
                    >
                      <ButtonBase
                        aria-label={item.scope}
                        className="shrink-0 rounded text-neutral-500 transition-colors hover:text-neutral-300"
                      >
                        <CircleAlertIcon size={15} aria-hidden="true" />
                      </ButtonBase>
                    </Tooltip>
                  </div>
                  <Switch
                    checked={status.policy[item.key]}
                    aria-labelledby={labelID}
                    loading={isSaving}
                    disabled={isSaving || isAuthorizing || pendingScriptDisablePolicy !== null}
                    onChange={(enabled) => requestPolicyChange(item.key, enabled)}
                  />
                </div>
              );
            })}
          </div>
        </div>
      )}
    </>
  );

  return (
    <section
      aria-labelledby="two-factor-protection-title"
      className="border border-neutral-700/80 bg-neutral-900/40 p-4 sm:p-5"
    >
      <Collapse
        ghost
        expandIconPosition="end"
        items={[
          {
            key: 'two-factor-protection',
            label: panelHeader,
            children: panelBody
          }
        ]}
      />

      <SensitiveActionModal
        open={isAuthorizationOpen}
        title={t('settings.account.twoFactorProtection.authorizeTitle')}
        description={t('settings.account.twoFactorProtection.authorizeDesc')}
        confirmText={t('auth.verify')}
        requirePassword={false}
        error={authorizationError}
        isLoading={isAuthorizing}
        isLocked={authorizationRateLimit.isLocked}
        onSubmit={submitAuthorization}
        onCancel={closeAuthorization}
        submitDisabled={!authorizationCode.trim()}
        extra={
          <Input
            autoFocus
            value={authorizationCode}
            autoComplete="one-time-code"
            placeholder={t('settings.account.authCodeOrRecoveryCode')}
            disabled={isAuthorizing || authorizationRateLimit.isLocked}
            onChange={(event) => setAuthorizationCode(event.target.value)}
            onPressEnter={submitAuthorization}
          />
        }
      />

      <Modal
        open={pendingScriptDisablePolicy !== null}
        title={t('settings.account.twoFactorProtection.disableScriptsWarningTitle')}
        onCancel={closeScriptDisableWarning}
        footer={[
          <Button key="cancel" onClick={closeScriptDisableWarning}>
            {t('settings.account.twoFactorProtection.cancelDisabling')}
          </Button>,
          <Button
            key="continue"
            type="primary"
            danger
            disabled={scriptDisableCountdown > 0 || isSaving}
            onClick={confirmScriptDisable}
          >
            {scriptDisableCountdown > 0
              ? t('settings.account.twoFactorProtection.continueDisablingCountdown', {
                  seconds: scriptDisableCountdown
                })
              : t('settings.account.twoFactorProtection.continueDisabling')}
          </Button>
        ]}
      >
        <p className="text-sm leading-6 text-neutral-300">
          {t('settings.account.twoFactorProtection.disableScriptsWarningDesc')}
        </p>
      </Modal>
    </section>
  );
};
