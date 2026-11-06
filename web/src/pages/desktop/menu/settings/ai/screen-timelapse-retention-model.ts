import type {
  ScreenTimelapseRetention,
  ScreenTimelapseRetentionCleanupConfig,
  ScreenTimelapseRetentionConfig,
  ScreenTimelapseRetentionPreview,
  SetScreenTimelapseRetentionRequest
} from '@/api/ai.ts';

export const screenTimelapseRetentionMiBPerGiB = 1024;

export type ScreenTimelapseRetentionFormValues = {
  retentionDays?: number;
  maxStorageGiB?: number;
};

function nonNegativeInteger(value: number | null | undefined, fallback: number): number {
  if (typeof value !== 'number' || !Number.isFinite(value)) return fallback;
  return Math.min(Number.MAX_SAFE_INTEGER, Math.max(0, Math.round(value)));
}

/**
 * Converts the integer value persisted by the API without display rounding.
 * Keeping the full fractional GiB value makes a config -> form -> config
 * round-trip recover the original MiB value instead of gradually drifting.
 */
export function screenTimelapseRetentionMiBToGiB(maxStorageMiB: number): number {
  return nonNegativeInteger(maxStorageMiB, 0) / screenTimelapseRetentionMiBPerGiB;
}

/**
 * The API accepts whole MiB. Rounding once at the conversion boundary keeps
 * decimal GiB input stable and preserves values originally returned by it.
 */
export function screenTimelapseRetentionGiBToMiB(maxStorageGiB: number): number {
  if (!Number.isFinite(maxStorageGiB) || maxStorageGiB < 0) return 0;
  return Math.min(
    Number.MAX_SAFE_INTEGER,
    Math.max(0, Math.round(maxStorageGiB * screenTimelapseRetentionMiBPerGiB))
  );
}

export function screenTimelapseRetentionCleanupToFormValues(
  cleanup: ScreenTimelapseRetentionCleanupConfig
): ScreenTimelapseRetentionFormValues {
  const retentionDays = nonNegativeInteger(cleanup.retentionDays, 0);
  const maxStorageMiB = nonNegativeInteger(cleanup.maxStorageMiB, 0);

  return {
    retentionDays: retentionDays || undefined,
    maxStorageGiB: maxStorageMiB > 0 ? screenTimelapseRetentionMiBToGiB(maxStorageMiB) : undefined
  };
}

export function screenTimelapseRetentionConfigToFormValues(
  config: ScreenTimelapseRetentionConfig
): ScreenTimelapseRetentionFormValues {
  return screenTimelapseRetentionCleanupToFormValues(config.cleanup);
}

/** Maps the server's normalized GET response data to editable form values. */
export function screenTimelapseRetentionToFormValues(
  retention: Pick<ScreenTimelapseRetention, 'config'>
): ScreenTimelapseRetentionFormValues {
  return screenTimelapseRetentionConfigToFormValues(retention.config);
}

export function screenTimelapseRetentionFormValuesToCleanup(
  values: ScreenTimelapseRetentionFormValues
): ScreenTimelapseRetentionCleanupConfig {
  const retentionDays = nonNegativeInteger(values.retentionDays, 0);
  const maxStorageMiB = screenTimelapseRetentionGiBToMiB(values.maxStorageGiB ?? 0);

  return {
    enabled: retentionDays > 0 || maxStorageMiB > 0,
    retentionDays,
    maxStorageMiB
  };
}

export function screenTimelapseRetentionFormValuesToUpdateRequest(
  revision: number,
  values: ScreenTimelapseRetentionFormValues,
  challenge?: string
): SetScreenTimelapseRetentionRequest {
  const request: SetScreenTimelapseRetentionRequest = {
    revision,
    cleanup: screenTimelapseRetentionFormValuesToCleanup(values)
  };
  if (challenge !== undefined) request.challenge = challenge;
  return request;
}

function hasEffectiveLimit(config: ScreenTimelapseRetentionCleanupConfig): boolean {
  return config.retentionDays > 0 || config.maxStorageMiB > 0;
}

/** A disabled policy may have no limits; an enabled policy must have at least one. */
export function isScreenTimelapseRetentionCleanupValid(
  config: ScreenTimelapseRetentionCleanupConfig
): boolean {
  return !config.enabled || hasEffectiveLimit(config);
}

function isLimitStricter(current: number, candidate: number): boolean {
  return candidate > 0 && (current === 0 || candidate < current);
}

/**
 * Returns true when applying candidate can delete recordings that the current
 * policy would retain. Relaxing one limit does not hide a stricter other one.
 */
export function isScreenTimelapseRetentionStricter(
  current: ScreenTimelapseRetentionCleanupConfig,
  candidate: ScreenTimelapseRetentionCleanupConfig
): boolean {
  if (!candidate.enabled || !hasEffectiveLimit(candidate)) return false;
  if (!current.enabled || !hasEffectiveLimit(current)) return true;

  return (
    isLimitStricter(current.retentionDays, candidate.retentionDays) ||
    isLimitStricter(current.maxStorageMiB, candidate.maxStorageMiB)
  );
}

export type ScreenTimelapseRetentionPreviewDisplayModel = {
  candidates: ScreenTimelapseRetentionPreview['candidates'];
  skipped: ScreenTimelapseRetentionPreview['skipped'];
  warnings: ScreenTimelapseRetentionPreview['warnings'];
  wouldDeleteDays: number;
  wouldDeleteBytes: number;
  isEmpty: boolean;
  hasSkipped: boolean;
  hasWarnings: boolean;
  remainsOverLimit: boolean;
};

/**
 * Normalizes a dry-run response into the values rendered by the settings UI.
 * Keeping this pure makes preview changes easy to test without a browser.
 */
export function screenTimelapseRetentionPreviewToDisplayModel(
  preview: ScreenTimelapseRetentionPreview
): ScreenTimelapseRetentionPreviewDisplayModel {
  const candidates = Array.isArray(preview.candidates) ? preview.candidates : [];
  const skipped = Array.isArray(preview.skipped) ? preview.skipped : [];
  const warnings = Array.isArray(preview.warnings) ? preview.warnings : [];
  const wouldDeleteDays = Number.isFinite(preview.wouldDeleteDays)
    ? Math.max(0, Math.floor(preview.wouldDeleteDays))
    : candidates.length;
  const wouldDeleteBytes = Number.isFinite(preview.wouldDeleteBytes)
    ? Math.max(0, preview.wouldDeleteBytes)
    : 0;

  return {
    candidates,
    skipped,
    warnings,
    wouldDeleteDays,
    wouldDeleteBytes,
    isEmpty: candidates.length === 0,
    hasSkipped: skipped.length > 0,
    hasWarnings: warnings.length > 0,
    remainsOverLimit: preview.overLimitAfterCleanup
  };
}
