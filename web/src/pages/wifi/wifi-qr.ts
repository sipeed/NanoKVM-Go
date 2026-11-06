export type QrWifi = {
  ssid: string;
  password: string;
};

const MAX_SSID_BYTES = 32;
// Mirrors the server-side limit in service/network/wifi.go connect().
const MAX_PASSWORD_LENGTH = 128;

/** Validate credentials from a QR code against what the device can accept. */
export function normalizeQrWifi(ssid: string | null, password: string): QrWifi | null {
  if (ssid === null || !ssid.trim()) return null;
  if (new TextEncoder().encode(ssid).length > MAX_SSID_BYTES) return null;
  if (password.length > MAX_PASSWORD_LENGTH) return null;
  if (/[\r\n]/.test(ssid) || /[\r\n]/.test(password)) return null;
  return { ssid, password };
}

/**
 * Parse the de-facto `WIFI:T:WPA;S:<ssid>;P:<password>;;` format that phones
 * show when sharing a network. `\`, `;`, `,`, `:` and `"` are backslash-escaped.
 */
function parseWifiScheme(text: string): QrWifi | null {
  const fields = new Map<string, string>();
  let key = '';
  let value = '';
  let inValue = false;
  // Some generators wrap values in double quotes; unescaped ones are delimiters.
  let quoted = false;

  const flush = () => {
    if (inValue && !fields.has(key)) fields.set(key, value);
    key = '';
    value = '';
    inValue = false;
  };

  for (let i = 5; i < text.length; i++) {
    let char = text[i];
    if (char === '\\' && i + 1 < text.length) {
      char = text[++i];
    } else if (char === '"' && inValue) {
      quoted = !quoted;
      continue;
    } else if (char === ';' && !quoted) {
      flush();
      quoted = false;
      continue;
    } else if (char === ':' && !inValue) {
      inValue = true;
      continue;
    }

    if (inValue) value += char;
    else key += char.toUpperCase();
  }
  flush();

  const type = (fields.get('T') ?? '').toLowerCase();
  const password = type === 'nopass' ? '' : (fields.get('P') ?? '');
  return normalizeQrWifi(fields.get('S') ?? null, password);
}

/** Parse this page's own provisioning URL: `http://<gateway>/#/wifi?ssid=..&psk=..`. */
function parseProvisioningUrl(text: string): QrWifi | null {
  let url: URL;
  try {
    url = new URL(text);
  } catch {
    return null;
  }

  const queryIndex = url.hash.indexOf('?');
  if (queryIndex < 0 || url.hash.slice(0, queryIndex) !== '#/wifi') return null;

  const params = new URLSearchParams(url.hash.slice(queryIndex + 1));
  return normalizeQrWifi(params.get('ssid'), params.get('psk') || '');
}

/** Extract Wi-Fi credentials from decoded QR text, or null if it holds none. */
export function parseWifiQrText(text: string): QrWifi | null {
  const trimmed = text.trim();
  if (trimmed.slice(0, 5).toUpperCase() === 'WIFI:') return parseWifiScheme(trimmed);
  return parseProvisioningUrl(trimmed);
}
