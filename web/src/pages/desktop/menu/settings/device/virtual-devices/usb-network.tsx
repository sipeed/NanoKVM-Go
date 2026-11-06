import { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Input, Select, Spin, Switch } from 'antd';
import clsx from 'clsx';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';
import type { UsbNetworkConfig as UsbNetworkConfigValue } from '@/api/vm.ts';

type Result = {
  type: 'success' | 'failed';
  message: string;
};

interface UsbNetworkConfigProps {
  disabled: boolean;
  getVirtualDevices: () => Promise<void>;
  onSavingChange: (isSaving: boolean) => void;
}

const defaultConfig: UsbNetworkConfigValue = {
  mode: 'server',
  ipv4: '',
  forwarding: false
};

export const UsbNetworkConfig = ({
  disabled,
  getVirtualDevices,
  onSavingChange
}: UsbNetworkConfigProps) => {
  const { t } = useTranslation();

  const [config, setConfig] = useState<UsbNetworkConfigValue>(defaultConfig);
  const [mode, setMode] = useState<UsbNetworkConfigValue['mode']>('server');
  const [forwardingEnabled, setForwardingEnabled] = useState(false);
  const [manualIpEnabled, setManualIpEnabled] = useState(false);
  const [ipv4, setIpv4] = useState('');
  const [isConfigLoading, setIsConfigLoading] = useState(true);
  const [loadError, setLoadError] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [result, setResult] = useState<Result | null>(null);

  const setDraft = useCallback((next: UsbNetworkConfigValue) => {
    setMode(next.mode);
    setForwardingEnabled(next.forwarding);
    setManualIpEnabled(next.ipv4 !== '');
    setIpv4(next.ipv4);
  }, []);

  const draft = useMemo<UsbNetworkConfigValue>(
    () => ({
      mode,
      ipv4: mode === 'server' && manualIpEnabled ? ipv4.trim() : '',
      forwarding: mode === 'server' && forwardingEnabled
    }),
    [forwardingEnabled, ipv4, manualIpEnabled, mode]
  );
  const hasChanges = !isSameConfig(config, draft);

  useEffect(() => {
    let isMounted = true;

    async function load() {
      try {
        const rsp = await api.getUsbNetwork();
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          if (isMounted) setLoadError(true);
          return;
        }

        const next = normalizeResponse(rsp.data);
        if (isMounted) {
          setLoadError(false);
          setConfig(next);
          setDraft(next);
        }
      } catch (err) {
        console.log(err);
        if (isMounted) setLoadError(true);
      } finally {
        if (isMounted) setIsConfigLoading(false);
      }
    }

    void load();
    return () => {
      isMounted = false;
      onSavingChange(false);
    };
  }, [onSavingChange, setDraft]);

  function resetResult() {
    setResult(null);
  }

  function cancel() {
    setDraft(config);
    resetResult();
  }

  function validate(next: UsbNetworkConfigValue) {
    if (mode === 'server' && manualIpEnabled && !validIpv4Cidr(next.ipv4)) {
      setResult({ type: 'failed', message: t('settings.device.usbNetwork.invalidIpv4') });
      return false;
    }
    return true;
  }

  async function save() {
    if (disabled || isSaving || !hasChanges || !validate(draft)) return;

    setIsSaving(true);
    onSavingChange(true);
    resetResult();
    try {
      const rsp = await api.setUsbNetwork(draft);
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        setResult({ type: 'failed', message: t('settings.device.usbNetwork.failed') });
        return;
      }

      const saved = normalizeResponse(rsp.data ?? draft);
      setConfig(saved);
      setDraft(saved);
      setResult({ type: 'success', message: t('settings.device.usbNetwork.success') });

      try {
        await getVirtualDevices();
      } catch (err) {
        console.log(err);
      }
    } catch (err) {
      console.log(err);
      setResult({ type: 'failed', message: t('settings.device.usbNetwork.failed') });
    } finally {
      onSavingChange(false);
      setIsSaving(false);
    }
  }

  return (
    <div className="flex flex-col">
      {isConfigLoading ? (
        <div className="flex min-h-24 items-center justify-center">
          <Spin size="small" />
        </div>
      ) : loadError ? (
        <div className="text-xs text-red-500">{t('settings.device.usbNetwork.loadFailed')}</div>
      ) : (
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-2">
            <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between md:gap-4">
              <span>{t('settings.device.usbNetwork.mode')}</span>
              <Select<UsbNetworkConfigValue['mode']>
                className="w-full md:w-40"
                value={mode}
                disabled={disabled || isSaving}
                options={[
                  { value: 'server', label: t('settings.device.usbNetwork.server') },
                  { value: 'client', label: t('settings.device.usbNetwork.client') }
                ]}
                onChange={(nextMode) => {
                  setMode(nextMode);
                  resetResult();
                }}
              />
            </div>
            <span className="text-xs text-neutral-500">
              {t(
                mode === 'server'
                  ? 'settings.device.usbNetwork.serverDesc'
                  : 'settings.device.usbNetwork.clientDesc'
              )}
            </span>
          </div>

          {mode === 'server' && (
            <>
              <div className="flex flex-col gap-3">
                <div className="flex items-center justify-between gap-4">
                  <span>{t('settings.device.usbNetwork.manualIp')}</span>
                  <Switch
                    checked={manualIpEnabled}
                    disabled={disabled || isSaving}
                    onChange={(enabled) => {
                      setManualIpEnabled(enabled);
                      resetResult();
                    }}
                  />
                </div>
                {manualIpEnabled && (
                  <Input
                    value={ipv4}
                    placeholder={t('settings.device.usbNetwork.ipv4Placeholder')}
                    disabled={disabled || isSaving}
                    onFocus={resetResult}
                    onChange={(event) => {
                      setIpv4(event.target.value);
                      resetResult();
                    }}
                  />
                )}
              </div>

              <div className="flex items-center justify-between gap-4">
                <span>{t('settings.device.usbNetwork.forwarding')}</span>
                <Switch
                  checked={forwardingEnabled}
                  disabled={disabled || isSaving}
                  onChange={(enabled) => {
                    setForwardingEnabled(enabled);
                    resetResult();
                  }}
                />
              </div>
            </>
          )}

          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            {result && (
              <span
                className={clsx(
                  'text-sm',
                  result.type === 'success' ? 'text-green-500' : 'text-red-500'
                )}
              >
                {result.message}
              </span>
            )}

            <div className="grid w-full grid-cols-2 gap-2 md:ml-auto md:flex md:w-auto md:justify-end">
              <Button
                className="md:min-w-20"
                disabled={disabled || isSaving || !hasChanges}
                onClick={cancel}
              >
                {t('settings.device.usbNetwork.cancelBtn')}
              </Button>
              <Button
                className="md:min-w-20"
                type="primary"
                disabled={disabled || !hasChanges}
                loading={isSaving}
                onClick={() => void save()}
              >
                {t('settings.device.usbNetwork.applyBtn')}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

function normalizeResponse(data: unknown): UsbNetworkConfigValue {
  const value = data && typeof data === 'object' ? (data as Record<string, unknown>) : {};
  const mode = value.mode === 'client' ? 'client' : 'server';
  return {
    mode,
    ipv4: mode === 'server' && typeof value.ipv4 === 'string' ? value.ipv4 : '',
    forwarding: mode === 'server' && value.forwarding === true
  };
}

function isSameConfig(left: UsbNetworkConfigValue, right: UsbNetworkConfigValue) {
  return (
    left.mode === right.mode && left.ipv4 === right.ipv4 && left.forwarding === right.forwarding
  );
}

function validIpv4Cidr(value: string) {
  const match = value.match(/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})\/(\d{1,2})$/);
  if (!match) return false;

  const octetText = match.slice(1, 5);
  const octets = octetText.map(Number);
  const prefixLength = Number(match[5]);
  if (octetText.some((octet, index) => String(octets[index]) !== octet)) return false;
  if (String(prefixLength) !== match[5]) return false;
  if (octets.some((octet) => octet < 0 || octet > 255)) return false;
  if (prefixLength < 1 || prefixLength > 30) return false;
  if (octets[0] === 0 || octets[0] === 127 || octets[0] >= 224) return false;
  if (octets[0] === 169 && octets[1] === 254) return false;
  return true;
}
