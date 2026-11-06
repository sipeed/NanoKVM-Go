import type {
  FingerRobotDebugCount,
  FingerRobotDebugData,
  FingerRobotDebugRange,
  FingerRobotDebugTest
} from '@/api/vm.ts';

export const fingerRobotDebugZeroUs = 1500;
export const fingerRobotDebugUnlimitedCount = 0;

export const fingerRobotDebugEditableFields = Object.freeze([
  'releaseUs',
  'pressUs',
  'count',
  'pressMs',
  'pressSpeedUsPerSecond'
] as const);

export type FingerRobotDebugEditableField = (typeof fingerRobotDebugEditableFields)[number];

export type FingerRobotDebugDraft = Record<FingerRobotDebugEditableField, number | null> & {
  unlimited: boolean;
};

export const fingerRobotDebugDefaults: FingerRobotDebugDraft = Object.freeze({
  releaseUs: 2050,
  pressUs: 899,
  count: 50,
  pressMs: 100,
  pressSpeedUsPerSecond: 2000,
  unlimited: false
});

export type FingerRobotDebugDraftError = FingerRobotDebugEditableField | 'direction' | null;

export type FingerRobotDebugPrimaryAction = 'start' | 'stop';

export type FingerRobotDebugApiResponse = {
  code: number;
  msg?: unknown;
};

export type FingerRobotDebugResetState = {
  running: boolean;
  isGateBusy: boolean;
  isStarting: boolean;
  isStopping: boolean;
  isResetting: boolean;
};

export function getFingerRobotDebugPrimaryAction(running: boolean): FingerRobotDebugPrimaryAction {
  return running ? 'stop' : 'start';
}

export function shouldPollFingerRobotDebugStatus(running: boolean, available: boolean): boolean {
  return running || !available;
}

export function getFingerRobotDebugChallengeResult<Response extends FingerRobotDebugApiResponse>(
  response: Response
): Response | undefined {
  const requiresVerification =
    (response.code === -3 || response.code === -4) &&
    typeof response.msg === 'string' &&
    response.msg.toLowerCase().includes('sensitive action verification');
  return requiresVerification ? response : undefined;
}

export function isFingerRobotDebugResetDisabled(state: FingerRobotDebugResetState): boolean {
  // `running` deliberately does not disable an explicit factory-zero request.
  return state.isGateBusy || state.isStarting || state.isStopping || state.isResetting;
}

export function runFingerRobotDebugStartIfActive<Result>(
  mounted: boolean,
  starting: boolean,
  execute: () => Result
): Result | undefined {
  if (!mounted || starting) return undefined;
  return execute();
}

export function formatFingerRobotDebugProgress(
  current: number,
  total: number,
  running: boolean,
  infinity: string
): string {
  const isUnlimited = total === fingerRobotDebugUnlimitedCount && (running || current > 0);
  return `${current} / ${isUnlimited ? infinity : total}`;
}

export function stopFingerRobotDebugOnUnmount(stop: () => Promise<unknown>): void {
  void stop().catch(() => undefined);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function readInteger(value: unknown): number | null {
  return typeof value === 'number' && Number.isSafeInteger(value) ? value : null;
}

function readNonNegativeInteger(value: unknown): number | null {
  const parsed = readInteger(value);
  return parsed !== null && parsed >= 0 ? parsed : null;
}

function readRange(value: unknown): FingerRobotDebugRange | null {
  if (!isRecord(value)) return null;

  const min = readInteger(value.min);
  const max = readInteger(value.max);
  const defaultValue = readInteger(value.default);
  if (
    min === null ||
    max === null ||
    defaultValue === null ||
    min > max ||
    defaultValue < min ||
    defaultValue > max
  ) {
    return null;
  }

  return { min, max, default: defaultValue };
}

function readCount(value: unknown): FingerRobotDebugCount | null {
  if (!isRecord(value)) return null;

  const min = readInteger(value.min);
  const defaultValue = readInteger(value.default);
  const unlimitedValue = readInteger(value.unlimitedValue);
  if (
    min === null ||
    min < 1 ||
    defaultValue === null ||
    unlimitedValue !== fingerRobotDebugUnlimitedCount ||
    (defaultValue !== unlimitedValue && defaultValue < min)
  ) {
    return null;
  }

  return { min, default: defaultValue, unlimitedValue };
}

/**
 * Fail closed when the mode/capability response is incomplete. In particular,
 * normal mode and authentication-disabled devices never expose the debug UI.
 * An enabled but temporarily unavailable controller still exposes its status,
 * safe ranges, and reset recovery action.
 */
export function parseFingerRobotDebugData(value: unknown): FingerRobotDebugData | null {
  if (!isRecord(value) || value.enabled !== true || typeof value.available !== 'boolean')
    return null;

  const zeroUs = readInteger(value.zeroUs);
  const releaseUs = readRange(value.releaseUs);
  const pressUs = readRange(value.pressUs);
  const count = readCount(value.count);
  const pressMs = readRange(value.pressMs);
  const pressSpeedUsPerSecond = readRange(value.pressSpeedUsPerSecond);
  const releaseIntervalMs = readNonNegativeInteger(value.releaseIntervalMs);
  const leaseTimeoutMs = readNonNegativeInteger(value.leaseTimeoutMs);
  if (
    zeroUs !== fingerRobotDebugZeroUs ||
    !releaseUs ||
    !pressUs ||
    !count ||
    !pressMs ||
    !pressSpeedUsPerSecond ||
    releaseIntervalMs === null ||
    leaseTimeoutMs === null ||
    leaseTimeoutMs === 0
  ) {
    return null;
  }

  const current = readNonNegativeInteger(value.current);
  const total = readNonNegativeInteger(value.total);
  if (
    current === null ||
    total === null ||
    typeof value.running !== 'boolean' ||
    typeof value.state !== 'string' ||
    typeof value.result !== 'string' ||
    (value.error !== undefined && typeof value.error !== 'string')
  ) {
    return null;
  }

  return {
    enabled: true,
    available: value.available,
    zeroUs,
    releaseUs,
    pressUs,
    count,
    pressMs,
    pressSpeedUsPerSecond,
    releaseIntervalMs,
    leaseTimeoutMs,
    running: value.running,
    current,
    total,
    state: value.state,
    result: value.result,
    error: value.error ?? ''
  };
}

export function createFingerRobotDebugDraft(data: FingerRobotDebugData): FingerRobotDebugDraft {
  const unlimited = data.count.default === data.count.unlimitedValue;
  return {
    releaseUs: data.releaseUs.default,
    pressUs: data.pressUs.default,
    count: unlimited ? data.count.min : data.count.default,
    pressMs: data.pressMs.default,
    pressSpeedUsPerSecond: data.pressSpeedUsPerSecond.default,
    unlimited
  };
}

function isValueInRange(value: number | null, range: FingerRobotDebugRange): value is number {
  return value !== null && Number.isSafeInteger(value) && value >= range.min && value <= range.max;
}

export function validateFingerRobotDebugDraft(
  draft: FingerRobotDebugDraft,
  data: FingerRobotDebugData
): FingerRobotDebugDraftError {
  for (const field of fingerRobotDebugEditableFields) {
    if (field === 'count') continue;
    if (!isValueInRange(draft[field], data[field])) return field;
  }

  if (
    !draft.unlimited &&
    (draft.count === null || !Number.isSafeInteger(draft.count) || draft.count < data.count.min)
  ) {
    return 'count';
  }

  // This robot's fixed direction presses by reducing the pulse width.
  if ((draft.pressUs as number) >= (draft.releaseUs as number)) return 'direction';

  return null;
}

export function snapshotFingerRobotDebugDraft(
  draft: FingerRobotDebugDraft,
  data: FingerRobotDebugData
): FingerRobotDebugTest | null {
  if (validateFingerRobotDebugDraft(draft, data) !== null) return null;

  return {
    releaseUs: draft.releaseUs as number,
    pressUs: draft.pressUs as number,
    count: draft.unlimited ? data.count.unlimitedValue : (draft.count as number),
    pressMs: draft.pressMs as number,
    pressSpeedUsPerSecond: draft.pressSpeedUsPerSecond as number
  };
}
