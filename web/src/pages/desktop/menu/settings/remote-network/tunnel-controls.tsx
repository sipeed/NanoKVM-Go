import type { ComponentType, ReactNode } from 'react';
import { DownloadOutlined } from '@ant-design/icons';
import {
  Button,
  Card,
  Divider,
  Form,
  Input,
  Popconfirm,
  Popover,
  Result,
  Tooltip,
  Typography
} from 'antd';
import clsx from 'clsx';
import {
  ArrowUpRightIcon,
  BookOpenIcon,
  CheckIcon,
  ChevronLeftIcon,
  EllipsisIcon,
  EraserIcon,
  ExternalLinkIcon,
  LoaderCircleIcon,
  PlayIcon,
  RotateCwIcon,
  SquareIcon,
  Trash2Icon,
  type LucideIcon
} from 'lucide-react';

import { ButtonBase } from '@/components/ui/button-base.tsx';

import { useRemoteNetworkMessages } from './messages.ts';
import type { RemoteNetworkComponentIconProps, RemoteNetworkStatus } from './types.ts';

type Provider = 'frp' | 'cloudflare' | 'zerotier' | 'netbird';

export function TunnelHeader({
  provider,
  name,
  Icon,
  wikiUrl,
  busy,
  installed,
  onBack,
  onUninstall
}: {
  provider: Provider;
  name: string;
  Icon: LucideIcon | ComponentType<RemoteNetworkComponentIconProps>;
  wikiUrl: string;
  busy: boolean;
  installed: boolean;
  onBack: () => void;
  onUninstall: () => void;
}) {
  const t = useRemoteNetworkMessages();
  return (
    <>
      <div className="flex items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2.5">
          <ButtonBase
            aria-label={t('settings.back')}
            disabled={busy}
            onClick={onBack}
            className="flex size-6 shrink-0 items-center justify-center rounded-lg text-neutral-400 enabled:hover:bg-neutral-800 enabled:hover:text-neutral-100"
          >
            <ChevronLeftIcon size={18} />
          </ButtonBase>
          <span
            className="hidden size-6 shrink-0 items-center justify-center rounded-lg border border-neutral-700/70 bg-neutral-800/60 text-neutral-200 min-[400px]:flex"
            aria-hidden="true"
          >
            <Icon size={16} />
          </span>
          <span className="min-w-0 truncate text-base font-bold text-neutral-100">{name}</span>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          <a
            href={wikiUrl}
            target="_blank"
            rel="noopener noreferrer"
            aria-label={`${name} Wiki`}
            className="inline-flex h-6 items-center gap-1.5 rounded-lg px-2 text-xs text-neutral-400 transition-colors hover:bg-neutral-800 hover:text-neutral-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
          >
            <BookOpenIcon size={14} aria-hidden="true" />
            Wiki
            <ArrowUpRightIcon size={12} aria-hidden="true" />
          </a>
          {installed && (
            <Popover
              trigger="click"
              placement="bottomRight"
              content={
                <Button
                  type="text"
                  danger
                  icon={<Trash2Icon size={15} />}
                  disabled={busy}
                  onClick={onUninstall}
                >
                  {t(`${provider}.uninstall`)}
                </Button>
              }
            >
              <ButtonBase
                aria-label={t(`${provider}.uninstall`)}
                disabled={busy}
                className="flex size-6 items-center justify-center rounded-lg text-neutral-400 enabled:hover:bg-neutral-800 enabled:hover:text-neutral-100"
              >
                <EllipsisIcon size={18} />
              </ButtonBase>
            </Popover>
          )}
        </div>
      </div>
      <Divider className="opacity-50" />
      <div className="h-10" />
    </>
  );
}

export function TunnelInstall({
  provider,
  installing,
  busy,
  onInstall
}: {
  provider: Provider;
  installing: boolean;
  busy: boolean;
  onInstall: () => void;
}) {
  const t = useRemoteNetworkMessages();
  return (
    <Card>
      <Result
        icon={<DownloadOutlined />}
        subTitle={t(`${provider}.notInstalled`)}
        extra={
          <Button type="primary" loading={busy} disabled={busy} onClick={onInstall}>
            {t(`${provider}.${installing ? 'installing' : 'install'}`)}
          </Button>
        }
      />
    </Card>
  );
}

export function TunnelConnection({
  provider,
  state,
  tone,
  active,
  busy,
  canStart,
  action,
  restartDescription,
  onAction
}: {
  provider: Provider | 'tailscale';
  state: string;
  tone: RemoteNetworkStatus;
  active: boolean;
  busy: boolean;
  canStart: boolean;
  action?: string;
  restartDescription?: string;
  onAction: (action: 'start' | 'stop' | 'restart') => void;
}) {
  const t = useRemoteNetworkMessages();
  const labelKey =
    state === 'notInstalled'
      ? 'statusNotInstalled'
      : state === 'notConfigured'
        ? 'statusNotConfigured'
        : state;
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-neutral-700/60 bg-neutral-800/35 px-4 py-3">
      <div role="status" className="flex min-w-0 items-center gap-2.5 text-sm">
        {tone === 'busy' ? (
          <LoaderCircleIcon
            size={15}
            className="shrink-0 animate-spin text-amber-400"
            aria-hidden="true"
          />
        ) : (
          <span
            aria-hidden="true"
            className={clsx('size-2 shrink-0 rounded-full', {
              'bg-emerald-400': tone === 'on',
              'bg-red-400': tone === 'bad',
              'bg-neutral-500': tone === 'off'
            })}
          />
        )}
        <span className={clsx('font-medium', tone === 'bad' ? 'text-red-400' : 'text-neutral-200')}>
          {t(`${provider}.${labelKey}`)}
        </span>
      </div>
      <div className="flex items-center gap-2">
        {active ? (
          <>
            <Popconfirm
              title={t(`${provider}.stop`)}
              description={t(`${provider}.disconnect`)}
              styles={{ root: { maxWidth: 'min(360px, calc(100vw - 32px))' } }}
              onConfirm={() => onAction('stop')}
              okText={t(`${provider}.confirm`)}
              cancelText={t(`${provider}.cancel`)}
              disabled={busy}
            >
              <Button
                danger
                icon={<SquareIcon size={12} />}
                loading={action === 'stop'}
                disabled={busy}
              >
                {t(`${provider}.stop`)}
              </Button>
            </Popconfirm>
            <Popconfirm
              title={t(`${provider}.restart`)}
              description={restartDescription ?? t(`${provider}.disconnect`)}
              styles={{ root: { maxWidth: 'min(360px, calc(100vw - 32px))' } }}
              onConfirm={() => onAction('restart')}
              okText={t(`${provider}.confirm`)}
              cancelText={t(`${provider}.cancel`)}
              disabled={busy || !canStart}
            >
              <Tooltip title={t(`${provider}.restart`)}>
                <Button
                  aria-label={t(`${provider}.restart`)}
                  icon={<RotateCwIcon size={14} />}
                  loading={action === 'restart'}
                  disabled={busy || !canStart}
                />
              </Tooltip>
            </Popconfirm>
          </>
        ) : (
          <Button
            type="primary"
            icon={<PlayIcon size={13} />}
            disabled={busy || !canStart}
            loading={action === 'start'}
            onClick={() => onAction('start')}
          >
            {t(`${provider}.start`)}
          </Button>
        )}
      </div>
    </div>
  );
}

export function TunnelTokenField({
  provider,
  label,
  value,
  saved,
  disabled,
  busy,
  clearing,
  onChange,
  onClear
}: {
  provider: Provider;
  label: string;
  value?: string;
  saved: boolean;
  disabled: boolean;
  busy: boolean;
  clearing: boolean;
  onChange: (value: string) => void;
  onClear: () => void;
}) {
  const t = useRemoteNetworkMessages();
  return (
    <Form.Item
      className="!mb-0"
      label={
        <span className="inline-flex items-center gap-2">
          {label}
          {saved && (
            <span className="inline-flex items-center gap-1 text-[11px] font-normal text-emerald-400">
              <CheckIcon size={12} aria-hidden="true" />
              {t(`${provider}.tokenStored`)}
            </span>
          )}
        </span>
      }
    >
      <div className="flex flex-wrap items-start justify-end gap-2">
        <Input.Password
          size="large"
          className="min-w-[140px] flex-1"
          aria-label={label}
          value={value ?? ''}
          autoComplete="new-password"
          maxLength={4096}
          allowClear
          placeholder={saved ? '••••••••' : undefined}
          disabled={disabled}
          onChange={(event) => onChange(event.target.value)}
        />
        {saved && (
          <Popconfirm
            title={t(`${provider}.clearToken`)}
            description={t(`${provider}.clearTokenConfirm`)}
            styles={{ root: { maxWidth: 'min(360px, calc(100vw - 32px))' } }}
            onConfirm={onClear}
            okText={t(`${provider}.confirm`)}
            cancelText={t(`${provider}.cancel`)}
            disabled={busy}
          >
            <Button
              size="large"
              danger
              icon={<EraserIcon size={15} />}
              disabled={busy}
              loading={clearing}
            >
              {t(`${provider}.clearToken`)}
            </Button>
          </Popconfirm>
        )}
      </div>
    </Form.Item>
  );
}

export function TunnelConfigPanel({
  children,
  footer
}: {
  children: ReactNode;
  footer: ReactNode;
}) {
  return (
    <div className="overflow-hidden rounded-xl border border-neutral-700/60 bg-neutral-900/30">
      <div className="p-4 sm:p-5">{children}</div>
      <div className="flex items-center justify-end gap-2 border-t border-neutral-700/50 bg-neutral-800/20 px-4 py-3 sm:px-5">
        {footer}
      </div>
    </div>
  );
}

export function TunnelAccessLink({ provider, url }: { provider: Provider; url: string }) {
  const t = useRemoteNetworkMessages();
  if (!url) return null;
  return (
    <div className="flex min-w-0 items-center gap-3 px-1">
      <ExternalLinkIcon size={16} className="shrink-0 text-neutral-500" aria-hidden="true" />
      <div className="min-w-0 flex-1">
        <div className="mb-0.5 text-xs text-neutral-500">{t(`${provider}.accessUrl`)}</div>
        <Typography.Paragraph copyable={{ text: url }} className="!mb-0 break-all !text-sm">
          <a href={url} target="_blank" rel="noopener noreferrer">
            {url}
          </a>
        </Typography.Paragraph>
      </div>
    </div>
  );
}
