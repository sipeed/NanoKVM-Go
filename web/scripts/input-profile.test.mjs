import assert from 'node:assert/strict';
import test from 'node:test';

import {
  applyFourPointCalibration,
  AUTO_INPUT_PROFILE_ID,
  calibrationTargetsFromRect,
  createAutomaticCropCalibration,
  createCalibrationFromPointPairs,
  createFourPointCalibration,
  createInputProfile,
  DEFAULT_INPUT_PROFILE_ID,
  getFourPointCalibrationMatrix,
  getNextInputProfileName,
  normalizeInputProfileStore
} from '../src/lib/input-profile.ts';

function closeTo(actual, expected, tolerance = 1e-9) {
  assert.ok(
    Math.abs(actual - expected) <= tolerance,
    `expected ${actual} to be within ${tolerance} of ${expected}`
  );
}

test('default safe-area calibration preserves the complete normalized frame', () => {
  const calibration = createFourPointCalibration();

  for (const point of [
    { x: 0, y: 0 },
    { x: 0.5, y: 0.5 },
    { x: 1, y: 1 }
  ]) {
    const mapped = applyFourPointCalibration(point, calibration);
    closeTo(mapped.x, point.x);
    closeTo(mapped.y, point.y);
  }
});

test('four point calibration maps every measured source point to its HID target', () => {
  const calibration = createFourPointCalibration();
  calibration.points['top-left'].source = { x: 0.2, y: 0.15 };
  calibration.points['top-right'].source = { x: 0.8, y: 0.1 };
  calibration.points['bottom-right'].source = { x: 0.85, y: 0.82 };
  calibration.points['bottom-left'].source = { x: 0.1, y: 0.9 };

  for (const point of Object.values(calibration.points)) {
    const mapped = applyFourPointCalibration(point.source, calibration);
    closeTo(mapped.x, point.target.x);
    closeTo(mapped.y, point.target.y);
  }
});

test('automatic absolute calibration expands the cropped source rectangle to full HID axes', () => {
  const calibration = createAutomaticCropCalibration('absolute', {
    sourceWidth: 1920,
    sourceHeight: 1080,
    contentRect: { left: 690, top: 60, width: 540, height: 960 },
    updatedAt: 123
  });

  assert.ok(calibration);
  assert.equal(calibration.updatedAt, 123);
  assert.deepEqual(applyFourPointCalibration({ x: 690 / 1920, y: 60 / 1080 }, calibration), {
    x: 0,
    y: 0
  });
  const bottomRight = applyFourPointCalibration({ x: 1230 / 1920, y: 1020 / 1080 }, calibration);
  closeTo(bottomRight.x, 1);
  closeTo(bottomRight.y, 1);
});

test('automatic Android calibration maps portrait content into the short X axis', () => {
  const calibration = createAutomaticCropCalibration('digitizer-stylus', {
    sourceWidth: 1920,
    sourceHeight: 1080,
    contentRect: { left: 690, top: 60, width: 540, height: 960 },
    updatedAt: 456
  });

  assert.ok(calibration);
  const bottomRight = applyFourPointCalibration({ x: 1230 / 1920, y: 1020 / 1080 }, calibration);
  closeTo(bottomRight.x, 540 / 960);
  closeTo(bottomRight.y, 1);
  assert.equal(
    createAutomaticCropCalibration('touch-screen', {
      sourceWidth: 1920,
      sourceHeight: 1080,
      contentRect: { left: 690, top: 60, width: 540, height: 960 },
      updatedAt: 456
    }),
    null
  );
});

test('automatic Android calibration maps landscape content into the short Y axis', () => {
  const calibration = createAutomaticCropCalibration('digitizer-stylus', {
    sourceWidth: 1920,
    sourceHeight: 1080,
    contentRect: { left: 160, top: 90, width: 1600, height: 900 },
    updatedAt: 789
  });

  assert.ok(calibration);
  const bottomRight = applyFourPointCalibration(
    { x: (160 + 1600) / 1920, y: (90 + 900) / 1080 },
    calibration
  );
  closeTo(bottomRight.x, 1);
  closeTo(bottomRight.y, 900 / 1600);
});

test('guided calibration maps visible control points to sampled raw HID values', () => {
  const controlPoints = calibrationTargetsFromRect({
    left: 0.1,
    top: 0.2,
    width: 0.8,
    height: 0.6
  });
  const hidSamples = {
    'top-left': { x: 0.2, y: 0.25 },
    'top-right': { x: 0.7, y: 0.25 },
    'bottom-right': { x: 0.7, y: 0.75 },
    'bottom-left': { x: 0.2, y: 0.75 }
  };
  const calibration = createCalibrationFromPointPairs(controlPoints, hidSamples, 123);

  for (const [corner, controlPoint] of Object.entries(controlPoints)) {
    const mapped = applyFourPointCalibration(controlPoint, calibration);
    closeTo(mapped.x, hidSamples[corner].x);
    closeTo(mapped.y, hidSamples[corner].y);
  }
  assert.equal(calibration.updatedAt, 123);
});

test('invalid four point geometry falls back to the uncalibrated point', () => {
  const calibration = createFourPointCalibration();
  for (const point of Object.values(calibration.points)) {
    point.source = { x: 0.5, y: 0.5 };
  }

  assert.deepEqual(applyFourPointCalibration({ x: 0.25, y: 0.75 }, calibration), {
    x: 0.25,
    y: 0.75
  });
});

test('profile matrix exposes the homography used by pointer mapping', () => {
  assert.deepEqual(getFourPointCalibrationMatrix(null), [1, 0, 0, 0, 1, 0, 0, 0, 1]);

  const calibration = createFourPointCalibration();
  const matrix = getFourPointCalibrationMatrix(calibration);
  assert.deepEqual(matrix, [1, 0, 0, 0, 1, 0, 0, 0, 1]);
});

test('profile storage keeps only supported modes and valid platform metadata', () => {
  const profile = createInputProfile('  Android test  ', 100);
  const calibration = createFourPointCalibration();
  const store = normalizeInputProfileStore({
    version: 99,
    selectedProfileId: profile.id,
    profiles: [
      {
        ...profile,
        origin: 'builtin',
        platform: 'android',
        calibrations: {
          absolute: calibration,
          relative: calibration
        }
      }
    ]
  });

  assert.equal(store.version, 1);
  assert.equal(store.selectedProfileId, profile.id);
  assert.equal(store.profiles[0].id, AUTO_INPUT_PROFILE_ID);
  assert.equal(store.profiles[0].origin, 'builtin');
  assert.equal(store.profiles[1].id, DEFAULT_INPUT_PROFILE_ID);
  assert.equal(store.profiles[2].name, 'Android test');
  assert.equal(store.profiles[2].origin, 'builtin');
  assert.equal(store.profiles[2].platform, 'android');
  assert.deepEqual(Object.keys(store.profiles[2].calibrations), ['absolute']);
});

test('profile storage migrates the legacy Android mode key to digitizer-stylus', () => {
  const profile = createInputProfile('Legacy mode', 100);
  const calibration = createFourPointCalibration();
  const store = normalizeInputProfileStore({
    version: 1,
    selectedProfileId: profile.id,
    profiles: [
      {
        ...profile,
        calibrations: {
          'absolute-android': calibration
        }
      }
    ]
  });

  const migrated = store.profiles.find((item) => item.id === profile.id);
  assert.ok(migrated);
  assert.ok(migrated.calibrations['digitizer-stylus']);
  assert.equal(migrated.calibrations['absolute-android'], undefined);
});

test('empty profile storage includes both built-in calibration profiles', () => {
  const store = normalizeInputProfileStore(null);

  assert.equal(store.selectedProfileId, AUTO_INPUT_PROFILE_ID);
  assert.deepEqual(
    store.profiles.map((profile) => profile.id),
    [AUTO_INPUT_PROFILE_ID, DEFAULT_INPUT_PROFILE_ID]
  );
});

test('generated profile names continue after the greatest remaining number', () => {
  const profiles = [
    createInputProfile('新配置1', 1),
    createInputProfile('New profile 3', 3),
    createInputProfile('Custom name', 4)
  ];

  assert.equal(getNextInputProfileName(profiles), 'New profile 4');
  assert.equal(
    getNextInputProfileName(profiles, (index) => `新配置 ${index}`),
    '新配置 4'
  );
});
