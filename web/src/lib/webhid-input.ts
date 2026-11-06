import {
  InputKind,
  inputRouter,
  InputRouterClient,
  InputSource,
  TouchscreenContact
} from '@/lib/input-router.ts';

export type WebHIDDeviceFilter = {
  vendorId?: number;
  productId?: number;
  usagePage?: number;
  usage?: number;
};

export type WebHIDDevice = {
  opened: boolean;
  productId: number;
  productName?: string;
  vendorId: number;
  open(): Promise<void>;
  close(): Promise<void>;
  addEventListener(type: 'inputreport', listener: WebHIDReportListener): void;
  removeEventListener(type: 'inputreport', listener: WebHIDReportListener): void;
};

export type WebHIDInputReport = {
  device: WebHIDDevice;
  reportId: number;
  data: DataView;
};

export type WebHIDReportListener = (event: WebHIDInputReport) => void;

export type WebHIDSemanticInput =
  | { kind: InputKind.KeyboardState; report: Uint8Array }
  | {
      kind: InputKind.RelativePointer;
      buttons: number;
      deltaX?: number;
      deltaY?: number;
      wheel?: number;
    }
  | {
      kind: InputKind.AbsolutePointer;
      buttons: number;
      x: number;
      y: number;
      wheel?: number;
    }
  | { kind: InputKind.Stylus; state: number; x: number; y: number; pressure?: number }
  | { kind: InputKind.Touchscreen; contacts: TouchscreenContact[] };

export type WebHIDProfile = {
  id: string;
  label?: string;
  filters: WebHIDDeviceFilter[];
  matches(device: WebHIDDevice): boolean;
  decode(report: WebHIDInputReport): WebHIDSemanticInput[];
  release?(): WebHIDSemanticInput[];
};

export type WebHIDProvider = {
  requestDevice(options: { filters: WebHIDDeviceFilter[] }): Promise<WebHIDDevice[]>;
  getDevices(): Promise<WebHIDDevice[]>;
  addEventListener?(type: 'disconnect', listener: (event: { device: WebHIDDevice }) => void): void;
  removeEventListener?(
    type: 'disconnect',
    listener: (event: { device: WebHIDDevice }) => void
  ): void;
};

type ActiveDevice = {
  profile: WebHIDProfile;
  listener: WebHIDReportListener;
};

export class WebHIDInputManager {
  private readonly profiles = new Map<string, WebHIDProfile>();
  private readonly active = new Map<WebHIDDevice, ActiveDevice>();
  private readonly subscribers = new Set<() => void>();
  private readonly handleDisconnect = (event: { device: WebHIDDevice }) => {
    void this.disconnect(event.device, false);
  };

  constructor(
    private readonly provider: WebHIDProvider | undefined = browserWebHID(),
    private readonly router: InputRouterClient = inputRouter,
    private readonly now: () => number = () => performance.now()
  ) {
    this.provider?.addEventListener?.('disconnect', this.handleDisconnect);
  }

  register(profile: WebHIDProfile) {
    if (!profile.id || profile.filters.length === 0) {
      throw new Error('WebHID profile requires an id and at least one filter');
    }
    if (this.profiles.has(profile.id)) {
      throw new Error(`WebHID profile already registered: ${profile.id}`);
    }
    this.profiles.set(profile.id, profile);
    this.notify();
    return () => this.unregister(profile.id);
  }

  async unregister(profileId: string) {
    const profile = this.profiles.get(profileId);
    if (!profile) return;
    this.profiles.delete(profileId);
    await Promise.all(
      Array.from(this.active)
        .filter(([, active]) => active.profile === profile)
        .map(([device]) => this.disconnect(device))
    );
    this.notify();
  }

  async requestProfile(profileId: string) {
    const provider = this.requireProvider();
    const profile = this.requireProfile(profileId);
    const devices = await provider.requestDevice({ filters: profile.filters });
    return this.connectMatching(devices, profile);
  }

  async connectGrantedDevices() {
    const provider = this.requireProvider();
    return this.connectMatching(await provider.getDevices());
  }

  async disconnect(device: WebHIDDevice, close = true) {
    const active = this.active.get(device);
    if (!active) return;
    this.active.delete(device);
    device.removeEventListener('inputreport', active.listener);
    const errors: unknown[] = [];
    try {
      this.dispatch(active.profile.release?.() ?? [], this.now());
    } catch (error) {
      errors.push(error);
    }
    try {
      if (close && device.opened) await device.close();
    } catch (error) {
      errors.push(error);
    } finally {
      this.notify();
    }
    if (errors.length > 0) {
      const error = new Error('failed to release WebHID input') as Error & { errors: unknown[] };
      error.errors = errors;
      throw error;
    }
  }

  async disconnectAll(close = true) {
    await Promise.all(
      Array.from(this.active.keys()).map((device) => this.disconnect(device, close))
    );
  }

  async close() {
    this.provider?.removeEventListener?.('disconnect', this.handleDisconnect);
    await this.disconnectAll();
    this.subscribers.clear();
  }

  subscribe(listener: () => void) {
    this.subscribers.add(listener);
    return () => this.subscribers.delete(listener);
  }

  get available() {
    return Boolean(this.provider);
  }

  get registeredProfiles() {
    return Array.from(this.profiles.values(), (profile) => ({
      id: profile.id,
      label: profile.label ?? profile.id,
      filters: profile.filters.map((filter) => ({ ...filter }))
    }));
  }

  get activeDevices() {
    return Array.from(this.active, ([device, active]) => ({
      profile: active.profile.id,
      productId: device.productId,
      productName: device.productName ?? '',
      vendorId: device.vendorId
    }));
  }

  private async connectMatching(devices: WebHIDDevice[], selected?: WebHIDProfile) {
    const connected: WebHIDDevice[] = [];
    for (const device of devices) {
      const profile =
        selected ?? Array.from(this.profiles.values()).find((item) => item.matches(device));
      if (!profile || !profile.matches(device) || this.active.has(device)) continue;
      if (!device.opened) await device.open();
      const listener: WebHIDReportListener = (report) => {
        try {
          this.dispatch(profile.decode(report), this.now());
        } catch (error) {
          console.warn(`[WebHID] Profile ${profile.id} rejected an input report`, error);
        }
      };
      device.addEventListener('inputreport', listener);
      this.active.set(device, { profile, listener });
      connected.push(device);
    }
    if (connected.length > 0) this.notify();
    return connected;
  }

  private dispatch(events: WebHIDSemanticInput[], capturedAt: number) {
    for (const event of events) dispatchWebHIDInput(this.router, event, capturedAt);
  }

  private requireProvider() {
    if (!this.provider) throw new Error('WebHID is unavailable in this browser');
    return this.provider;
  }

  private requireProfile(profileId: string) {
    const profile = this.profiles.get(profileId);
    if (!profile) throw new Error(`unknown WebHID profile: ${profileId}`);
    return profile;
  }

  private notify() {
    for (const subscriber of this.subscribers) subscriber();
  }
}

export const webHIDInput = new WebHIDInputManager();

export function registerWebHIDProfile(profile: WebHIDProfile) {
  return webHIDInput.register(profile);
}

export function requestWebHIDProfile(profileId: string) {
  return webHIDInput.requestProfile(profileId);
}

export function dispatchWebHIDInput(
  router: InputRouterClient,
  event: WebHIDSemanticInput,
  capturedAt: number
) {
  switch (event.kind) {
    case InputKind.KeyboardState:
      return router.sendKeyboard(InputSource.WebHID, event.report, capturedAt);
    case InputKind.RelativePointer:
      return router.sendRelativePointer(
        InputSource.WebHID,
        event.buttons,
        event.deltaX,
        event.deltaY,
        event.wheel,
        capturedAt
      );
    case InputKind.AbsolutePointer:
      return router.sendAbsolutePointer(
        InputSource.WebHID,
        event.buttons,
        event.x,
        event.y,
        event.wheel,
        capturedAt
      );
    case InputKind.Stylus:
      return router.sendStylus(
        InputSource.WebHID,
        event.state,
        event.x,
        event.y,
        event.pressure,
        capturedAt
      );
    case InputKind.Touchscreen:
      return router.sendTouchscreen(InputSource.WebHID, event.contacts, capturedAt);
  }
}

function browserWebHID() {
  return (globalThis.navigator as (Navigator & { hid?: WebHIDProvider }) | undefined)?.hid;
}
