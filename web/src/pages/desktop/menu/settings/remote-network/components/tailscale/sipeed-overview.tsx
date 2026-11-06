import { useCallback, useEffect, useRef, useState } from 'react';
import { CheckOutlined, CopyOutlined } from '@ant-design/icons';
import { Button, Card, Divider, Dropdown, Input, Modal } from 'antd';
import { CircleCheckIcon, EllipsisIcon, PlusIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/extensions/tailscale.ts';
import { getIntlLocale } from '@/i18n/locale.ts';
import { feedback } from '@/lib/feedback.tsx';
import { useProtectionChallengeGate } from '@/hooks/useProtectionChallengeGate.tsx';

import { useRemoteNetworkMessages } from '../../messages.ts';
import { SipeedClaimCode } from './sipeed-claim-code.tsx';
import type { Status } from './types.ts';

const PENDING_POLL_MIN_INTERVAL_MS = 1000;
const PENDING_POLL_RETRY_INTERVAL_MS = 3000;
const SIPEED_CONTROL_URL = 'https://control.tailnet.sipeed.com';

type ResponseEnvelope<T = unknown> = {
  code: number;
  msg?: string;
  data?: T;
};

type SipeedOverviewProps = {
  isAddPageOpen: boolean;
  status: Status;
  onAddPageOpenChange: (isOpen: boolean) => void;
  onIdentityExpired: () => void;
};

function isAbortError(error: unknown): boolean {
  return error instanceof Error && (error.name === 'AbortError' || error.name === 'CanceledError');
}

function retryAfterSeconds(error: unknown): number {
  const response = (error as { response?: { headers?: Record<string, string>; status?: number } })
    ?.response;
  if (response?.status !== 429) return 0;
  const value = response.headers?.['retry-after'];
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? Math.ceil(parsed) : 60;
}

function formatLastSeen(
  value: string | undefined,
  locale: string,
  now: number,
  justNow: string
): string {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;

  const elapsed = Math.max(0, now - date.getTime());
  if (elapsed < 60_000) return justNow;

  const units: Array<[Intl.RelativeTimeFormatUnit, number]> = [
    ['year', 365 * 24 * 60 * 60 * 1000],
    ['month', 30 * 24 * 60 * 60 * 1000],
    ['day', 24 * 60 * 60 * 1000],
    ['hour', 60 * 60 * 1000],
    ['minute', 60 * 1000]
  ];
  const [unit, milliseconds] =
    units.find(([, threshold]) => elapsed >= threshold) ?? units[units.length - 1];

  try {
    return new Intl.RelativeTimeFormat(locale, { numeric: 'auto' }).format(
      -Math.floor(elapsed / milliseconds),
      unit
    );
  } catch {
    return date.toLocaleString(locale);
  }
}

function waitForAbortableDelay(delay: number, signal: AbortSignal): Promise<void> {
  if (delay === 0 || signal.aborted) return Promise.resolve();

  return new Promise((resolve) => {
    let timer: number;
    const finish = () => {
      window.clearTimeout(timer);
      signal.removeEventListener('abort', finish);
      resolve();
    };
    timer = window.setTimeout(finish, delay);
    signal.addEventListener('abort', finish, { once: true });
  });
}

function waitForPendingPoll(startedAt: number, signal: AbortSignal): Promise<void> {
  return waitForAbortableDelay(
    Math.max(0, PENDING_POLL_MIN_INTERVAL_MS - (Date.now() - startedAt)),
    signal
  );
}

function isValidBackupCode(value: string): boolean {
  return /^[23456789ABCDEFGHJKMNPQRSTVWXYZ]{8}$/.test(value.replace(/-/g, ''));
}

export const SipeedOverview = ({
  isAddPageOpen,
  status,
  onAddPageOpenChange,
  onIdentityExpired
}: SipeedOverviewProps) => {
  const t = useRemoteNetworkMessages();
  const { i18n } = useTranslation();
  const { runWithChallenge, modal, isGateBusy } = useProtectionChallengeGate();
  const [isCopied, setIsCopied] = useState(false);
  const [isClaimCopied, setIsClaimCopied] = useState(false);
  const [claim, setClaim] = useState<api.SipeedClaimCode | null>(null);
  const [claimNow, setClaimNow] = useState(() => Date.now());
  const [pending, setPending] = useState<api.SipeedPendingRequest[]>([]);
  const [pendingNames, setPendingNames] = useState<Record<string, string>>({});
  const [backupCode, setBackupCode] = useState('');
  const [backupName, setBackupName] = useState('');
  const [devices, setDevices] = useState<api.SipeedDevice[]>([]);
  const [isLoadingClaim, setIsLoadingClaim] = useState(false);
  const [isAuthorizing, setIsAuthorizing] = useState(false);
  const [busyPendingId, setBusyPendingId] = useState('');
  const [editingNodeId, setEditingNodeId] = useState('');
  const [editingName, setEditingName] = useState('');
  const [removingDevice, setRemovingDevice] = useState<api.SipeedDevice | null>(null);
  const [error, setError] = useState('');
  const [rateLimitUntil, setRateLimitUntil] = useState(0);
  const mountedRef = useRef(true);
  const claimRequestRef = useRef(0);
  const claimInFlightRef = useRef(false);

  const rateLimitSeconds = Math.max(0, Math.ceil((rateLimitUntil - claimNow) / 1000));
  const secondsLeft = claim
    ? Math.max(0, Math.ceil((new Date(claim.expiresAt).getTime() - claimNow) / 1000))
    : 0;
  const isRateLimited = rateLimitSeconds > 0;
  const clockInterval = isAddPageOpen || isRateLimited ? 1000 : 60_000;
  const isBackupCodeValid = isValidBackupCode(backupCode);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  useEffect(() => {
    setClaimNow(Date.now());
    const timer = window.setInterval(() => setClaimNow(Date.now()), clockInterval);
    return () => window.clearInterval(timer);
  }, [clockInterval]);

  useEffect(() => {
    if (isAddPageOpen) return;
    claimRequestRef.current += 1;
    claimInFlightRef.current = false;
    setClaim(null);
    setIsLoadingClaim(false);
  }, [isAddPageOpen]);

  const recordRetryAfter = useCallback((seconds: number) => {
    if (seconds <= 0) return;
    const now = Date.now();
    setClaimNow(now);
    setRateLimitUntil((current) => Math.max(current, now + seconds * 1000));
  }, []);

  const handleEnvelope = useCallback(
    <T,>(
      response: ResponseEnvelope<T>,
      failureKey = 'settings.tailscale.sipeed.managementFailed'
    ): boolean => {
      if (response.code === 0) return true;
      const data = response.data as Partial<Status> | undefined;
      if (response.code === -2 && data?.managedState === 'disabled') {
        onIdentityExpired();
        return false;
      }
      if (response.code === -2 && typeof data?.retryAfterSeconds === 'number') {
        recordRetryAfter(data.retryAfterSeconds);
      }
      setError(
        response.code === -2 && isRateLimitData(data)
          ? t('settings.tailscale.sipeed.rateLimited', {
              seconds: Math.max(1, data?.retryAfterSeconds ?? 1)
            })
          : t(failureKey)
      );
      return false;
    },
    [onIdentityExpired, recordRetryAfter, t]
  );

  const handleRequestError = useCallback(
    (requestError: unknown, failureKey = 'settings.tailscale.sipeed.managementFailed') => {
      if (isAbortError(requestError)) return;
      const seconds = retryAfterSeconds(requestError);
      if (seconds > 0) recordRetryAfter(seconds);
      const statusCode = (requestError as { response?: { status?: number } })?.response?.status;
      if (statusCode === 401) {
        onIdentityExpired();
        return;
      }
      setError(t(failureKey));
    },
    [onIdentityExpired, recordRetryAfter, t]
  );

  const refreshDevices = useCallback(async () => {
    if (isRateLimited) return;
    try {
      const response = (await api.getSipeedDevices()) as ResponseEnvelope<api.SipeedDevices>;
      if (
        !mountedRef.current ||
        !handleEnvelope(response, 'settings.tailscale.sipeed.refreshFailed')
      )
        return;
      setDevices(response.data?.devices ?? []);
    } catch (requestError) {
      if (mountedRef.current) {
        handleRequestError(requestError, 'settings.tailscale.sipeed.refreshFailed');
      }
    }
  }, [handleEnvelope, handleRequestError, isRateLimited]);

  useEffect(() => {
    void refreshDevices();
    const timer = window.setInterval(() => void refreshDevices(), 15_000);
    return () => window.clearInterval(timer);
  }, [refreshDevices]);

  const closeAddPage = useCallback(() => {
    claimRequestRef.current += 1;
    claimInFlightRef.current = false;
    onAddPageOpenChange(false);
    setClaim(null);
    setIsClaimCopied(false);
    setIsLoadingClaim(false);
    setPending([]);
    setPendingNames({});
    setBackupCode('');
    setBackupName('');
  }, [onAddPageOpenChange]);

  const requestClaimCode = useCallback(
    (refresh = false) => {
      if (claimInFlightRef.current || isGateBusy || isRateLimited) return;
      const requestId = ++claimRequestRef.current;
      setClaim(null);
      setIsClaimCopied(false);
      setError('');
      void runWithChallenge('tailscale_login', 'tailscale', async (challenge) => {
        if (!mountedRef.current || requestId !== claimRequestRef.current) return;
        claimInFlightRef.current = true;
        setIsLoadingClaim(true);
        try {
          const response = (await api.getSipeedClaimCode(
            challenge,
            refresh
          )) as ResponseEnvelope<api.SipeedClaimCode>;
          if (!mountedRef.current || requestId !== claimRequestRef.current) return;
          if (response.code === -3) return response;
          if (!handleEnvelope(response)) return response;
          setClaim(response.data ?? null);
          setClaimNow(Date.now());
          setPending([]);
          setPendingNames({});
          return response;
        } catch (requestError) {
          if (mountedRef.current && requestId === claimRequestRef.current) {
            handleRequestError(requestError);
          }
          return { code: -1 };
        } finally {
          if (mountedRef.current && requestId === claimRequestRef.current) {
            claimInFlightRef.current = false;
            setIsLoadingClaim(false);
          }
        }
      });
    },
    [handleEnvelope, handleRequestError, isGateBusy, isRateLimited, runWithChallenge]
  );

  const openAddPage = useCallback(() => {
    setClaim(null);
    setPending([]);
    setPendingNames({});
    setBackupCode('');
    setBackupName('');
    onAddPageOpenChange(true);
    requestClaimCode();
  }, [onAddPageOpenChange, requestClaimCode]);

  useEffect(() => {
    // The upstream returns immediately while any pending request exists, so a
    // long poll must only live while the one-time claim code is visible.  The
    // code is consumed by the first claim and is hidden as soon as that batch
    // arrives; keeping the loop alive after that would turn it into a one
    // request-per-second short poll until the user acts.
    if (!isAddPageOpen || claim === null || pending.length > 0) return;
    const requestId = claimRequestRef.current;
    const controller = new AbortController();
    let cancelled = false;

    const poll = async () => {
      while (!cancelled) {
        if (isRateLimited) return;
        const startedAt = Date.now();
        try {
          const response = (await api.getSipeedPending(controller.signal)) as ResponseEnvelope<{
            pending: api.SipeedPendingRequest[];
          }>;
          if (cancelled || !mountedRef.current || requestId !== claimRequestRef.current) return;
          if (!handleEnvelope(response)) {
            // -2 represents an invalid identity or a persisted Retry-After
            // deadline. Both require user/state recovery rather than another
            // request. Ordinary upstream failures are transient and mirror the
            // reference console's paced retry.
            if (response.code === -2) return;
            await waitForAbortableDelay(PENDING_POLL_RETRY_INTERVAL_MS, controller.signal);
            continue;
          }
          const nextPending = response.data?.pending ?? [];
          setPending(nextPending);
          setPendingNames((current) => {
            const next: Record<string, string> = {};
            for (const item of nextPending) {
              if (current[item.pendingId] !== undefined) {
                next[item.pendingId] = current[item.pendingId];
              }
            }
            return next;
          });
          if (nextPending.length > 0) {
            setClaim(null);
          }
          await waitForPendingPoll(startedAt, controller.signal);
        } catch (requestError) {
          if (cancelled || controller.signal.aborted || requestId !== claimRequestRef.current)
            return;
          const statusCode = (requestError as { response?: { status?: number } })?.response?.status;
          const seconds = retryAfterSeconds(requestError);
          handleRequestError(requestError);
          if (statusCode === 401 || seconds > 0) return;
          await waitForAbortableDelay(PENDING_POLL_RETRY_INTERVAL_MS, controller.signal);
        }
      }
    };

    void poll();
    return () => {
      cancelled = true;
      controller.abort();
    };
  }, [claim, handleEnvelope, handleRequestError, isAddPageOpen, isRateLimited, pending.length]);

  const copyIP = useCallback(async () => {
    if (!status.ip) return;
    try {
      await navigator.clipboard.writeText(status.ip);
      setIsCopied(true);
      window.setTimeout(() => setIsCopied(false), 2000);
      feedback.success(t('settings.tailscale.sipeed.ipCopied'));
    } catch {
      feedback.error(t('clipboard.copyFailed'));
    }
  }, [status.ip, t]);

  const copyClaimCode = useCallback(async () => {
    if (!claim || secondsLeft === 0) return;
    try {
      await navigator.clipboard.writeText(claim.code);
      setIsClaimCopied(true);
      window.setTimeout(() => setIsClaimCopied(false), 2000);
      feedback.success(t('settings.tailscale.sipeed.claimCopied'));
    } catch {
      feedback.error(t('clipboard.copyFailed'));
    }
  }, [claim, secondsLeft, t]);

  const approve = useCallback(
    (pendingId: string) => {
      if (busyPendingId || isRateLimited) return;
      void runWithChallenge('tailscale_login', 'tailscale', async (challenge) => {
        setBusyPendingId(pendingId);
        setError('');
        try {
          const response = (await api.approveSipeedPending(
            pendingId,
            pendingNames[pendingId],
            challenge
          )) as ResponseEnvelope;
          if (response.code === -3) return response;
          if (!handleEnvelope(response)) return response;
          closeAddPage();
          await refreshDevices();
          feedback.success(t('settings.tailscale.sipeed.pendingAllow'));
          return response;
        } catch (requestError) {
          handleRequestError(requestError);
          return { code: -1 };
        } finally {
          setBusyPendingId('');
        }
      });
    },
    [
      busyPendingId,
      closeAddPage,
      handleEnvelope,
      handleRequestError,
      isRateLimited,
      pendingNames,
      refreshDevices,
      runWithChallenge,
      t
    ]
  );

  const reject = useCallback(
    (pendingId: string) => {
      if (busyPendingId || isRateLimited) return;
      void runWithChallenge('tailscale_login', 'tailscale', async (challenge) => {
        setBusyPendingId(pendingId);
        setError('');
        try {
          const response = (await api.rejectSipeedPending(
            pendingId,
            challenge
          )) as ResponseEnvelope;
          if (response.code === -3) return response;
          if (!handleEnvelope(response)) return response;
          setPending((current) => current.filter((item) => item.pendingId !== pendingId));
          setPendingNames((current) => {
            const next = { ...current };
            delete next[pendingId];
            return next;
          });
          return response;
        } catch (requestError) {
          handleRequestError(requestError);
          return { code: -1 };
        } finally {
          setBusyPendingId('');
        }
      });
    },
    [busyPendingId, handleEnvelope, handleRequestError, isRateLimited, runWithChallenge]
  );

  const authorizeBackup = useCallback(() => {
    if (isAuthorizing || isRateLimited || !isValidBackupCode(backupCode)) return;
    void runWithChallenge('tailscale_login', 'tailscale', async (challenge) => {
      setIsAuthorizing(true);
      setError('');
      try {
        const response = (await api.authorizeSipeed(
          backupCode,
          backupName,
          challenge
        )) as ResponseEnvelope;
        if (response.code === -3) return response;
        if (!handleEnvelope(response, 'settings.tailscale.sipeed.authorizeFailed')) {
          return response;
        }
        setBackupCode('');
        setBackupName('');
        closeAddPage();
        await refreshDevices();
        feedback.success(t('settings.tailscale.sipeed.pendingAllow'));
        return response;
      } catch (requestError) {
        handleRequestError(requestError, 'settings.tailscale.sipeed.authorizeFailed');
        return { code: -1 };
      } finally {
        setIsAuthorizing(false);
      }
    });
  }, [
    backupCode,
    backupName,
    closeAddPage,
    handleEnvelope,
    handleRequestError,
    isAuthorizing,
    isRateLimited,
    refreshDevices,
    runWithChallenge,
    t
  ]);

  const rename = useCallback(
    (nodeId: string) => {
      if (!editingName.trim() || isRateLimited) return;
      void runWithChallenge('tailscale_login', 'tailscale', async (challenge) => {
        try {
          const response = (await api.renameSipeedDevice(
            nodeId,
            editingName.trim(),
            challenge
          )) as ResponseEnvelope;
          if (response.code === -3) return response;
          if (!handleEnvelope(response)) return response;
          setEditingNodeId('');
          setEditingName('');
          await refreshDevices();
          return response;
        } catch (requestError) {
          handleRequestError(requestError);
          return { code: -1 };
        }
      });
    },
    [
      editingName,
      handleEnvelope,
      handleRequestError,
      isRateLimited,
      refreshDevices,
      runWithChallenge
    ]
  );

  const remove = useCallback(
    (device: api.SipeedDevice) => {
      if (device.isSelf || isRateLimited) return;
      void runWithChallenge('tailscale_login', 'tailscale', async (challenge) => {
        try {
          const response = (await api.removeSipeedDevice(
            device.nodeId,
            challenge
          )) as ResponseEnvelope;
          if (response.code === -3) return response;
          if (!handleEnvelope(response)) return response;
          await refreshDevices();
          return response;
        } catch (requestError) {
          handleRequestError(requestError);
          return { code: -1 };
        }
      });
    },
    [handleEnvelope, handleRequestError, isRateLimited, refreshDevices, runWithChallenge]
  );

  const locale = getIntlLocale(i18n.resolvedLanguage || i18n.language);

  if (isAddPageOpen) {
    return (
      <>
        <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
          <Card>
            <div className="flex flex-col gap-5">
              {pending.length === 0 && (
                <>
                  <p className="text-sm text-neutral-400">
                    {t('settings.tailscale.sipeed.claimHint', {
                      server: SIPEED_CONTROL_URL
                    })}
                  </p>
                  <SipeedClaimCode
                    code={claim?.code.replace(/\s/g, '') ?? ''}
                    secondsLeft={secondsLeft}
                    isCopied={isClaimCopied}
                    isLoading={isLoadingClaim || isGateBusy}
                    onCopy={copyClaimCode}
                    isRenewDisabled={isRateLimited || isGateBusy}
                    onRenew={() => requestClaimCode(true)}
                  />
                </>
              )}

              {pending.map((item) => (
                <div
                  aria-live="polite"
                  className="flex flex-col gap-3 rounded border border-neutral-700 p-4"
                  key={item.pendingId}
                >
                  <div className="font-medium">{t('settings.tailscale.sipeed.pendingTitle')}</div>
                  <p className="text-sm text-neutral-400">
                    {t('settings.tailscale.sipeed.pendingDesc')}
                  </p>
                  <div
                    aria-label={`${t('settings.tailscale.sipeed.pendingSas')}: ${item.sas.join(' ')}`}
                    className="flex justify-center gap-3 py-1"
                    role="group"
                  >
                    {item.sas.map((digit, index) => (
                      <span
                        aria-hidden="true"
                        className="flex h-16 w-14 items-center justify-center rounded-xl border border-amber-400/40 bg-amber-400/10 font-mono text-3xl font-semibold text-amber-100"
                        key={`${item.pendingId}-${index}`}
                      >
                        {digit}
                      </span>
                    ))}
                  </div>
                  <p className="text-sm font-medium text-red-300">
                    {t('settings.tailscale.sipeed.pendingMismatch')}
                  </p>
                  <div className="text-sm text-neutral-400">
                    {(item.clientUa || item.clientIp) && (
                      <div>
                        {t('settings.tailscale.sipeed.pendingClient')}:{' '}
                        {[item.clientUa, item.clientIp].filter(Boolean).join(' · ')}
                      </div>
                    )}
                  </div>
                  <Input
                    aria-label={t('settings.tailscale.sipeed.pendingName')}
                    onChange={(event) =>
                      setPendingNames((current) => ({
                        ...current,
                        [item.pendingId]: event.target.value
                      }))
                    }
                    placeholder={t('settings.tailscale.sipeed.pendingName')}
                    value={pendingNames[item.pendingId] ?? ''}
                  />
                  <div className="flex flex-wrap gap-2">
                    <Button
                      disabled={isRateLimited || !!busyPendingId}
                      loading={busyPendingId === item.pendingId}
                      onClick={() => approve(item.pendingId)}
                      type="primary"
                    >
                      {t('settings.tailscale.sipeed.pendingAllow')}
                    </Button>
                    <Button
                      disabled={isRateLimited || !!busyPendingId}
                      onClick={() => reject(item.pendingId)}
                      danger
                    >
                      {t('settings.tailscale.sipeed.pendingReject')}
                    </Button>
                  </div>
                </div>
              ))}

              <Divider className="my-1 opacity-50" />
              <div className="flex flex-col gap-3">
                <div>
                  <div className="font-medium">{t('settings.tailscale.sipeed.backupCode')}</div>
                  <div className="text-sm text-neutral-500">
                    {t('settings.tailscale.sipeed.backupCodeDesc')}
                  </div>
                </div>
                <Input
                  aria-label={t('settings.tailscale.sipeed.backupCode')}
                  autoCapitalize="characters"
                  maxLength={9}
                  onChange={(event) => setBackupCode(event.target.value.toUpperCase())}
                  placeholder={t('settings.tailscale.sipeed.backupCode')}
                  value={backupCode}
                />
                <Input
                  aria-label={t('settings.tailscale.sipeed.pendingName')}
                  onChange={(event) => setBackupName(event.target.value)}
                  placeholder={t('settings.tailscale.sipeed.pendingName')}
                  value={backupName}
                />
                <Button
                  disabled={isRateLimited || !isBackupCodeValid}
                  loading={isAuthorizing}
                  onClick={authorizeBackup}
                >
                  {t('settings.tailscale.sipeed.authorize')}
                </Button>
              </div>
            </div>
          </Card>

          {isRateLimited && (
            <div className="text-sm text-amber-300" role="status">
              {t('settings.tailscale.sipeed.rateLimited', { seconds: rateLimitSeconds })}
            </div>
          )}
          {error && (
            <div className="text-sm text-red-400" role="alert">
              {error}
            </div>
          )}
        </div>
        {modal}
      </>
    );
  }

  return (
    <>
      <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
        <Card>
          <div className="flex flex-col gap-4">
            <div className="flex items-center justify-between gap-4">
              <span className="text-neutral-400">{t('settings.remoteNetwork.mode.heading')}</span>
              <span>{t('settings.tailscale.mode.sipeed')}</span>
            </div>
            <div className="flex items-center justify-between gap-4">
              <span className="text-neutral-400">{t('settings.tailscale.sipeed.localStatus')}</span>
              <span className="flex items-center gap-2 text-green-400">
                <CircleCheckIcon aria-hidden="true" size={16} />
                {t('settings.tailscale.sipeed.connected')}
              </span>
            </div>
            <div className="flex items-center justify-between gap-4">
              <span className="text-neutral-400">{t('settings.tailscale.deviceName')}</span>
              <span className="min-w-0 truncate" title={status.name}>
                {status.name}
              </span>
            </div>
            <div className="flex items-center justify-between gap-4">
              <span className="text-neutral-400">{t('settings.tailscale.deviceIP')}</span>
              <div className="flex min-w-0 items-center gap-2">
                <span className="truncate font-mono" title={status.ip}>
                  {status.ip}
                </span>
                <Button
                  aria-label={t('settings.tailscale.sipeed.copyIP')}
                  disabled={!status.ip}
                  icon={isCopied ? <CheckOutlined className="text-green-500" /> : <CopyOutlined />}
                  onClick={copyIP}
                  size="small"
                  type="text"
                />
              </div>
            </div>
          </div>
        </Card>

        <Divider className="opacity-50" style={{ margin: '8px 0 0' }} />

        <section aria-labelledby="sipeed-devices-heading" className="flex flex-col gap-3">
          <div className="flex items-center justify-between gap-3">
            <h3 id="sipeed-devices-heading" className="text-sm font-semibold text-neutral-200">
              {t('settings.tailscale.sipeed.devices')}
            </h3>
            <Button icon={<PlusIcon size={16} />} onClick={openAddPage} size="small" type="primary">
              {t('settings.tailscale.sipeed.addDevice')}
            </Button>
          </div>
          {devices.length === 0 ? (
            <div className="rounded-lg border border-dashed border-neutral-700 px-4 py-10 text-center text-sm text-neutral-500">
              {t('settings.tailscale.sipeed.emptyDevices')}
            </div>
          ) : (
            <div className="flex flex-col gap-2">
              {devices.map((device) => {
                const statusText = device.online
                  ? t('settings.tailscale.sipeed.deviceOnline')
                  : device.lastSeen
                    ? t('settings.tailscale.sipeed.lastSeen', {
                        time: formatLastSeen(
                          device.lastSeen,
                          locale,
                          claimNow,
                          t('settings.tailscale.sipeed.justNow')
                        )
                      })
                    : t('settings.tailscale.sipeed.deviceOffline');

                return (
                  <Card
                    key={device.nodeId}
                    size="small"
                    styles={{ body: { padding: '11px 12px' } }}
                  >
                    {editingNodeId === device.nodeId ? (
                      <div className="flex items-center gap-2">
                        <Input
                          aria-label={t('settings.tailscale.sipeed.rename')}
                          autoFocus
                          className="min-w-0 flex-1"
                          onChange={(event) => setEditingName(event.target.value)}
                          onPressEnter={() => rename(device.nodeId)}
                          size="small"
                          value={editingName}
                        />
                        <Button
                          disabled={!editingName.trim() || isRateLimited}
                          onClick={() => rename(device.nodeId)}
                          size="small"
                          type="primary"
                        >
                          {t('settings.tailscale.sipeed.save')}
                        </Button>
                        <Button
                          onClick={() => {
                            setEditingNodeId('');
                            setEditingName('');
                          }}
                          size="small"
                        >
                          {t('settings.tailscale.sipeed.cancel')}
                        </Button>
                      </div>
                    ) : (
                      <div className="flex min-w-0 items-center justify-between gap-3">
                        <div className="flex min-w-0 items-baseline gap-2">
                          <span
                            className="truncate text-sm font-semibold text-neutral-100"
                            title={device.name}
                          >
                            {device.name}
                          </span>
                          <span
                            className="truncate font-mono text-xs text-neutral-500"
                            title={device.ips.join(', ')}
                          >
                            {device.ips.join(', ')}
                          </span>
                        </div>

                        <div className="flex shrink-0 items-center gap-1">
                          <span
                            className={`flex max-w-[200px] items-center gap-1.5 text-xs ${
                              device.online ? 'text-green-400' : 'text-neutral-500'
                            }`}
                            title={statusText}
                          >
                            <span
                              aria-hidden="true"
                              className={`size-2 shrink-0 rounded-full ${
                                device.online ? 'bg-green-500' : 'border border-neutral-500'
                              }`}
                            />
                            <span className="truncate">{statusText}</span>
                          </span>
                          <Dropdown
                            menu={{
                              items: [
                                {
                                  key: 'rename',
                                  label: t('settings.tailscale.sipeed.rename'),
                                  disabled: isRateLimited
                                },
                                {
                                  key: 'remove',
                                  label: t('settings.tailscale.sipeed.remove'),
                                  danger: true,
                                  disabled: device.isSelf || isRateLimited
                                }
                              ],
                              onClick: ({ key }) => {
                                if (key === 'rename') {
                                  setEditingNodeId(device.nodeId);
                                  setEditingName(device.name);
                                } else if (key === 'remove') {
                                  setRemovingDevice(device);
                                }
                              }
                            }}
                            placement="bottomRight"
                            trigger={['click']}
                          >
                            <Button
                              aria-label={`${device.name}: ${t(
                                'settings.tailscale.sipeed.rename'
                              )}, ${t('settings.tailscale.sipeed.remove')}`}
                              className="text-neutral-400"
                              disabled={isRateLimited}
                              icon={<EllipsisIcon aria-hidden="true" size={18} />}
                              size="small"
                              type="text"
                            />
                          </Dropdown>
                        </div>
                      </div>
                    )}
                  </Card>
                );
              })}
            </div>
          )}
        </section>

        <Modal
          cancelText={t('settings.tailscale.cancelBtn')}
          okButtonProps={{ danger: true, disabled: isRateLimited }}
          okText={t('settings.tailscale.okBtn')}
          onCancel={() => setRemovingDevice(null)}
          onOk={() => {
            const device = removingDevice;
            setRemovingDevice(null);
            if (device) remove(device);
          }}
          open={removingDevice !== null}
          title={t('settings.tailscale.sipeed.remove')}
        >
          <p>{t('settings.tailscale.sipeed.removeConfirm')}</p>
        </Modal>

        {isRateLimited && (
          <div className="text-sm text-amber-300" role="status">
            {t('settings.tailscale.sipeed.rateLimited', { seconds: rateLimitSeconds })}
          </div>
        )}
        {error && (
          <div className="text-sm text-red-400" role="alert">
            {error}
          </div>
        )}
      </div>
      {modal}
    </>
  );
};

function isRateLimitData(
  value: Partial<Status> | undefined
): value is Partial<Status> & { retryAfterSeconds: number } {
  return typeof value?.retryAfterSeconds === 'number' && value.retryAfterSeconds > 0;
}
