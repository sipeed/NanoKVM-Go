import { http } from '@/lib/http.ts';

type StorageResponse<T> = {
  code: number;
  msg: string;
  data: T;
};

export type ImageDownloadStatus = {
  status: string;
  file: string;
  percentage: string;
  downloadedBytes: number;
  totalBytes: number;
  bytesPerSecond: number;
  remainingSeconds: number;
};

// get image list
export function getImages() {
  return http.get('/api/storage/image');
}

// get mounted image
export function getMountedImage() {
  return http.get('/api/storage/image/mounted');
}

// mount/unmount image
export function mountImage(file?: string, cdrom?: boolean, readOnly?: boolean) {
  const data = {
    file: file ? file : '',
    cdrom: cdrom,
    readOnly: readOnly
  };
  return http.post('/api/storage/image/mount', data);
}

export type ImageUploadChunk = {
  uploadId: string;
  filename: string;
  chunkSize: number;
  chunkIndex: number;
  totalChunks: number;
  sha256?: string;
};

export type ImageUploadOptions = {
  signal?: AbortSignal;
  onProgress?: (uploadedBytes: number) => void;
};

// Upload the chunk as a raw body so the device can stream it directly to disk.
export function uploadImage(
  chunk: Blob,
  metadata: ImageUploadChunk,
  { signal, onProgress }: ImageUploadOptions = {}
) {
  return http.request({
    url: '/api/storage/image/upload',
    method: 'post',
    params: metadata,
    signal,
    headers: {
      'Content-Type': 'application/octet-stream'
    },
    onUploadProgress: (event) => {
      onProgress?.(Math.min(chunk.size, Math.max(0, event.loaded)));
    },
    data: chunk
  });
}

export type ImageUploadCancellationStatus = {
  status: 'cancelled' | 'idle';
  uploadId: string;
};

export function cancelImageUpload(
  uploadId: string
): Promise<StorageResponse<ImageUploadCancellationStatus>> {
  return http.post('/api/storage/image/upload/cancel', { uploadId }) as Promise<
    StorageResponse<ImageUploadCancellationStatus>
  >;
}

// delete image (optional challenge when advanced deleteImage protection is on)
export function deleteImage(file: string, challenge?: string) {
  return challenge
    ? http.post('/api/storage/image/delete', { file, challenge })
    : http.post('/api/storage/image/delete', { file });
}

// download image
export function downloadImage(file?: string, sha256?: string) {
  const data = {
    file: file ? file : '',
    sha256: sha256 || ''
  };
  return http.post('/api/storage/download/image', data);
}

export function cancelImageDownload(): Promise<StorageResponse<ImageDownloadStatus>> {
  return http.post('/api/storage/download/image/cancel') as Promise<
    StorageResponse<ImageDownloadStatus>
  >;
}

export function pauseImageDownload(): Promise<StorageResponse<ImageDownloadStatus>> {
  return http.post('/api/storage/download/image/pause') as Promise<
    StorageResponse<ImageDownloadStatus>
  >;
}

export function resumeImageDownload(): Promise<StorageResponse<ImageDownloadStatus>> {
  return http.post('/api/storage/download/image/resume') as Promise<
    StorageResponse<ImageDownloadStatus>
  >;
}

// get image download status
export function statusImage(): Promise<StorageResponse<ImageDownloadStatus>> {
  return http.get('/api/storage/download/image/status') as Promise<
    StorageResponse<ImageDownloadStatus>
  >;
}

// check if download is supported
export function imageEnabled() {
  return http.get('/api/storage/download/image/enabled');
}
