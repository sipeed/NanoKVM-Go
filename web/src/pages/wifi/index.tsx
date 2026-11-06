import { useCallback, useEffect, useRef, useState } from 'react';
import type { ChangeEvent, ReactNode } from 'react';
import {
  CheckCircleFilled,
  KeyOutlined,
  LockOutlined,
  ReloadOutlined,
  WifiOutlined
} from '@ant-design/icons';
import { Button, Empty, Form, Input, Spin, Steps } from 'antd';
import {
  CameraIcon,
  CheckIcon,
  ImageIcon,
  ListCollapseIcon,
  ListIcon,
  LockIcon,
  SettingsIcon,
  ToggleRightIcon,
  WifiHighIcon,
  WifiIcon,
  WifiLowIcon,
  WifiOffIcon,
  WifiZeroIcon
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/network.ts';
import type { WifiNetwork } from '@/api/network.ts';
import { getWifiPortalViewMode, setWifiPortalViewMode } from '@/lib/localstorage.ts';
import type { WifiPortalViewMode } from '@/lib/localstorage.ts';
import { Head } from '@/components/head.tsx';
import { LanguagePicker } from '@/components/language-picker.tsx';
import { ButtonBase } from '@/components/ui/button-base.tsx';
import { normalizeQrWifi, parseWifiQrText } from '@/pages/wifi/wifi-qr.ts';
import type { QrWifi } from '@/pages/wifi/wifi-qr.ts';
import { consumeWifiUrlParamsFromCurrentUrl } from '@/pages/wifi/wifi-url.ts';
import { getWifiBand, getWifiSignalLevel } from '@/pages/wifi/wifi-visual.ts';
import type { WifiSignalLevel } from '@/pages/wifi/wifi-visual.ts';

type ConnectionState = '' | 'loading' | 'submitted' | 'failed' | 'denied';
type VerifyState = '' | 'failed' | 'denied';
type ScanState = 'idle' | 'loading' | 'ready' | 'empty' | 'failed';
type SetupMode = 'scan' | 'manual';
type QrError = '' | 'notFound' | 'invalid';

type VerifyFormValues = {
  apPassword?: unknown;
};

type PageShellProps = {
  children: ReactNode;
  footer?: ReactNode;
};

const VERIFY_FORM_ID = 'wifi-ap-verify';
const CONNECT_FORM_ID = 'wifi-connect-selected';
const MANUAL_FORM_ID = 'wifi-connect-manual';

// Captive portal windows (Android's sign-in page, iOS's login sheet) often
// cannot open a file picker or camera. If the page neither blurs nor hides
// shortly after the tap, assume nothing opened and suggest a real browser.
const PICKER_OPEN_TIMEOUT_MS = 1500;

// Footer action buttons must let long locale copy wrap and grow vertically
// instead of overflowing the mobile viewport. The ! utilities beat the runtime-
// injected antd button styles (white-space: nowrap and the fixed large height).
const FOOTER_BUTTON_CLASS =
  'min-h-12 !h-auto min-w-0 flex-1 !whitespace-normal break-words py-2 leading-tight';

// Keep keys literal so scripts/audit-i18n.mjs can discover them.
const SUBMITTED_TIP_KEYS = [
  'wifi.submitted.tipSwitch',
  'wifi.submitted.tipCheck',
  'wifi.submitted.tipRetry'
] as const;

function readInputString(value: unknown): string {
  return typeof value === 'string' ? value : '';
}

function isOpenNetwork(network: WifiNetwork): boolean {
  return network.security.toLowerCase() === 'open';
}

const SIGNAL_LEVEL_ICONS: Record<WifiSignalLevel, LucideIcon> = {
  full: WifiIcon,
  high: WifiHighIcon,
  low: WifiLowIcon,
  zero: WifiZeroIcon
};

/** Decorative signal-strength fan icon; the row text carries the network name. */
function NetworkSignalIcon({ signal }: { signal: number }) {
  const LevelIcon = SIGNAL_LEVEL_ICONS[getWifiSignalLevel(signal)];
  return <LevelIcon size={18} className="shrink-0 text-neutral-300" aria-hidden="true" />;
}

/** Small outlined band chip (2.4G/5G/6G); hidden when the band is unknown. */
function NetworkBandChip({ frequency }: { frequency: string }) {
  const band = getWifiBand(frequency);
  if (!band) return null;

  return (
    <span className="shrink-0 rounded border border-neutral-600 px-1.5 py-0.5 text-xs leading-none text-neutral-400">
      {band}
    </span>
  );
}

/** Padlock marker for secured networks; open networks render nothing. */
function NetworkLockIcon({ network }: { network: WifiNetwork }) {
  if (isOpenNetwork(network)) return null;
  return <LockIcon size={14} className="shrink-0 text-neutral-400" aria-hidden="true" />;
}

/**
 * App-like single-column shell for the public AP provisioning page: a fixed
 * header, a scrollable content column and a bottom action bar that stays above
 * the mobile browser chrome and the device safe area.
 */
function PageShell({ children, footer }: PageShellProps) {
  return (
    <div className="flex h-screen w-screen flex-col supports-[height:100dvh]:h-dvh">
      <header className="flex shrink-0 items-center justify-between gap-4 px-5 pb-2 pt-[max(1rem,env(safe-area-inset-top,0px))]">
        <div className="flex items-center gap-2 text-neutral-100">
          <WifiIcon size={20} aria-hidden="true" />
          <span className="text-base font-semibold">NanoKVM</span>
        </div>
        <LanguagePicker />
      </header>

      <main className="flex-1 overflow-y-auto px-5 pb-4">
        <div className="mx-auto flex w-full max-w-md flex-col">{children}</div>
      </main>

      {footer && (
        <footer className="shrink-0 px-5 pb-[max(1rem,env(safe-area-inset-bottom,0px))] pt-2">
          <div className="mx-auto flex w-full max-w-md gap-3">{footer}</div>
        </footer>
      )}
    </div>
  );
}

export const Wifi = () => {
  const { t } = useTranslation();
  const [connectionState, setConnectionState] = useState<ConnectionState>('');
  const [apPassword, setApPassword] = useState('');
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [verifying, setVerifying] = useState(false);
  const [verifyState, setVerifyState] = useState<VerifyState>('');
  const [scanState, setScanState] = useState<ScanState>('idle');
  const [setupMode, setSetupMode] = useState<SetupMode>('scan');
  const [wifiList, setWifiList] = useState<WifiNetwork[]>([]);
  const [selectedWifi, setSelectedWifi] = useState<WifiNetwork | null>(null);
  const [manualSsid, setManualSsid] = useState('');
  const [password, setPassword] = useState('');
  const [viewMode, setViewMode] = useState<WifiPortalViewMode>(() => getWifiPortalViewMode());
  const [qrWifi, setQrWifi] = useState<QrWifi | null>(null);
  const [qrError, setQrError] = useState<QrError>('');
  const [decodingQr, setDecodingQr] = useState(false);
  const [pickerUnavailable, setPickerUnavailable] = useState(false);

  const attemptedUrlVerification = useRef(false);
  const verifyingRef = useRef(false);
  const connectingRef = useRef(false);
  const scanRequestVersion = useRef(0);
  const activeScan = useRef(false);
  // Two inputs: `capture` opens the camera directly, the plain one lets the
  // user pick a screenshot. Live viewfinder scanning (getUserMedia) is not
  // available because the portal is served over plain http.
  const qrCameraInput = useRef<HTMLInputElement>(null);
  const qrLibraryInput = useRef<HTMLInputElement>(null);
  const stopPickerWatch = useRef<(() => void) | null>(null);

  const invalidateScan = useCallback(() => {
    scanRequestVersion.current += 1;
    activeScan.current = false;
  }, []);

  const scanNearbyNetworks = useCallback(async () => {
    if (activeScan.current || connectingRef.current) return;

    const requestVersion = ++scanRequestVersion.current;
    activeScan.current = true;
    setScanState('loading');

    try {
      const rsp = await api.scanWiFi(apPassword);
      if (requestVersion !== scanRequestVersion.current) return;

      if (!rsp || rsp.code !== 0) {
        setWifiList([]);
        setScanState('failed');
        return;
      }

      const networks = api.normalizeWifiList(rsp.data);
      setWifiList(networks);
      setScanState(networks.length > 0 ? 'ready' : 'empty');
    } catch {
      if (requestVersion !== scanRequestVersion.current) return;
      setWifiList([]);
      setScanState('failed');
    } finally {
      if (requestVersion === scanRequestVersion.current) activeScan.current = false;
    }
  }, [apPassword]);

  useEffect(() => {
    if (attemptedUrlVerification.current) return;
    attemptedUrlVerification.current = true;

    const urlParams = consumeWifiUrlParamsFromCurrentUrl();
    if (urlParams.ssid !== null) {
      const credentials = normalizeQrWifi(urlParams.ssid, urlParams.psk);
      if (credentials) setQrWifi(credentials);
      else setQrError('invalid');
    }

    // Without an AP password the device still admits phones joined to its AP,
    // so try silently and fall back to the password form.
    void verifyPassword(urlParams.apPassword, !urlParams.apPassword);
  }, []);

  useEffect(() => {
    if (!isAuthenticated || connectionState === 'submitted') return;
    if (setupMode !== 'scan' || selectedWifi !== null || qrWifi !== null) return;
    void scanNearbyNetworks();
  }, [connectionState, isAuthenticated, qrWifi, scanNearbyNetworks, selectedWifi, setupMode]);

  useEffect(() => {
    return () => {
      invalidateScan();
      stopPickerWatch.current?.();
    };
  }, [invalidateScan]);

  async function verifyPassword(candidate: string, silent = false) {
    if (verifyingRef.current) return;

    verifyingRef.current = true;
    setVerifying(true);
    setVerifyState('');

    try {
      const rsp = await api.verifyApLogin(candidate);
      if (rsp?.code === 0) {
        setApPassword(candidate);
        setIsAuthenticated(true);
      } else if (rsp?.code === -1) {
        setVerifyState('denied');
      } else if (!silent) {
        setVerifyState('failed');
      }
    } catch {
      if (!silent) setVerifyState('failed');
    } finally {
      verifyingRef.current = false;
      setVerifying(false);
    }
  }

  async function onVerifyFinish(values: VerifyFormValues) {
    const candidate = readInputString(values.apPassword);
    if (!candidate) return;
    await verifyPassword(candidate);
  }

  async function connectNetwork(ssid: string, networkPassword: string) {
    const normalizedSsid = ssid.trim();
    if (connectingRef.current || !normalizedSsid || !isAuthenticated) return;

    connectingRef.current = true;
    setConnectionState('loading');

    // The device tears its AP down before joining the target network, so the
    // response may never reach the browser. An accepted request and a dropped
    // transport are indistinguishable here: both mean the credentials were
    // handed over and the device is now trying on its own.
    const markSubmitted = () => {
      setPassword('');
      setApPassword('');
      setQrWifi(null);
      setConnectionState('submitted');
    };

    try {
      const rsp = await api.connectWifiNoAuth(normalizedSsid, networkPassword, apPassword);
      if (rsp?.code === 0) {
        markSubmitted();
        return;
      }

      setPassword('');
      setConnectionState(rsp?.code === -1 || rsp?.code === -4 ? 'denied' : 'failed');
    } catch {
      markSubmitted();
    } finally {
      connectingRef.current = false;
    }
  }

  function toggleViewMode() {
    setViewMode((current) => {
      const next: WifiPortalViewMode = current === 'simple' ? 'detailed' : 'simple';
      setWifiPortalViewMode(next);
      return next;
    });
  }

  function selectWifi(network: WifiNetwork) {
    if (connectingRef.current) return;

    invalidateScan();
    setConnectionState('');
    setSelectedWifi(network);
    setPassword('');

    if (isOpenNetwork(network)) {
      void connectNetwork(network.ssid, '');
    }
  }

  function enterManualSetup() {
    if (connectingRef.current) return;

    invalidateScan();
    setSetupMode('manual');
    setSelectedWifi(null);
    setManualSsid('');
    setPassword('');
    setConnectionState('');
  }

  function returnToNetworks() {
    if (connectingRef.current) return;

    invalidateScan();
    setQrWifi(null);
    setQrError('');
    setSetupMode('scan');
    setSelectedWifi(null);
    setManualSsid('');
    setPassword('');
    setConnectionState('');
    setScanState('idle');
  }

  function openQrPicker(input: HTMLInputElement | null) {
    if (!input || decodingQr || connectingRef.current) return;

    setQrError('');
    setPickerUnavailable(false);
    stopPickerWatch.current?.();

    function stop() {
      window.clearTimeout(timer);
      window.removeEventListener('blur', stop);
      document.removeEventListener('visibilitychange', onVisibilityChange);
      stopPickerWatch.current = null;
    }
    function onVisibilityChange() {
      if (document.visibilityState === 'hidden') stop();
    }
    const timer = window.setTimeout(() => {
      stop();
      setPickerUnavailable(true);
    }, PICKER_OPEN_TIMEOUT_MS);

    window.addEventListener('blur', stop);
    document.addEventListener('visibilitychange', onVisibilityChange);
    stopPickerWatch.current = stop;
    input.click();
  }

  async function onQrFileChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = '';
    stopPickerWatch.current?.();
    setPickerUnavailable(false);
    if (!file) return;

    setDecodingQr(true);
    setQrError('');
    try {
      const { decodeQrImage } = await import('@/pages/wifi/wifi-qr-image.ts');
      const text = await decodeQrImage(file);
      if (!text) {
        setQrError('notFound');
        return;
      }

      if (connectingRef.current) return;

      const credentials = parseWifiQrText(text);
      if (!credentials) {
        setQrError('invalid');
        return;
      }

      invalidateScan();
      setConnectionState('');
      setQrWifi(credentials);
    } catch {
      setQrError('notFound');
    } finally {
      setDecodingQr(false);
    }
  }

  function renderConnectingStatus() {
    if (connectionState !== 'loading') return null;

    return (
      <div className="mt-4 flex items-center gap-2 text-xs text-neutral-500" role="status">
        <Spin size="small" />
        {t('wifi.scan.connecting')}
      </div>
    );
  }

  function renderSelectedNetwork() {
    if (!selectedWifi) return null;

    const openNetwork = isOpenNetwork(selectedWifi);
    const isConnecting = connectionState === 'loading';

    return (
      <>
        <div className="flex flex-col gap-1 pt-4">
          <span className="break-all text-xl font-semibold text-neutral-100">
            {t('wifi.scan.connectTitle', { ssid: selectedWifi.ssid })}
          </span>
          {viewMode === 'simple' ? (
            <span className="flex items-center gap-2 pt-1">
              <NetworkSignalIcon signal={selectedWifi.signal} />
              <NetworkLockIcon network={selectedWifi} />
              <NetworkBandChip frequency={selectedWifi.frequency} />
            </span>
          ) : (
            <>
              <span className="text-xs text-neutral-500">
                {selectedWifi.frequency} · {selectedWifi.signal} dBm · {selectedWifi.security}
              </span>
              <span className="font-mono text-xs text-neutral-500">BSSID {selectedWifi.bssid}</span>
            </>
          )}
        </div>

        {!openNetwork && (
          <Form
            id={CONNECT_FORM_ID}
            className="mt-6"
            onFinish={() => {
              if (password.length <= 128) void connectNetwork(selectedWifi.ssid, password);
            }}
          >
            <Form.Item label={t('wifi.passwordPlaceholder')}>
              <Input.Password
                size="large"
                prefix={<LockOutlined />}
                placeholder={t('wifi.passwordPlaceholder')}
                autoComplete="current-password"
                maxLength={128}
                value={password}
                disabled={isConnecting}
                onChange={(event) => setPassword(event.target.value)}
              />
            </Form.Item>
          </Form>
        )}

        {renderConnectingStatus()}
      </>
    );
  }

  function renderNetworkItem(network: WifiNetwork) {
    const identity = (
      <>
        <NetworkLockIcon network={network} />
        <NetworkBandChip frequency={network.frequency} />
      </>
    );

    return (
      <Button
        key={network.bssid}
        className="h-auto min-h-14 w-full px-4 py-3 text-left"
        disabled={connectionState === 'loading' || decodingQr}
        onClick={() => selectWifi(network)}
      >
        <span className="flex w-full min-w-0 items-center gap-3">
          <NetworkSignalIcon signal={network.signal} />
          {viewMode === 'simple' ? (
            <>
              <span className="min-w-0 flex-1 truncate text-base font-medium">{network.ssid}</span>
              {identity}
            </>
          ) : (
            <span className="flex min-w-0 flex-1 flex-col gap-0.5">
              <span className="flex min-w-0 items-center gap-2">
                <span className="min-w-0 flex-1 truncate text-base font-medium">
                  {network.ssid}
                </span>
                {identity}
              </span>
              <span className="text-xs text-neutral-500">
                {network.frequency} · {network.signal} dBm · {network.security}
              </span>
              <span className="font-mono text-xs text-neutral-500">BSSID {network.bssid}</span>
            </span>
          )}
        </span>
      </Button>
    );
  }

  function renderScanSetup() {
    const viewModeToggleLabel = t(
      viewMode === 'simple' ? 'wifi.scan.viewDetailed' : 'wifi.scan.viewSimple'
    );

    return (
      <>
        <div className="flex items-start justify-between gap-3 pt-4">
          <div className="flex min-w-0 flex-col gap-1">
            <span className="text-xl font-semibold text-neutral-100">{t('wifi.scan.title')}</span>
            <span className="text-xs text-neutral-500">{t('wifi.scan.description')}</span>
          </div>
          <ButtonBase
            className="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg text-neutral-300 hover:bg-neutral-800 hover:text-neutral-100"
            aria-label={viewModeToggleLabel}
            title={viewModeToggleLabel}
            onClick={toggleViewMode}
          >
            {viewMode === 'simple' ? (
              <ListIcon size={20} aria-hidden="true" />
            ) : (
              <ListCollapseIcon size={20} aria-hidden="true" />
            )}
          </ButtonBase>
        </div>

        {renderQrScan()}

        {(scanState === 'idle' || scanState === 'loading') && (
          <div className="flex items-center gap-2 py-8 text-xs text-neutral-500" role="status">
            <Spin size="small" />
            {t('wifi.scan.loading')}
          </div>
        )}

        {scanState === 'ready' && (
          <div className="mt-4 flex flex-col gap-2" role="list">
            {wifiList.map((network) => renderNetworkItem(network))}
          </div>
        )}

        {scanState === 'empty' && (
          <div role="status">
            <Empty
              className="py-6"
              image={
                <div className="flex h-full items-center justify-center">
                  <WifiOffIcon size={40} strokeWidth={1.2} className="text-neutral-600" />
                </div>
              }
              description={t('wifi.scan.empty')}
            />
          </div>
        )}

        {scanState === 'failed' && (
          <span className="block py-6 text-sm text-red-500" role="alert">
            {t('wifi.scan.failed')}
          </span>
        )}
      </>
    );
  }

  function renderQrScan() {
    const portalUrl = `${window.location.origin}/#/wifi`;
    const pickerDisabled = decodingQr || connectionState === 'loading';

    return (
      <div className="mt-4 flex flex-col gap-2">
        <div className="flex gap-2">
          <Button
            className="h-auto min-h-12 min-w-0 flex-1 !whitespace-normal py-2"
            size="large"
            icon={<CameraIcon size={18} aria-hidden="true" />}
            disabled={pickerDisabled}
            onClick={() => openQrPicker(qrCameraInput.current)}
          >
            {t('wifi.qr.cameraBtn')}
          </Button>
          <Button
            className="h-auto min-h-12 min-w-0 flex-1 !whitespace-normal py-2"
            size="large"
            icon={<ImageIcon size={18} aria-hidden="true" />}
            disabled={pickerDisabled}
            onClick={() => openQrPicker(qrLibraryInput.current)}
          >
            {t('wifi.qr.pickBtn')}
          </Button>
        </div>
        <input
          ref={qrCameraInput}
          className="hidden"
          type="file"
          accept="image/*"
          capture="environment"
          onChange={(event) => void onQrFileChange(event)}
        />
        <input
          ref={qrLibraryInput}
          className="hidden"
          type="file"
          accept="image/*"
          onChange={(event) => void onQrFileChange(event)}
        />
        {decodingQr && (
          <div className="flex items-center gap-2 text-xs text-neutral-500" role="status">
            <Spin size="small" />
            {t('wifi.qr.decoding')}
          </div>
        )}
        <span className="text-xs text-neutral-500">{t('wifi.qr.scanHint')}</span>

        {qrError && (
          <span className="text-sm text-red-500" role="alert">
            {t(qrError === 'notFound' ? 'wifi.qr.notFound' : 'wifi.qr.invalid')}
          </span>
        )}

        {pickerUnavailable && (
          <div
            className="flex flex-col gap-1 rounded-lg border border-neutral-800 bg-neutral-900 p-3"
            role="status"
          >
            <span className="text-sm text-neutral-300">{t('wifi.qr.pickerUnavailable')}</span>
            <span className="select-all break-all font-mono text-sm text-neutral-100">
              {portalUrl}
            </span>
          </div>
        )}
      </div>
    );
  }

  function renderQrConfirm(credentials: QrWifi) {
    return (
      <>
        <div className="flex flex-col gap-1 pt-4">
          <span className="break-all text-xl font-semibold text-neutral-100">
            {t('wifi.qr.confirmTitle', { ssid: credentials.ssid })}
          </span>
          <span className="text-xs text-neutral-500">{t('wifi.qr.confirmDescription')}</span>
        </div>

        {credentials.password ? (
          <Form className="mt-6">
            <Form.Item label={t('wifi.passwordPlaceholder')}>
              <Input.Password
                size="large"
                prefix={<LockOutlined />}
                autoComplete="off"
                readOnly
                value={credentials.password}
              />
            </Form.Item>
          </Form>
        ) : (
          <span className="mt-6 text-sm text-neutral-400">{t('wifi.qr.openNetwork')}</span>
        )}

        {renderConnectingStatus()}
      </>
    );
  }

  function renderManualSetup() {
    const isConnecting = connectionState === 'loading';

    return (
      <>
        <div className="flex flex-col gap-1 pt-4">
          <span className="text-xl font-semibold text-neutral-100">{t('wifi.scan.manualBtn')}</span>
          <span className="text-xs text-neutral-500">{t('wifi.description')}</span>
        </div>

        <Form
          id={MANUAL_FORM_ID}
          className="mt-6"
          onFinish={() => {
            if (manualSsid.trim() && password.length <= 128) {
              void connectNetwork(manualSsid, password);
            }
          }}
        >
          <Form.Item label={t('wifi.ssidPlaceholder')}>
            <Input
              size="large"
              prefix={<WifiOutlined />}
              placeholder={t('wifi.ssidPlaceholder')}
              autoComplete="off"
              maxLength={128}
              required
              value={manualSsid}
              disabled={isConnecting}
              onChange={(event) => setManualSsid(event.target.value)}
            />
          </Form.Item>

          <Form.Item label={t('wifi.passwordPlaceholder')}>
            <Input.Password
              size="large"
              prefix={<LockOutlined />}
              placeholder={t('wifi.passwordPlaceholder')}
              autoComplete="current-password"
              maxLength={128}
              value={password}
              disabled={isConnecting}
              onChange={(event) => setPassword(event.target.value)}
            />
          </Form.Item>
        </Form>

        {renderConnectingStatus()}
      </>
    );
  }

  function renderSubmitted() {
    return (
      <div className="flex flex-col items-center pt-10" role="status">
        <CheckCircleFilled className="text-[44px] text-green-500" aria-hidden="true" />
        <span className="mt-4 text-center text-xl font-semibold text-neutral-100">
          {t('wifi.submitted.title')}
        </span>

        <div className="mt-6 flex w-full flex-col gap-3 rounded-xl border border-neutral-800 bg-neutral-900 p-4">
          <span className="text-sm font-medium text-neutral-200">
            {t('wifi.submitted.tipsTitle')}
          </span>
          <div className="flex flex-col gap-2.5">
            {SUBMITTED_TIP_KEYS.map((tipKey) => (
              <div key={tipKey} className="flex items-start gap-2.5">
                <CheckIcon
                  size={16}
                  className="mt-0.5 shrink-0 text-green-500"
                  aria-hidden="true"
                />
                <span className="text-sm leading-relaxed text-neutral-400">{t(tipKey)}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    );
  }

  function renderDenied() {
    return (
      <div className="flex flex-col items-center gap-5 pt-10">
        <span className="text-center text-sm text-red-500" role="alert">
          {t('wifi.invalidMode')}
        </span>
        <Steps
          className="w-full max-w-[400px]"
          size="small"
          responsive={false}
          items={[
            { title: t('wifi.steps.settings'), status: 'wait', icon: <SettingsIcon size={24} /> },
            { title: t('wifi.steps.wifi'), status: 'wait', icon: <WifiIcon size={24} /> },
            { title: t('wifi.steps.enable'), status: 'wait', icon: <ToggleRightIcon size={24} /> }
          ]}
        />
      </div>
    );
  }

  if (!isAuthenticated) {
    return (
      <>
        <Head title={t('head.wifi')} />

        <PageShell
          footer={
            <Button
              className={FOOTER_BUTTON_CLASS}
              htmlType="submit"
              form={VERIFY_FORM_ID}
              type="primary"
              size="large"
              loading={verifying}
            >
              {t('wifi.ap.verifyBtn')}
            </Button>
          }
        >
          <div className="flex flex-col gap-1 pt-6">
            <span className="text-2xl font-semibold text-neutral-100">
              {t('wifi.ap.authTitle')}
            </span>
            <span className="text-xs text-neutral-500">{t('wifi.ap.authDescription')}</span>
          </div>

          <Form id={VERIFY_FORM_ID} className="mt-6" onFinish={onVerifyFinish}>
            <Form.Item
              label={t('wifi.ap.passPlaceholder')}
              name="apPassword"
              rules={[{ required: true }]}
            >
              <Input.Password
                size="large"
                prefix={<KeyOutlined />}
                placeholder={t('wifi.ap.passPlaceholder')}
                autoComplete="current-password"
                disabled={verifying}
              />
            </Form.Item>
          </Form>

          {verifyState === 'failed' && (
            <span className="text-sm text-red-500" role="alert">
              {t('wifi.ap.authFailed')}
            </span>
          )}
          {verifyState === 'denied' && (
            <span className="text-sm text-red-500" role="alert">
              {t('wifi.invalidMode')}
            </span>
          )}
        </PageShell>
      </>
    );
  }

  const isConnecting = connectionState === 'loading';

  let content: ReactNode;
  let footer: ReactNode = null;

  if (connectionState === 'submitted') {
    content = renderSubmitted();
  } else if (connectionState === 'denied') {
    content = renderDenied();
  } else if (qrWifi) {
    content = renderQrConfirm(qrWifi);
    footer = (
      <>
        <Button
          className={FOOTER_BUTTON_CLASS}
          size="large"
          disabled={isConnecting}
          onClick={returnToNetworks}
        >
          {t('wifi.qr.otherBtn')}
        </Button>
        <Button
          className={FOOTER_BUTTON_CLASS}
          size="large"
          type="primary"
          loading={isConnecting}
          disabled={isConnecting}
          onClick={() => void connectNetwork(qrWifi.ssid, qrWifi.password)}
        >
          {t('wifi.qr.connectBtn')}
        </Button>
      </>
    );
  } else if (setupMode === 'scan') {
    if (selectedWifi) {
      content = renderSelectedNetwork();
      footer = (
        <>
          <Button
            className={FOOTER_BUTTON_CLASS}
            size="large"
            disabled={isConnecting}
            onClick={returnToNetworks}
          >
            {t('wifi.scan.backBtn')}
          </Button>
          {isOpenNetwork(selectedWifi) ? (
            <Button
              className={FOOTER_BUTTON_CLASS}
              size="large"
              type="primary"
              loading={isConnecting}
              disabled={isConnecting}
              onClick={() => void connectNetwork(selectedWifi.ssid, '')}
            >
              {t('wifi.confirmBtn')}
            </Button>
          ) : (
            <Button
              className={FOOTER_BUTTON_CLASS}
              htmlType="submit"
              form={CONNECT_FORM_ID}
              size="large"
              type="primary"
              loading={isConnecting}
              disabled={isConnecting}
            >
              {t('wifi.confirmBtn')}
            </Button>
          )}
        </>
      );
    } else {
      content = renderScanSetup();
      footer = (
        <>
          <Button
            className={FOOTER_BUTTON_CLASS}
            size="large"
            icon={<ReloadOutlined />}
            disabled={isConnecting}
            onClick={() => void scanNearbyNetworks()}
          >
            {t('wifi.scan.retryBtn')}
          </Button>
          <Button
            className={FOOTER_BUTTON_CLASS}
            size="large"
            disabled={isConnecting}
            onClick={enterManualSetup}
          >
            {t('wifi.scan.manualBtn')}
          </Button>
        </>
      );
    }
  } else {
    content = renderManualSetup();
    footer = (
      <>
        <Button
          className={FOOTER_BUTTON_CLASS}
          size="large"
          disabled={isConnecting}
          onClick={returnToNetworks}
        >
          {t('wifi.scan.backBtn')}
        </Button>
        <Button
          className={FOOTER_BUTTON_CLASS}
          htmlType="submit"
          form={MANUAL_FORM_ID}
          size="large"
          type="primary"
          loading={isConnecting}
          disabled={isConnecting}
        >
          {t('wifi.confirmBtn')}
        </Button>
      </>
    );
  }

  return (
    <>
      <Head title={t('head.wifi')} />

      <PageShell footer={footer}>
        {content}

        {connectionState === 'failed' && (
          <span className="mt-4 block text-center text-sm text-red-500" role="alert">
            {t('wifi.errors.connect')}
          </span>
        )}
      </PageShell>
    </>
  );
};
