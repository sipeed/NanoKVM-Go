import { http } from '@/lib/http.ts';

export type ApplicationRegion = 'cn' | 'global';

export type ApplicationRegionResponse = {
  region: ApplicationRegion;
  cloudCleanupFailed?: boolean;
};

export type ApplicationUpdateSource = '' | 'online' | 'offline';

export type ApplicationUpdatePhase =
  | 'idle'
  | 'downloading'
  | 'uploading'
  | 'verifying'
  | 'staging'
  | 'installing'
  | 'restarting'
  | 'succeeded'
  | 'failed';

export type ApplicationUpdateStatus = {
  operationId: string;
  source: ApplicationUpdateSource;
  version: string;
  phase: ApplicationUpdatePhase;
  progress: number;
  errorCode: string;
  startedAt: string;
  updatedAt: string;
};

export type ApplicationResponse<T> = {
  code: number;
  msg: string;
  data: T;
};

export type OfflineUpdateOptions = {
  signal?: AbortSignal;
  onProgress?: (progress: number) => void;
};

export type SignatureVerificationStatus = {
  enabled: boolean;
};

export type UpdateServerSettings = {
  server: string;
};

export function getUpdateServer(): Promise<ApplicationResponse<UpdateServerSettings>> {
  return http.get('/api/application/update/server') as Promise<
    ApplicationResponse<UpdateServerSettings>
  >;
}

export function setUpdateServer(
  server: string
): Promise<ApplicationResponse<UpdateServerSettings>> {
  return http.post('/api/application/update/server', { server }) as Promise<
    ApplicationResponse<UpdateServerSettings>
  >;
}

// get application version
export function getVersion() {
  return http.get('/api/application/version');
}

// update application to latest version
export function update(): Promise<ApplicationResponse<ApplicationUpdateStatus | null>> {
  return http.request({
    method: 'post',
    url: '/api/application/update',
    timeout: 15 * 60 * 1000 // 15 minutes
  }) as Promise<ApplicationResponse<ApplicationUpdateStatus | null>>;
}

// Do not set Content-Type here: the browser adds the multipart boundary for
// this FormData body.
export function updateOffline(
  file: File,
  { signal, onProgress }: OfflineUpdateOptions = {}
): Promise<ApplicationResponse<ApplicationUpdateStatus | null>> {
  const data = new FormData();
  data.append('file', file);

  return http.post('/api/application/update/offline', data, {
    signal,
    timeout: 0,
    onUploadProgress: (event) => {
      const total = event.total && event.total > 0 ? event.total : file.size;
      if (total <= 0) return;
      onProgress?.(Math.min(100, Math.round((event.loaded / total) * 100)));
    }
  }) as Promise<ApplicationResponse<ApplicationUpdateStatus | null>>;
}

export function getUpdateStatus(): Promise<ApplicationResponse<ApplicationUpdateStatus>> {
  return http.get('/api/application/update/status') as Promise<
    ApplicationResponse<ApplicationUpdateStatus>
  >;
}

export function acknowledgeUpdateStatus(
  operationId: string
): Promise<ApplicationResponse<ApplicationUpdateStatus>> {
  return http.post('/api/application/update/status/acknowledge', { operationId }) as Promise<
    ApplicationResponse<ApplicationUpdateStatus>
  >;
}

export function getSignatureVerification(): Promise<
  ApplicationResponse<SignatureVerificationStatus>
> {
  return http.get('/api/application/signature-verification') as Promise<
    ApplicationResponse<SignatureVerificationStatus>
  >;
}

// Refresh the local feature manifest before loading remote-network settings.
export function syncServerFeatures(): Promise<ApplicationResponse<null>> {
  return http.post('/api/application/features/sync') as Promise<ApplicationResponse<null>>;
}

export function getRegion(): Promise<ApplicationResponse<ApplicationRegionResponse>> {
  return http.get('/api/application/region') as Promise<
    ApplicationResponse<ApplicationRegionResponse>
  >;
}

export function setRegion(
  region: ApplicationRegion
): Promise<ApplicationResponse<ApplicationRegionResponse>> {
  return http.post('/api/application/region', { region }) as Promise<
    ApplicationResponse<ApplicationRegionResponse>
  >;
}

export function setSignatureVerification(
  enabled: boolean,
  challenge?: string
): Promise<ApplicationResponse<SignatureVerificationStatus>> {
  const data = challenge ? { enabled, challenge } : { enabled };
  return http.post('/api/application/signature-verification', data) as Promise<
    ApplicationResponse<SignatureVerificationStatus>
  >;
}

// enable/disable preview updates
export function setPreviewUpdates(enable: boolean) {
  const data = {
    enable
  };
  return http.post('/api/application/preview', data);
}

// get preview updates state
export function getPreviewUpdates() {
  return http.get('/api/application/preview');
}
