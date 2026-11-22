import { useEffect, useState } from 'react';
import { Button, Input, Modal, Switch } from 'antd';
import clsx from 'clsx';

type BootStringConfig = {
  isEnabled: boolean;
  value: string;
};

type Result = {
  type: 'success' | 'failed';
  message: string;
};

type BootStringApi = {
  get: () => Promise<{ code: number; msg?: string; data?: { enabled: boolean; value: string } }>;
  set: (enabled: boolean, value: string) => Promise<{ code: number; msg?: string; data?: unknown }>;
};

export type BootStringLabels = {
  title: string;
  config: string;
  tip: string;
  enable: string;
  placeholder: string;
  notEmpty: string;
  notValid: string;
  failed: string;
  success: string;
  okBtn: string;
  cancelBtn: string;
};

type Props = {
  labels: BootStringLabels;
  maxLength: number;
  asciiOnly?: boolean;
  api: BootStringApi;
};

export const BootStringSetting = ({ labels, maxLength, asciiOnly = false, api }: Props) => {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [config, setConfig] = useState<BootStringConfig>({ isEnabled: false, value: '' });
  const [isEnabled, setIsEnabled] = useState(false);
  const [value, setValue] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [result, setResult] = useState<Result | null>(null);

  useEffect(() => {
    load();
  }, []);

  function openModal() {
    setIsEnabled(config.isEnabled);
    setValue(config.value);
    setResult(null);
    setIsModalOpen(true);
  }

  function load() {
    api.get().then((rsp) => {
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        return;
      }

      const next = {
        isEnabled: rsp.data?.enabled ?? false,
        value: rsp.data?.value ?? ''
      };
      setConfig(next);
      setIsEnabled(next.isEnabled);
      setValue(next.value);
    });
  }

  async function save() {
    const nextValue = isEnabled ? value.trim() : '';

    if (isEnabled && !validContent(nextValue)) {
      return;
    }

    if (config.isEnabled === isEnabled && config.value === nextValue) {
      setIsModalOpen(false);
      return;
    }

    if (isLoading) return;
    setIsLoading(true);
    setResult(null);

    try {
      const rsp = await api.set(isEnabled, nextValue);
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        setIsLoading(false);
        setResult({ type: 'failed', message: labels.failed });
        return;
      }

      setConfig({ isEnabled, value: nextValue });
      setValue(nextValue);
      setIsLoading(false);
      setResult({ type: 'success', message: labels.success });
    } catch (err) {
      console.log(err);
      setResult({ type: 'failed', message: labels.failed });
      setIsLoading(false);
    }
  }

  function validContent(nextValue: string) {
    if (nextValue === '') {
      setResult({ type: 'failed', message: labels.notEmpty });
      return false;
    }
    if ([...nextValue].length > maxLength) {
      setResult({ type: 'failed', message: labels.notValid });
      return false;
    }
    for (let i = 0; i < nextValue.length; i += 1) {
      const code = nextValue.charCodeAt(i);
      if (code < 0x20 || code === 0x7f || (asciiOnly && code > 0x7e)) {
        setResult({ type: 'failed', message: labels.notValid });
        return false;
      }
    }
    return true;
  }

  const footer = [
    <Button key="cancel" disabled={isLoading} onClick={() => setIsModalOpen(false)}>
      {labels.cancelBtn}
    </Button>,
    <Button key="ok" type="primary" loading={isLoading} onClick={save}>
      {labels.okBtn}
    </Button>
  ];

  return (
    <>
      <div className="flex items-center justify-between">
        <span>{labels.title}</span>

        <Button type={config.isEnabled ? 'primary' : 'default'} onClick={openModal}>
          {labels.config}
        </Button>
      </div>

      <Modal
        open={isModalOpen}
        closable={false}
        centered={true}
        footer={footer}
        onCancel={() => setIsModalOpen(false)}
      >
        <div className="flex flex-col space-y-3">
          <div className="flex flex-col rounded-lg bg-neutral-800 p-3">
            <div className="flex items-center">
              <span className="text-lg">{labels.title}</span>
            </div>
            <span className="text-xs text-neutral-500">{labels.tip}</span>
          </div>

          <div className="flex justify-between rounded-lg bg-neutral-800 p-3">
            <span>{labels.enable}</span>
            <Switch value={isEnabled} disabled={isLoading} onChange={setIsEnabled} />
          </div>

          {isEnabled && (
            <Input
              autoFocus
              value={value}
              placeholder={labels.placeholder}
              maxLength={asciiOnly ? maxLength : undefined}
              disabled={isLoading}
              onFocus={() => setResult(null)}
              onChange={(e) => setValue(e.target.value)}
            />
          )}

          <div className="h-7">
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
          </div>
        </div>
      </Modal>
    </>
  );
};
