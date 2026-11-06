import { atom } from 'jotai';

// mouse mode: absolute (GD mouse), digitizer-stylus (stylus), touchscreen, or relative
export type MouseMode = 'absolute' | 'digitizer-stylus' | 'touch-screen' | 'relative';
export type PointerMode = Exclude<MouseMode, 'relative'>;

export type PointerModeDeviceStatus = {
  mode: PointerMode | 'unknown' | null;
  switching: boolean;
  degraded: boolean;
  ready: boolean;
  error: string;
};

const LEGACY_DIGITIZER_STYLUS_MODE = 'absolute-android';

export function normalizeMouseMode(value: string | null | undefined): MouseMode {
  if (value === LEGACY_DIGITIZER_STYLUS_MODE) return 'digitizer-stylus';
  if (
    value === 'absolute' ||
    value === 'digitizer-stylus' ||
    value === 'touch-screen' ||
    value === 'relative'
  ) {
    return value;
  }
  return 'absolute';
}

export const mouseModeAtom = atom<MouseMode>('absolute');

// The device profile is separate from the browser-local mouse mode because
// `relative` is a local UI mode and must never rewrite HID2. Until the first
// authoritative GET succeeds, all input producers remain paused.
export const pointerModeDeviceStatusAtom = atom<PointerModeDeviceStatus>({
  mode: null,
  switching: false,
  degraded: false,
  ready: false,
  error: ''
});

export const pointerModeOperationAtom = atom(false);

export const pointerModeInputEnabledAtom = atom((get) => {
  const status = get(pointerModeDeviceStatusAtom);
  return status.ready && !status.switching && !status.degraded && !get(pointerModeOperationAtom);
});

// mouse scroll direction: -1 or 1
export const scrollDirectionAtom = atom(-1);

// mouse scroll interval (unit: ms)
export const scrollIntervalAtom = atom(0);
