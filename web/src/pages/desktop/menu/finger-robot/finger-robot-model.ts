import type {
  FingerRobotArm,
  FingerRobotConfig,
  FingerRobotData,
  FingerRobotProfile,
  FingerRobotRange,
  FingerRobotSpeed,
  FingerRobotStatus
} from '@/api/vm.ts';

export const fingerRobotConfigVersion = 1;
export const fingerRobotArms = ['long', 'short'] as const;
export const fingerRobotSpeeds = ['slow', 'fast'] as const;
export const fingerRobotStrengthRange = { min: 1, max: 5, default: 3 } as const;

const fingerRobotStates = new Set(['idle', 'releasing', 'pressing', 'holding', 'resetting']);
const fingerRobotResults = new Set([
  'idle',
  'running',
  'success',
  'cancelled',
  'failed',
  'reset',
  'shutdown'
]);

export type FingerRobotDisplayState =
  | 'idle'
  | 'moving'
  | 'holding'
  | 'releasing'
  | 'completed'
  | 'armReplacement'
  | 'busy'
  | 'failed';

type FingerRobotDisplayStateOptions = {
  busy?: boolean;
  failed?: boolean;
  pressPending?: boolean;
  releasePending?: boolean;
  replaceArmPending?: boolean;
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function readSafeInteger(value: unknown): number | null {
  return typeof value === 'number' && Number.isSafeInteger(value) ? value : null;
}

function readNonEmptyString(value: unknown): string | null {
  return typeof value === 'string' && value.trim() ? value : null;
}

function isFingerRobotArm(value: unknown): value is FingerRobotArm {
  return typeof value === 'string' && fingerRobotArms.includes(value as FingerRobotArm);
}

function isFingerRobotSpeed(value: unknown): value is FingerRobotSpeed {
  return typeof value === 'string' && fingerRobotSpeeds.includes(value as FingerRobotSpeed);
}

function readRange(value: unknown): FingerRobotRange | null {
  if (!isRecord(value)) return null;

  const min = readSafeInteger(value.min);
  const max = readSafeInteger(value.max);
  const defaultValue = readSafeInteger(value.default);
  if (
    min === null ||
    max === null ||
    defaultValue === null ||
    min > defaultValue ||
    defaultValue > max
  ) {
    return null;
  }

  return { min, max, default: defaultValue };
}

function readProfiles(value: unknown, strength: FingerRobotRange): FingerRobotProfile[] | null {
  if (!Array.isArray(value) || value.length !== fingerRobotArms.length) return null;

  const profiles: FingerRobotProfile[] = [];
  const seen = new Set<FingerRobotArm>();
  for (const item of value) {
    if (!isRecord(item) || !isFingerRobotArm(item.id) || seen.has(item.id)) return null;

    const displayName = readNonEmptyString(item.displayName);
    const defaultStrength = readSafeInteger(item.defaultStrength);
    if (
      displayName === null ||
      defaultStrength === null ||
      defaultStrength < strength.min ||
      defaultStrength > strength.max
    ) {
      return null;
    }

    seen.add(item.id);
    profiles.push({ id: item.id, displayName, defaultStrength });
  }

  return fingerRobotArms.every((arm) => seen.has(arm)) ? profiles : null;
}

function readSpeedOptions(value: unknown): FingerRobotSpeed[] | null {
  if (!Array.isArray(value) || value.length !== fingerRobotSpeeds.length) return null;

  const options: FingerRobotSpeed[] = [];
  const seen = new Set<FingerRobotSpeed>();
  for (const option of value) {
    if (!isFingerRobotSpeed(option) || seen.has(option)) return null;
    seen.add(option);
    options.push(option);
  }

  return fingerRobotSpeeds.every((speed) => seen.has(speed)) ? options : null;
}

function readStatus(value: unknown): FingerRobotStatus | null {
  if (!isRecord(value)) return null;

  const { running, state, result, error } = value;
  if (
    typeof running !== 'boolean' ||
    typeof state !== 'string' ||
    !fingerRobotStates.has(state) ||
    typeof result !== 'string' ||
    !fingerRobotResults.has(result) ||
    typeof error !== 'string'
  ) {
    return null;
  }

  return { running, state, result, error };
}

export function parseFingerRobotConfig(
  value: unknown,
  strength: FingerRobotRange = { min: 1, max: 5, default: 3 },
  pressMS: FingerRobotRange = { min: 100, max: 30_000, default: 800 },
  speedOptions: readonly FingerRobotSpeed[] = fingerRobotSpeeds
): FingerRobotConfig | null {
  if (!isRecord(value)) return null;

  const version = readSafeInteger(value.version);
  const revision = readSafeInteger(value.revision);
  const pressMs = readSafeInteger(value.pressMs);
  const strengthByArm = value.strengthByArm;
  if (
    version !== fingerRobotConfigVersion ||
    revision === null ||
    revision < 1 ||
    !isFingerRobotArm(value.activeArm) ||
    !isRecord(strengthByArm) ||
    pressMs === null ||
    pressMs < pressMS.min ||
    pressMs > pressMS.max ||
    !isFingerRobotSpeed(value.speed) ||
    !speedOptions.includes(value.speed)
  ) {
    return null;
  }

  const long = readSafeInteger(strengthByArm.long);
  const short = readSafeInteger(strengthByArm.short);
  if (
    long === null ||
    short === null ||
    long < strength.min ||
    long > strength.max ||
    short < strength.min ||
    short > strength.max
  ) {
    return null;
  }

  return {
    version,
    revision,
    activeArm: value.activeArm,
    strengthByArm: { long, short },
    pressMs,
    speed: value.speed
  };
}

export function parseFingerRobotData(value: unknown): FingerRobotData | null {
  if (
    !isRecord(value) ||
    typeof value.enabled !== 'boolean' ||
    typeof value.available !== 'boolean'
  ) {
    return null;
  }

  const strength = readRange(value.strength);
  const pressMS = readRange(value.pressMs);
  const speedOptions = readSpeedOptions(value.speedOptions);
  const status = readStatus(value.status);
  if (
    !strength ||
    !pressMS ||
    !speedOptions ||
    !status ||
    !isFingerRobotSpeed(value.defaultSpeed)
  ) {
    return null;
  }
  if (
    strength.min !== fingerRobotStrengthRange.min ||
    strength.max !== fingerRobotStrengthRange.max ||
    strength.default !== fingerRobotStrengthRange.default
  ) {
    return null;
  }
  if (!speedOptions.includes(value.defaultSpeed)) return null;

  const profiles = readProfiles(value.profiles, strength);
  if (!profiles) return null;

  let config: FingerRobotConfig | undefined;
  if (value.config !== undefined && value.config !== null) {
    const parsed = parseFingerRobotConfig(value.config, strength, pressMS, speedOptions);
    if (!parsed) return null;
    config = parsed;
  }

  return {
    enabled: value.enabled,
    available: value.available,
    ...(config ? { config } : {}),
    profiles,
    strength,
    pressMs: pressMS,
    speedOptions,
    defaultSpeed: value.defaultSpeed,
    status
  };
}

export function cloneFingerRobotConfig(config: FingerRobotConfig): FingerRobotConfig {
  return { ...config, strengthByArm: { ...config.strengthByArm } };
}

export function selectFingerRobotArm(
  config: FingerRobotConfig,
  activeArm: FingerRobotArm
): FingerRobotConfig {
  return { ...cloneFingerRobotConfig(config), activeArm };
}

export function selectFingerRobotStrength(
  config: FingerRobotConfig,
  strength: number,
  limits: FingerRobotRange
): FingerRobotConfig | null {
  if (!Number.isSafeInteger(strength) || strength < limits.min || strength > limits.max)
    return null;

  return {
    ...cloneFingerRobotConfig(config),
    strengthByArm: { ...config.strengthByArm, [config.activeArm]: strength }
  };
}

export function selectFingerRobotPressMS(
  config: FingerRobotConfig,
  pressMS: number,
  limits: FingerRobotRange
): FingerRobotConfig | null {
  if (!Number.isSafeInteger(pressMS) || pressMS < limits.min || pressMS > limits.max) return null;
  return { ...cloneFingerRobotConfig(config), pressMs: pressMS };
}

export function selectFingerRobotSpeed(
  config: FingerRobotConfig,
  speed: FingerRobotSpeed,
  options: readonly FingerRobotSpeed[]
): FingerRobotConfig | null {
  if (!options.includes(speed)) return null;
  return { ...cloneFingerRobotConfig(config), speed };
}

export function getActiveFingerRobotStrength(config: FingerRobotConfig): number {
  return config.strengthByArm[config.activeArm];
}

export function resolveFingerRobotDisplayState(
  status: FingerRobotStatus,
  options: FingerRobotDisplayStateOptions = {}
): FingerRobotDisplayState {
  if (options.failed || status.error.trim() || status.result === 'failed') return 'failed';
  if (options.releasePending) return 'releasing';
  if (options.replaceArmPending) return 'moving';

  if (status.running) {
    if (status.state === 'holding') return 'holding';
    if (status.state === 'releasing') return 'releasing';
    return 'moving';
  }
  if (options.pressPending) return 'moving';
  if (options.busy) return 'busy';
  if (status.result === 'reset') return 'armReplacement';
  if (status.result === 'success' || status.result === 'cancelled') {
    return 'completed';
  }
  if (status.state === 'releasing') return 'releasing';
  return 'idle';
}

export function shouldPollFingerRobotStatus(
  panelOpen: boolean,
  pressPending: boolean,
  releasePending: boolean,
  statusRunning: boolean
): boolean {
  return panelOpen || pressPending || releasePending || statusRunning;
}

export function isFingerRobotChallengeResponse(code: number): boolean {
  return code === -3 || code === -4;
}
