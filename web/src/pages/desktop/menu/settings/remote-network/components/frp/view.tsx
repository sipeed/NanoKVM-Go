import { useCallback, useEffect, useRef, useState } from 'react';
import { Alert, Button, Form, Input, InputNumber, Modal, Spin } from 'antd';
import { CableIcon, SaveIcon, Trash2Icon } from 'lucide-react';

import * as api from '@/api/extensions/frp.ts';
import { useProtectionChallengeGate } from '@/hooks/useProtectionChallengeGate.tsx';

import { useRemoteNetworkMessages } from '../../messages.ts';
import {
  TunnelAccessLink,
  TunnelConfigPanel,
  TunnelConnection,
  TunnelHeader,
  TunnelInstall,
  TunnelTokenField
} from '../../tunnel-controls.tsx';
import type { RemoteNetworkViewProps } from '../../types.ts';
import { errorMessageKey, toRemoteNetworkStatus, validateConfig } from './model.ts';

type Action = api.FRPAction | 'configure';

export const FRPView = ({ onBack, setLocked }: RemoteNetworkViewProps) => {
  const t = useRemoteNetworkMessages();
  const { runWithChallenge, modal, isGateBusy } = useProtectionChallengeGate();
  const [status, setStatus] = useState<api.FRPStatus>();
  const [saved, setSaved] = useState<api.FRPConfig>();
  const [draft, setDraft] = useState<api.FRPConfigInput>({
    serverAddr: '',
    serverPort: 7000,
    remotePort: 8443
  });
  const [dirty, setDirty] = useState(false);
  const [action, setAction] = useState<Action>();
  const [error, setError] = useState('');
  const [uninstallOpen, setUninstallOpen] = useState(false);
  const mounted = useRef(true);
  const busy = Boolean(action || isGateBusy || status?.busy);

  const refreshConfig = useCallback(
    async (signal?: AbortSignal, pendingDraft?: api.FRPConfigInput) => {
      const rsp = await api.getConfig(signal);
      if (!mounted.current || signal?.aborted) return;
      if (rsp.code !== 0) {
        setError(rsp.msg);
        return;
      }
      const config = rsp.data as api.FRPConfig;
      setSaved(config);
      const next = pendingDraft
        ? { ...pendingDraft, token: undefined }
        : {
            serverAddr: config.serverAddr,
            serverPort: config.serverPort,
            remotePort: config.remotePort
          };
      setDraft(next);
      setDirty(
        next.serverAddr !== config.serverAddr ||
          next.serverPort !== config.serverPort ||
          next.remotePort !== config.remotePort
      );
    },
    []
  );

  useEffect(() => {
    mounted.current = true;
    const controller = new AbortController();
    let timer: ReturnType<typeof setTimeout>;
    const poll = async () => {
      try {
        const rsp = await api.getStatus(controller.signal);
        if (controller.signal.aborted) return;
        if (rsp.code === 0) setStatus(rsp.data as api.FRPStatus);
        else setError(rsp.msg);
      } catch {
        if (!controller.signal.aborted) setError('failed');
      } finally {
        if (!controller.signal.aborted) timer = setTimeout(poll, 3000);
      }
    };
    void poll();
    void refreshConfig(controller.signal).catch(() => {
      if (!controller.signal.aborted) setError('failed');
    });
    return () => {
      mounted.current = false;
      controller.abort();
      clearTimeout(timer);
    };
  }, [refreshConfig]);

  useEffect(() => {
    setLocked(busy);
    return () => setLocked(false);
  }, [busy, setLocked]);

  function updateDraft(update: Partial<api.FRPConfigInput>) {
    setDraft((current) => ({ ...current, ...update }));
    setDirty(true);
  }

  function perform(next: Action) {
    if (busy) return;
    if (next === 'configure') {
      const invalid = validateConfig(draft);
      if (invalid) {
        setError(invalid);
        return;
      }
    }
    const protectionAction = next === 'clear-token' ? 'configure' : next;
    void runWithChallenge(`frp_${protectionAction}`, 'frp', async (challenge) => {
      setAction(next);
      setError('');
      try {
        const rsp =
          next === 'configure'
            ? await api.setConfig(draft, challenge)
            : await api.runAction(next, challenge);
        if (!mounted.current) return rsp;
        if (rsp.code === -3 || rsp.code === -4) return rsp;
        if (rsp.code !== 0) {
          setError(rsp.msg);
          return rsp;
        }
        setUninstallOpen(false);
        if (next === 'configure' || next === 'install' || next === 'clear-token') {
          await refreshConfig(undefined, next === 'clear-token' ? draft : undefined);
        }
        const current = await api.getStatus();
        if (mounted.current && current.code === 0) setStatus(current.data as api.FRPStatus);
        return rsp;
      } catch {
        if (mounted.current) setError(next === 'install' ? 'installFailed' : 'failed');
        return { code: -1 };
      } finally {
        if (mounted.current) setAction(undefined);
      }
    });
  }

  const needsInstall = status?.state === 'notInstalled' || status?.state === 'installing';
  const canStart = Boolean(saved?.serverAddr && !dirty);

  return (
    <>
      <TunnelHeader
        provider="frp"
        name="FRP"
        Icon={CableIcon}
        wikiUrl="https://wiki.sipeed.com/hardware/en/kvm/NanoKVM_Go/network/frp.html"
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
            provider="frp"
            installing={status.state === 'installing' || action === 'install'}
            busy={busy}
            onInstall={() => perform('install')}
          />
        ) : (
          <>
            <TunnelConnection
              provider="frp"
              state={status.state}
              tone={toRemoteNetworkStatus(status)}
              active={status.enabled}
              busy={busy}
              canStart={canStart}
              action={action}
              onAction={perform}
            />
            {status.error && (
              <Alert type="error" showIcon message={t(errorMessageKey(status.error))} />
            )}
            <Form layout="vertical" requiredMark={false} onFinish={() => perform('configure')}>
              <TunnelConfigPanel
                footer={
                  <>
                    {dirty && (
                      <Button
                        disabled={busy}
                        onClick={() => void refreshConfig().catch(() => setError('failed'))}
                      >
                        {t('frp.cancel')}
                      </Button>
                    )}
                    <Button
                      type="primary"
                      htmlType="submit"
                      icon={<SaveIcon size={15} />}
                      disabled={busy || !dirty || !saved || needsInstall}
                      loading={action === 'configure'}
                    >
                      {t('frp.save')}
                    </Button>
                  </>
                }
              >
                <Form.Item label={t('frp.serverAddr')} htmlFor="frp-server-address">
                  <Input
                    id="frp-server-address"
                    size="large"
                    value={draft.serverAddr}
                    placeholder="frp.example.com"
                    maxLength={253}
                    disabled={busy || !saved || needsInstall}
                    onChange={(event) => updateDraft({ serverAddr: event.target.value })}
                  />
                </Form.Item>
                <div className="grid grid-cols-1 gap-x-4 sm:grid-cols-2">
                  <Form.Item label={t('frp.serverPort')} htmlFor="frp-server-port">
                    <InputNumber
                      id="frp-server-port"
                      size="large"
                      min={1}
                      max={65535}
                      precision={0}
                      value={draft.serverPort}
                      className="w-full"
                      disabled={busy || !saved || needsInstall}
                      onChange={(value) => updateDraft({ serverPort: value ?? 0 })}
                    />
                  </Form.Item>
                  <Form.Item label={t('frp.remotePort')} htmlFor="frp-remote-port">
                    <InputNumber
                      id="frp-remote-port"
                      size="large"
                      min={1}
                      max={65535}
                      precision={0}
                      value={draft.remotePort}
                      className="w-full"
                      disabled={busy || !saved || needsInstall}
                      onChange={(value) => updateDraft({ remotePort: value ?? 0 })}
                    />
                  </Form.Item>
                </div>
                <TunnelTokenField
                  provider="frp"
                  label="Token"
                  value={draft.token}
                  saved={Boolean(saved?.tokenSet)}
                  disabled={busy || !saved || needsInstall}
                  busy={busy}
                  clearing={action === 'clear-token'}
                  onChange={(value) => updateDraft({ token: value || undefined })}
                  onClear={() => perform('clear-token')}
                />
              </TunnelConfigPanel>
            </Form>
            {status.enabled && <TunnelAccessLink provider="frp" url={status.accessUrl} />}
          </>
        )}
      </div>
      <Modal
        title={
          <span className="flex items-center gap-2 text-red-500">
            <Trash2Icon size={18} />
            {t('frp.uninstall')}
          </span>
        }
        open={uninstallOpen}
        okType="danger"
        okText={t('frp.confirm')}
        cancelText={t('frp.cancel')}
        confirmLoading={busy}
        onOk={() => perform('uninstall')}
        onCancel={() => {
          if (!busy) setUninstallOpen(false);
        }}
      >
        <p className="py-4">{t('frp.uninstallDesc')}</p>
      </Modal>
      {modal}
    </>
  );
};
