import type { ScreenTimelapseRetention } from '@/api/ai.ts';

const bytesPerMiB = 1024 * 1024;

export const screenTimelapseRetentionStorageWarningRatio = 0.9;

export type ScreenTimelapseStorageWarning = {
  usedBytes: number;
  limitBytes: number;
};

export function getScreenTimelapseStorageWarning(
  retention: Pick<ScreenTimelapseRetention, 'config' | 'status'> | null | undefined
): ScreenTimelapseStorageWarning | null {
  if (!retention) return null;

  const cleanup = retention.config?.cleanup;
  const usageBytes = retention.status?.usageBytes;
  const maxStorageMiB = cleanup?.maxStorageMiB;

  if (
    cleanup?.enabled !== true ||
    typeof maxStorageMiB !== 'number' ||
    !Number.isSafeInteger(maxStorageMiB) ||
    maxStorageMiB <= 0 ||
    typeof usageBytes !== 'number' ||
    !Number.isSafeInteger(usageBytes) ||
    usageBytes < 0 ||
    maxStorageMiB > Math.floor(Number.MAX_SAFE_INTEGER / bytesPerMiB)
  ) {
    return null;
  }

  const limitBytes = maxStorageMiB * bytesPerMiB;
  if (
    usageBytes < limitBytes * screenTimelapseRetentionStorageWarningRatio &&
    retention.status.overLimit !== true
  ) {
    return null;
  }

  return { usedBytes: usageBytes, limitBytes };
}
