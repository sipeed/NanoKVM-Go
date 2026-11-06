import { ChangeEvent, useEffect, useRef, useState } from 'react';
import { Divider } from 'antd';
import {
  ArrowUpFromLineIcon,
  BookIcon,
  CheckIcon,
  LoaderCircleIcon,
  RatioIcon,
  Trash2Icon
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';
import { feedback } from '@/lib/feedback.tsx';
import { useEdidUpdateConfirmation } from '@/hooks/useEdidUpdateConfirmation.tsx';
import { MenuSubmenu } from '@/components/menu-item.tsx';

import { getEdidResponseErrorTranslationKey, runEdidSwitch, runEdidUpload } from './edid-model.ts';

export const Edid = () => {
  const { t } = useTranslation();
  const { confirm: confirmEdidUpdate, contextHolder } = useEdidUpdateConfirmation();

  const [currentEdid, setCurrentEdid] = useState('');
  const [customEdidList, setCustomEdidList] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isUploading, setIsUploading] = useState(false);

  const inputRef = useRef<HTMLInputElement>(null);

  const edidList = [
    { key: 'nanokvm-go-3840x2160-50', width: 3840, height: 2160, rate: '50Hz' },
    { key: 'nanokvm-go-3840x2160-30', width: 3840, height: 2160, rate: '30Hz' },
    { key: 'nanokvm-go-3440x1440-60', width: 3440, height: 1440, rate: '60Hz' },
    { key: 'nanokvm-go-2560x1440-60', width: 2560, height: 1440, rate: '60Hz' },
    { key: 'nanokvm-go-1920x1080-60', width: 1920, height: 1080, rate: '60Hz' }
  ];

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
          t(getEdidResponseErrorTranslationKey(rsp, 'settings.screen.edid.errors.uploadFailed'))
        );
        return;
      }

      setCustomEdidList(Array.isArray(rsp.data?.edidList) ? rsp.data.edidList : []);
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

  function selectFile() {
    if (inputRef.current) {
      inputRef.current.value = '';
    }

    inputRef.current?.click();
  }

  function uploadFile(e: ChangeEvent<HTMLInputElement>) {
    if (!e.target?.files?.length) return;
    const file = e.target.files[0];

    if (isUploading) return;
    void runEdidUpload({
      file,
      request: () => {
        const formData = new FormData();
        formData.append('file', file);
        return api.uploadEdid(formData);
      },
      refreshCustomList: getCustomEdidList,
      setLoading: setIsUploading,
      onError: (key) => feedback.error(t(key))
    });
  }

  async function deleteEdid(e: { stopPropagation: () => void }, edid: string) {
    e.stopPropagation();

    try {
      const rsp = await api.deleteEdid(edid);
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        feedback.error(t('settings.screen.edid.errors.deleteFailed'));
        return;
      }

      setCustomEdidList((current) => current.filter((customEdid) => customEdid !== edid));
    } catch (err) {
      console.log(err);
      feedback.error(t('settings.screen.edid.errors.networkFailed'));
    }
  }

  const content = (
    <>
      {/* default EDID list */}
      {edidList.map((edid) => (
        <div
          key={edid.key}
          className="flex cursor-pointer select-none items-center rounded py-1.5 pl-1 pr-6 hover:bg-neutral-700/70"
          onClick={() => confirmUpdateEdid(edid.key)}
        >
          <div className="flex h-[14px] w-[20px] items-end text-blue-500">
            {edid.key === currentEdid && <CheckIcon size={14} />}
          </div>

          <div className="flex w-[80px] items-center justify-between">
            <span>{edid.width}</span>
            <span className="text-xs">x</span>
            <span>{edid.height}</span>
          </div>
          <div className="pl-3">{edid.rate}</div>
        </div>
      ))}

      <Divider style={{ margin: '5px 0' }} />

      {customEdidList.length > 0 ? (
        // custom EDID list
        <>
          {customEdidList.map((edid) => (
            <div
              key={edid}
              className="group flex cursor-pointer select-none items-center justify-between rounded py-1.5 pl-1 pr-6 hover:bg-neutral-700/70"
              onClick={() => confirmUpdateEdid(edid)}
            >
              <div className="flex items-center">
                <div className="flex h-[14px] w-[20px] items-end text-blue-500">
                  {edid === currentEdid && <CheckIcon size={14} />}
                </div>
                <div className="max-w-[100px] select-none truncate">{edid.replace('.bin', '')}</div>
              </div>

              <div
                className="hidden h-[16px] w-[16px] items-center text-red-500 hover:text-red-500/60 group-hover:flex"
                onClick={(e) => deleteEdid(e, edid)}
              >
                <Trash2Icon size={14} />
              </div>
            </div>
          ))}
        </>
      ) : (
        <a
          className="flex cursor-pointer select-none items-center rounded px-1 py-1.5 text-neutral-300 hover:bg-neutral-700/70 hover:text-neutral-300"
          href="https://wiki.sipeed.com/hardware/en/kvm/NanoKVM_Go/resolution.html"
          target="_blank"
        >
          <div className="h-[14px] w-[20px]" />
          <div className="flex items-center space-x-1">
            <BookIcon size={14} />
            <span>{t('settings.screen.edid.document')}</span>
          </div>
        </a>
      )}

      <Divider style={{ margin: '5px 0' }} />

      <input ref={inputRef} type="file" accept=".bin" className="hidden" onChange={uploadFile} />

      <div
        className="flex cursor-pointer select-none items-center rounded px-1 py-1.5 hover:bg-neutral-700/70"
        onClick={selectFile}
      >
        <div className="h-[14px] w-[20px]" />
        <div className="flex items-center space-x-1">
          {isUploading ? (
            <LoaderCircleIcon className="animate-spin" size={14} />
          ) : (
            <ArrowUpFromLineIcon size={14} />
          )}
          <span>{t('screen.upload')}</span>
        </div>
      </div>
    </>
  );

  return (
    <>
      {contextHolder}
      <MenuSubmenu
        title="EDID"
        content={content}
        popoverProps={{ placement: 'rightTop', arrow: false, align: { offset: [14, 0] } }}
      >
        <div className="flex h-[30px] cursor-pointer items-center space-x-2 rounded pl-3 pr-6 text-neutral-300 hover:bg-neutral-700/70">
          <RatioIcon size={18} />
          <span className="select-none text-sm">EDID</span>
        </div>
      </MenuSubmenu>
    </>
  );
};
