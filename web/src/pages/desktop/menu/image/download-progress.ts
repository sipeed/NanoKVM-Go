export type DownloadTransferMetrics = {
  downloadedBytes?: number;
  totalBytes?: number;
  bytesPerSecond?: number;
  remainingSeconds?: number;
};

export type DownloadTransferDetails = {
  speed?: string;
  transferred: string;
  remaining?: string;
};

const byteUnits = ['B', 'KB', 'MB', 'GB', 'TB'] as const;

export function parseDownloadProgress(value?: string): number | undefined {
  const progress = Number.parseFloat(value ?? '');
  if (!Number.isFinite(progress)) return undefined;

  return Math.min(100, Math.max(0, progress));
}

export function formatDownloadBytes(bytes: number, locale: string): string {
  let value = Math.max(0, bytes);
  let unitIndex = 0;

  while (value >= 1024 && unitIndex < byteUnits.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }

  return `${new Intl.NumberFormat(locale, {
    maximumFractionDigits: unitIndex === 0 ? 0 : 1
  }).format(value)} ${byteUnits[unitIndex]}`;
}

export function formatRemainingDuration(seconds: number, locale: string): string {
  let value: number;
  let unit: Intl.NumberFormatOptions['unit'];

  if (seconds < 60) {
    value = Math.ceil(seconds);
    unit = 'second';
  } else if (seconds < 60 * 60) {
    value = Math.ceil(seconds / 60);
    unit = 'minute';
  } else {
    value = Math.ceil(seconds / (60 * 60));
    unit = 'hour';
  }

  const normalizedLocale = locale.toLowerCase();
  if (normalizedLocale.startsWith('zh-cn') || normalizedLocale === 'zh') {
    const units = {
      second: '秒钟',
      minute: '分钟',
      hour: '小时'
    } as const;
    return `${new Intl.NumberFormat(locale).format(value)} ${units[unit as keyof typeof units]}`;
  }
  if (normalizedLocale.startsWith('zh-tw')) {
    const units = {
      second: '秒鐘',
      minute: '分鐘',
      hour: '小時'
    } as const;
    return `${new Intl.NumberFormat(locale).format(value)} ${units[unit as keyof typeof units]}`;
  }

  return new Intl.NumberFormat(locale, {
    style: 'unit',
    unit,
    unitDisplay: 'long',
    maximumFractionDigits: 0
  }).format(value);
}

export function getDownloadTransferDetails(
  metrics: DownloadTransferMetrics,
  locale: string
): DownloadTransferDetails | undefined {
  const downloadedBytes = metrics.downloadedBytes;
  if (downloadedBytes === undefined || !Number.isFinite(downloadedBytes) || downloadedBytes < 0) {
    return undefined;
  }

  const totalBytes = metrics.totalBytes;
  const hasKnownTotal = totalBytes !== undefined && Number.isFinite(totalBytes) && totalBytes > 0;
  const transferred = hasKnownTotal
    ? `${formatDownloadBytes(downloadedBytes, locale)}/${formatDownloadBytes(totalBytes, locale)}`
    : formatDownloadBytes(downloadedBytes, locale);

  const bytesPerSecond = metrics.bytesPerSecond;
  const speed =
    bytesPerSecond !== undefined && Number.isFinite(bytesPerSecond) && bytesPerSecond > 0
      ? `${formatDownloadBytes(bytesPerSecond, locale)}/s`
      : undefined;

  const remainingSeconds = metrics.remainingSeconds;
  const remaining =
    hasKnownTotal &&
    remainingSeconds !== undefined &&
    Number.isFinite(remainingSeconds) &&
    remainingSeconds >= 0
      ? formatRemainingDuration(remainingSeconds, locale)
      : undefined;

  return { speed, transferred, remaining };
}

export function formatDownloadTransferSummary(
  details: DownloadTransferDetails,
  locale: string,
  remainingLabel?: string
): string {
  const transfer = details.speed
    ? `${details.speed} - ${details.transferred}`
    : details.transferred;
  const separator = locale.toLowerCase().startsWith('zh') ? '，' : ', ';

  return remainingLabel ? `${transfer}${separator}${remainingLabel}` : transfer;
}
