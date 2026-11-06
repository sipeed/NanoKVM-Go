type ReplaceCurrentUrl = (url: string) => void;

export type WifiUrlParams = {
  /** AP password from the device's captive portal redirect (`p`/`P`). */
  apPassword: string;
  /** Target network from a provisioning QR code (`ssid`), or null when absent. */
  ssid: string | null;
  /** Target network password from a provisioning QR code (`psk`). */
  psk: string;
};

const SECRET_PARAMS = ['p', 'P', 'ssid', 'psk'];

/**
 * Consume the AP password and QR provisioning credentials from the hash route
 * and synchronously replace the current history entry without any of them.
 */
export function consumeWifiUrlParams(
  currentUrl: string,
  replaceCurrentUrl: ReplaceCurrentUrl
): WifiUrlParams {
  const params: WifiUrlParams = { apPassword: '', ssid: null, psk: '' };

  const url = new URL(currentUrl);
  const queryIndex = url.hash.indexOf('?');
  if (queryIndex < 0) return params;

  const route = url.hash.slice(0, queryIndex);
  const searchParams = new URLSearchParams(url.hash.slice(queryIndex + 1));
  if (!SECRET_PARAMS.some((name) => searchParams.has(name))) return params;

  params.apPassword = searchParams.get('p') || searchParams.get('P') || '';
  params.ssid = searchParams.get('ssid');
  params.psk = searchParams.get('psk') || '';

  SECRET_PARAMS.forEach((name) => searchParams.delete(name));
  const safeQuery = searchParams.toString();
  url.hash = safeQuery ? `${route}?${safeQuery}` : route;

  replaceCurrentUrl(url.href);
  return params;
}

export function consumeWifiUrlParamsFromCurrentUrl(): WifiUrlParams {
  return consumeWifiUrlParams(window.location.href, (safeUrl) => {
    window.history.replaceState(window.history.state, '', safeUrl);
  });
}
