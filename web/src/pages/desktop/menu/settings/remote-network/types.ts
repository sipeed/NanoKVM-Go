import type { ComponentType } from 'react';

import type { SupportedLanguage } from '@/i18n/locale.ts';

/** Coarse state a component reports for the home card badge. */
export type RemoteNetworkStatus = 'off' | 'busy' | 'on' | 'bad';

export type RemoteNetworkViewProps = {
  /** Leave the component page and return to the remote network home. */
  onBack: () => void;
  /**
   * Report a long running operation to the shell. The shell owns the lock:
   * while locked the settings modal is locked and `onBack` is ignored.
   */
  setLocked: (isLocked: boolean) => void;
};

export type RemoteNetworkComponentIconProps = {
  /** Pixel size of the brand mark. */
  size?: number;
};

/** Flat `key → text` bundle for one language. */
export type RemoteNetworkMessageBundle = Record<string, string>;

/** Component strings; every supported language ships with the component. */
export type RemoteNetworkComponentMessages = Record<SupportedLanguage, RemoteNetworkMessageBundle>;

/**
 * Contract every remote network provider (Tailscale, FRP, ...) implements.
 * The shell only knows this type: adding a component means adding its folder and
 * one registration in `registry.ts`, with no change to the home page, the shell,
 * the global i18n bundles or the locale files.
 */
export type RemoteNetworkComponent = {
  id: string;
  /** Brand name shown on the home card. Product names are not translated. */
  name: string;
  /** Brand mark rendered in the home card icon block. */
  Icon: ComponentType<RemoteNetworkComponentIconProps>;
  /** Optional status badge for the home card; the component owns its own data. */
  Status?: ComponentType;
  /** Component owned strings, resolved by `useRemoteNetworkMessages`. */
  messages: RemoteNetworkComponentMessages;
  /** The component's own settings view. */
  View: ComponentType<RemoteNetworkViewProps>;
};
