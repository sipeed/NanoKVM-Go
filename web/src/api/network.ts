import { http } from '@/lib/http.ts';

export type WifiNetwork = {
  ssid: string;
  bssid: string;
  signal: number;
  frequency: string;
  security: string;
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function normalizeWifiFrequency(value: unknown): string | null {
  if (typeof value === 'string' && value.trim()) return value.trim();
  if (typeof value === 'number' && Number.isFinite(value)) return `${value} MHz`;
  return null;
}

function normalizeWifiNetwork(value: unknown): WifiNetwork | null {
  if (!isRecord(value)) return null;

  const ssid = value.ssid;
  const bssid = value.bssid;
  const signal = value.signal;
  const frequency = normalizeWifiFrequency(value.frequency);
  const security = value.security;
  if (
    typeof ssid !== 'string' ||
    !ssid.trim() ||
    typeof bssid !== 'string' ||
    !bssid.trim() ||
    typeof signal !== 'number' ||
    !Number.isFinite(signal) ||
    frequency === null ||
    typeof security !== 'string' ||
    !security.trim()
  ) {
    return null;
  }

  return { ssid: ssid.trim(), bssid: bssid.trim(), signal, frequency, security: security.trim() };
}

export function normalizeWifiList(data: unknown): WifiNetwork[] {
  if (!isRecord(data) || !Array.isArray(data.wifiList)) return [];
  return data.wifiList.flatMap((wifi) => {
    const normalized = normalizeWifiNetwork(wifi);
    return normalized ? [normalized] : [];
  });
}

// wake on lan
export function wol(mac: string) {
  const data = {
    mac
  };
  return http.post('/api/network/wol', data);
}

// get wake-on-lan macs history
export function getWolMacs() {
  return http.get('/api/network/wol/mac');
}

export function deleteWolMac(mac: string) {
  const data = {
    mac
  };
  return http.delete('/api/network/wol/mac', data);
}

// set Mac name
export function setWolMacName(mac: string, name: string) {
  return http.post('/api/network/wol/mac/name', { mac, name });
}

// get static IP
export function getStaticIp() {
  return http.get('/api/network/static-ip');
}

// set static IP
export function setStaticIP(enabled: boolean, ip: string) {
  const data = {
    enabled: enabled,
    ip: ip
  };
  return http.post('/api/network/static-ip', data);
}

// scan wifi
// AP portal requests always carry this header: a station on the device's AP is
// authorized without the AP password, and the custom header keeps other sites
// from forging such requests.
function apPortalHeaders(apPassword: string) {
  return {
    'X-NanoKVM-AP-Portal': '1',
    ...(apPassword ? { 'X-AP-Key': apPassword } : {})
  };
}

// Pass apPassword (even empty) only from the AP portal; the settings page omits it.
export function scanWiFi(apPassword?: string) {
  return http.request({
    method: 'get',
    url: '/api/network/wifi/scan',
    headers: apPassword === undefined ? undefined : apPortalHeaders(apPassword)
  });
}

// connect wifi without auth (only available in wifi configuration mode)
export function connectWifiNoAuth(ssid: string, password: string, apPassword: string) {
  const data = {
    ssid,
    password
  };
  return http.post('/api/network/wifi', data, {
    headers: apPortalHeaders(apPassword)
  });
}

// verify ap login
export function verifyApLogin(apPassword: string) {
  return http.post(
    '/api/network/wifi/verify',
    {},
    {
      headers: apPortalHeaders(apPassword)
    }
  );
}

// connect wifi
export function connectWifi(ssid: string, password: string) {
  const data = {
    ssid,
    password
  };
  return http.post('/api/network/wifi/connect', data);
}

// disconnect wifi
export function disconnectWifi() {
  return http.post('/api/network/wifi/disconnect');
}

// pre-store target wifi credentials (overwrite the previous preset)
export function presetWifi(ssid: string, password: string) {
  const data = {
    ssid,
    password
  };
  return http.post('/api/network/wifi/preset', data);
}

// get the pre-stored wifi ssid (empty string when absent)
export function getWifiPreset() {
  return http.get('/api/network/wifi/preset');
}

// remove the pre-stored wifi
export function removeWifiPreset() {
  return http.post('/api/network/wifi/preset/remove');
}

// get connected wifi information
export function getWiFi() {
  return http.get('/api/network/wifi');
}
