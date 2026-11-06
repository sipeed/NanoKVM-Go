export type UploadTransferMetrics = {
  uploadedBytes: number;
  totalBytes: number;
  bytesPerSecond?: number;
  remainingSeconds?: number;
};

const minimumSpeedSampleMilliseconds = 250;

export function getUploadTransferMetrics(
  uploadedBytes: number,
  totalBytes: number,
  elapsedMilliseconds: number
): UploadTransferMetrics {
  const normalizedTotal = Number.isFinite(totalBytes) && totalBytes > 0 ? totalBytes : 0;
  const normalizedUploaded = Math.min(
    normalizedTotal || Number.POSITIVE_INFINITY,
    Math.max(0, Number.isFinite(uploadedBytes) ? uploadedBytes : 0)
  );
  const hasSpeedSample =
    elapsedMilliseconds >= minimumSpeedSampleMilliseconds && normalizedUploaded > 0;
  const bytesPerSecond = hasSpeedSample
    ? normalizedUploaded / (elapsedMilliseconds / 1000)
    : undefined;
  const remainingSeconds =
    bytesPerSecond && normalizedTotal > 0
      ? Math.max(0, normalizedTotal - normalizedUploaded) / bytesPerSecond
      : undefined;

  return {
    uploadedBytes: normalizedUploaded,
    totalBytes: normalizedTotal,
    bytesPerSecond,
    remainingSeconds
  };
}
