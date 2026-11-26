import { useEffect, useState } from 'react';
import { Select } from 'antd';
import { useSetAtom } from 'jotai';
import { ExternalLinkIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';
import { feedback } from '@/lib/feedback.tsx';
import { validScreenResolution } from '@/lib/no-signal-screen.ts';
import { edidResolutionAtom } from '@/jotai/screen.ts';
import { useEdidUpdateConfirmation } from '@/hooks/useEdidUpdateConfirmation.tsx';
import {
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

export const Edid = () => {
  const { t } = useTranslation();
  const { confirm: confirmEdidUpdate, contextHolder } = useEdidUpdateConfirmation();
  const setEdidResolution = useSetAtom(edidResolutionAtom);

  const [currentEdid, setCurrentEdid] = useState('');
  const [edidList, setEdidList] = useState<any[]>(defaultEdidList);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    void getCurrentEdid();
    void getCustomEdidList();
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
      onError: (key) => feedback.error(t(key))
    });
  }

  function confirmUpdateEdid(value: string) {
    if (value === currentEdid || isLoading) return;

    confirmEdidUpdate(() => update(value));
  }

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
    </>
  );
};
