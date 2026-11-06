import { http } from '@/lib/http.ts';
import type { InputProfileStore } from '@/lib/input-profile.ts';
import { PortalConfig, ScreenCropState } from '@/lib/screen-crop.ts';

// Consume a Web-session-bound challenge and mint a one-time terminal grant.
export function terminalAuth(challenge: string) {
  return http.post('/api/vm/terminal/auth', { challenge });
}

// get device information
export function getInfo() {
  return http.get('/api/vm/info');
}

// get hardware information
export function getHardware() {
  return http.get('/api/vm/hardware');
}

export type FingerRobotArm = 'long' | 'short';
export type FingerRobotSpeed = 'slow' | 'fast';

export type FingerRobotStrengthByArm = {
  long: number;
  short: number;
};

export type FingerRobotConfig = {
  version: number;
  revision: number;
  activeArm: FingerRobotArm;
  strengthByArm: FingerRobotStrengthByArm;
  pressMs: number;
  speed: FingerRobotSpeed;
};

export type FingerRobotProfile = {
  id: FingerRobotArm;
  displayName: string;
  defaultStrength: number;
};

export type FingerRobotRange = {
  min: number;
  max: number;
  default: number;
};

export type FingerRobotStatus = {
  running: boolean;
  state: string;
  result: string;
  error: string;
};

export type FingerRobotData = {
  enabled: boolean;
  available: boolean;
  config?: FingerRobotConfig;
  profiles: FingerRobotProfile[];
  strength: FingerRobotRange;
  pressMs: FingerRobotRange;
  speedOptions: FingerRobotSpeed[];
  defaultSpeed: FingerRobotSpeed;
  status: FingerRobotStatus;
};

// Read the product-level finger robot configuration and runtime state.
export function getFingerRobot() {
  return http.get('/api/vm/finger-robot');
}

// Replace the complete versioned configuration. Updating it never moves the servo.
export function setFingerRobot(config: FingerRobotConfig, challenge?: string) {
  const data = challenge ? { ...config, challenge } : config;
  return http.request({ method: 'put', url: '/api/vm/finger-robot', data });
}

// Run one complete press from the currently persisted product configuration.
export function pressFingerRobot(challenge?: string) {
  return http.post('/api/vm/finger-robot/press', challenge ? { challenge } : {});
}

// Cancel any active operation and return to the absolute release point.
export function releaseFingerRobot(challenge?: string) {
  return http.post('/api/vm/finger-robot/release', challenge ? { challenge } : {});
}

export function replaceFingerRobotArm(challenge?: string) {
  return http.post('/api/vm/finger-robot/replace-arm', challenge ? { challenge } : {});
}

export type FingerRobotDebugRange = {
  min: number;
  max: number;
  default: number;
};

export type FingerRobotDebugCount = {
  min: number;
  default: number;
  unlimitedValue: 0;
};

export type FingerRobotDebugData = {
  enabled: boolean;
  available: boolean;
  zeroUs: number;
  releaseUs: FingerRobotDebugRange;
  pressUs: FingerRobotDebugRange;
  count: FingerRobotDebugCount;
  pressMs: FingerRobotDebugRange;
  pressSpeedUsPerSecond: FingerRobotDebugRange;
  releaseIntervalMs: number;
  leaseTimeoutMs: number;
  running: boolean;
  current: number;
  total: number;
  state: string;
  result: string;
  error: string;
};

export type FingerRobotDebugTest = {
  releaseUs: number;
  pressUs: number;
  count: number;
  pressMs: number;
  pressSpeedUsPerSecond: number;
};

// Read the engineering debug capability, safe ranges, and current sequence state.
export function getFingerRobotDebug() {
  return http.get('/api/vm/finger-robot/debug');
}

// Start a debug sequence. This reuses the target_power sensitive-action challenge.
export function testFingerRobotDebug(params: FingerRobotDebugTest, challenge?: string) {
  return challenge
    ? http.post('/api/vm/finger-robot/debug/test', { ...params, challenge })
    : http.post('/api/vm/finger-robot/debug/test', params);
}

// Stop the active sequence and return to the release point selected for that sequence.
export function stopFingerRobotDebug() {
  return http.post('/api/vm/finger-robot/debug/stop');
}

// Factory zero deliberately remains separate from stop and does not require a challenge.
export function resetFingerRobotDebug() {
  return http.post('/api/vm/finger-robot/debug/reset');
}

// get HDMI capture status
export function getHdmiCapture() {
  return http.get('/api/vm/hdmi/capture');
}

// set HDMI capture status
export function setHdmiCapture(enabled: boolean) {
  return http.post('/api/vm/hdmi/capture', { enabled });
}

// get SSH state
export function getSSHState() {
  return http.get('/api/vm/ssh');
}

// enable SSH (optional challenge when advanced ssh protection is on)
export function enableSSH(challenge?: string) {
  return challenge
    ? http.post('/api/vm/ssh/enable', { challenge })
    : http.post('/api/vm/ssh/enable');
}

// disable SSH (optional challenge when advanced ssh protection is on)
export function disableSSH(challenge?: string) {
  return challenge
    ? http.post('/api/vm/ssh/disable', { challenge })
    : http.post('/api/vm/ssh/disable');
}

// get mouse jiggler
export function getMouseJiggler() {
  return http.get('/api/vm/mouse-jiggler');
}

// set mouse jiggler
export function setMouseJiggler(enabled: boolean, mode: string) {
  return http.post('/api/vm/mouse-jiggler', { enabled, mode });
}

// get Hostname
export function getHostname() {
  return http.get('/api/vm/hostname');
}

// set Hostname
export function setHostname(hostname: string) {
  return http.post('/api/vm/hostname', { hostname });
}

// get WebTitle
export function getWebTitle() {
  return http.get('/api/vm/web-title');
}

// set WebTitle
export function setWebTitle(title: string) {
  return http.post('/api/vm/web-title', { title });
}

// get mDNS state
export function getMdnsState() {
  return http.get('/api/vm/mdns');
}

// enable mDNS
export function enableMdns() {
  return http.post('/api/vm/mdns/enable');
}

// disable mDNS
export function disableMdns() {
  return http.post('/api/vm/mdns/disable');
}

// get low power mode
export function getLowPowerMode() {
  return http.get('/api/vm/low-power');
}

// set low power mode
export function setLowPowerMode(enable: boolean) {
  return http.post('/api/vm/low-power', { enable });
}

// get swap mode
export function getSwapMode() {
  return http.get('/api/vm/swap');
}

// set swap mode
export function setSwapMode(enable: boolean) {
  return http.post('/api/vm/swap', { enable });
}

// get USB device identity
export function getUsbId() {
  return http.get('/api/vm/usb-id');
}

export type UsbIdFields = {
  vid: string;
  pid: string;
  bcdDevice: string;
  serialnumber: string;
  manufacturer: string;
  product: string;
};

// set USB device identity
export function setUsbId(enabled: boolean, fields: UsbIdFields) {
  return http.post('/api/vm/usb-id', { enabled, ...fields });
}

export type UsbNetworkConfig = {
  mode: 'server' | 'client';
  ipv4: string;
  forwarding: boolean;
};

// get / set USB network configuration
export function getUsbNetwork() {
  return http.get('/api/vm/usb-network');
}

export function setUsbNetwork(config: UsbNetworkConfig) {
  return http.post('/api/vm/usb-network', config);
}

// get / set mass storage inquiry string
export function getDiskInquiry() {
  return http.get('/api/vm/disk-inquiry');
}

export function setDiskInquiry(enabled: boolean, value: string) {
  return http.post('/api/vm/disk-inquiry', { enabled, value });
}

// get / set CD-ROM inquiry string
export function getDiskInquiryCdrom() {
  return http.get('/api/vm/disk-inquiry-cdrom');
}

export function setDiskInquiryCdrom(enabled: boolean, value: string) {
  return http.post('/api/vm/disk-inquiry-cdrom', { enabled, value });
}

// get / set UAC2 function name
export function getUac2FunctionName() {
  return http.get('/api/vm/uac2-function-name');
}

export function setUac2FunctionName(enabled: boolean, value: string) {
  return http.post('/api/vm/uac2-function-name', { enabled, value });
}

// reboot NanoKVM itself (optionally protected by an advanced 2FA challenge)
export function reboot(challenge?: string) {
  return challenge
    ? http.post('/api/vm/system/reboot', { challenge })
    : http.post('/api/vm/system/reboot');
}

// reboot NanoKVM into USB flashing mode (optionally protected by an advanced 2FA challenge)
export function rebootToFlashing(challenge?: string) {
  return challenge
    ? http.post('/api/vm/system/reboot-to-flashing', { challenge })
    : http.post('/api/vm/system/reboot-to-flashing');
}

export const edidErrorReasons = [
  'bad_request',
  'invalid_file_name',
  'edid_too_large',
  'invalid_edid_length',
  'invalid_edid_header',
  'unsupported_edid_version',
  'edid_extension_count_mismatch',
  'invalid_edid_checksum',
  'invalid_cta_extension',
  'current_edid_cannot_be_overwritten',
  'save_failed'
] as const;

export type EdidErrorReason = (typeof edidErrorReasons)[number];

export type EdidResponse<T extends object = object> = {
  code: number;
  msg: string;
  data?: Partial<T> & {
    reason?: EdidErrorReason;
  };
};

export function updateEdid(edid: string): Promise<EdidResponse> {
  return http.post('/api/vm/edid', {
    edid
  });
}

export type EdidMonitorNameResponse = EdidResponse<{ monitorName: string }>;

const edidMonitorNameUnsupportedMessages = new Set([
  'monitor name descriptor not found',
  'multiple monitor name descriptors found'
]);

export function isEdidMonitorNameUnsupportedError(message: string) {
  return edidMonitorNameUnsupportedMessages.has(message);
}

export function getEdidMonitorName(): Promise<EdidMonitorNameResponse> {
  return http.get('/api/vm/edid/monitor-name');
}

export function setEdidMonitorName(monitorName: string): Promise<EdidMonitorNameResponse> {
  return http.post('/api/vm/edid/monitor-name', { monitorName });
}

export function uploadEdid(formData: FormData): Promise<EdidResponse<{ file: string }>> {
  return http.request({
    method: 'post',
    url: '/api/vm/edid/upload',
    data: formData,
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  });
}

export function getCustomEdidList(): Promise<EdidResponse<{ edidList: string[] }>> {
  return http.get('/api/vm/edid/custom');
}

export function deleteEdid(edid: string): Promise<EdidResponse> {
  return http.post('/api/vm/edid/delete', { edid });
}

export type EdidData = {
  edid: string;
  width: number;
  height: number;
};

export function getEdid(): Promise<EdidResponse<EdidData>> {
  return http.get('/api/vm/edid');
}

export type DialBackgroundImage = {
  name: string;
  size: number;
  modTime: number;
  url: string;
};

export function getDialBackgroundImages() {
  return http.get('/api/vm/dial-background');
}

export function switchDialBackground(imageName: string) {
  return http.post('/api/vm/dial-background', { imageName });
}

export function uploadDialBackgroundImage(formData: FormData) {
  return http.request({
    method: 'post',
    url: '/api/vm/dial-background/upload',
    data: formData,
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  });
}

export function getTimezone() {
  return http.get('/api/vm/timezone');
}

export function setTimezone(timezone: string) {
  return http.post('/api/vm/timezone', {
    timezone
  });
}

export function getTimeStatus() {
  return http.get('/api/vm/time/status');
}

export function syncTime() {
  return http.post('/api/vm/time/sync');
}

export function setTime(datetime: string) {
  return http.post('/api/vm/time', { datetime });
}

// get menu bar config
export function getMenuBarConfig() {
  return http.get('/api/vm/menubar');
}

// set menu bar config
export function setMenuBarConfig(disabledItems: string[]) {
  return http.post('/api/vm/menubar', { disabledItems });
}

// get onboarding completion state (device-level, from server.yaml)
export function getOnboarding() {
  return http.get('/api/vm/onboarding');
}

// mark the onboarding tour version as completed or dismissed
export function setOnboardingCompleted(completedVersion: string) {
  return http.post('/api/vm/onboarding', { completedVersion });
}

// mark the mobile portrait onboarding tour version as completed or dismissed
// (independent device-level namespace from the desktop tour)
export function setMobileOnboardingCompleted(mobileCompletedVersion: string) {
  return http.post('/api/vm/onboarding', { mobileCompletedVersion });
}

export function getPortalConfig() {
  return http.get('/api/vm/portal');
}

// report the auto-detected content area so MCP sees the same crop as the UI
export function reportPortalDetectedArea(
  sourceWidth: number,
  sourceHeight: number,
  contentRect: ScreenCropState['contentRect']
) {
  return http.post('/api/vm/portal/area', { sourceWidth, sourceHeight, contentRect });
}

export function setPortalConfig(config: PortalConfig) {
  return http.post('/api/vm/portal', config);
}

export function getInputProfileConfig() {
  return http.get('/api/vm/input-profiles');
}

let inputProfileWriteQueue = Promise.resolve();

export function setInputProfileConfig(config: InputProfileStore) {
  const request = inputProfileWriteQueue
    .catch(() => undefined)
    .then(() => http.post('/api/vm/input-profiles', config));
  inputProfileWriteQueue = request.then(
    () => undefined,
    () => undefined
  );
  return request;
}
