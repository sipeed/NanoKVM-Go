/**
 * Pure visual-mapping helpers for the AP provisioning portal Wi-Fi list.
 * Kept DOM-free so scripts/wifi-visual.test.mjs can exercise them with
 * node --test (same pattern as wifi-url.ts).
 */

export type WifiSignalLevel = 'full' | 'high' | 'low' | 'zero';

export type WifiBand = '2.4G' | '5G' | '6G';

/**
 * Map a dBm reading to one of four signal levels driving the lucide
 * Wifi/WifiHigh/WifiLow/WifiZero icons. Thresholds follow the common
 * mobile convention: -50 dBm or better is excellent, -60 good, -70 usable,
 * anything weaker is marginal. A missing/invalid reading degrades to the
 * weakest level instead of throwing.
 */
export function getWifiSignalLevel(signal: number): WifiSignalLevel {
  if (!Number.isFinite(signal)) return 'zero';
  if (signal >= -50) return 'full';
  if (signal >= -60) return 'high';
  if (signal >= -70) return 'low';
  return 'zero';
}

/**
 * Derive the band chip label from the normalized frequency string. The API
 * layer delivers either a backend-provided label like "2.4 GHz" / "5 GHz" or
 * a numeric channel frequency formatted as "2412 MHz". Unknown formats
 * return null so the chip is simply not rendered.
 */
export function getWifiBand(frequency: string): WifiBand | null {
  const value = frequency.trim().toLowerCase();
  if (!value) return null;

  const mhzMatch = value.match(/^(\d+(?:\.\d+)?)\s*mhz$/);
  if (mhzMatch) {
    const mhz = Number(mhzMatch[1]);
    if (mhz >= 2400 && mhz < 2500) return '2.4G';
    if (mhz >= 5000 && mhz < 5925) return '5G';
    if (mhz >= 5925 && mhz <= 7125) return '6G';
    return null;
  }

  const ghzMatch = value.match(/(\d+(?:\.\d+)?)\s*ghz/);
  if (ghzMatch) {
    const ghz = Number(ghzMatch[1]);
    if (ghz >= 2 && ghz < 3) return '2.4G';
    if (ghz >= 5 && ghz < 6) return '5G';
    if (ghz >= 6) return '6G';
    return null;
  }

  return null;
}
