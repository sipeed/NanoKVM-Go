export const CALIBRATABLE_MOUSE_MODES = ['absolute', 'digitizer-stylus', 'touch-screen'] as const;

const LEGACY_DIGITIZER_STYLUS_MODE = 'absolute-android';

export type CalibratableMouseMode = (typeof CALIBRATABLE_MOUSE_MODES)[number];

export type NormalizedPoint = {
  x: number;
  y: number;
};

export type CalibrationCorner = 'top-left' | 'top-right' | 'bottom-right' | 'bottom-left';

export type CalibrationPoint = {
  source: NormalizedPoint;
  target: NormalizedPoint;
};

export type FourPointCalibration = {
  points: Record<CalibrationCorner, CalibrationPoint>;
  updatedAt: number;
};

export type CalibrationMatrix = readonly [
  number,
  number,
  number,
  number,
  number,
  number,
  number,
  number,
  number
];

export type CalibrationTargetRect = {
  left: number;
  top: number;
  width: number;
  height: number;
};

export type InputProfile = {
  id: string;
  name: string;
  origin: 'user' | 'builtin';
  platform?: 'ios' | 'android' | 'windows' | 'macos' | 'linux';
  calibrations: Partial<Record<CalibratableMouseMode, FourPointCalibration>>;
  createdAt: number;
  updatedAt: number;
};

export type InputProfileStore = {
  version: 1;
  selectedProfileId: string;
  profiles: InputProfile[];
};

type Homography = readonly [number, number, number, number, number, number, number, number];

const homographyCache = new WeakMap<FourPointCalibration, Homography | null>();
const generatedProfileName = /^(?:new profile|新配置)\s*(\d+)$/i;

export const DEFAULT_CALIBRATION_POINTS: Record<CalibrationCorner, NormalizedPoint> = {
  'top-left': { x: 0.1, y: 0.1 },
  'top-right': { x: 0.9, y: 0.1 },
  'bottom-right': { x: 0.9, y: 0.9 },
  'bottom-left': { x: 0.1, y: 0.9 }
};

export const DEFAULT_CALIBRATION_TARGET_RECT: CalibrationTargetRect = {
  left: 0.1,
  top: 0.1,
  width: 0.8,
  height: 0.8
};

export const DEFAULT_INPUT_PROFILE_ID = 'default-no-calibration';
export const AUTO_INPUT_PROFILE_ID = 'automatic-crop-calibration';

export const DEFAULT_INPUT_PROFILE: InputProfile = {
  id: DEFAULT_INPUT_PROFILE_ID,
  name: 'No calibration',
  origin: 'builtin',
  calibrations: {},
  createdAt: 0,
  updatedAt: 0
};

export const AUTO_INPUT_PROFILE: InputProfile = {
  id: AUTO_INPUT_PROFILE_ID,
  name: 'Automatic calibration',
  origin: 'builtin',
  calibrations: {},
  createdAt: 0,
  updatedAt: 0
};

export const EMPTY_INPUT_PROFILE_STORE: InputProfileStore = {
  version: 1,
  selectedProfileId: AUTO_INPUT_PROFILE_ID,
  profiles: [AUTO_INPUT_PROFILE, DEFAULT_INPUT_PROFILE]
};

export function createAutomaticCropCalibration(
  mode: CalibratableMouseMode,
  crop: {
    sourceWidth: number;
    sourceHeight: number;
    contentRect: { left: number; top: number; width: number; height: number } | null;
    contentAspectRatio?: number;
    updatedAt: number;
  }
): FourPointCalibration | null {
  const rect = crop.contentRect;
  if (
    (mode !== 'absolute' && mode !== 'digitizer-stylus') ||
    crop.sourceWidth <= 0 ||
    crop.sourceHeight <= 0 ||
    !rect ||
    rect.width <= 0 ||
    rect.height <= 0
  ) {
    return null;
  }

  const sourcePoints = calibrationTargetsFromRect({
    left: rect.left / crop.sourceWidth,
    top: rect.top / crop.sourceHeight,
    width: rect.width / crop.sourceWidth,
    height: rect.height / crop.sourceHeight
  });
  const contentAspectRatio =
    typeof crop.contentAspectRatio === 'number' &&
    Number.isFinite(crop.contentAspectRatio) &&
    crop.contentAspectRatio > 0
      ? crop.contentAspectRatio
      : rect.width / rect.height;
  const targetRect =
    mode === 'digitizer-stylus'
      ? contentAspectRatio < 1
        ? { left: 0, top: 0, width: contentAspectRatio, height: 1 }
        : { left: 0, top: 0, width: 1, height: 1 / contentAspectRatio }
      : { left: 0, top: 0, width: 1, height: 1 };
  const targetPoints = calibrationTargetsFromRect({
    ...targetRect
  });
  return createCalibrationFromPointPairs(sourcePoints, targetPoints, crop.updatedAt);
}

export function createFourPointCalibration(
  targetPoints: Partial<Record<CalibrationCorner, NormalizedPoint>> = {}
): FourPointCalibration {
  const points = {} as Record<CalibrationCorner, CalibrationPoint>;
  for (const corner of Object.keys(DEFAULT_CALIBRATION_POINTS) as CalibrationCorner[]) {
    points[corner] = {
      source: { ...DEFAULT_CALIBRATION_POINTS[corner] },
      target: normalizePoint(targetPoints[corner] ?? DEFAULT_CALIBRATION_POINTS[corner])
    };
  }
  return { points, updatedAt: Date.now() };
}

export function createCalibrationFromPointPairs(
  sourcePoints: Record<CalibrationCorner, NormalizedPoint>,
  targetPoints: Record<CalibrationCorner, NormalizedPoint>,
  now = Date.now()
): FourPointCalibration {
  const calibration = createFourPointCalibration(targetPoints);
  for (const corner of Object.keys(DEFAULT_CALIBRATION_POINTS) as CalibrationCorner[]) {
    calibration.points[corner].source = normalizePoint(sourcePoints[corner]);
  }
  calibration.updatedAt = now;
  return calibration;
}

export function calibrationTargetsFromRect(
  value: CalibrationTargetRect
): Record<CalibrationCorner, NormalizedPoint> {
  const rect = normalizeCalibrationTargetRect(value);
  const right = rect.left + rect.width;
  const bottom = rect.top + rect.height;
  return {
    'top-left': { x: rect.left, y: rect.top },
    'top-right': { x: right, y: rect.top },
    'bottom-right': { x: right, y: bottom },
    'bottom-left': { x: rect.left, y: bottom }
  };
}

export function normalizeCalibrationTargetRect(
  value: CalibrationTargetRect
): CalibrationTargetRect {
  const left = clamp(value.left);
  const top = clamp(value.top);
  return {
    left,
    top,
    width: Math.max(0, Math.min(1 - left, finiteNumber(value.width, 0))),
    height: Math.max(0, Math.min(1 - top, finiteNumber(value.height, 0)))
  };
}

export function applyFourPointCalibration(
  point: NormalizedPoint,
  calibration: FourPointCalibration | null | undefined
): NormalizedPoint {
  if (!calibration) return normalizePoint(point);
  const homography = getHomography(calibration);
  if (!homography) return normalizePoint(point);

  const denominator = homography[6] * point.x + homography[7] * point.y + 1;
  if (!Number.isFinite(denominator) || Math.abs(denominator) < Number.EPSILON) {
    return normalizePoint(point);
  }

  return normalizePoint({
    x: (homography[0] * point.x + homography[1] * point.y + homography[2]) / denominator,
    y: (homography[3] * point.x + homography[4] * point.y + homography[5]) / denominator
  });
}

export function isFourPointCalibrationUsable(calibration: FourPointCalibration): boolean {
  return getHomography(calibration) !== null;
}

export function getFourPointCalibrationMatrix(
  calibration: FourPointCalibration | null | undefined
): CalibrationMatrix {
  const homography = calibration ? getHomography(calibration) : null;
  if (!homography) return [1, 0, 0, 0, 1, 0, 0, 0, 1];
  const value = (index: number) => (Object.is(homography[index], -0) ? 0 : homography[index]);
  return [value(0), value(1), value(2), value(3), value(4), value(5), value(6), value(7), 1];
}

export function createInputProfile(name: string, now = Date.now()): InputProfile {
  return {
    id: createProfileId(now),
    name: name.trim(),
    origin: 'user',
    calibrations: {},
    createdAt: now,
    updatedAt: now
  };
}

export function getNextInputProfileName(
  profiles: readonly InputProfile[],
  formatName: (index: number) => string = (index) => `New profile ${index}`
) {
  const maximum = profiles.reduce((currentMaximum, profile) => {
    const match = generatedProfileName.exec(profile.name);
    return match ? Math.max(currentMaximum, Number(match[1])) : currentMaximum;
  }, 0);
  const index = maximum + 1;
  return formatName(index).trim() || `New profile ${index}`;
}

export function normalizeInputProfileStore(value: unknown): InputProfileStore {
  if (!isRecord(value)) return EMPTY_INPUT_PROFILE_STORE;
  const storedProfiles = Array.isArray(value.profiles)
    ? value.profiles.map(normalizeProfile).filter((profile): profile is InputProfile => !!profile)
    : [];
  const profiles = [
    AUTO_INPUT_PROFILE,
    DEFAULT_INPUT_PROFILE,
    ...storedProfiles.filter(
      (profile) => profile.id !== DEFAULT_INPUT_PROFILE_ID && profile.id !== AUTO_INPUT_PROFILE_ID
    )
  ];
  const selectedProfileId =
    typeof value.selectedProfileId === 'string' &&
    profiles.some((profile) => profile.id === value.selectedProfileId)
      ? value.selectedProfileId
      : AUTO_INPUT_PROFILE_ID;
  return { version: 1, selectedProfileId, profiles };
}

export function normalizePoint(point: NormalizedPoint): NormalizedPoint {
  return { x: clamp(point.x), y: clamp(point.y) };
}

function normalizeProfile(value: unknown): InputProfile | null {
  if (!isRecord(value) || typeof value.id !== 'string' || typeof value.name !== 'string') {
    return null;
  }
  const now = Date.now();
  const calibrations: InputProfile['calibrations'] = {};
  if (isRecord(value.calibrations)) {
    for (const mode of CALIBRATABLE_MOUSE_MODES) {
      const storedCalibration =
        mode === 'digitizer-stylus' && value.calibrations[mode] === undefined
          ? value.calibrations[LEGACY_DIGITIZER_STYLUS_MODE]
          : value.calibrations[mode];
      const calibration = normalizeCalibration(storedCalibration);
      if (calibration) calibrations[mode] = calibration;
    }
  }
  return {
    id: value.id,
    name: value.name.trim() || value.id,
    origin: value.origin === 'builtin' ? 'builtin' : 'user',
    platform: normalizePlatform(value.platform),
    calibrations,
    createdAt: finiteNumber(value.createdAt, now),
    updatedAt: finiteNumber(value.updatedAt, now)
  };
}

function normalizeCalibration(value: unknown): FourPointCalibration | null {
  if (!isRecord(value) || !isRecord(value.points)) return null;
  const points = {} as Record<CalibrationCorner, CalibrationPoint>;
  for (const corner of Object.keys(DEFAULT_CALIBRATION_POINTS) as CalibrationCorner[]) {
    const point = value.points[corner];
    if (!isRecord(point) || !isRecord(point.source) || !isRecord(point.target)) return null;
    points[corner] = {
      source: normalizePoint(readPoint(point.source)),
      target: normalizePoint(readPoint(point.target))
    };
  }
  return { points, updatedAt: finiteNumber(value.updatedAt, Date.now()) };
}

function readPoint(value: Record<string, unknown>): NormalizedPoint {
  return {
    x: finiteNumber(value.x, 0),
    y: finiteNumber(value.y, 0)
  };
}

function createProfileId(now: number) {
  return `profile-${now.toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
}

function isRecord(value: unknown): value is Record<string, any> {
  return !!value && typeof value === 'object' && !Array.isArray(value);
}

function normalizePlatform(value: unknown): InputProfile['platform'] {
  if (
    value === 'ios' ||
    value === 'android' ||
    value === 'windows' ||
    value === 'macos' ||
    value === 'linux'
  ) {
    return value;
  }
  return undefined;
}

function finiteNumber(value: unknown, fallback: number) {
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback;
}

function clamp(value: number) {
  return Math.max(0, Math.min(1, Number.isFinite(value) ? value : 0));
}

function getHomography(calibration: FourPointCalibration): Homography | null {
  const cached = homographyCache.get(calibration);
  if (cached !== undefined) return cached;

  const rows: number[][] = [];
  const values: number[] = [];
  for (const corner of Object.keys(DEFAULT_CALIBRATION_POINTS) as CalibrationCorner[]) {
    const { source, target } = calibration.points[corner];
    rows.push([source.x, source.y, 1, 0, 0, 0, -target.x * source.x, -target.x * source.y]);
    values.push(target.x);
    rows.push([0, 0, 0, source.x, source.y, 1, -target.y * source.x, -target.y * source.y]);
    values.push(target.y);
  }

  const solution = solveLinearSystem(rows, values);
  const homography = solution?.every(Number.isFinite) ? (solution as unknown as Homography) : null;
  homographyCache.set(calibration, homography);
  return homography;
}

function solveLinearSystem(coefficients: number[][], values: number[]): number[] | null {
  const size = values.length;
  const rows = coefficients.map((row, rowIndex) => [...row, values[rowIndex]]);

  for (let column = 0; column < size; column += 1) {
    let pivotRow = column;
    for (let rowIndex = column + 1; rowIndex < size; rowIndex += 1) {
      if (Math.abs(rows[rowIndex][column]) > Math.abs(rows[pivotRow][column])) {
        pivotRow = rowIndex;
      }
    }
    if (Math.abs(rows[pivotRow][column]) < 1e-10) return null;

    [rows[column], rows[pivotRow]] = [rows[pivotRow], rows[column]];
    const pivot = rows[column][column];
    for (let index = column; index <= size; index += 1) {
      rows[column][index] /= pivot;
    }

    for (let rowIndex = 0; rowIndex < size; rowIndex += 1) {
      if (rowIndex === column) continue;
      const factor = rows[rowIndex][column];
      for (let index = column; index <= size; index += 1) {
        rows[rowIndex][index] -= factor * rows[column][index];
      }
    }
  }

  return rows.map((row) => row[size]);
}
