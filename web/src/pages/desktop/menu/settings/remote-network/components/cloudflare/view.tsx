import { useCallback, useEffect, useRef, useState } from 'react';
import { Alert, Button, Form, Input, Modal, Spin } from 'antd';
import { CloudIcon, SaveIcon, Trash2Icon } from 'lucide-react';

import * as api from '@/api/extensions/cloudflare.ts';
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
import { ChooseMode } from './choose-mode.tsx';
import { errorMessageKey, toRemoteNetworkStatus, validateConfig } from './model.ts';

type Action = api.CloudflareAction | 'configure' | 'mode';

export const CloudflareView = ({ onBack, setLocked }: RemoteNetworkViewProps) => {
  const t = useRemoteNetworkMessages();
  const { runWithChallenge, modal, isGateBusy } = useProtectionChallengeGate();
  const [status, setStatus] = useState<api.CloudflareStatus>();
  const [saved, setSaved] = useState<api.CloudflareConfig>();
  const [draft, setDraft] = useState<api.CloudflareConfigInput>({
    accessUrl: ''
  });
  const [choosingMode, setChoosingMode] = useState<boolean>();
  const [selectedMode, setSelectedMode] = useState<api.CloudflareMode>('named');
  const [dirty, setDirty] = useState(false);
  const [action, setAction] = useState<Action>();
  const [error, setError] = useState('');
  const [uninstallOpen, setUninstallOpen] = useState(false);
  const mounted = useRef(true);
  const busy = Boolean(action || isGateBusy || status?.busy);

  const refreshConfig = useCallback(
    async (signal?: AbortSignal, pendingDraft?: api.CloudflareConfigInput) => {
      const rsp = await api.getConfig(signal);
      if (!mounted.current || signal?.aborted) return;
      if (rsp.code !== 0) {
        setError(rsp.msg);
        return;
      }
      const config = rsp.data as api.CloudflareConfig;
      setSaved(config);
      setSelectedMode(config.mode);
      const next = pendingDraft
        ? { ...pendingDraft, token: undefined }
        : { accessUrl: config.accessUrl };
      setDraft(next);
      setDirty(next.accessUrl !== config.accessUrl);
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
        if (rsp.code === 0) {
          const current = rsp.data as api.CloudflareStatus;
          setStatus(current);
          // Restore an active tunnel on entry; subsequent polls preserve an
          // explicit visit to the mode picker.
          setChoosingMode((previous) => previous ?? !(current.enabled || current.running));
        } else setError(rsp.msg);
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

  function updateDraft(update: Partial<api.CloudflareConfigInput>) {
    setDraft((current) => ({ ...current, ...update }));
    setDirty(true);
  }

  function perform(next: Action) {
    if (busy) return;
    if (next === 'configure') {
      const invalid = validateConfig(draft, saved?.tokenSet);
      if (invalid) {
        setError(invalid);
        return;
      }
    }
    const protectionAction = next === 'clear-token' || next === 'mode' ? 'configure' : next;
    void runWithChallenge(`cloudflare_${protectionAction}`, 'cloudflare', async (challenge) => {
      setAction(next);
      setError('');
      try {
        const rsp =
          next === 'configure'
            ? await api.setConfig(draft, challenge)
            : next === 'mode'
              ? await api.setMode(selectedMode, challenge)
              : await api.runAction(next, challenge);
        if (!mounted.current) return rsp;
        if (rsp.code === -3 || rsp.code === -4) return rsp;
        if (rsp.code !== 0) {
          setError(rsp.msg);
          return rsp;
        }
        setUninstallOpen(false);
        if (next === 'mode') setChoosingMode(false);
        if (
          next === 'configure' ||
          next === 'install' ||
          next === 'clear-token' ||
          next === 'mode'
        ) {
          await refreshConfig(undefined, next === 'clear-token' ? draft : undefined);
        }
        const current = await api.getStatus();
        if (mounted.current && current.code === 0) setStatus(current.data as api.CloudflareStatus);
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
  const quick = saved?.mode === 'quick';
  const canStart = quick || Boolean(saved?.tokenSet && !dirty);
  const active = Boolean(status?.enabled || status?.running);

  return (
    <>
      <TunnelHeader
        provider="cloudflare"
        name="Cloudflare Tunnel"
        Icon={CloudIcon}
        wikiUrl="https://wiki.sipeed.com/hardware/en/kvm/NanoKVM_Go/network/cloudflare_tunnel.html"
        busy={busy}
        installed={Boolean(status && !needsInstall)}
        onBack={
          active || choosingMode !== false
            ? onBack
            : () => {
                setSelectedMode(saved?.mode ?? 'named');
                setChoosingMode(true);
              }
        }
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
            provider="cloudflare"
            installing={status.state === 'installing' || action === 'install'}
            busy={busy}
            onInstall={() => perform('install')}
          />
        ) : !saved ? (
          <div className="flex justify-center py-10">
            <Spin />
          </div>
        ) : choosingMode ? (
          <ChooseMode
            value={selectedMode}
            busy={busy}
            onChange={setSelectedMode}
            confirmDisconnect={active && selectedMode !== saved.mode}
            onConfirm={() => {
              if (selectedMode === saved.mode) setChoosingMode(false);
              else perform('mode');
            }}
          />
        ) : (
          <>
            <TunnelConnection
              provider="cloudflare"
              state={status.state}
              tone={toRemoteNetworkStatus(status)}
              active={status.enabled || status.running}
              busy={busy}
              canStart={canStart}
              action={action}
              onAction={perform}
            />
            {status.error && (
              <Alert type="error" showIcon message={t(errorMessageKey(status.error))} />
            )}
            {(!quick || (needsInstall && saved.tokenSet)) && (
              <Form layout="vertical" requiredMark={false} onFinish={() => perform('configure')}>
                <TunnelConfigPanel
                  footer={
                    <>
                      {dirty && (
                        <Button
                          disabled={busy}
                          onClick={() => void refreshConfig().catch(() => setError('failed'))}
                        >
                          {t('cloudflare.cancel')}
                        </Button>
                      )}
                      <Button
                        type="primary"
                        htmlType="submit"
                        icon={<SaveIcon size={15} />}
                        disabled={busy || !dirty || !saved || needsInstall}
                        loading={action === 'configure'}
                      >
                        {t('cloudflare.save')}
                      </Button>
                    </>
                  }
                >
                  <Form.Item label={t('cloudflare.accessUrl')} htmlFor="cloudflare-access-url">
                    <Input
                      id="cloudflare-access-url"
                      size="large"
                      value={draft.accessUrl}
                      placeholder="https://kvm.example.com"
                      maxLength={2048}
                      disabled={busy || !saved || needsInstall}
                      onChange={(event) => updateDraft({ accessUrl: event.target.value })}
                    />
                  </Form.Item>
                  <TunnelTokenField
                    provider="cloudflare"
                    label="Tunnel Token"
                    value={draft.token}
                    saved={Boolean(saved?.tokenSet)}
                    disabled={busy || !saved || needsInstall}
                    busy={busy}
                    clearing={action === 'clear-token'}
                    onChange={(value) => updateDraft({ token: value.trim() || undefined })}
                    onClear={() => perform('clear-token')}
                  />
                </TunnelConfigPanel>
              </Form>
            )}
            {status.enabled && <TunnelAccessLink provider="cloudflare" url={status.accessUrl} />}
          </>
        )}
      </div>
      <Modal
        title={
          <span className="flex items-center gap-2 text-red-500">
            <Trash2Icon size={18} />
            {t('cloudflare.uninstall')}
          </span>
        }
        open={uninstallOpen}
        okType="danger"
        okText={t('cloudflare.confirm')}
        cancelText={t('cloudflare.cancel')}
        confirmLoading={busy}
        onOk={() => perform('uninstall')}
        onCancel={() => {
          if (!busy) setUninstallOpen(false);
        }}
      >
        <p className="py-4">{t('cloudflare.uninstallDesc')}</p>
      </Modal>
      {modal}
    </>
  );
};
