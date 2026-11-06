import { useEffect, useRef, useState } from 'react';
import { Alert, Button, Modal, Popconfirm, Spin, Typography } from 'antd';
import { ArrowUpRightIcon, LogInIcon, LogOutIcon, ShieldCheckIcon } from 'lucide-react';

import * as api from '@/api/extensions/netbird.ts';
import { useProtectionChallengeGate } from '@/hooks/useProtectionChallengeGate.tsx';
import { NetBird as NetBirdIcon } from '@/components/icons/netbird';

import { useRemoteNetworkMessages } from '../../messages.ts';
import {
  TunnelAccessLink,
  TunnelConnection,
  TunnelHeader,
  TunnelInstall
} from '../../tunnel-controls.tsx';
import type { RemoteNetworkViewProps } from '../../types.ts';
import { errorMessageKey, isLoginAvailable, toRemoteNetworkStatus } from './model.ts';

export const NetBirdView = ({ onBack, setLocked }: RemoteNetworkViewProps) => {
  const t = useRemoteNetworkMessages();
  const { runWithChallenge, modal, isGateBusy } = useProtectionChallengeGate();
  const [status, setStatus] = useState<api.NetBirdStatus>();
  const [action, setAction] = useState<api.NetBirdAction>();
  const [error, setError] = useState('');
  const [uninstallOpen, setUninstallOpen] = useState(false);
  const mounted = useRef(true);
  const inFlight = useRef(false);
  const revision = useRef(0);
  const busy = Boolean(action || isGateBusy || status?.busy);

  useEffect(() => {
    mounted.current = true;
    const controller = new AbortController();
    let timer: ReturnType<typeof setTimeout>;
    const poll = async () => {
      const currentRevision = revision.current;
      try {
        const rsp = await api.getStatus(controller.signal);
        if (controller.signal.aborted || inFlight.current || currentRevision !== revision.current)
          return;
        if (rsp.code === 0) setStatus(rsp.data as api.NetBirdStatus);
        else setError('failed');
      } catch {
        if (!controller.signal.aborted) setError('failed');
      } finally {
        if (!controller.signal.aborted) timer = setTimeout(poll, 3000);
      }
    };
    void poll();
    return () => {
      mounted.current = false;
      controller.abort();
      clearTimeout(timer);
    };
  }, []);

  useEffect(() => {
    // Browser authorization is pending server-side and does not lock navigation.
    setLocked(busy);
    return () => setLocked(false);
  }, [busy, setLocked]);

  function perform(next: api.NetBirdAction) {
    if (busy || inFlight.current) return;
    if (next === 'login' && !isLoginAvailable(status)) {
      setError('startRequired');
      return;
    }
    inFlight.current = true;
    void runWithChallenge(`netbird_${next}`, 'netbird', async (challenge) => {
      inFlight.current = true;
      revision.current++;
      setAction(next);
      setError('');
      try {
        const rsp = await api.runAction(next, challenge);
        if (!mounted.current) return rsp;
        if (rsp.code === -3 || rsp.code === -4) return rsp;
        if (rsp.code !== 0) {
          setError(rsp.msg);
          return rsp;
        }
        setUninstallOpen(false);
        const current = await api.getStatus();
        if (mounted.current && current.code === 0) setStatus(current.data as api.NetBirdStatus);
        return rsp;
      } catch {
        if (mounted.current) setError(next === 'install' ? 'installFailed' : 'failed');
        return { code: -1 };
      } finally {
        inFlight.current = false;
        if (mounted.current) setAction(undefined);
      }
    }).finally(() => {
      inFlight.current = false;
    });
  }

  const needsInstall = status?.state === 'notInstalled' || status?.state === 'installing';
  const canLogin = status && ['stopped', 'notLogin'].includes(status.state);

  return (
    <>
      <TunnelHeader
        provider="netbird"
        name="NetBird"
        Icon={NetBirdIcon}
        wikiUrl="https://docs.netbird.io/how-to/getting-started"
        busy={busy}
        installed={Boolean(status && !needsInstall)}
        onBack={onBack}
        onUninstall={() => setUninstallOpen(true)}
      />
      <div className="space-y-4">
        {(error || status?.error) && (
          <Alert
            type="error"
            showIcon
            message={t(errorMessageKey(error || status?.error || 'failed'))}
          />
        )}
        {!status ? (
          <div className="flex justify-center py-10">
            <Spin />
          </div>
        ) : needsInstall ? (
          <TunnelInstall
            provider="netbird"
            installing={status.state === 'installing' || action === 'install'}
            busy={busy}
            onInstall={() => perform('install')}
          />
        ) : (
          <>
            <TunnelConnection
              provider="netbird"
              state={status.state}
              tone={toRemoteNetworkStatus(status)}
              active={status.enabled || status.running}
              busy={busy}
              canStart={status.state !== 'authorizing'}
              action={action}
              onAction={perform}
            />
            {canLogin ? (
              <section className="space-y-5 rounded-2xl border border-solid border-neutral-700/60 bg-neutral-900/40 p-5">
                <p className="m-0 text-sm leading-6 text-neutral-400">
                  {t(isLoginAvailable(status) ? 'netbird.loginHint' : 'netbird.startRequired')}
                </p>
                <div className="flex justify-end">
                  <Button
                    type="primary"
                    className="!rounded-lg"
                    icon={<LogInIcon size={15} />}
                    disabled={busy || !isLoginAvailable(status)}
                    loading={action === 'login'}
                    onClick={() => perform('login')}
                  >
                    {t('netbird.login')}
                  </Button>
                </div>
              </section>
            ) : status.state === 'authorizing' ? (
              <section className="overflow-hidden rounded-2xl border border-solid border-neutral-700/60 bg-neutral-900/40">
                <div className="space-y-4 p-5">
                  <div className="flex items-center gap-2.5 text-sm font-medium text-neutral-200">
                    <ShieldCheckIcon size={18} className="shrink-0 text-neutral-400" />
                    {t('netbird.authorizing')}
                  </div>
                  <p className="m-0 text-sm leading-6 text-neutral-400">
                    {t('netbird.authorizationHint')}
                  </p>
                  {status.userCode && (
                    <div className="flex flex-wrap items-center justify-between gap-3 text-sm">
                      <span className="text-neutral-400">{t('netbird.userCode')}</span>
                      <Typography.Paragraph
                        copyable={{ text: status.userCode }}
                        className="!mb-0 font-mono !text-sm"
                      >
                        <span className="me-3 tracking-wider">{status.userCode}</span>
                      </Typography.Paragraph>
                    </div>
                  )}
                  <a
                    href={status.authUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-1.5 text-sm text-blue-400 transition-colors hover:text-blue-300"
                  >
                    {t('netbird.openLogin')}
                    <ArrowUpRightIcon size={15} />
                  </a>
                </div>
                <div className="flex justify-end border-0 border-t border-solid border-neutral-700/40 px-3 py-2">
                  <Button
                    type="text"
                    className="!text-xs !text-neutral-400"
                    disabled={busy}
                    loading={action === 'login_cancel'}
                    onClick={() => perform('login_cancel')}
                  >
                    {t('netbird.cancelLogin')}
                  </Button>
                </div>
              </section>
            ) : (
              <section className="overflow-hidden rounded-2xl border border-solid border-neutral-700/60 bg-neutral-900/40">
                <div className="space-y-4 p-5">
                  {status.name && (
                    <div className="break-all text-sm text-neutral-200">{status.name}</div>
                  )}
                  {status.ip && (
                    <div className="flex flex-wrap items-center justify-between gap-3 text-sm">
                      <span className="text-neutral-400">NetBird IP</span>
                      <Typography.Paragraph
                        copyable={{ text: status.ip }}
                        className="!mb-0 font-mono !text-sm"
                      >
                        <span className="me-3">{status.ip}</span>
                      </Typography.Paragraph>
                    </div>
                  )}
                  {status.enabled && status.state === 'connected' && status.accessUrl ? (
                    <TunnelAccessLink provider="netbird" url={status.accessUrl} />
                  ) : (
                    <p role="status" className="m-0 text-sm text-neutral-400">
                      {t(`netbird.${status.state}`)}
                    </p>
                  )}
                </div>
                <div className="flex justify-end border-0 border-t border-solid border-neutral-700/40 px-3 py-2">
                  <Popconfirm
                    title={t('netbird.logout')}
                    description={t('netbird.logoutConfirm')}
                    disabled={busy}
                    okText={t('netbird.confirm')}
                    cancelText={t('netbird.cancel')}
                    onConfirm={() => perform('logout')}
                    styles={{ root: { maxWidth: 'min(360px, calc(100vw - 32px))' } }}
                  >
                    <Button
                      type="text"
                      className="!text-xs !text-neutral-400 enabled:hover:!text-red-400"
                      icon={<LogOutIcon size={13} />}
                      disabled={busy}
                      loading={action === 'logout'}
                    >
                      {t('netbird.logout')}
                    </Button>
                  </Popconfirm>
                </div>
              </section>
            )}
          </>
        )}
      </div>
      <Modal
        title={t('netbird.uninstall')}
        open={uninstallOpen}
        okType="danger"
        okText={t('netbird.confirm')}
        cancelText={t('netbird.cancel')}
        confirmLoading={busy}
        onOk={() => perform('uninstall')}
        onCancel={() => {
          if (!busy) setUninstallOpen(false);
        }}
      >
        <p className="py-4">{t('netbird.uninstallDesc')}</p>
      </Modal>
      {modal}
    </>
  );
};
