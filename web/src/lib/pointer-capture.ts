export type PointerCaptureMode = 'pointer-lock' | 'normal-pointer';

export type PointerCapabilities = {
  hasTouch: boolean;
  coarsePointer: boolean;
  finePointer: boolean;
  canPointerLock: boolean;
};

export function getPointerCapabilities(): PointerCapabilities {
  if (typeof window === 'undefined') {
    return {
      hasTouch: false,
      coarsePointer: false,
      finePointer: false,
      canPointerLock: false
    };
  }

  return {
    hasTouch: navigator.maxTouchPoints > 0,
    coarsePointer: window.matchMedia?.('(pointer: coarse)').matches ?? false,
    finePointer: window.matchMedia?.('(pointer: fine)').matches ?? false,
    canPointerLock: 'requestPointerLock' in HTMLElement.prototype
  };
}

export function resolvePointerCapture(
  mouseMode: 'absolute' | 'relative',
  capabilities: PointerCapabilities = getPointerCapabilities()
): PointerCaptureMode {
  if (mouseMode === 'absolute') return 'normal-pointer';
  if (capabilities.hasTouch && capabilities.coarsePointer && !capabilities.finePointer) {
    return 'normal-pointer';
  }
  return capabilities.canPointerLock ? 'pointer-lock' : 'normal-pointer';
}
