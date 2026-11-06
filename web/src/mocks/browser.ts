import { getDefaultStore } from 'jotai';
import { isCommonAssetRequest } from 'msw';
import { setupWorker } from 'msw/browser';

import { createHttpHandlers } from './handlers/index.ts';
import { createMockHubs, type MockHubs } from './hubs.ts';
import { createMockState, type MockState } from './state.ts';
import { applyMockTarget, parseMockFeatures } from './target.ts';
import { createTransportHandlers } from './transports.ts';

export const worker = setupWorker();

let activeHubs: MockHubs | undefined;

function reportUnhandledRequest(request: Request, print: { error: () => void }) {
  if (isCommonAssetRequest(request)) return;

  const url = new URL(request.url);
  if (url.origin === window.location.origin && url.pathname.startsWith('/api/')) {
    print.error();
  }
}

/**
 * Starts a fresh, page-scoped mock runtime before React is allowed to mount.
 * It intentionally does not persist state, tokens, or credentials between a
 * reload and normal development is never allowed to call this entry point.
 */
export async function startMockedFrontend(): Promise<MockState> {
  activeHubs?.dispose();

  const state = createMockState();
  const hubs = createMockHubs();
  const origin = window.location.origin;

  // Dev-only feature switches for direct debugging; invalid selectors are
  // ignored so this never affects the production or plain-dev entry points.
  const featureOverrides = parseMockFeatures(window.location.hash);
  if (featureOverrides.tailscaleCnEnabled !== undefined) {
    state.application.tailscaleCnEnabled = featureOverrides.tailscaleCnEnabled;
  }

  worker.resetHandlers(
    ...createHttpHandlers(origin, state, hubs),
    ...createTransportHandlers(origin, state, hubs)
  );

  try {
    await worker.start({ quiet: true, onUnhandledRequest: reportUnhandledRequest });
    activeHubs = hubs;
    applyMockTarget(window.location.hash, getDefaultStore());
    return state;
  } catch (error) {
    hubs.dispose();
    throw error;
  }
}
