import type { RequestHandler } from 'msw';

import type { MockHubs } from '../hubs.ts';
import type { MockState } from '../state.ts';
import { createAIHandlers } from './ai.ts';
import { createApplicationHandlers } from './application.ts';
import { createAuthHandlers } from './auth.ts';
import { createCloudflareHandlers } from './cloudflare.ts';
import { createDeviceHandlers } from './device.ts';
import { createFRPHandlers } from './frp.ts';
import { createNetBirdHandlers } from './netbird.ts';
import { createStorageHandlers } from './storage.ts';
import { createTailscaleHandlers } from './tailscale.ts';
import { createZeroTierHandlers } from './zerotier.ts';

/**
 * Builds exact-origin HTTP handlers. Keeping the origin as an argument lets a
 * Vite server move between localhost, 127.0.0.1, or another development port
 * without accidentally intercepting an unrelated site's API calls.
 */
export function createHttpHandlers(
  origin: string,
  state: MockState,
  hubs: MockHubs
): RequestHandler[] {
  return [
    ...createAuthHandlers(origin, state),
    ...createApplicationHandlers(origin, state, hubs),
    ...createStorageHandlers(origin, state),
    ...createDeviceHandlers(origin, state),
    ...createAIHandlers(origin, state, hubs),
    ...createTailscaleHandlers(origin, state),
    ...createFRPHandlers(origin, state),
    ...createZeroTierHandlers(origin, state),
    ...createNetBirdHandlers(origin, state),
    ...createCloudflareHandlers(origin, state)
  ];
}
