export type State = 'installing' | 'notInstall' | 'notRunning' | 'notLogin' | 'stopped' | 'running';

export type TailscaleModeId = 'official' | 'sipeed';

export type ManagedState =
  | 'disabled'
  | 'enrolling'
  | 'enabledConnecting'
  | 'enabledConnected'
  | 'degraded'
  | 'invalidIdentity';

export type Status = {
  state: State;
  name: string;
  ip: string;
  account: string;
  routingAvailable: boolean;
  advertiseExitNode: boolean;
  advertiseSubnetRoute: boolean;
  lanSubnet: string;
  availableModes: TailscaleModeId[];
  activeMode: TailscaleModeId | null;
  managedState: ManagedState;
  managedError: string;
  retryAfterSeconds: number;
  sipeedDisclaimerAccepted: boolean;
};
