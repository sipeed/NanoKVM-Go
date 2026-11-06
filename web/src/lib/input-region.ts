export const INPUT_REGION_ID = 'input-region';
export const INPUT_REGION_HOST_ATTRIBUTE = 'data-input-region-host';

// The #input-region node is destroyed and recreated whenever its portal host switches between
// the normal and no-signal screens, so consumers must not capture it once at mount. This
// registry always points at the currently attached element; subscribers are called with the
// current value immediately and again on every identity change.

type InputRegionListener = (element: HTMLElement | null) => void;

let currentInputRegion: HTMLElement | null = null;
const inputRegionListeners = new Set<InputRegionListener>();

export function registerInputRegion(element: HTMLElement | null) {
  if (currentInputRegion === element) return;
  currentInputRegion = element;
  for (const listener of inputRegionListeners) listener(element);
}

export function getInputRegionElement(): HTMLElement | null {
  return currentInputRegion;
}

export function subscribeInputRegion(listener: InputRegionListener) {
  inputRegionListeners.add(listener);
  listener(currentInputRegion);
  return () => {
    inputRegionListeners.delete(listener);
  };
}
