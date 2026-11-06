import type { ComponentType } from 'react';

import { Sipeed as SipeedMark } from '@/components/icons/sipeed';
import { Tailscale as TailscaleMark } from '@/components/icons/tailscale';

import { getAllowedModeIds } from './capability.ts';

export type TailscaleModeId = 'official' | 'sipeed';

export type TailscaleMode = {
  id: TailscaleModeId;
  /** Component message keys, resolved by `useRemoteNetworkMessages`. */
  nameKey: string;
  descriptionKey: string;
  badgeKey?: string;
  Icon: ComponentType<{ size?: number }>;
  /** Marked mode is preselected when the picker opens. */
  recommended?: boolean;
};

/**
 * Every mode this component can offer, in display order. Availability is the
 * authenticated backend response.
 */
const allModes: TailscaleMode[] = [
  {
    id: 'sipeed',
    nameKey: 'settings.tailscale.mode.sipeed',
    descriptionKey: 'settings.tailscale.mode.sipeedDesc',
    badgeKey: 'settings.remoteNetwork.mode.experimental',
    Icon: SipeedMark,
    recommended: true
  },
  {
    id: 'official',
    nameKey: 'settings.tailscale.mode.official',
    descriptionKey: 'settings.tailscale.mode.officialDesc',
    Icon: TailscaleMark
  }
];

export function getTailscaleModes(backendModes: readonly string[] | undefined): TailscaleMode[] {
  const allowed = new Set(getAllowedModeIds(backendModes));
  return allModes.filter((mode) => allowed.has(mode.id));
}

/** The mode selected when the picker opens: the recommended one wins. */
export function getDefaultTailscaleModeId(modes: TailscaleMode[]): TailscaleModeId {
  return (modes.find((mode) => mode.recommended) ?? modes[0])?.id ?? 'official';
}

export function getTailscaleMode(modes: TailscaleMode[], id: TailscaleModeId): TailscaleMode {
  return modes.find((mode) => mode.id === id) ?? modes[0];
}
