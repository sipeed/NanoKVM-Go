export const screenTimelapseQPMin = 36;
export const screenTimelapseQPMax = 51;
export const screenTimelapseQPDefault = 42;
export const screenTimelapseQualityMin = 0;
export const screenTimelapseQualityMax = 100;

const screenTimelapseQPSpan = screenTimelapseQPMax - screenTimelapseQPMin;

export const screenTimelapseQualitySteps = Array.from(
  { length: screenTimelapseQPSpan + 1 },
  (_, index) => ({
    qp: screenTimelapseQPMax - index,
    quality: Math.round((index * screenTimelapseQualityMax) / screenTimelapseQPSpan)
  })
);
export const screenTimelapseQualityDefault = screenTimelapseQualitySteps.find(
  (step) => step.qp === screenTimelapseQPDefault
)!.quality;

export function parseScreenTimelapseQP(value: unknown): number | null {
  if (
    typeof value !== 'number' ||
    !Number.isInteger(value) ||
    value < screenTimelapseQPMin ||
    value > screenTimelapseQPMax
  ) {
    return null;
  }
  return value;
}

export function normalizeScreenTimelapseQP(value: unknown): number {
  return parseScreenTimelapseQP(value) ?? screenTimelapseQPDefault;
}

export function parseScreenTimelapseQuality(value: unknown): number | null {
  if (
    typeof value !== 'number' ||
    !Number.isInteger(value) ||
    value < screenTimelapseQualityMin ||
    value > screenTimelapseQualityMax
  ) {
    return null;
  }
  return value;
}

/** Maps internal fixed QP 51..36 onto user-facing recording quality 0..100. */
export function screenTimelapseQPToQuality(value: unknown): number {
  const qp = normalizeScreenTimelapseQP(value);
  return screenTimelapseQualitySteps.find((step) => step.qp === qp)!.quality;
}

/** Maps user-facing recording quality 0..100 back onto internal fixed QP 51..36. */
export function screenTimelapseQualityToQP(value: unknown): number | null {
  const quality = parseScreenTimelapseQuality(value);
  if (quality === null) return null;

  return screenTimelapseQualitySteps.find((step) => step.quality === quality)?.qp ?? null;
}
