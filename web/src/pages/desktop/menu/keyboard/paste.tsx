import { ChangeEvent, useRef, useState } from 'react';
import { Button, Input, Modal, type InputRef } from 'antd';
import clsx from 'clsx';
import { useSetAtom } from 'jotai';
import { ClipboardIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { paste } from '@/api/hid';
import { isKeyboardEnableAtom } from '@/jotai/keyboard.ts';

const { TextArea } = Input;

export const Paste = () => {
  const { t } = useTranslation();
  const setIsKeyboardEnable = useSetAtom(isKeyboardEnableAtom);

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [inputValue, setInputValue] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [errMsg, setErrMsg] = useState('');

  const inputRef = useRef<InputRef>(null);

  function onChange(e: ChangeEvent<HTMLTextAreaElement>) {
    const value = e.target.value;
    setErrMsg('');
    setInputValue(value);
  }

  function submit() {
    if (isLoading || !inputValue) return;
    setIsLoading(true);

    paste(inputValue)
      .then((rsp) => {
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          setErrMsg(rsp.code === -5 ? rsp.msg : t('keyboard.errors.paste'));
          return;
        }

        setInputValue('');
        setErrMsg('');
        setIsModalOpen(false);
      })
      .catch((err) => {
        console.log(err);
        setErrMsg(t('keyboard.errors.paste'));
      })
      .finally(() => {
        setIsLoading(false);
      });
  }

  function afterOpenChange(open: boolean) {
    if (open) {
      inputRef.current?.focus();
    }

    setIsKeyboardEnable(!open);
  }

  return (
    <>
      <div
        className={clsx(
          'flex cursor-pointer select-none items-center space-x-2 rounded py-1 pl-2 pr-5 hover:bg-neutral-700/70'
        )}
        onClick={() => setIsModalOpen(true)}
      >
        <ClipboardIcon size={18} />
        <span>{t('keyboard.paste')}</span>
      </div>

      <Modal
        open={isModalOpen}
        centered={false}
        title={t('keyboard.paste')}
        footer={null}
        onCancel={() => setIsModalOpen(false)}
        afterOpenChange={afterOpenChange}
      >
        <div className="pb-3 text-xs text-neutral-500">{t('keyboard.tips')}</div>

        <TextArea
          ref={inputRef}
          value={inputValue}
          status={errMsg ? 'error' : undefined}
          showCount
          maxLength={1024}
          autoSize={{ minRows: 5, maxRows: 12 }}
          placeholder={t('keyboard.placeholder')}
          onChange={onChange}
        />

        {errMsg && <div className="pt-1 text-sm text-red-500">{errMsg}</div>}

        <div className="flex justify-center py-3">
          <Button
            type="primary"
            loading={isLoading}
            disabled={!inputValue}
            htmlType="submit"
            onClick={submit}
          >
            {t('keyboard.submit')}
          </Button>
        </div>
      </Modal>
    </>
  );
};
