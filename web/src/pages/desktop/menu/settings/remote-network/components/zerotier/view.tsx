import { useEffect, useRef, useState } from 'react';
import { Alert, Button, Form, Input, Modal, Popconfirm, Spin, Typography } from 'antd';
import {
  ArrowUpRightIcon,
  CheckIcon,
  LogInIcon,
  LogOutIcon,
  NetworkIcon,
  ShieldCheckIcon,
  Trash2Icon
} from 'lucide-react';

import * as api from '@/api/extensions/zerotier.ts';
import { useProtectionChallengeGate } from '@/hooks/useProtectionChallengeGate.tsx';

import { useRemoteNetworkMessages } from '../../messages.ts';
import {
  TunnelAccessLink,
  TunnelConnection,
  TunnelHeader,
  TunnelInstall
} from '../../tunnel-controls.tsx';
import type { RemoteNetworkViewProps } from '../../types.ts';
import {
  errorMessageKey,
  networkMessageKey,
  normalizeNetworkID,
  toRemoteNetworkStatus
} from './model.ts';

export const ZeroTierView = ({ onBack, setLocked }: RemoteNetworkViewProps) => {
  const t = useRemoteNetworkMessages();
  const { runWithChallenge, modal, isGateBusy } = useProtectionChallengeGate();
  const [status, setStatus] = useState<api.ZeroTierStatus>();
  const [networkId, setNetworkId] = useState('');
  const [action, setAction] = useState<api.ZeroTierAction>();
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
        if (rsp.code === 0) setStatus(rsp.data as api.ZeroTierStatus);
        else setError(rsp.msg);
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
    setLocked(busy);
    return () => setLocked(false);
  }, [busy, setLocked]);

  function perform(next: api.ZeroTierAction, id?: string) {
    if (busy || inFlight.current) return;
    const normalized = id === undefined ? undefined : normalizeNetworkID(id);
    if ((next === 'join' || next === 'leave') && !normalized) {
      setError('invalidNetworkID');
      return;
    }
    inFlight.current = true;
    void runWithChallenge(`zerotier_${next}`, 'zerotier', async (challenge) => {
      inFlight.current = true;
      revision.current++;
      setAction(next);
      setError('');
      try {
        const rsp = await api.runAction(next, challenge, normalized);
        if (!mounted.current) return rsp;
        if (rsp.code === -3 || rsp.code === -4) return rsp;
        if (rsp.code !== 0) {
          setError(rsp.msg);
          return rsp;
        }
        setUninstallOpen(false);
        if (next === 'join') setNetworkId('');
        const current = await api.getStatus();
        if (mounted.current && current.code === 0) setStatus(current.data as api.ZeroTierStatus);
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

  return (
    <>
      <TunnelHeader
        provider="zerotier"
        name="ZeroTier"
        Icon={NetworkIcon}
        wikiUrl="https://docs.zerotier.com/start/"
        busy={busy}
        installed={Boolean(status && !needsInstall)}
        onBack={onBack}
        onUninstall={() => setUninstallOpen(true)}
      />
      <div className="space-y-4">
        {error && (
          <Alert
            type="error"
            showIcon
            closable
            message={t(errorMessageKey(error))}
            onClose={() => setError('')}
          />
        )}
        {!status ? (
          <div className="flex justify-center py-10">
            <Spin />
          </div>
        ) : needsInstall ? (
          <TunnelInstall
            provider="zerotier"
            installing={status.state === 'installing' || action === 'install'}
            busy={busy}
            onInstall={() => perform('install')}
          />
        ) : (
          <>
            <TunnelConnection
              provider="zerotier"
              state={status.state}
              tone={toRemoteNetworkStatus(status)}
              active={status.enabled || status.running}
              busy={busy}
              canStart={!needsInstall}
              action={action}
              onAction={perform}
            />
            {status.error && (
              <Alert type="error" showIcon message={t(errorMessageKey(status.error))} />
            )}
            <section className="overflow-hidden rounded-2xl border border-solid border-neutral-700/60 bg-neutral-900/40">
              {status.nodeId && (
                <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 border-0 border-b border-solid border-neutral-700/50 bg-neutral-800/40 px-5 py-3.5">
                  <span className="text-xs text-neutral-400">{t('zerotier.nodeId')}</span>
                  <Typography.Paragraph
                    copyable={{ text: status.nodeId }}
                    className="!mb-0 font-mono !text-sm !text-neutral-200"
                  >
                    <span className="me-3 tracking-wider">{status.nodeId}</span>
                  </Typography.Paragraph>
                </div>
              )}
              {status.networks.length === 0 ? (
                <Form
                  layout="vertical"
                  requiredMark={false}
                  onFinish={() => perform('join', networkId)}
                >
                  <div className="space-y-4 p-5">
                    <Form.Item
                      className="!mb-0"
                      label={t('zerotier.networkId')}
                      htmlFor="zerotier-network-id"
                    >
                      <Input
                        id="zerotier-network-id"
                        size="large"
                        className="!rounded-lg font-mono"
                        value={networkId}
                        autoComplete="off"
                        spellCheck={false}
                        placeholder="0123456789abcdef"
                        maxLength={32}
                        disabled={busy || needsInstall}
                        onChange={(event) => setNetworkId(event.target.value)}
                      />
                    </Form.Item>
                    <div className="flex justify-end">
                      <Button
                        type="primary"
                        htmlType="submit"
                        className="!rounded-lg"
                        icon={<LogInIcon size={15} />}
                        disabled={busy || !status.running || !normalizeNetworkID(networkId)}
                        loading={action === 'join'}
                      >
                        {t('zerotier.join')}
                      </Button>
                    </div>
                  </div>
                </Form>
              ) : (
                status.networks.map((network) => (
                  <div
                    key={network.id}
                    className="border-0 border-solid border-neutral-700/50 [&+&]:border-t"
                  >
                    <div className="space-y-5 p-5">
                      {['REQUESTING_CONFIGURATION', 'ACCESS_DENIED'].includes(network.status) ? (
                        <>
                          <div className="flex items-start gap-3.5">
                            <span className="box-border flex size-10 shrink-0 items-center justify-center rounded-xl border border-solid border-blue-400/15 bg-blue-500/10 text-blue-300">
                              <ShieldCheckIcon size={20} strokeWidth={1.5} aria-hidden="true" />
                            </span>
                            <div className="min-w-0 space-y-1.5">
                              <div role="status" className="text-sm font-medium text-neutral-100">
                                {t('zerotier.authorizationTitle')}
                              </div>
                              <p className="m-0 text-xs leading-relaxed text-neutral-400">
                                {t('zerotier.authorizationHint')}
                              </p>
                            </div>
                          </div>
                          <a
                            href="https://central.zerotier.com/"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="group box-border flex min-h-12 items-center justify-between gap-3 rounded-xl border border-solid border-blue-400/20 bg-blue-500/10 px-4 py-3 text-sm font-medium text-blue-200 no-underline transition-colors hover:border-blue-400/40 hover:bg-blue-500/15 hover:text-blue-100 hover:no-underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-400 focus-visible:ring-offset-2 focus-visible:ring-offset-neutral-900"
                          >
                            <span>{t('zerotier.authorize')}</span>
                            <ArrowUpRightIcon
                              size={17}
                              className="shrink-0 text-blue-400 transition-colors group-hover:text-blue-200"
                              aria-hidden="true"
                            />
                          </a>
                        </>
                      ) : status.running && status.online && network.status === 'OK' ? (
                        <>
                          <div className="flex items-center gap-2.5 text-sm font-medium text-emerald-300">
                            <span className="flex size-7 items-center justify-center rounded-full bg-emerald-500/10">
                              <CheckIcon size={16} aria-hidden="true" />
                            </span>
                            {t('zerotier.connected')}
                          </div>
                          <div className="space-y-3">
                            {network.accessUrls.map((url) => (
                              <TunnelAccessLink key={url} provider="zerotier" url={url} />
                            ))}
                          </div>
                        </>
                      ) : (
                        <div
                          role="status"
                          className="flex items-center gap-2.5 text-sm text-neutral-400"
                        >
                          <NetworkIcon size={17} className="shrink-0" aria-hidden="true" />
                          {t(networkMessageKey(network.status))}
                        </div>
                      )}
                    </div>
                    <div className="flex justify-end border-0 border-t border-solid border-neutral-700/40 px-3 py-2">
                      <Popconfirm
                        title={t('zerotier.leave')}
                        description={t('zerotier.leaveConfirm')}
                        disabled={busy}
                        okText={t('zerotier.confirm')}
                        cancelText={t('zerotier.cancel')}
                        onConfirm={() => perform('leave', network.id)}
                        styles={{ root: { maxWidth: 'min(360px, calc(100vw - 32px))' } }}
                      >
                        <Button
                          type="text"
                          className="!text-xs !text-neutral-400 enabled:hover:!text-red-400"
                          icon={<LogOutIcon size={13} />}
                          disabled={busy}
                          loading={action === 'leave'}
                        >
                          {t('zerotier.leave')}
                        </Button>
                      </Popconfirm>
                    </div>
                  </div>
                ))
              )}
            </section>
          </>
        )}
      </div>
      <Modal
        title={
          <span className="flex items-center gap-2 text-red-500">
            <Trash2Icon size={18} />
            {t('zerotier.uninstall')}
          </span>
        }
        open={uninstallOpen}
        okType="danger"
        okText={t('zerotier.confirm')}
        cancelText={t('zerotier.cancel')}
        confirmLoading={busy}
        onOk={() => perform('uninstall')}
        onCancel={() => {
          if (!busy) setUninstallOpen(false);
        }}
      >
        <p className="py-4">{t('zerotier.uninstallDesc')}</p>
      </Modal>
      {modal}
    </>
  );
};
