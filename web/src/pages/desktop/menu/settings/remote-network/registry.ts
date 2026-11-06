import { cloudflareComponent } from './components/cloudflare';
import { frpComponent } from './components/frp';
import { netbirdComponent } from './components/netbird';
import { tailscaleComponent } from './components/tailscale';
import { zerotierComponent } from './components/zerotier';
import type { RemoteNetworkComponent } from './types.ts';

const components: RemoteNetworkComponent[] = [];

/**
 * Register a remote network provider with the settings hub.
 *
 * Adding a provider is a three step change and never touches
 * the home page, the shell or the settings tab:
 *   1. create `components/<id>/` exporting a `RemoteNetworkComponent`,
 *   2. import it here,
 *   3. call `registerRemoteNetworkComponent` below.
 *
 * Components are listed on the home page in registration order; ids are unique.
 */
export function registerRemoteNetworkComponent(component: RemoteNetworkComponent): void {
  if (components.some((item) => item.id === component.id)) {
    return;
  }

  components.push(component);
}

export function getRemoteNetworkComponents(): readonly RemoteNetworkComponent[] {
  return components;
}

export function getRemoteNetworkComponent(id: string): RemoteNetworkComponent | undefined {
  return components.find((component) => component.id === id);
}

registerRemoteNetworkComponent(tailscaleComponent);
registerRemoteNetworkComponent(netbirdComponent);
registerRemoteNetworkComponent(zerotierComponent);
registerRemoteNetworkComponent(cloudflareComponent);
registerRemoteNetworkComponent(frpComponent);
