import { useEffect, useRef, useState } from 'react';
import { LockOutlined, WifiOutlined } from '@ant-design/icons';
import { Alert, Button, Divider, Input, Modal, Popover } from 'antd';
import {
  LinkIcon,
  LoaderCircleIcon,
  LoaderIcon,
  LockIcon,
  PlusIcon,
  Trash2Icon,
  WifiHighIcon,
  WifiIcon,
  WifiLowIcon,
  WifiOffIcon,
  WifiPenIcon
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/network.ts';

type WiFi = {
  ssid: string;
  bssid: string;
  signal: number;
  frequency: string;
  security: string;
};

export const Wifi = () => {
  const { t } = useTranslation();

  const [isSupported, setIsSupported] = useState(false);
  const [isAPMode, setIsAPMode] = useState(false);
  const [connectedWifi, setConnectedWifi] = useState<WiFi | null>();

  const [wifiList, setWifiList] = useState<WiFi[]>([]);
  const [isScanning, setIsScanning] = useState(false);

  const [isPopoverOpen, setIsPopoverOpen] = useState(false);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [selectedWifi, setSelectedWifi] = useState<WiFi | null>(null);

  const [ssid, setSsid] = useState('');
  const [password, setPassword] = useState('');
  const [isConnecting, setIsConnecting] = useState(false);
  const [connectingBssid, setConnectingBssid] = useState('');
  const [message, setMessage] = useState('');

  const [isDisconnectModalOpen, setIsDisconnectModalOpen] = useState(false);
  const [isDisconnecting, setIsDisconnecting] = useState(false);

  const [presetSsid, setPresetSsid] = useState<string | null>(null);
  const [isPresetModalOpen, setIsPresetModalOpen] = useState(false);
  const [presetInputSsid, setPresetInputSsid] = useState('');
  const [presetInputPassword, setPresetInputPassword] = useState('');
  const [isPresetSaving, setIsPresetSaving] = useState(false);
  const [presetMessage, setPresetMessage] = useState('');
  const [isPresetRemoveModalOpen, setIsPresetRemoveModalOpen] = useState(false);
  const [isPresetRemoving, setIsPresetRemoving] = useState(false);
  const [presetRemoveMessage, setPresetRemoveMessage] = useState('');

  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    getWiFi();
    scanWiFi();
    getPreset();
  }, []);

  useEffect(() => {
    if (isPopoverOpen && !intervalRef.current) {
      intervalRef.current = setInterval(scanWiFi, 10 * 1000);
    }

    if (!isPopoverOpen && intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
  }, [isPopoverOpen]);

  function getWiFi() {
    api.getWiFi().then((rsp) => {
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        return;
      }

      setIsSupported(!!rsp.data?.supported);
      setIsAPMode(!!rsp.data?.apMode);

      if (rsp.data?.connected && rsp.data?.wifi) {
        setConnectedWifi(rsp.data.wifi);
      } else {
        setConnectedWifi(null);
      }
    });
  }

  function scanWiFi() {
    if (isScanning || isConnecting) return;
    setIsScanning(true);

    api
      .scanWiFi()
      .then((rsp) => {
        if (rsp.code !== 0) {
          setIsAPMode(rsp.code === -1);
          return;
        }

        if (rsp.data?.wifiList) {
          const list = sortWifiList(rsp.data.wifiList);
          setWifiList(list);
        }
      })
      .finally(() => {
        setIsScanning(false);
      });
  }

  function connectWifi(name?: string, pwd?: string) {
    setMessage('');

    const wifiSSID = name ? name : ssid;
    const wifiPwd = name ? pwd || '' : password;

    if (isConnecting || !wifiSSID) return;
    setIsConnecting(true);

    api
      .connectWifi(wifiSSID, wifiPwd)
      .then((rsp) => {
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          setMessage(t('wifi.errors.connect'));
          return;
        }

        getWiFi();
        scanWiFi();
        setIsModalOpen(false);
      })
      .finally(() => {
        setIsConnecting(false);
        setConnectingBssid('');
      });
  }

  function disconnectWifi() {
    if (isDisconnecting) return;
    setIsDisconnecting(true);

    api
      .disconnectWifi()
      .then((rsp) => {
        if (rsp.code !== 0) {
          console.log(rsp.msg);
        }

        getWiFi();
        scanWiFi();
        setIsDisconnectModalOpen(false);
      })
      .finally(() => {
        setIsDisconnecting(false);
      });
  }

  function getPreset() {
    api.getWifiPreset().then((rsp) => {
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        return;
      }

      setPresetSsid(rsp.data?.ssid || '');
    });
  }

  function openPresetModal() {
    setPresetMessage('');
    setPresetInputSsid('');
    setPresetInputPassword('');
    setIsPresetModalOpen(true);
  }

  function closePresetModal() {
    if (isPresetSaving) return;

    setIsPresetModalOpen(false);
    setPresetInputPassword('');
    setPresetMessage('');
  }

  function savePreset() {
    setPresetMessage('');

    if (isPresetSaving || !presetInputSsid.trim()) return;
    setIsPresetSaving(true);

    api
      .presetWifi(presetInputSsid, presetInputPassword)
      .then((rsp) => {
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          setPresetMessage(t('settings.device.wifi.preset.saveFailed'));
          return;
        }

        getPreset();
        setIsPresetModalOpen(false);
        setPresetInputPassword('');
      })
      .catch((err) => {
        console.log(err);
        setPresetMessage(t('settings.device.wifi.preset.saveFailed'));
      })
      .finally(() => {
        setIsPresetSaving(false);
      });
  }

  function openPresetRemoveModal() {
    setPresetRemoveMessage('');
    setIsPresetRemoveModalOpen(true);
  }

  function closePresetRemoveModal() {
    if (isPresetRemoving) return;

    setIsPresetRemoveModalOpen(false);
    setPresetRemoveMessage('');
  }

  function removePreset() {
    setPresetRemoveMessage('');

    if (isPresetRemoving) return;
    setIsPresetRemoving(true);

    api
      .removeWifiPreset()
      .then((rsp) => {
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          setPresetRemoveMessage(t('settings.device.wifi.preset.removeFailed'));
          return;
        }

        setPresetSsid('');
        setIsPresetRemoveModalOpen(false);
      })
      .catch((err) => {
        console.log(err);
        setPresetRemoveMessage(t('settings.device.wifi.preset.removeFailed'));
      })
      .finally(() => {
        setIsPresetRemoving(false);
      });
  }

  function sortWifiList(list: WiFi[]) {
    const arr = Array.isArray(list) ? [...list] : [];

    arr.sort((a, b) => {
      const sa = (a?.ssid || '').toLowerCase();
      const sb = (b?.ssid || '').toLowerCase();
      if (sa < sb) return -1;
      if (sa > sb) return 1;
      return 0;
    });

    return arr;
  }

  function getWifiSignalIcon(signal: number) {
    if (signal >= -60) {
      return <WifiIcon size={14} />;
    } else if (signal >= -70 && signal < -60) {
      return <WifiHighIcon size={14} />;
    } else {
      return <WifiLowIcon size={14} />;
    }
  }

  function handleOpenChange(open: boolean) {
    if (open) {
      scanWiFi();
      setIsPopoverOpen(true);
      return;
    }

    if (isModalOpen || isDisconnectModalOpen) {
      return;
    }

    setIsPopoverOpen(false);
  }

  function handleSelect(wifi: WiFi | null) {
    setMessage('');

    if (isConnecting) return;
    if (wifi && wifi.bssid === connectedWifi?.bssid) return;

    setSsid(wifi ? wifi.ssid : '');
    setPassword('');
    setConnectingBssid(wifi ? wifi.bssid : '');

    if (wifi && wifi.security.toLowerCase() === 'open') {
      connectWifi(wifi.ssid, '');
      return;
    }

    setSelectedWifi(wifi);
    setIsModalOpen(true);
  }

  // Wi-Fi list
  const content = (
    <div className="flex max-h-[400px] w-[280px] flex-col space-y-1 overflow-y-auto p-3">
      {/* connected Wi-Fi */}
      {connectedWifi && (
        <>
          <div
            className="flex cursor-pointer items-center justify-between rounded-lg px-2 py-1.5 hover:bg-neutral-500/20"
            onClick={() => setIsDisconnectModalOpen(true)}
          >
            <div className="w-2/3 truncate">{connectedWifi.ssid}</div>
            <div className="flex w-1/3 items-center justify-end space-x-2 text-blue-500">
              {connectedWifi.security.toLowerCase() !== 'open' && <LockIcon size={14} />}
              {getWifiSignalIcon(connectedWifi.signal)}
            </div>
          </div>

          <Divider />
        </>
      )}

      {wifiList.map((wifi) => (
        <div
          key={wifi.bssid}
          className="flex cursor-pointer items-center justify-between rounded-lg px-2 py-1.5 hover:bg-neutral-500/20"
          onClick={() => handleSelect(wifi)}
        >
          <div className="w-2/3 truncate">{wifi.ssid}</div>
          <div className="flex w-1/3 items-center justify-end space-x-2">
            {wifi.security.toLowerCase() !== 'open' && <LockIcon size={14} />}
            {isConnecting && connectingBssid === wifi.bssid ? (
              <LoaderCircleIcon className="animate-spin text-green-500" size={14} />
            ) : (
              getWifiSignalIcon(wifi.signal)
            )}
          </div>
        </div>
      ))}

      {isScanning && (
        <div className="flex justify-center py-1.5 text-neutral-500">
          <LoaderIcon className="animate-spin" size={16} />
        </div>
      )}

      <div
        className="flex cursor-pointer items-center rounded-lg px-2 py-1.5 hover:bg-neutral-500/20"
        onClick={() => handleSelect(null)}
      >
        {t('settings.device.wifi.others')}
      </div>
    </div>
  );

  if (!isSupported) {
    return <></>;
  }

  return (
    <>
      <div className="flex flex-col space-y-6">
        {isAPMode ? (
          <div className="flex items-center justify-between">
            <div className="flex flex-col">
              <span>{t('settings.device.wifi.title')}</span>
              <span className="text-xs text-neutral-500">{t('settings.device.wifi.apMode')}</span>
            </div>

            <Button shape="round" onClick={() => handleSelect(null)}>
              <LinkIcon size={14} />
            </Button>
          </div>
        ) : (
          <div className="flex items-center justify-between">
            <div className="flex flex-col space-y-1">
              <span>{t('settings.device.wifi.title')}</span>
              <span className="text-xs text-neutral-500">
                {t('settings.device.wifi.description')}
              </span>
            </div>

            <Popover
              className="p-3"
              styles={{ body: { padding: 0 } }}
              placement="bottomRight"
              trigger="click"
              content={content}
              arrow={false}
              open={isPopoverOpen}
              onOpenChange={handleOpenChange}
            >
              {connectedWifi ? (
                <Button type="primary" shape="round" size="small">
                  <div className="flex items-center justify-center px-1.5">
                    <WifiIcon size={16} />
                  </div>
                </Button>
              ) : (
                <Button shape="round" size="small">
                  <div className="flex items-center justify-center px-1.5">
                    <WifiIcon size={16} />
                  </div>
                </Button>
              )}
            </Popover>
          </div>
        )}

        {/* Wi-Fi preset (hidden in AP mode: saving a preset starts wpa_supplicant, which conflicts with the running hostapd) */}
        {!isAPMode && (
          <div className="flex items-center justify-between">
            <div className="flex min-w-0 flex-col space-y-1 pr-4">
              <span>{t('settings.device.wifi.preset.title')}</span>
              <span className="text-xs text-neutral-500">
                {t('settings.device.wifi.preset.description')}
              </span>
            </div>

            <div className="flex flex-shrink-0 items-center space-x-2">
              {presetSsid === null ? (
                <LoaderIcon
                  className="animate-spin text-neutral-500"
                  size={16}
                  aria-hidden="true"
                />
              ) : presetSsid ? (
                <span className="max-w-[140px] truncate text-sm text-neutral-300">
                  {presetSsid}
                </span>
              ) : (
                <span className="text-xs text-neutral-500">
                  {t('settings.device.wifi.preset.empty')}
                </span>
              )}

              {presetSsid === null ? null : presetSsid ? (
                <Button
                  danger
                  shape="round"
                  size="small"
                  aria-label={t('settings.device.wifi.preset.removeBtn')}
                  onClick={openPresetRemoveModal}
                >
                  <div className="flex items-center justify-center px-1.5">
                    <Trash2Icon size={16} />
                  </div>
                </Button>
              ) : (
                <Button
                  shape="round"
                  size="small"
                  aria-label={t('settings.device.wifi.preset.addBtn')}
                  onClick={openPresetModal}
                >
                  <div className="flex items-center justify-center px-1.5">
                    <PlusIcon size={16} />
                  </div>
                </Button>
              )}
            </div>
          </div>
        )}
      </div>

      <Modal
        closable={false}
        open={isModalOpen}
        centered={true}
        onOk={() => connectWifi()}
        onCancel={() => {
          if (!isConnecting) {
            setIsModalOpen(false);
          }
        }}
        okText={t('settings.device.wifi.joinBtn')}
        cancelText={t('settings.device.wifi.cancelBtn')}
        confirmLoading={isConnecting}
      >
        {/* title */}
        <div className="flex items-center space-x-5">
          <div className="h-[64px] w-[64px]">
            <WifiPenIcon size={64} className="text-blue-400" />
          </div>

          {!selectedWifi ? (
            <div className="flex flex-col">
              <span className="text-lg font-bold">{t('settings.device.wifi.connect')}</span>
              <span className="text-xs text-neutral-500">
                {t('settings.device.wifi.connectDesc1')}
              </span>
            </div>
          ) : (
            <div className="flex flex-col space-y-1">
              <span className="break-all text-lg font-bold">{selectedWifi.ssid}</span>
              <span className="text-xs text-neutral-500">
                {t('settings.device.wifi.connectDesc2')}
              </span>
            </div>
          )}
        </div>

        {/* form */}
        <div className="flex flex-col items-center space-y-3 py-6">
          {!selectedWifi && (
            <Input
              value={ssid}
              style={{ width: '300px' }}
              prefix={<WifiOutlined />}
              placeholder={t('settings.device.wifi.ssid')}
              onChange={(e) => setSsid(e.target.value)}
            />
          )}
          <Input.Password
            value={password}
            style={{ width: '300px' }}
            prefix={<LockOutlined />}
            placeholder={t('settings.device.wifi.password')}
            onChange={(e) => setPassword(e.target.value)}
          />

          {!!message && <span className="text-sm text-red-500">{message}</span>}
        </div>
      </Modal>

      <Modal
        closable={false}
        open={isDisconnectModalOpen}
        centered={true}
        okText={t('settings.device.wifi.confirmBtn')}
        cancelText={t('settings.device.wifi.cancelBtn')}
        onOk={disconnectWifi}
        onCancel={() => {
          if (!isDisconnecting) {
            setIsDisconnectModalOpen(false);
          }
        }}
        okType="danger"
        confirmLoading={isDisconnecting}
      >
        <div className="flex items-center space-x-5">
          <div className="h-[64px] w-[64px]">
            <WifiOffIcon size={64} className="text-blue-400" />
          </div>

          <div className="flex flex-col space-y-1">
            <span className="break-all text-lg font-bold">{connectedWifi?.ssid}</span>
            <span className="text-xs text-neutral-500">{t('settings.device.wifi.disconnect')}</span>
          </div>
        </div>
      </Modal>

      <Modal
        closable={false}
        open={isPresetModalOpen}
        centered={true}
        onOk={savePreset}
        onCancel={closePresetModal}
        okText={t('settings.device.wifi.preset.saveBtn')}
        cancelText={t('settings.device.wifi.cancelBtn')}
        confirmLoading={isPresetSaving}
      >
        {/* title */}
        <div className="flex items-center space-x-5">
          <div className="h-[64px] w-[64px]">
            <WifiPenIcon size={64} className="text-blue-400" />
          </div>

          <div className="flex flex-col space-y-1">
            <span className="text-lg font-bold">{t('settings.device.wifi.preset.saveTitle')}</span>
            <span className="text-xs text-neutral-500">
              {t('settings.device.wifi.preset.saveDesc')}
            </span>
          </div>
        </div>

        {/* persistent preset warning */}
        <Alert
          type="warning"
          showIcon
          style={{ width: '100%' }}
          message={t('settings.device.wifi.preset.warning')}
        />

        {/* form */}
        <div className="flex flex-col items-center space-y-3 py-6">
          <Input
            value={presetInputSsid}
            style={{ width: '300px' }}
            prefix={<WifiOutlined />}
            placeholder={t('settings.device.wifi.ssid')}
            onChange={(e) => setPresetInputSsid(e.target.value)}
          />
          <Input.Password
            value={presetInputPassword}
            style={{ width: '300px' }}
            prefix={<LockOutlined />}
            placeholder={t('settings.device.wifi.preset.passwordPlaceholder')}
            onChange={(e) => setPresetInputPassword(e.target.value)}
          />

          {!!presetMessage && <span className="text-sm text-red-500">{presetMessage}</span>}
        </div>
      </Modal>

      <Modal
        closable={false}
        open={isPresetRemoveModalOpen}
        centered={true}
        okText={t('settings.device.wifi.confirmBtn')}
        cancelText={t('settings.device.wifi.cancelBtn')}
        onOk={removePreset}
        onCancel={closePresetRemoveModal}
        okType="danger"
        confirmLoading={isPresetRemoving}
      >
        <div className="flex items-center space-x-5">
          <div className="h-[64px] w-[64px]">
            <Trash2Icon size={64} className="text-blue-400" />
          </div>

          <div className="flex flex-col space-y-1">
            <span className="break-all text-lg font-bold">{presetSsid}</span>
            <span className="text-xs text-neutral-500">
              {t('settings.device.wifi.preset.removeConfirm')}
            </span>
            {!!presetRemoveMessage && (
              <span className="text-sm text-red-500">{presetRemoveMessage}</span>
            )}
          </div>
        </div>
      </Modal>
    </>
  );
};
