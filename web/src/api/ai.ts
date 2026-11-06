import { http } from '@/lib/http.ts';

export type AIConfig = {
  mcpEnabled: boolean;
  /** Safe-to-display MCP API key fingerprint; the complete key is never returned here. */
  mcpApiKeyMasked?: string;
};

export type MCPAPIKey = {
  mcpApiKey: string;
};

export type MemoryFabricState = {
  enabled: boolean;
  running?: boolean;
};

export type MemoryFabricModelConfig = {
  model: string;
  baseUrl: string;
  apiKeyConfigured: boolean;
  /** Omit to preserve the current key, set a value to replace it, or use an empty string to clear it. */
  apiKey?: string;
  maxTokens: number;
  timeoutSeconds: number;
};

export type MemoryFabricConfig = {
  revision: string;
  models: {
    default: MemoryFabricModelConfig;
  };
};

export type GetMemoryFabricConfigResponse = {
  code: number;
  msg: string;
  data: {
    config: MemoryFabricConfig;
  };
};

export type SetMemoryFabricConfigResponse = {
  code: number;
  msg: string;
  data: {
    config: MemoryFabricConfig;
    changed: boolean;
    restarted: boolean;
  };
};

export type ScreenTimelapseState = {
  enabled: boolean;
};

export type ScreenTimelapseRecordingConfig = {
  qp: number;
};

export type SetScreenTimelapseRecordingResult = ScreenTimelapseRecordingConfig & {
  changed: boolean;
  restarted: boolean;
};

export type GetScreenTimelapseRecordingResponse = {
  code: number;
  msg: string;
  data: ScreenTimelapseRecordingConfig;
};

export type SetScreenTimelapseRecordingResponse = {
  code: number;
  msg: string;
  data: SetScreenTimelapseRecordingResult;
};

export type ScreenTimelapseRetentionCleanupConfig = {
  enabled: boolean;
  retentionDays: number;
  maxStorageMiB: number;
};

export type ScreenTimelapseRetentionConfig = {
  revision: number;
  cleanup: ScreenTimelapseRetentionCleanupConfig;
};

export type ScreenTimelapseRetentionStatus = {
  running: boolean;
  scannedAt: string;
  usageBytes: number;
  managedDays: number;
  overLimit: boolean;
  blockedReason: string;
  lastCleanupAt: string;
  lastTrigger: string;
  lastDeletedDays: number;
  lastDeletedBytes: number;
  lastError: string;
};

export type ScreenTimelapseRetention = {
  config: ScreenTimelapseRetentionConfig;
  status: ScreenTimelapseRetentionStatus;
};

export type ScreenTimelapseRetentionCandidate = {
  date: string;
  bytes: number;
  reasons: string[];
};

export type ScreenTimelapseRetentionSkipped = {
  date: string;
  bytes?: number;
  reason: string;
};

export type ScreenTimelapseRetentionWarning = {
  path: string;
  reason: string;
  error?: string;
};

export type ScreenTimelapseRetentionPreview = {
  dryRun: true;
  beforeBytes: number;
  projectedAfterBytes: number;
  overLimitAfterCleanup: boolean;
  wouldDeleteDays: number;
  wouldDeleteBytes: number;
  candidates: ScreenTimelapseRetentionCandidate[];
  skipped: ScreenTimelapseRetentionSkipped[];
  warnings: ScreenTimelapseRetentionWarning[];
};

export type SetScreenTimelapseRetentionRequest = {
  revision: number;
  cleanup: ScreenTimelapseRetentionCleanupConfig;
  challenge?: string;
};

export type SetScreenTimelapseRetentionResult = {
  config: ScreenTimelapseRetentionConfig;
  changed: boolean;
  cleanupTriggered: boolean;
};

export type TriggerScreenTimelapseRetentionCleanupResult = {
  accepted: boolean;
};

export type GetScreenTimelapseRetentionResponse = {
  code: number;
  msg: string;
  data: ScreenTimelapseRetention;
};

export type SetScreenTimelapseRetentionResponse = {
  code: number;
  msg: string;
  data: SetScreenTimelapseRetentionResult;
};

export type PreviewScreenTimelapseRetentionResponse = {
  code: number;
  msg: string;
  data: ScreenTimelapseRetentionPreview;
};

export type TriggerScreenTimelapseRetentionCleanupResponse = {
  code: number;
  msg: string;
  data: TriggerScreenTimelapseRetentionCleanupResult;
};

export type AIServiceStateEntry = {
  enabled: boolean;
  updatedAt?: string;
  source?: string;
};

export type AIServiceStateSnapshotEvent = {
  type: 'snapshot';
  revision: number;
  services: Record<string, AIServiceStateEntry>;
};

export type AIServiceStateChangedEvent = {
  type: 'service_changed';
  revision: number;
  service: string;
  enabled: boolean;
  source: string;
  updatedAt?: string;
};

export type AIServiceCapability = {
  id: string;
  supported: boolean;
  minMemoryMiB?: number;
};

export type AIServices = {
  memoryMiB: number;
  memoryTier: string;
  services: AIServiceCapability[];
};

export const aiServiceEventsPath = '/api/ai/services/events';

export function getAIServices() {
  return http.get('/api/ai/services');
}

export function getAIConfig() {
  return http.get('/api/ai/config');
}

export function setAIConfig(mcpEnabled: boolean, challenge?: string) {
  return http.post('/api/ai/config', challenge ? { mcpEnabled, challenge } : { mcpEnabled });
}

export function getMCPAPIKey(challenge?: string) {
  return http.post('/api/ai/mcp/key', challenge ? { challenge } : undefined);
}

export function copyMCPAPIKey(challenge?: string) {
  return http.post('/api/ai/mcp/key/copy', challenge ? { challenge } : undefined);
}

export function regenerateMCPAPIKey(challenge?: string) {
  return http.post('/api/ai/mcp/key/regenerate', challenge ? { challenge } : undefined);
}

export function getMemoryFabricState() {
  return http.get('/api/ai/memory-fabric');
}

export function setMemoryFabricEnabled(enabled: boolean) {
  return http.post('/api/ai/memory-fabric', { enabled });
}

export function getMemoryFabricConfig(): Promise<GetMemoryFabricConfigResponse> {
  return http.get('/api/ai/memory-fabric/config');
}

export function setMemoryFabricConfig(
  config: MemoryFabricConfig
): Promise<SetMemoryFabricConfigResponse> {
  return http.request({
    method: 'put',
    url: '/api/ai/memory-fabric/config',
    data: config
  });
}

export function getScreenTimelapseState() {
  return http.get('/api/ai/screen-timelapse/state');
}

export function setScreenTimelapseEnabled(enabled: boolean, challenge?: string) {
  return http.post(
    '/api/ai/screen-timelapse/state',
    challenge ? { enabled, challenge } : { enabled }
  );
}

export function getScreenTimelapseRecording(): Promise<GetScreenTimelapseRecordingResponse> {
  return http.get('/api/ai/screen-timelapse/recording');
}

export function setScreenTimelapseRecording(
  qp: number
): Promise<SetScreenTimelapseRecordingResponse> {
  return http.request({
    method: 'put',
    url: '/api/ai/screen-timelapse/recording',
    data: { qp }
  });
}

export function authorizeScreenTimelapsePlayback(challenge?: string) {
  return http.post('/api/ai/screen-timelapse/access', challenge ? { challenge } : undefined);
}

export function getScreenTimelapseRetention(): Promise<GetScreenTimelapseRetentionResponse> {
  return http.get('/api/ai/screen-timelapse/retention');
}

export function setScreenTimelapseRetention(
  request: SetScreenTimelapseRetentionRequest
): Promise<SetScreenTimelapseRetentionResponse> {
  return http.request({
    method: 'put',
    url: '/api/ai/screen-timelapse/retention',
    data: request
  });
}

export function previewScreenTimelapseRetention(
  cleanup?: ScreenTimelapseRetentionCleanupConfig
): Promise<PreviewScreenTimelapseRetentionResponse> {
  const data = cleanup === undefined ? { dryRun: true } : { dryRun: true, config: cleanup };
  return http.post('/api/ai/screen-timelapse/cleanup', data);
}

export function triggerScreenTimelapseRetentionCleanup(
  challenge?: string
): Promise<TriggerScreenTimelapseRetentionCleanupResponse> {
  const data = challenge === undefined ? { dryRun: false } : { dryRun: false, challenge };
  return http.post('/api/ai/screen-timelapse/cleanup', data);
}

export type ScreenTimelapseDay = {
  date: string;
  videoUrl: string;
  startTime: string;
  endTime: string;
  durationMs: number;
  frameCount: number;
  filesize: number;
  deletable: boolean;
  width?: number;
  height?: number;
};

export type ScreenTimelapseFrame = {
  pts: number;
  currentTime: number;
  capturedAt: string;
};

export type ScreenTimelapseFrames = {
  date: string;
  durationMs: number;
  frameCount: number;
  frames: ScreenTimelapseFrame[];
};

export type ScreenTimelapseSearchResult = {
  capturedAt: string;
  date: string;
  pts?: number;
  currentTime?: number;
  matchedFrameTime?: string;
  snippet?: string;
  available: boolean;
  unavailableReason?: string;
};

export type ScreenTimelapseSearchResponse = {
  query: string;
  count: number;
  results: ScreenTimelapseSearchResult[];
  skippedMissingFileCount: number;
};

export function getScreenTimelapseDays(recent = 30) {
  return http.get('/api/ai/screen-timelapse/days', { recent });
}

export function getScreenTimelapseFrames(date: string) {
  return http.get('/api/ai/screen-timelapse/frames', { date });
}

export function deleteScreenTimelapseDay(date: string) {
  return http.delete(`/api/ai/screen-timelapse/day?date=${encodeURIComponent(date)}`);
}

export function searchScreenTimelapse(query: string, limit = 20) {
  return http.get('/api/ai/screen-timelapse/search', { query, limit });
}
