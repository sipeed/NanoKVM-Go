import { useEffect, useState } from 'react';
import { Button, Input, Modal, Switch } from 'antd';
import clsx from 'clsx';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';

type UsbIdConfig = {
  isEnabled: boolean;
  vid: string;
  pid: string;
  bcdDevice: string;
  serialnumber: string;
  manufacturer: string;
  product: string;
};

type Result = {
  type: 'success' | 'failed';
  message: string;
};

const emptyConfig: UsbIdConfig = {
  isEnabled: false,
  vid: '',
  pid: '',
  bcdDevice: '',
  serialnumber: '',
  manufacturer: '',
  product: ''
};

const usbIdHexRegex = /^(0x)?[0-9a-fA-F]{1,4}$/;

export const UsbId = () => {
  const { t } = useTranslation();

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [config, setConfig] = useState<UsbIdConfig>(emptyConfig);
  const [isEnabled, setIsEnabled] = useState(false);
  const [vid, setVid] = useState('');
  const [pid, setPid] = useState('');
  const [bcdDevice, setBcdDevice] = useState('');
  const [serialnumber, setSerialnumber] = useState('');
  const [manufacturer, setManufacturer] = useState('');
  const [product, setProduct] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [result, setResult] = useState<Result | null>(null);

  useEffect(() => {
    getUsbId();
  }, []);

  function openModal() {
    setIsEnabled(config.isEnabled);
    setVid(config.vid);
    setPid(config.pid);
    setBcdDevice(config.bcdDevice);
    setSerialnumber(config.serialnumber);
    setManufacturer(config.manufacturer);
    setProduct(config.product);
    setResult(null);
    setIsModalOpen(true);
  }

  function getUsbId() {
    api.getUsbId().then((rsp) => {
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        return;
      }

      const next: UsbIdConfig = {
        isEnabled: rsp.data.enabled,
        vid: rsp.data.vid ?? '',
        pid: rsp.data.pid ?? '',
        bcdDevice: rsp.data.bcdDevice ?? '',
        serialnumber: rsp.data.serialnumber ?? '',
        manufacturer: rsp.data.manufacturer ?? '',
        product: rsp.data.product ?? ''
      };
      setConfig(next);
      setIsEnabled(next.isEnabled);
      setVid(next.vid);
      setPid(next.pid);
      setBcdDevice(next.bcdDevice);
      setSerialnumber(next.serialnumber);
      setManufacturer(next.manufacturer);
      setProduct(next.product);
    });
  }

  async function updateUsbId() {
    const nextVid = isEnabled ? vid.trim() : '';
    const nextPid = isEnabled ? pid.trim() : '';
    const nextBcd = isEnabled ? bcdDevice.trim() : '';
    const nextSerial = isEnabled ? serialnumber.trim() : '';
    const nextMfg = isEnabled ? manufacturer.trim() : '';
    const nextProduct = isEnabled ? product.trim() : '';

    if (
      config.isEnabled === isEnabled &&
      config.vid === nextVid &&
      config.pid === nextPid &&
      config.bcdDevice === nextBcd &&
      config.serialnumber === nextSerial &&
      config.manufacturer === nextMfg &&
      config.product === nextProduct
    ) {
      setIsModalOpen(false);
      return;
    }

    if (isEnabled && !validContent()) {
      return;
    }

    if (isLoading) return;
    setIsLoading(true);
    setResult(null);

    try {
      const rsp = await api.setUsbId(isEnabled, {
        vid: nextVid,
        pid: nextPid,
        bcdDevice: nextBcd,
        serialnumber: nextSerial,
        manufacturer: nextMfg,
        product: nextProduct
      });
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        setIsLoading(false);
        setResult({ type: 'failed', message: t('settings.device.usbId.failed') });
        return;
      }

      setConfig({
        isEnabled,
        vid: nextVid,
        pid: nextPid,
        bcdDevice: nextBcd,
        serialnumber: nextSerial,
        manufacturer: nextMfg,
        product: nextProduct
      });
      setVid(nextVid);
      setPid(nextPid);
      setBcdDevice(nextBcd);
      setSerialnumber(nextSerial);
      setManufacturer(nextMfg);
      setProduct(nextProduct);
      setIsLoading(false);
      setResult({ type: 'success', message: t('settings.device.usbId.success') });
    } catch (err) {
      console.log(err);
      setResult({ type: 'failed', message: t('settings.device.usbId.failed') });
      setIsLoading(false);
    }
  }

  function validContent() {
    if (vid.trim() === '' || pid.trim() === '') {
      setResult({ type: 'failed', message: t('settings.device.usbId.notEmpty') });
      return false;
    }

    if (!usbIdHexRegex.test(vid.trim()) || !usbIdHexRegex.test(pid.trim())) {
      setResult({ type: 'failed', message: t('settings.device.usbId.notValid') });
      return false;
    }

    if (bcdDevice.trim() !== '' && !usbIdHexRegex.test(bcdDevice.trim())) {
      setResult({ type: 'failed', message: t('settings.device.usbId.bcdNotValid') });
      return false;
    }

    return true;
  }

  const footer = [
    <Button key="cancel" disabled={isLoading} onClick={() => setIsModalOpen(false)}>
      {t('settings.device.usbId.cancelBtn')}
    </Button>,
    <Button key="ok" type="primary" loading={isLoading} onClick={updateUsbId}>
      {t('settings.device.usbId.okBtn')}
    </Button>
  ];

  return (
    <>
      <div className="flex items-center justify-between">
        <span>{t('settings.device.usbId.title')}</span>

        <Button onClick={openModal}>{t('settings.device.usbId.config')}</Button>
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
              <span className="text-lg">{t('settings.device.usbId.title')}</span>
            </div>
            <span className="text-xs text-neutral-500">{t('settings.device.usbId.tip')}</span>
          </div>

          <div className="flex justify-between rounded-lg bg-neutral-800 p-3">
            <span>{t('settings.device.usbId.enable')}</span>
            <Switch value={isEnabled} disabled={isLoading} onChange={setIsEnabled} />
          </div>

          {isEnabled && (
            <div className="flex flex-col space-y-3 rounded-lg bg-neutral-800 p-3">
              <div className="flex flex-col space-y-1">
                <span className="text-xs text-neutral-500">{t('settings.device.usbId.vid')}</span>
                <Input
                  autoFocus
                  value={vid}
                  placeholder={t('settings.device.usbId.vidPlaceholder')}
                  disabled={isLoading}
                  onFocus={() => setResult(null)}
                  onChange={(e) => setVid(e.target.value)}
                />
              </div>
              <div className="flex flex-col space-y-1">
                <span className="text-xs text-neutral-500">{t('settings.device.usbId.pid')}</span>
                <Input
                  value={pid}
                  placeholder={t('settings.device.usbId.pidPlaceholder')}
                  disabled={isLoading}
                  onFocus={() => setResult(null)}
                  onChange={(e) => setPid(e.target.value)}
                />
              </div>
              <div className="flex flex-col space-y-1">
                <span className="text-xs text-neutral-500">
                  {t('settings.device.usbId.bcdDevice')}
                </span>
                <Input
                  value={bcdDevice}
                  placeholder={t('settings.device.usbId.bcdDevicePlaceholder')}
                  disabled={isLoading}
                  onFocus={() => setResult(null)}
                  onChange={(e) => setBcdDevice(e.target.value)}
                />
              </div>
              <div className="flex flex-col space-y-1">
                <span className="text-xs text-neutral-500">
                  {t('settings.device.usbId.serialnumber')}
                </span>
                <Input
                  value={serialnumber}
                  placeholder={t('settings.device.usbId.serialnumberPlaceholder')}
                  disabled={isLoading}
                  onFocus={() => setResult(null)}
                  onChange={(e) => setSerialnumber(e.target.value)}
                />
              </div>
              <div className="flex flex-col space-y-1">
                <span className="text-xs text-neutral-500">
                  {t('settings.device.usbId.manufacturer')}
                </span>
                <Input
                  value={manufacturer}
                  placeholder={t('settings.device.usbId.manufacturerPlaceholder')}
                  disabled={isLoading}
                  onFocus={() => setResult(null)}
                  onChange={(e) => setManufacturer(e.target.value)}
                />
              </div>
              <div className="flex flex-col space-y-1">
                <span className="text-xs text-neutral-500">
                  {t('settings.device.usbId.product')}
                </span>
                <Input
                  value={product}
                  placeholder={t('settings.device.usbId.productPlaceholder')}
                  disabled={isLoading}
                  onFocus={() => setResult(null)}
                  onChange={(e) => setProduct(e.target.value)}
                />
              </div>
            </div>
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
