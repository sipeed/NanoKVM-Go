import { useEffect, useState } from 'react';
import { Button, Input, Select } from 'antd';
import { useSetAtom } from 'jotai';
import { ExternalLinkIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';
import { feedback } from '@/lib/feedback.tsx';
import { validScreenResolution } from '@/lib/no-signal-screen.ts';
import { edidResolutionAtom } from '@/jotai/screen.ts';
import { useEdidUpdateConfirmation } from '@/hooks/useEdidUpdateConfirmation.tsx';
import {
  EdidErrorTranslationKey,
  getEdidResponseErrorTranslationKey,
  runEdidSwitch
} from '@/pages/desktop/menu/screen/edid-model.ts';

import { SettingRow } from './setting-row.tsx';

const defaultEdidList = [
  { value: 'nanokvm-go-3840x2160-50', label: '3840 x 2160 50Hz' },
  { value: 'nanokvm-go-3840x2160-30', label: '3840 x 2160 30Hz' },
  { value: 'nanokvm-go-3440x1440-60', label: '3440 x 1440 60Hz' },
  { value: 'nanokvm-go-2560x1440-60', label: '2560 x 1440 60Hz' },
  { value: 'nanokvm-go-1920x1080-60', label: '1920 x 1080 60Hz' }
];

const MAX_MONITOR_NAME_BYTES = 13;

function isValidMonitorName(value: string) {
  if (value.length > MAX_MONITOR_NAME_BYTES) return false;

  for (let i = 0; i < value.length; i += 1) {
    const code = value.charCodeAt(i);
    if (code < 0x20 || code > 0x7e) return false;
  }

  return true;
}

export const Edid = () => {
  const { t } = useTranslation();
  const { confirm: confirmEdidUpdate, contextHolder } = useEdidUpdateConfirmation();
  const setEdidResolution = useSetAtom(edidResolutionAtom);

  const [currentEdid, setCurrentEdid] = useState('');
  const [edidList, setEdidList] = useState<any[]>(defaultEdidList);
  const [isLoading, setIsLoading] = useState(false);
  const [monitorName, setMonitorName] = useState('');
  const [savedMonitorName, setSavedMonitorName] = useState('');
  const [isMonitorNameLoaded, setIsMonitorNameLoaded] = useState(false);
  const [isMonitorNameLoading, setIsMonitorNameLoading] = useState(true);
  const [isMonitorNameSaving, setIsMonitorNameSaving] = useState(false);
  const [monitorNameError, setMonitorNameError] = useState<EdidErrorTranslationKey | null>(null);

  useEffect(() => {
    void getCurrentEdid();
    void getCustomEdidList();
    void getMonitorName();
  }, []);

  async function getCurrentEdid() {
    try {
      const rsp = await api.getEdid();
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        feedback.error(
          t(getEdidResponseErrorTranslationKey(rsp, 'settings.screen.edid.errors.switchFailed'))
        );
        return;
      }

      if (typeof rsp.data?.edid === 'string') {
        setCurrentEdid(rsp.data.edid);
      }
      const resolution = validScreenResolution(rsp.data?.width, rsp.data?.height);
      if (resolution) setEdidResolution(resolution);
    } catch (err) {
      console.log(err);
      feedback.error(t('settings.screen.edid.errors.networkFailed'));
    }
  }

  async function getCustomEdidList() {
    try {
      const rsp = await api.getCustomEdidList();
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        feedback.error(
          t(getEdidResponseErrorTranslationKey(rsp, 'settings.screen.edid.errors.switchFailed'))
        );
        return;
      }

      if (rsp.data?.edidList?.length) {
        setEdidList([
          {
            label: <span>{t('settings.screen.edid.default')}</span>,
            title: 'default',
            options: defaultEdidList
          },
          {
            label: <span>{t('settings.screen.edid.custom')}</span>,
            title: 'custom',
            options: rsp.data.edidList.map((edid: string) => {
              return {
                value: edid,
                label: edid
              };
            })
          }
        ]);
      } else {
        setEdidList(defaultEdidList);
      }
    } catch (err) {
      console.log(err);
      feedback.error(t('settings.screen.edid.errors.networkFailed'));
    }
  }

  function update(value: string) {
    if (value === currentEdid || isLoading) return;

    void runEdidSwitch({
      request: () => api.updateEdid(value),
      refreshCurrent: getCurrentEdid,
      setLoading: setIsLoading,
      onError: (key) => {
        feedback.error(t(key));
        if (key === 'settings.screen.edid.monitorNameUnsupported') {
          setMonitorNameError(key);
        }
      },
      onSuccess: () => {
        if (monitorNameError === 'settings.screen.edid.monitorNameUnsupported') {
          setMonitorNameError(null);
        }
      }
    });
  }

  function confirmUpdateEdid(value: string) {
    if (value === currentEdid || isLoading) return;

    confirmEdidUpdate(() => update(value));
  }

  async function getMonitorName() {
    setIsMonitorNameLoading(true);
    setMonitorNameError(null);

    try {
      const rsp = await api.getEdidMonitorName();
      const value = rsp.data?.monitorName;
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        const key = getEdidResponseErrorTranslationKey(
          rsp,
          'settings.screen.edid.monitorNameLoadFailed'
        );
        setMonitorNameError(key);
        feedback.error(t(key));
        return;
      }
      if (typeof value !== 'string' || !isValidMonitorName(value)) {
        console.log(rsp.msg);
        setMonitorNameError('settings.screen.edid.monitorNameLoadFailed');
        return;
      }

      setMonitorName(value);
      setSavedMonitorName(value);
      setIsMonitorNameLoaded(true);
    } catch (err) {
      console.log(err);
      setMonitorNameError('settings.screen.edid.monitorNameLoadFailed');
      feedback.error(t('settings.screen.edid.errors.networkFailed'));
    } finally {
      setIsMonitorNameLoading(false);
    }
  }

  async function saveMonitorName(value: string) {
    setIsMonitorNameSaving(true);
    setMonitorNameError(null);

    try {
      const rsp = await api.setEdidMonitorName(value);
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        const key = getEdidResponseErrorTranslationKey(
          rsp,
          'settings.screen.edid.monitorNameUpdateFailed'
        );
        setMonitorNameError(key);
        feedback.error(t(key));
        return;
      }

      const nextValue = rsp.data?.monitorName;
      const appliedValue =
        typeof nextValue === 'string' && isValidMonitorName(nextValue) ? nextValue : value;
      setMonitorName(appliedValue);
      setSavedMonitorName(appliedValue);
      setMonitorNameError(null);
    } catch (err) {
      console.log(err);
      setMonitorNameError('settings.screen.edid.monitorNameUpdateFailed');
      feedback.error(t('settings.screen.edid.errors.networkFailed'));
    } finally {
      setIsMonitorNameSaving(false);
    }
  }

  function confirmMonitorNameUpdate() {
    if (
      !isMonitorNameLoaded ||
      isLoading ||
      isMonitorNameLoading ||
      isMonitorNameSaving ||
      monitorName === savedMonitorName ||
      !isValidMonitorName(monitorName)
    ) {
      return;
    }

    const value = monitorName;
    setMonitorNameError(null);
    confirmEdidUpdate(() => saveMonitorName(value));
  }

  const isMonitorNameValid = isValidMonitorName(monitorName);
  const monitorNameErrorMessage = !isMonitorNameValid
    ? t('settings.screen.edid.monitorNameInvalid')
    : monitorNameError
      ? t(monitorNameError)
      : '';

  return (
    <>
      {contextHolder}
      <SettingRow
        label={
          <div className="flex items-center space-x-2">
            <span>EDID</span>
            <a
              className="flex items-center text-neutral-500 hover:text-blue-500"
              href="https://wiki.sipeed.com/hardware/en/kvm/NanoKVM_Go/resolution.html"
              target="_blank"
            >
              <ExternalLinkIcon size={15} />
            </a>
          </div>
        }
        description={t('settings.screen.edid.description')}
        control={
          <Select
            value={currentEdid}
            style={{ width: '100%' }}
            loading={isLoading}
            options={edidList}
            onSelect={confirmUpdateEdid}
          />
        }
      />
      <SettingRow
        label={t('settings.screen.edid.monitorName')}
        description={
          <span className="whitespace-pre-line">
            {t('settings.screen.edid.monitorNameDescription')}
          </span>
        }
        control={
          <div className="flex flex-col space-y-1">
            <div className="flex space-x-2">
              <Input
                className="min-w-0 flex-1"
                value={monitorName}
                maxLength={MAX_MONITOR_NAME_BYTES}
                status={isMonitorNameValid ? undefined : 'error'}
                placeholder={t('settings.screen.edid.monitorNamePlaceholder')}
                disabled={!isMonitorNameLoaded || isMonitorNameLoading || isMonitorNameSaving}
                onChange={(event) => {
                  setMonitorName(event.target.value);
                  setMonitorNameError(null);
                }}
                onPressEnter={confirmMonitorNameUpdate}
              />
              <Button
                type="primary"
                loading={isMonitorNameSaving}
                disabled={
                  !isMonitorNameLoaded ||
                  isLoading ||
                  isMonitorNameLoading ||
                  isMonitorNameSaving ||
                  monitorName === savedMonitorName ||
                  !isMonitorNameValid
                }
                onClick={confirmMonitorNameUpdate}
              >
                {t('settings.screen.edid.monitorNameApply')}
              </Button>
            </div>
            {monitorNameErrorMessage && (
              <span className="text-xs text-red-500">{monitorNameErrorMessage}</span>
            )}
          </div>
        }
      />
    </>
  );
};
