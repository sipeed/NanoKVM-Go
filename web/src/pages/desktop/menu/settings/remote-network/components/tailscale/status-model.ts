import type { RemoteNetworkStatus } from '../../types.ts';
import type { Status } from './types.ts';

/** Managed state wins; otherwise preserve the legacy Tailscale mapping. */
export function toRemoteNetworkStatus(
  value: Status | Status['state'] | undefined
): RemoteNetworkStatus {
  if (typeof value === 'object' && value !== null) {
    switch (value.managedState) {
      case 'enrolling':
      case 'enabledConnecting':
        return 'busy';
      case 'enabledConnected':
        return 'on';
      case 'degraded':
      case 'invalidIdentity':
        return 'bad';
      default:
        break;
    }
  }
  const state = typeof value === 'string' ? value : value?.state;
  switch (state) {
    case 'running':
      return 'on';
    case 'installing':
      return 'busy';
    case 'notInstall':
    case 'notRunning':
    case 'notLogin':
    case 'stopped':
      return 'off';
    default:
      return 'bad';
  }
}
