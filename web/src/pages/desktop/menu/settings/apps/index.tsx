import { useCallback, useEffect, useRef, useState } from 'react';
import { Button, Card, Checkbox, Empty, Input, Modal, Popconfirm, Spin, Tabs, Tag } from 'antd';
import {
  CopyIcon,
  DownloadIcon,
  PlusIcon,
  RefreshCwIcon,
  ServerIcon,
  Trash2Icon,
  UploadIcon
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/apps.ts';
import type { AppInfo, AppServer, CatalogApp } from '@/api/apps.ts';
import { feedback } from '@/lib/feedback.tsx';

import { requiresLargeAppUploadConfirmation } from './upload-confirmation.ts';

type PendingInstall = {
  source: 'upload' | 'remote' | 'edit';
  file?: File;
  serverID?: string;
  app: AppInfo;
};

const inputClass = 'bg-neutral-800';

export const Apps = () => {
  const { t } = useTranslation();
  const [modal, modalContextHolder] = Modal.useModal();
  const fileInput = useRef<HTMLInputElement>(null);
  const logEnd = useRef<HTMLDivElement>(null);
  const installAbort = useRef<AbortController | null>(null);
  const [installed, setInstalled] = useState<AppInfo[]>([]);
  const [servers, setServers] = useState<AppServer[]>([]);
  const [catalog, setCatalog] = useState<CatalogApp[]>([]);
  const [selectedServer, setSelectedServer] = useState('');
  const [loading, setLoading] = useState(false);
  const [storeLoading, setStoreLoading] = useState(false);
  const [pending, setPending] = useState<PendingInstall | null>(null);
  const [env, setEnv] = useState<Record<string, string>>({});
  const [serverModal, setServerModal] = useState(false);
  const [serverDrafts, setServerDrafts] = useState<AppServer[]>([]);
  const [logModal, setLogModal] = useState(false);
  const [installing, setInstalling] = useState(false);
  const [cancellingInstall, setCancellingInstall] = useState(false);
  const [installLog, setInstallLog] = useState('');

  const fail = useCallback(
    (rsp: any, fallback = t('settings.apps.operationFailed')) => {
      feedback.error(rsp?.msg || fallback);
    },
    [t]
  );

  const refreshInstalled = useCallback(async () => {
    const rsp = await api.listApps();
    if (rsp.code === 0) setInstalled(rsp.data ?? []);
    else fail(rsp);
  }, [fail]);

  useEffect(() => {
    void refreshInstalled();
  }, [refreshInstalled]);

  useEffect(() => {
    logEnd.current?.scrollIntoView({ block: 'end' });
  }, [installLog]);

  const prepareInstall = (install: PendingInstall) => {
    const values: Record<string, string> = {};
    for (const [name, spec] of Object.entries(install.app.manifest.env ?? {})) {
      values[name] = spec.value || spec.default || '';
    }
    setEnv(values);
    setPending(install);
  };

  const inspectUpload = async (file?: File) => {
    if (!file) return;
    setLoading(true);
    try {
      const rsp = await api.inspectApp(file);
      if (rsp.code !== 0) return fail(rsp, t('settings.apps.inspectFailed'));
      prepareInstall({ source: 'upload', file, app: rsp.data });
    } catch {
      feedback.error(t('settings.apps.inspectFailed'));
    } finally {
      setLoading(false);
      if (fileInput.current) fileInput.current.value = '';
    }
  };

  const selectUpload = (file?: File) => {
    if (fileInput.current) fileInput.current.value = '';
    if (!file) return;

    if (!requiresLargeAppUploadConfirmation(file.size)) {
      void inspectUpload(file);
      return;
    }

    modal.confirm({
      title: t('settings.apps.largeUploadTitle'),
      content: t('settings.apps.largeUploadDesc', {
        name: file.name,
        size: (file.size / 1024 / 1024).toFixed(1)
      }),
      okText: t('settings.apps.largeUploadConfirm'),
      cancelText: t('settings.apps.largeUploadCancel'),
      autoFocusButton: 'cancel',
      centered: true,
      onOk: () => void inspectUpload(file)
    });
  };

  const install = async () => {
    if (!pending) return;
    for (const [name, spec] of Object.entries(pending.app.manifest.env ?? {})) {
      if (
        spec.required &&
        !env[name]?.trim() &&
        !(pending.source === 'edit' && spec.secret && spec.configured)
      ) {
        feedback.error(t('settings.apps.invalidRequired'));
        return;
      }
    }
    const target = pending;
    if (target.source === 'edit') {
      setLoading(true);
      try {
        const rsp = await api.updateAppEnv(target.app.directory, env);
        if (rsp.code !== 0) return fail(rsp);
        setPending(null);
        feedback.success(t('settings.apps.envSaved'));
        await refreshInstalled();
      } finally {
        setLoading(false);
      }
      return;
    }
    setInstallLog('');
    setLogModal(true);
    setInstalling(true);
    setCancellingInstall(false);
    setPending(null);
    const controller = new AbortController();
    installAbort.current = controller;
    const appendLog = (text: string) => setInstallLog((current) => current + text);
    try {
      const rsp =
        target.source === 'upload'
          ? await api.installApp(
              target.file!,
              target.app.directory,
              env,
              appendLog,
              controller.signal
            )
          : await api.installRemote(
              target.serverID!,
              target.app.manifest.app_id,
              env,
              appendLog,
              controller.signal
            );
      if (rsp.code !== 0) {
        appendLog(`\nERROR: ${rsp.msg || t('settings.apps.operationFailed')}\n`);
        return fail(rsp);
      }
      feedback.success(t('settings.apps.installSuccess'));
      await refreshInstalled();
    } catch (error) {
      if (controller.signal.aborted) {
        const detail = t('settings.apps.installCancelled');
        appendLog(`\n${detail}\n`);
        feedback.info(detail);
        return;
      }
      const detail = error instanceof Error ? error.message : t('settings.apps.operationFailed');
      appendLog(`\nERROR: ${detail}\n`);
      feedback.error(detail);
    } finally {
      if (installAbort.current === controller) installAbort.current = null;
      setInstalling(false);
      setCancellingInstall(false);
    }
  };

  const remove = async (id: string) => {
    const rsp = await api.removeApp(id);
    if (rsp.code === 0) await refreshInstalled();
    else fail(rsp);
  };

  const loadCatalog = useCallback(
    async (serverID: string) => {
      const rsp = await api.getCatalog(serverID);
      if (rsp.code === 0) {
        setCatalog(rsp.data?.apps ?? []);
        return;
      }
      setCatalog([]);
      fail(rsp);
    },
    [fail]
  );

  const browse = async (serverID = selectedServer) => {
    if (!serverID) return;
    setStoreLoading(true);
    setSelectedServer(serverID);
    try {
      await loadCatalog(serverID);
    } catch {
      setCatalog([]);
      feedback.error(t('settings.apps.storeLoadFailed'));
    } finally {
      setStoreLoading(false);
    }
  };

  const refreshStores = useCallback(async () => {
    setStoreLoading(true);
    try {
      const rsp = await api.getServers();
      if (rsp.code !== 0) {
        fail(rsp);
        return;
      }
      const values: AppServer[] = rsp.data ?? [];
      setServers(values);
      const selected = values.some((server) => server.id === selectedServer)
        ? selectedServer
        : values[0]?.id || '';
      setSelectedServer(selected);
      if (selected) await loadCatalog(selected);
      else setCatalog([]);
    } catch {
      setCatalog([]);
      feedback.error(t('settings.apps.storeLoadFailed'));
    } finally {
      setStoreLoading(false);
    }
  }, [fail, loadCatalog, selectedServer, t]);

  const saveServers = async () => {
    const rsp = await api.putServers(serverDrafts);
    if (rsp.code !== 0) return fail(rsp);
    setServerModal(false);
    await refreshStores();
  };

  const appCard = (app: AppInfo, remoteServer?: string) => (
    <Card key={app.manifest.app_id} size="small" className="border-neutral-700 bg-neutral-800/60">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="truncate font-medium text-neutral-100">{app.manifest.name}</div>
          <div className="mt-1 text-xs text-neutral-500">
            {app.manifest.creator} · {app.manifest.version}
          </div>
          {app.manifest.desc && (
            <div className="mt-2 text-xs text-neutral-300">{app.manifest.desc}</div>
          )}
          <div className="mt-2">
            <Tag>{app.manifest.app_id}</Tag>
            {(app.manifest.pre_script || app.manifest.post_script) && (
              <Tag color="orange">{t('settings.apps.installScripts')}</Tag>
            )}
          </div>
        </div>
        <div className="flex shrink-0 gap-1">
          {remoteServer ? (
            <Button
              size="small"
              type="primary"
              onClick={() => prepareInstall({ source: 'remote', serverID: remoteServer, app })}
            >
              {t('settings.apps.install')}
            </Button>
          ) : (
            <>
              {Object.keys(app.manifest.env ?? {}).length > 0 && (
                <Button size="small" onClick={() => prepareInstall({ source: 'edit', app })}>
                  {t('settings.apps.editEnv')}
                </Button>
              )}
              <Button
                size="small"
                href={api.downloadURL(app.directory)}
                icon={<DownloadIcon size={14} />}
              >
                {t('settings.apps.download')}
              </Button>
              <Popconfirm
                title={t('settings.apps.removeConfirm')}
                onConfirm={() => remove(app.directory)}
              >
                <Button
                  size="small"
                  danger
                  icon={<Trash2Icon size={14} />}
                  aria-label={t('settings.apps.remove')}
                />
              </Popconfirm>
            </>
          )}
        </div>
      </div>
    </Card>
  );

  const installedPane = (
    <div>
      <div className="mb-3 flex items-center justify-between gap-2">
        <div className="text-xs text-neutral-500">{t('settings.apps.zipHint')}</div>
        <Button
          type="primary"
          icon={<UploadIcon size={15} />}
          onClick={() => fileInput.current?.click()}
        >
          {t('settings.apps.upload')}
        </Button>
        <input
          ref={fileInput}
          type="file"
          accept=".zip,application/zip"
          className="hidden"
          onChange={(event) => selectUpload(event.target.files?.[0])}
        />
      </div>
      <div className="space-y-2">
        {installed.length ? (
          installed.map((app) => appCard(app))
        ) : (
          <Empty className="pt-20" description={t('settings.apps.empty')} />
        )}
      </div>
    </div>
  );

  const storePane = (
    <div>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        {servers.map((server) => (
          <Button
            key={server.id}
            type={selectedServer === server.id ? 'primary' : 'default'}
            disabled={storeLoading}
            onClick={() => void browse(server.id)}
          >
            {server.official && <Tag color="blue">{t('settings.apps.official')}</Tag>}
            {server.name}
          </Button>
        ))}
        <Button
          icon={<ServerIcon size={15} />}
          onClick={() => {
            setServerDrafts(servers);
            setServerModal(true);
          }}
        >
          {t('settings.apps.servers')}
        </Button>
        <Button
          icon={<RefreshCwIcon size={15} />}
          loading={storeLoading}
          onClick={() => void refreshStores()}
        />
      </div>
      <Spin spinning={storeLoading}>
        {!servers.length ? (
          <Empty className="pt-20" description={t('settings.apps.noServers')} />
        ) : (
          <div className="space-y-2">
            {catalog.length ? (
              catalog.map((app) => appCard(app, selectedServer))
            ) : (
              <Empty className="pt-20" description={t('settings.apps.noCatalog')} />
            )}
          </div>
        )}
      </Spin>
    </div>
  );

  return (
    <Spin spinning={loading}>
      {modalContextHolder}
      <div className="text-base font-bold">{t('settings.apps.title')}</div>
      <Tabs
        onChange={(key) => {
          if (key === 'store') void refreshStores();
        }}
        items={[
          { key: 'installed', label: t('settings.apps.installed'), children: installedPane },
          { key: 'store', label: t('settings.apps.store'), children: storePane }
        ]}
      />

      <Modal
        open={!!pending}
        title={pending?.app.manifest.name}
        okText={t(pending?.source === 'edit' ? 'settings.apps.saveEnv' : 'settings.apps.install')}
        onOk={() => void install()}
        onCancel={() => setPending(null)}
        confirmLoading={loading}
      >
        {pending && (
          <>
            <label className="mb-1 block text-xs text-neutral-500">
              {t('settings.apps.appId')}
            </label>
            <Input className={inputClass} value={pending.app.manifest.app_id} disabled />
            {Object.keys(pending.app.manifest.env ?? {}).length > 0 && (
              <div className="mt-4 text-sm font-medium">{t('settings.apps.environment')}</div>
            )}
            {Object.entries(pending.app.manifest.env ?? {}).map(([name, spec]) => (
              <div className="mt-3" key={name}>
                <label className="mb-1 flex gap-2 text-xs text-neutral-300">
                  <span>{spec.label || name}</span>
                  {spec.required && <Tag color="red">{t('settings.apps.required')}</Tag>}
                  {spec.secret && <Tag>{t('settings.apps.secret')}</Tag>}
                </label>
                {spec.secret ? (
                  <Input.Password
                    value={env[name] ?? ''}
                    placeholder={spec.configured ? '••••••••' : spec.default}
                    onChange={(event) =>
                      setEnv((values) => ({ ...values, [name]: event.target.value }))
                    }
                  />
                ) : (
                  <Input
                    value={env[name] ?? ''}
                    placeholder={spec.default}
                    onChange={(event) =>
                      setEnv((values) => ({ ...values, [name]: event.target.value }))
                    }
                  />
                )}
                {pending.source === 'edit' && spec.secret && spec.configured && !env[name] && (
                  <div className="mt-1 text-xs text-neutral-500">
                    {t('settings.apps.configured')}
                  </div>
                )}
                {spec.description && (
                  <div className="mt-1 text-xs text-neutral-500">{spec.description}</div>
                )}
              </div>
            ))}
          </>
        )}
      </Modal>

      <Modal
        open={serverModal}
        title={t('settings.apps.servers')}
        okText={t('settings.apps.saveServers')}
        onOk={() => void saveServers()}
        onCancel={() => setServerModal(false)}
      >
        <div className="space-y-3">
          {serverDrafts.map((server, index) => (
            <Card key={`${server.id}-${index}`} size="small">
              <div className="space-y-2">
                <Input
                  placeholder="official"
                  value={server.id}
                  disabled={server.builtin}
                  onChange={(event) =>
                    setServerDrafts((items) =>
                      items.map((item, i) =>
                        i === index ? { ...item, id: event.target.value } : item
                      )
                    )
                  }
                />
                <Input
                  placeholder={t('settings.apps.serverName')}
                  value={server.name}
                  disabled={server.builtin}
                  onChange={(event) =>
                    setServerDrafts((items) =>
                      items.map((item, i) =>
                        i === index ? { ...item, name: event.target.value } : item
                      )
                    )
                  }
                />
                <Input
                  placeholder={t('settings.apps.serverURL')}
                  value={server.url}
                  disabled={server.builtin}
                  onChange={(event) =>
                    setServerDrafts((items) =>
                      items.map((item, i) =>
                        i === index ? { ...item, url: event.target.value } : item
                      )
                    )
                  }
                />
                <div className="flex items-center justify-between">
                  <Checkbox
                    checked={server.official}
                    disabled={server.builtin}
                    onChange={(event) =>
                      setServerDrafts((items) =>
                        items.map((item, i) =>
                          i === index ? { ...item, official: event.target.checked } : item
                        )
                      )
                    }
                  >
                    {t('settings.apps.official')}
                  </Checkbox>
                  <Button
                    danger
                    size="small"
                    icon={<Trash2Icon size={14} />}
                    disabled={server.builtin}
                    onClick={() => setServerDrafts((items) => items.filter((_, i) => i !== index))}
                  />
                </div>
              </div>
            </Card>
          ))}
          <Button
            block
            icon={<PlusIcon size={15} />}
            onClick={() =>
              setServerDrafts((items) => [
                ...items,
                { id: '', name: '', url: '', official: false, builtin: false }
              ])
            }
          >
            {t('settings.apps.addServer')}
          </Button>
        </div>
      </Modal>

      <Modal
        open={logModal}
        width={760}
        title={t('settings.apps.installLog')}
        closable={!installing}
        maskClosable={false}
        keyboard={!installing}
        onCancel={() => !installing && setLogModal(false)}
        footer={[
          <Button
            key="copy"
            icon={<CopyIcon size={14} />}
            onClick={() => void navigator.clipboard.writeText(installLog)}
          >
            {t('settings.apps.copyLog')}
          </Button>,
          installing && (
            <Button
              key="cancel"
              danger
              loading={cancellingInstall}
              onClick={() => {
                setCancellingInstall(true);
                installAbort.current?.abort();
              }}
            >
              {t(
                cancellingInstall
                  ? 'settings.apps.cancellingInstall'
                  : 'settings.apps.cancelInstall'
              )}
            </Button>
          ),
          <Button
            key="close"
            type="primary"
            disabled={installing}
            onClick={() => setLogModal(false)}
          >
            {t('settings.apps.closeLog')}
          </Button>
        ]}
      >
        <div className="mb-2 text-xs text-neutral-500">
          {t(installing ? 'settings.apps.installing' : 'settings.apps.installFinished')}
        </div>
        <div className="max-h-[55vh] overflow-auto whitespace-pre-wrap break-all rounded bg-black p-3 font-mono text-xs leading-5 text-neutral-200">
          {installLog || t('settings.apps.installing')}
          <div ref={logEnd} />
        </div>
      </Modal>
    </Spin>
  );
};
