import type { ImageDownloadStatus } from '@/api/storage.ts';
import { http } from '@/lib/http.ts';
import { getBaseUrl } from '@/lib/service.ts';

export type MediaType = 'image' | 'partition';

// What the emulated USB disk currently exposes to the target machine. "none"
// is a medium that was really released, "unsupported" means the gadget has no
// mass storage at all because the virtual disk is off or the device runs in
// HID-only mode.
export type MediaState = 'none' | 'image' | 'partition' | 'unsupported';

export interface MediaItem {
  type: MediaType;
  rel: string;
  name: string;
  size: number;
  fs: string;
  browsable: boolean;
  attached: boolean;
  modifiedAt: string;
}

// An optical image is one the emulated disk can carry as a CD-ROM: an ISO 9660
// image or a UDF one, which the server reports under its own file system name.
export function isOpticalImage(item: MediaItem): boolean {
  return item.fs === 'iso9660' || item.fs === 'udf';
}

export interface MediaCapabilities {
  loop: boolean;
}

export interface MediaStateInfo {
  sharedDiskInitialized: boolean;
  state: MediaState;
  type?: MediaType;
  rel?: string;
  hidOnly: boolean;
  open: string[];
  // True while the attached medium is served as a CD-ROM.
  cdrom: boolean;
  readOnly: boolean;
  localWritable: boolean;
  localStorageError?: string;
}

export interface MediaFileEntry {
  name: string;
  size: number;
  isDir: boolean;
  modifiedAt: string;
}

export interface MediaFiles {
  path: string;
  entries: MediaFileEntry[];
  free: number;
  total: number;
  // True while the medium is seeing writes: a zero length entry in that state
  // is a transfer in progress, not an empty file.
  writing?: boolean;
}

export interface MediaStatus {
  status: 'idle' | 'in_progress' | 'success' | 'failed';
  name: string;
  percentage: string;
  // Bytes transferred so far and the expected total; total is 0 while the
  // server does not know the size yet.
  loaded: number;
  total: number;
  message: string;
}

// Error codes the media API reports; the UI reacts to these instead of
// matching on message text.
export const codeStorageDisabled = -10;
export const codeMediumLocked = -11;
export const codeMediumBusy = -12;
export const codeNotBrowsable = -13;
export const codeInsufficientSpace = -14;
// A CD-ROM was asked for a medium that is not an ISO 9660 image.
export const codeNotOptical = -15;

export function getMedia() {
  return http.get('/api/storage/media');
}

export function getMediaState() {
  return http.get('/api/storage/media/state');
}

export function getMediaStatus() {
  return http.get('/api/storage/media/status');
}

export function deleteMedia(data: { type: MediaType; rel: string; challenge?: string }) {
  return http.post('/api/storage/media/delete', data);
}

export function mountMedia(data: {
  type: MediaType;
  rel: string;
  cdrom: boolean;
  readOnly: boolean;
  force?: boolean;
}) {
  return http.post('/api/storage/media/mount', data);
}

export function ejectMedia(force = false) {
  return http.post('/api/storage/media/eject', { force });
}

export function listMediaFiles(params: { type: MediaType; rel: string; path: string }) {
  return http.get('/api/storage/media/files', params);
}

export function mkdirMediaFile(data: { type: MediaType; rel: string; path: string }) {
  return http.post('/api/storage/media/files/mkdir', data);
}

export function deleteMediaFile(data: {
  type: MediaType;
  rel: string;
  path: string;
  challenge?: string;
}) {
  return http.post('/api/storage/media/files/delete', data);
}

export function mediaQuery(type: MediaType, rel: string, path = ''): string {
  const params = new URLSearchParams();
  params.set('type', type);
  if (type === 'image') {
    params.set('rel', rel);
  }
  if (path) {
    params.set('path', path);
  }
  return params.toString();
}

// Downloads go through the browser so it can show its own progress: an anchor
// with the session cookie is all a file transfer needs.
export function mediaFileDownloadUrl(type: MediaType, rel: string, path: string): string {
  return `${getBaseUrl('http')}/api/storage/media/files/download?${mediaQuery(type, rel, path)}`;
}

export function mediaImageDownloadUrl(rel: string): string {
  return `${getBaseUrl('http')}/api/storage/media/download?${mediaQuery('image', rel)}`;
}

// The fetch requests below carry the session cookie the way the axios client
// does: VITE_WITH_CREDENTIALS=false is the only setting that leaves it out.
export function fetchCredentials(): 'omit' | 'include' {
  return 'include';
}

// Keep each request at 8 MiB, matching image uploads. The server retains the
// upload and storage ownership between requests, and publishes only at the end.
export async function uploadMediaFile(
  type: MediaType,
  rel: string,
  path: string,
  file: File,
  signal?: AbortSignal,
  onProgress?: (loaded: number, total: number) => void
): Promise<{ code: number; msg: string; data: unknown }> {
  const uploadId = Array.from(crypto.getRandomValues(new Uint8Array(16)), (byte) =>
    byte.toString(16).padStart(2, '0')
  ).join('');
  const chunkSize = 8 * 1024 * 1024;
  const totalChunks = Math.max(1, Math.ceil(file.size / chunkSize));
  let complete = false;
  try {
    for (let chunkIndex = 0; chunkIndex < totalChunks; chunkIndex++) {
      signal?.throwIfAborted();
      const params = new URLSearchParams(mediaQuery(type, rel, path));
      params.set('uploadId', uploadId);
      params.set('filename', file.name);
      params.set('size', String(file.size));
      params.set('chunkSize', String(chunkSize));
      params.set('chunkIndex', String(chunkIndex));
      params.set('totalChunks', String(totalChunks));
      const response = await fetch(
        `${getBaseUrl('http')}/api/storage/media/files/upload?${params}`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/octet-stream' },
          body: file.slice(
            chunkIndex * chunkSize,
            Math.min((chunkIndex + 1) * chunkSize, file.size)
          ),
          credentials: fetchCredentials(),
          signal
        }
      );
      const rsp = await response.json();
      if (rsp.code !== 0) return rsp;
      complete = rsp.data.complete;
      onProgress?.(rsp.data.written, file.size);
      if (complete) return rsp;
    }
    throw new Error('file upload did not complete');
  } finally {
    if (!complete) {
      // The aborted chunk request releases first; this then removes its
      // partial file and releases the ownership retained between chunks.
      await fetch(`${getBaseUrl('http')}/api/storage/media/files/upload/cancel`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ uploadId }),
        credentials: fetchCredentials()
      }).catch(() => {});
    }
  }
}

export function getImageTransferStatus(): Promise<{
  code: number;
  msg: string;
  data: ImageDownloadStatus;
}> {
  return http.get('/api/storage/media/transfer');
}
