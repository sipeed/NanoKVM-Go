import type {
  ScreenTimelapseDay,
  ScreenTimelapseRecordingConfig,
  ScreenTimelapseRetentionConfig,
  ScreenTimelapseRetentionStatus
} from '@/api/ai.ts';
import type { ApplicationUpdateStatus } from '@/api/application.ts';
import type { FingerRobotDebugData } from '@/api/vm.ts';
import { getDefaultMenuDisabledItems } from '../lib/localstorage.ts';

export type MockFingerRobotArmID = 'long' | 'short';

export type MockFingerRobotConfig = {
  version: 1;
  revision: number;
  activeArm: MockFingerRobotArmID;
  strengthByArm: Record<MockFingerRobotArmID, number>;
  pressMs: number;
  speed: 'slow' | 'fast';
};

export type MockFingerRobotData = {
  enabled: boolean;
  available: boolean;
  config: MockFingerRobotConfig;
  profiles: Array<{
    id: MockFingerRobotArmID;
    displayName: string;
    defaultStrength: number;
  }>;
  strength: { min: number; max: number; default: number };
  pressMs: { min: number; max: number; default: number };
  speedOptions: Array<'slow' | 'fast'>;
  defaultSpeed: 'slow';
  status: {
    running: boolean;
    state: 'idle' | 'releasing' | 'pressing' | 'holding' | 'resetting';
    result: 'idle' | 'running' | 'success' | 'cancelled' | 'failed' | 'reset' | 'shutdown';
    error: string;
  };
  /** Mirrors the controller's private debug ownership bit. */
  owner: 'normal' | 'debug';
  /** Identifies the operation that most recently took controller ownership. */
  operationKind: 'idle' | 'press' | 'release' | 'test' | 'stop' | 'reset';
  /** Internal cancellation generation; omitted from API responses. */
  operationRevision: number;
};

export type MockWifi = {
  ssid: string;
  bssid: string;
  signal: number;
  frequency: string;
  security: string;
};

export type MockTwoFactorProtection = {
  frp: boolean;
  cloudflare: boolean;
  zerotier: boolean;
  netbird: boolean;
  login: boolean;
  webTerminal: boolean;
  ssh: boolean;
  tailscale: boolean;
  mcp: boolean;
  targetPower: boolean;
  nanoKVMRestart: boolean;
  scripts: boolean;
  screenTimelapse: boolean;
  deleteImage: boolean;
};

export type MockScreenTimelapseRetentionDay = {
  date: string;
  bytes: number;
};

export function createMockTwoFactorProtection(): MockTwoFactorProtection {
  return {
    frp: false,
    cloudflare: false,
    zerotier: false,
    netbird: false,
    login: false,
    webTerminal: false,
    ssh: false,
    tailscale: false,
    mcp: false,
    targetPower: false,
    nanoKVMRestart: false,
    scripts: true,
    screenTimelapse: false,
    deleteImage: false
  };
}

export type MockState = {
  netbird: {
    installed: boolean;
    enabled: boolean;
    authorized: boolean;
    state: 'notLogin' | 'authorizing' | 'connected' | 'offline';
    authorizationPollsRemaining: number;
    busy: boolean;
    error: string;
  };
  zerotier: {
    installed: boolean;
    enabled: boolean;
    online: boolean;
    nodeId: string;
    networks: Record<string, { name: string; status: string; addresses: string[] }>;
  };
  cloudflare: {
    mode: 'named' | 'quick';
    quickUrl: string;
    quickRuns: number;
    installed: boolean;
    enabled: boolean;
    token: string;
    accessUrl: string;
    error: string;
  };
  frp: {
    installed: boolean;
    enabled: boolean;
    error: string;
    serverAddr: string;
    serverPort: number;
    remotePort: number;
    token: string;
  };
  auth: {
    username: string;
    passwordUpdated: boolean;
    twoFactorEnabled: boolean;
    recoveryEnabled: boolean;
    recoveryRemaining: number;
    recoveryCodes: string[];
    twoFactorProtection: MockTwoFactorProtection;
    twoFactorProtectionAuthorizedUntil: number;
    mcpAuthorizationUntil: number;
    screenTimelapseAuthorizationUntil: number;
    targetPowerChallenge: string | null;
    nanoKVMRestartChallenge: string | null;
    deleteImageChallenge: string | null;
    screenTimelapseChallenge: string | null;
    screenTimelapsePlaybackChallenge: string | null;
    webTerminalChallenge: string | null;
    tailscaleExitNodeChallenge: string | null;
    tailscaleSubnetRouteChallenge: string | null;
    tailscaleLoginChallenge: string | null;
    frpChallenge: { action: string; token: string } | null;
    cloudflareChallenge: { action: string; token: string } | null;
    zerotierChallenge: { action: string; token: string } | null;
    netbirdChallenge: { action: string; token: string } | null;
    updateSignatureVerificationChallenge: string | null;
  };
  application: {
    current: string;
    latest: string;
    region: 'cn' | 'global';
    /** Mirrors /etc/kvm/server-feature.json `features.tailscale_cn.enabled`. */
    tailscaleCnEnabled: boolean;
    previewEnabled: boolean;
    updateServer: string;
    signatureVerificationEnabled: boolean;
    update: ApplicationUpdateStatus;
    nextOperation: number;
  };
  menu: {
    disabledItems: string[];
  };
  /** Mirrors server.yaml `onboarding.*` (empty means first-run). */
  onboarding: {
    completedVersion: string;
    mobileCompletedVersion: string;
  };
  stream: {
    rateControl: 'cbr' | 'vbr';
    bitrate: number;
    gop: number;
    fps: number;
    mode: string;
    edid: string;
    customEdids: string[];
  };
  vm: {
    hostname: string;
    webTitle: string;
    sshEnabled: boolean;
    mdnsEnabled: boolean;
    hdmiCaptureEnabled: boolean;
    mouseJiggler: { enabled: boolean; mode: string; maxHours: number; startedAt: number };
    lowPowerEnabled: boolean;
    swapEnabled: boolean;
    timezone: string;
    lastSyncTime: number;
    staticIp: { enabled: boolean; ip: string };
    usbId: {
      enabled: boolean;
      vid: string;
      pid: string;
      bcdDevice: string;
      serialnumber: string;
      manufacturer: string;
      product: string;
      configurationName: string;
    };
    usbNetwork: {
      mode: 'server' | 'client';
      ipv4: string;
      forwarding: boolean;
    };
    diskInquiry: { enabled: boolean; value: string };
    diskInquiryCdrom: { enabled: boolean; value: string };
    uac2FunctionName: { enabled: boolean; value: string };
    webcamFunctionName: { enabled: boolean; value: string };
    dialBackground: {
      current: string;
      files: Array<{ name: string; size: number; modTime: number; url: string }>;
    };
    fingerRobot: MockFingerRobotData;
    fingerRobotDebug: FingerRobotDebugData;
  };
  network: {
    wifi: {
      supported: boolean;
      apMode: boolean;
      connected: boolean;
      current: MockWifi | null;
      available: MockWifi[];
      /** Pre-stored target SSID (empty when absent); passwords are never kept in mock state. */
      preset: string;
    };
    /** Backend string-list contract: each line is "mac" or "mac name" (server/service/network/wol.go). */
    wolMacs: string[];
  };
  hid: {
    mode: string;
    pointerMode: {
      mode: 'absolute' | 'digitizer-stylus' | 'touch-screen' | 'relative' | 'unknown';
      singleEndpoint?: boolean;
      switching: boolean;
      degraded: boolean;
      error: string;
    };
    keyboardLayout: string;
    leaderKey: string;
    shortcuts: Array<{ id: string; keys: string[] }>;
  };
  virtualDevice: {
    isWebcamEnabled: boolean;
    isNetworkEnabled: boolean;
    isDiskEnabled: boolean;
    isSdCardExist: boolean;
    diskMinSizeMiB: number;
    diskMaxSizeMiB: number;
    isMicEnabled: boolean;
    isSpeakerEnabled: boolean;
    hidMode: string;
    mountedImage: string;
    mountedCdrom: boolean;
    mountedReadOnly: boolean;
    imageFiles: string[];
  };
  ai: {
    mcpEnabled: boolean;
    /** Synthetic fixture only; it must never be used as a production credential. */
    mcpApiKey: string;
    memoryFabric: { enabled: boolean; running: boolean };
    memoryFabricConfig: {
      revision: string;
      models: {
        default: {
          model: string;
          baseUrl: string;
          apiKeyConfigured: boolean;
          maxTokens: number;
          timeoutSeconds: number;
        };
      };
    };
    screenTimelapseEnabled: boolean;
    screenTimelapseRecording: ScreenTimelapseRecordingConfig;
    screenTimelapseDays: ScreenTimelapseDay[];
    screenTimelapseRetention: {
      config: ScreenTimelapseRetentionConfig;
      status: ScreenTimelapseRetentionStatus;
      days: MockScreenTimelapseRetentionDay[];
    };
    serviceRevision: number;
  };
  tailscale: {
    state: 'notInstall' | 'notRunning' | 'notLogin' | 'stopped' | 'running';
    login: boolean;
    hostname: string;
    ip: string;
    routingAvailable: boolean;
    advertiseExitNode: boolean;
    advertiseSubnetRoute: boolean;
    lanSubnet: string;
    activeMode: 'official' | 'sipeed' | null;
    managedState:
      | 'disabled'
      | 'enrolling'
      | 'enabledConnecting'
      | 'enabledConnected'
      | 'degraded'
      | 'invalidIdentity';
    managedError: string;
    retryAfterSeconds: number;
    managedPollsRemaining: number;
    managedNetworkRegistered: boolean;
    sipeedDisclaimerAccepted: boolean;
    managedEnrollRequests: number;
    managedJoinKeyRequests: number;
    sipeedClaimCode: { code: string; expiresAt: string; singleUse: boolean } | null;
    sipeedPending: Array<{
      pendingId: string;
      sas: string[];
      createdAt: string;
      clientIp?: string;
      clientUa?: string;
    }>;
    sipeedDevices: Array<{
      nodeId: string;
      name: string;
      ips: string[];
      online: boolean;
      lastSeen?: string;
      isSelf: boolean;
    }>;
    sipeedNextDeviceId: number;
    sipeedManagementRateLimitUntil: number;
    sipeedIdentityValid: boolean;
    sipeedCloudCleanupFailed: boolean;
  };
  clock: {
    epochMs: number;
  };
};

export function createMockState(): MockState {
  return {
    netbird: {
      installed: false,
      enabled: false,
      authorized: false,
      state: 'notLogin',
      authorizationPollsRemaining: 0,
      busy: false,
      error: ''
    },
    zerotier: { installed: false, enabled: false, online: true, nodeId: '', networks: {} },
    cloudflare: {
      mode: 'named',
      quickUrl: '',
      quickRuns: 0,
      installed: false,
      enabled: false,
      token: '',
      accessUrl: '',
      error: ''
    },
    frp: {
      installed: false,
      enabled: false,
      error: '',
      serverAddr: '',
      serverPort: 7000,
      remotePort: 8443,
      token: ''
    },
    auth: {
      username: 'mock-admin',
      passwordUpdated: true,
      twoFactorEnabled: false,
      recoveryEnabled: false,
      recoveryRemaining: 0,
      recoveryCodes: ['MOCK-ALPHA-0001', 'MOCK-BRAVO-0002', 'MOCK-CHARLIE-0003'],
      twoFactorProtection: createMockTwoFactorProtection(),
      twoFactorProtectionAuthorizedUntil: 0,
      mcpAuthorizationUntil: 0,
      screenTimelapseAuthorizationUntil: 0,
      targetPowerChallenge: null,
      nanoKVMRestartChallenge: null,
      deleteImageChallenge: null,
      screenTimelapseChallenge: null,
      screenTimelapsePlaybackChallenge: null,
      webTerminalChallenge: null,
      tailscaleExitNodeChallenge: null,
      tailscaleSubnetRouteChallenge: null,
      tailscaleLoginChallenge: null,
      frpChallenge: null,
      cloudflareChallenge: null,
      zerotierChallenge: null,
      netbirdChallenge: null,
      updateSignatureVerificationChallenge: null
    },
    application: {
      current: 'v2.0.0-mock',
      latest: 'v2.1.0-mock',
      region: 'cn',
      tailscaleCnEnabled: true,
      previewEnabled: false,
      updateServer: '',
      signatureVerificationEnabled: true,
      nextOperation: 1,
      update: {
        operationId: '',
        source: '',
        version: '',
        phase: 'idle',
        progress: 0,
        errorCode: '',
        startedAt: '',
        updatedAt: ''
      }
    },
    menu: {
      disabledItems: getDefaultMenuDisabledItems()
    },
    onboarding: {
      completedVersion: '',
      mobileCompletedVersion: ''
    },
    stream: {
      rateControl: 'vbr',
      bitrate: 8000,
      gop: 50,
      fps: 60,
      mode: 'h264-webrtc',
      edid: 'nanokvm-go-1920x1080-60',
      customEdids: ['mock-development-display.bin']
    },
    vm: {
      hostname: 'mock-nanokvm',
      webTitle: 'NanoKVM Mock',
      sshEnabled: true,
      mdnsEnabled: true,
      hdmiCaptureEnabled: true,
      mouseJiggler: { enabled: false, mode: 'relative', maxHours: 8, startedAt: 0 },
      lowPowerEnabled: false,
      swapEnabled: true,
      timezone: 'Asia/Hong_Kong',
      lastSyncTime: Date.UTC(2026, 0, 2, 3, 4, 5),
      staticIp: { enabled: false, ip: '192.0.2.20/24' },
      usbId: {
        enabled: false,
        vid: '',
        pid: '',
        bcdDevice: '',
        serialnumber: '',
        manufacturer: '',
        product: '',
        configurationName: ''
      },
      usbNetwork: {
        mode: 'server',
        ipv4: '',
        forwarding: false
      },
      diskInquiry: { enabled: false, value: 'LogitechFlash Drive     1.00' },
      diskInquiryCdrom: { enabled: false, value: 'LogitechOptical Drive   1.00' },
      uac2FunctionName: { enabled: false, value: '' },
      webcamFunctionName: { enabled: false, value: '' },
      dialBackground: {
        current: 'mock-grid.bmp',
        files: [
          {
            name: 'mock-grid.bmp',
            size: 64_000,
            modTime: Date.UTC(2026, 0, 2, 3, 4, 5),
            url: '/mock-dial-background.bmp'
          }
        ]
      },
      fingerRobot: {
        enabled: true,
        available: true,
        config: {
          version: 1,
          revision: 1,
          activeArm: 'short',
          strengthByArm: { long: 3, short: 3 },
          pressMs: 800,
          speed: 'slow'
        },
        profiles: [
          { id: 'long', displayName: '长摆臂', defaultStrength: 3 },
          { id: 'short', displayName: '短摆臂', defaultStrength: 3 }
        ],
        strength: { min: 1, max: 5, default: 3 },
        pressMs: { min: 100, max: 30_000, default: 800 },
        speedOptions: ['slow', 'fast'],
        defaultSpeed: 'slow',
        status: {
          running: false,
          state: 'idle',
          result: 'idle',
          error: ''
        },
        owner: 'normal',
        operationKind: 'idle',
        operationRevision: 0
      },
      fingerRobotDebug: {
        enabled: true,
        available: true,
        zeroUs: 1500,
        releaseUs: { min: 889, max: 2050, default: 2050 },
        pressUs: { min: 889, max: 2000, default: 899 },
        count: { min: 1, default: 50, unlimitedValue: 0 },
        pressMs: { min: 100, max: 30_000, default: 100 },
        pressSpeedUsPerSecond: { min: 200, max: 2000, default: 2000 },
        releaseIntervalMs: 500,
        leaseTimeoutMs: 3000,
        running: false,
        current: 0,
        total: 0,
        state: 'idle',
        result: 'idle',
        error: ''
      }
    },
    network: {
      wifi: {
        supported: true,
        apMode: false,
        connected: true,
        current: {
          ssid: 'MockLab',
          bssid: '02:00:00:00:00:01',
          signal: -48,
          frequency: '5 GHz',
          security: 'WPA2'
        },
        available: [
          {
            ssid: 'MockLab',
            bssid: '02:00:00:00:00:01',
            signal: -48,
            frequency: '5 GHz',
            security: 'WPA2'
          },
          {
            ssid: 'Guest Mock Network',
            bssid: '02:00:00:00:00:02',
            signal: -66,
            frequency: '2.4 GHz',
            security: 'Open'
          },
          {
            ssid: 'Mock Attic Repeater',
            bssid: '02:00:00:00:00:03',
            signal: -78,
            frequency: '2.4 GHz',
            security: 'WPA3'
          },
          {
            ssid: 'MockLab 6G',
            bssid: '02:00:00:00:00:04',
            signal: -55,
            frequency: '6 GHz',
            security: 'WPA3'
          }
        ],
        preset: ''
      },
      wolMacs: ['02:00:00:00:00:10 Mock workstation']
    },
    hid: {
      mode: 'hid',
      pointerMode: {
        mode: 'absolute',
        singleEndpoint: true,
        switching: false,
        degraded: false,
        error: ''
      },
      keyboardLayout: 'en-US',
      leaderKey: 'MetaLeft',
      shortcuts: []
    },
    virtualDevice: {
      isWebcamEnabled: false,
      isNetworkEnabled: true,
      isDiskEnabled: true,
      isSdCardExist: true,
      diskMinSizeMiB: 5120,
      diskMaxSizeMiB: 49152,
      isMicEnabled: true,
      isSpeakerEnabled: false,
      hidMode: 'hid',
      mountedImage: '',
      mountedCdrom: false,
      mountedReadOnly: true,
      imageFiles: [
        'debian-12.7.0-amd64-netinst.iso',
        'ubuntu-24.04.1-desktop-amd64.iso',
        'rescue-system.img'
      ]
    },
    ai: {
      mcpEnabled: true,
      mcpApiKey: 'nag_mcp_mock-development-key-not-a-secret-000000000',
      memoryFabric: { enabled: true, running: true },
      memoryFabricConfig: {
        revision: 'mock-revision-1',
        models: {
          default: {
            model: 'mock-local-model',
            baseUrl: 'http://192.0.2.42:11434/v1',
            apiKeyConfigured: false,
            maxTokens: 4096,
            timeoutSeconds: 30
          }
        }
      },
      screenTimelapseEnabled: true,
      screenTimelapseRecording: { qp: 42 },
      screenTimelapseDays: [
        {
          date: '2026-01-02',
          videoUrl: '/api/ai/screen-timelapse/video?date=2026-01-02',
          startTime: '2026-01-02T03:00:00Z',
          endTime: '2026-01-02T03:05:00Z',
          durationMs: 300_000,
          frameCount: 5,
          filesize: 1024,
          deletable: true,
          width: 1920,
          height: 1080
        }
      ],
      screenTimelapseRetention: {
        config: {
          revision: 0,
          cleanup: {
            enabled: false,
            retentionDays: 180,
            maxStorageMiB: 48 * 1024
          }
        },
        status: {
          running: false,
          scannedAt: '',
          usageBytes: 6 * 1024 * 1024 * 1024,
          managedDays: 4,
          overLimit: false,
          blockedReason: '',
          lastCleanupAt: '',
          lastTrigger: '',
          lastDeletedDays: 0,
          lastDeletedBytes: 0,
          lastError: ''
        },
        days: [
          { date: '2026-07-12', bytes: 2 * 1024 * 1024 * 1024 },
          { date: '2026-07-17', bytes: 1024 * 1024 * 1024 },
          { date: '2026-07-19', bytes: 1024 * 1024 * 1024 },
          { date: '2026-07-20', bytes: 2 * 1024 * 1024 * 1024 }
        ]
      },
      serviceRevision: 1
    },
    tailscale: {
      state: 'running',
      login: true,
      hostname: 'mock-nanokvm',
      ip: '100.64.0.42',
      routingAvailable: true,
      advertiseExitNode: false,
      advertiseSubnetRoute: false,
      lanSubnet: '192.168.0.0/22',
      activeMode: 'official',
      managedState: 'disabled',
      managedError: '',
      retryAfterSeconds: 0,
      managedPollsRemaining: 0,
      managedNetworkRegistered: false,
      sipeedDisclaimerAccepted: false,
      managedEnrollRequests: 0,
      managedJoinKeyRequests: 0,
      sipeedClaimCode: null,
      sipeedPending: [],
      sipeedDevices: [],
      sipeedNextDeviceId: 1,
      sipeedManagementRateLimitUntil: 0,
      sipeedIdentityValid: true,
      sipeedCloudCleanupFailed: false
    },
    clock: {
      epochMs: Date.UTC(2026, 0, 2, 3, 4, 5)
    }
  };
}
