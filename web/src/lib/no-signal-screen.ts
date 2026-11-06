export const NO_SIGNAL_SCREEN_ID = 'no-signal-screen';

export type ScreenResolution = {
  width: number;
  height: number;
};

export function validScreenResolution(width: unknown, height: unknown): ScreenResolution | null {
  if (
    typeof width !== 'number' ||
    typeof height !== 'number' ||
    !Number.isFinite(width) ||
    !Number.isFinite(height) ||
    width <= 0 ||
    height <= 0
  ) {
    return null;
  }

  return { width, height };
}
