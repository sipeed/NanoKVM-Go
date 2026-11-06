import { http, HttpResponse } from 'msw';

import type { MediaFileEntry, MediaFiles, MediaItem } from '@/api/media.ts';

import { fail, ok, readJsonObject, readString, type MockEnvelope } from '../response.ts';
import type { MockState } from '../state.ts';

export function createStorageHandlers(origin: string, state: MockState) {
  const images: MediaItem[] = [
    {
      type: 'image',
      rel: 'rescue.img',
      name: 'rescue.img',
      size: 32 << 20,
      fs: 'fat32',
      browsable: true,
      attached: false,
      modifiedAt: '2026-10-01T00:00:00Z'
    },
    {
      type: 'image',
      rel: 'rescue.iso',
      name: 'rescue.iso',
      size: 64 << 20,
      fs: 'iso9660',
      browsable: false,
      attached: false,
      modifiedAt: '2026-10-01T00:00:00Z'
    }
  ];
  const files = new Map<string, MediaFileEntry[]>([
    [
      'partition::/',
      [
        { name: 'Documents', size: 0, isDir: true, modifiedAt: '2026-09-12T08:30:00Z' },
        { name: 'Photos', size: 0, isDir: true, modifiedAt: '2026-10-02T15:41:00Z' },
        { name: 'Backups', size: 0, isDir: true, modifiedAt: '2026-10-18T22:05:00Z' },
        { name: 'hello.txt', size: 11, isDir: false, modifiedAt: '2026-10-01T00:00:00Z' },
        {
          name: 'release-notes.pdf',
          size: 481_200,
          isDir: false,
          modifiedAt: '2026-10-11T09:24:00Z'
        },
        {
          name: 'holiday.mp4',
          size: 732_000_000,
          isDir: false,
          modifiedAt: '2026-08-19T17:02:00Z'
        },
        { name: 'logo.png', size: 28_400, isDir: false, modifiedAt: '2026-09-30T11:12:00Z' },
        {
          name: 'firmware-backup.zip',
          size: 18_300_000,
          isDir: false,
          modifiedAt: '2026-10-20T03:45:00Z'
        },
        { name: 'song.flac', size: 24_100_000, isDir: false, modifiedAt: '2026-07-05T20:18:00Z' }
      ]
    ],
    [
      'partition::Documents',
      [
        { name: 'manuals', size: 0, isDir: true, modifiedAt: '2026-09-12T08:31:00Z' },
        {
          name: 'setup-guide.docx',
          size: 92_000,
          isDir: false,
          modifiedAt: '2026-09-14T10:00:00Z'
        },
        { name: 'pins.csv', size: 1_200, isDir: false, modifiedAt: '2026-09-15T16:40:00Z' }
      ]
    ],
    [
      'partition::Documents/manuals',
      [
        { name: 'nanokvm.pdf', size: 1_450_000, isDir: false, modifiedAt: '2026-09-16T12:00:00Z' },
        { name: 'warranty.txt', size: 3_400, isDir: false, modifiedAt: '2026-09-16T12:05:00Z' }
      ]
    ],
    [
      'partition::Photos',
      [
        { name: 'lab.jpg', size: 3_240_000, isDir: false, modifiedAt: '2026-10-02T15:42:00Z' },
        { name: 'desk.webp', size: 812_000, isDir: false, modifiedAt: '2026-10-02T15:43:00Z' }
      ]
    ],
    ['partition::Backups', []],
    [
      'image:rescue.img:/',
      [{ name: 'readme.txt', size: 11, isDir: false, modifiedAt: '2026-10-01T00:00:00Z' }]
    ]
  ]);
  const current = () =>
    state.hid.mode === 'hid-only'
      ? 'unsupported'
      : state.virtualDevice.mountedImage
        ? 'image'
        : state.virtualDevice.isDiskEnabled
          ? 'partition'
          : 'none';
  const rel = () => state.virtualDevice.mountedImage.replace(/^\/data\//, '');
  const validPath = (path: string) => !path.split('/').includes('..') && !path.includes('\\');
  const directory = (type: string, medium: string, path: string) =>
    `${type}:${medium}:${path === '/' ? '/' : path.replace(/^\//, '')}`;
  const uploads = new Map<
    string,
    { key: string; name: string; size: number; loaded: number; next: number; chunks: number }
  >();
  function protectedDelete(body: Record<string, unknown>) {
    if (!state.auth.twoFactorEnabled || !state.auth.twoFactorProtection.deleteImage) return false;
    if (
      !state.auth.deleteImageChallenge ||
      readString(body, 'challenge') !== state.auth.deleteImageChallenge
    )
      return true;
    state.auth.deleteImageChallenge = null;
    return false;
  }
  return [
    http.get(`${origin}/api/storage/media`, () =>
      ok({
        media: [
          ...images.map((item) => ({
            ...item,
            attached: current() === 'image' && rel() === item.rel
          })),
          ...(state.virtualDevice.isSdCardExist
            ? [
                {
                  type: 'partition',
                  rel: '',
                  name: 'Shared folder',
                  size: 0,
                  fs: 'exfat',
                  browsable: true,
                  attached: current() === 'partition',
                  modifiedAt: '2026-10-01T00:00:00Z'
                }
              ]
            : [])
        ],
        capabilities: { loop: true }
      })
    ),
    http.get(`${origin}/api/storage/media/state`, () =>
      ok({
        sharedDiskInitialized: state.virtualDevice.isSdCardExist,
        state: current(),
        type: current() === 'image' ? 'image' : 'partition',
        rel: rel(),
        hidOnly: state.hid.mode === 'hid-only',
        open: [],
        cdrom: current() === 'image' && state.virtualDevice.mountedCdrom,
        readOnly: current() === 'image' && state.virtualDevice.mountedReadOnly,
        localWritable: current() !== 'partition'
      })
    ),
    http.get(`${origin}/api/storage/media/status`, () =>
      ok({ status: 'idle', name: '', loaded: 0, total: 0, percentage: '', message: '' })
    ),
    http.get(`${origin}/api/storage/media/transfer`, () =>
      ok({ status: 'idle', file: '', downloadedBytes: 0, totalBytes: 0, percentage: '' })
    ),
    http.post(`${origin}/api/storage/media/mount`, async ({ request }) => {
      const body = await readJsonObject(request);
      if (uploads.size > 0) return fail(-12, 'a file upload is running');
      if (current() === 'unsupported') return fail(-10, 'HID-only mode');
      if (body.type === 'partition') {
        if (!state.virtualDevice.isSdCardExist)
          return fail(-10, 'initialize the virtual disk first');
        if (body.cdrom) return fail(-15, 'not optical');
        state.virtualDevice.mountedImage = '';
        state.virtualDevice.isDiskEnabled = true;
      } else {
        const medium = images.find((item) => item.rel === body.rel);
        if (!medium) return fail(-1, 'invalid medium');
        if (body.cdrom && medium.fs !== 'iso9660' && medium.fs !== 'udf')
          return fail(-15, 'not optical');
        state.virtualDevice.mountedImage = `/data/${medium.rel}`;
        state.virtualDevice.mountedCdrom = body.cdrom === true;
        state.virtualDevice.mountedReadOnly = body.cdrom === true || body.readOnly !== false;
        state.virtualDevice.isDiskEnabled = false;
      }
      return ok(null);
    }),
    http.post(`${origin}/api/storage/media/eject`, () => {
      state.virtualDevice.mountedImage = '';
      state.virtualDevice.isDiskEnabled = false;
      return ok(null);
    }),
    http.post(`${origin}/api/storage/media/delete`, async ({ request }) => {
      const body = await readJsonObject(request);
      if (protectedDelete(body)) return fail(-3, 'sensitive action verification required');
      if (body.type === 'partition') return fail(-1, 'the shared disk cannot be deleted');
      if (current() === 'partition' || rel() === body.rel) return fail(-12, 'medium is in use');
      const index = images.findIndex((item) => item.rel === body.rel);
      if (index < 0) return fail(-1, 'invalid medium');
      images.splice(index, 1);
      return ok(null);
    }),
    http.get(`${origin}/api/storage/media/files`, ({ request }) => {
      const params = new URL(request.url).searchParams;
      const path = params.get('path') || '/';
      if (!validPath(path)) return fail<MediaFiles>(-1, 'invalid path');
      const type = params.get('type') || 'image',
        medium = params.get('rel') || '';
      if (type === 'partition' && !state.virtualDevice.isSdCardExist)
        return fail<MediaFiles>(-10, 'initialize the virtual disk first');
      if (type === 'image' && (current() === 'partition' || rel() === medium))
        return fail<MediaFiles>(-12, 'medium is in use');
      return ok({
        path,
        entries: files.get(directory(type, medium, path)) || [],
        free: 8 * 1024 ** 3,
        total: 16 * 1024 ** 3,
        writing: false
      });
    }),
    http.post(
      `${origin}/api/storage/media/files/upload`,
      async ({ request }): Promise<HttpResponse<MockEnvelope<unknown>>> => {
        if (current() === 'partition') return fail(-12, 'eject the shared disk first');
        const params = new URL(request.url).searchParams;
        if (params.get('type') === 'partition' && !state.virtualDevice.isSdCardExist)
          return fail(-10, 'initialize the virtual disk first');
        const path = params.get('path') || '/';
        if (!validPath(path)) return fail(-1, 'invalid path');
        const uploadId = params.get('uploadId');
        if (uploadId) {
          const key = directory(params.get('type') || 'image', params.get('rel') || '', path);
          const name = params.get('filename') || '';
          const size = Number(params.get('size'));
          const index = Number(params.get('chunkIndex'));
          const chunks = Number(params.get('totalChunks'));
          const chunkSize = Number(params.get('chunkSize'));
          if (
            !/^[a-f0-9]{32}$/.test(uploadId) ||
            !name ||
            /[\\/]/.test(name) ||
            !Number.isSafeInteger(size) ||
            size < 0 ||
            chunkSize !== 8 * 1024 * 1024 ||
            chunks !== Math.max(1, Math.ceil(size / chunkSize)) ||
            !Number.isInteger(index) ||
            index < 0 ||
            index >= chunks
          )
            return fail(-1, 'invalid upload metadata');
          let upload = uploads.get(uploadId);
          if (!upload) {
            if (index !== 0) return fail(-1, 'upload must start with the first chunk');
            if ((files.get(key) || []).some((entry) => entry.name === name))
              return fail(-2, 'file already exists');
            upload = { key, name, size, loaded: 0, next: 0, chunks };
            uploads.set(uploadId, upload);
          }
          if (
            upload.key !== key ||
            upload.name !== name ||
            upload.size !== size ||
            upload.chunks !== chunks ||
            upload.next !== index
          )
            return fail(-1, 'upload chunk out of order');
          const body = await request.arrayBuffer();
          if (body.byteLength !== Math.min(chunkSize, size - upload.loaded)) {
            uploads.delete(uploadId);
            return fail(-1, 'invalid upload chunk length');
          }
          upload.loaded += body.byteLength;
          upload.next++;
          const complete = upload.next === chunks;
          if (complete) {
            const entries = files.get(key) || [];
            entries.push({ name, size, isDir: false, modifiedAt: new Date().toISOString() });
            files.set(key, entries);
            uploads.delete(uploadId);
          }
          return ok({ uploadId, written: upload.loaded, complete });
        }
        const file = (await request.formData()).get('file');
        if (!(file instanceof File)) return fail(-1, 'missing file');
        const key = directory(params.get('type') || 'image', params.get('rel') || '', path);
        const entries = files.get(key) || [];
        if (entries.some((entry) => entry.name === file.name))
          return fail(-2, 'file already exists');
        entries.push({
          name: file.name,
          size: file.size,
          isDir: false,
          modifiedAt: new Date().toISOString()
        });
        files.set(key, entries);
        return ok(null);
      }
    ),
    http.post(`${origin}/api/storage/media/files/upload/cancel`, async ({ request }) => {
      const body = await readJsonObject(request);
      const uploadId = readString(body, 'uploadId') || '';
      if (!/^[a-f0-9]{32}$/.test(uploadId)) return fail(-1, 'invalid upload id');
      uploads.delete(uploadId);
      return ok(null);
    }),
    http.post(`${origin}/api/storage/media/files/mkdir`, async ({ request }) => {
      const body = await readJsonObject(request);
      if (body.type === 'partition' && !state.virtualDevice.isSdCardExist)
        return fail(-10, 'initialize the virtual disk first');
      if (current() === 'partition') return fail(-12, 'eject the shared disk first');
      const path = readString(body, 'path') || '';
      if (!path || path === '/' || !validPath(path)) return fail(-1, 'invalid path');
      const parts = path.replace(/^\//, '').split('/'),
        name = parts.pop();
      const key = directory(
        readString(body, 'type') || 'image',
        readString(body, 'rel') || '',
        parts.join('/') || '/'
      );
      const entries = files.get(key);
      if (!entries) return fail(-1, 'parent directory does not exist');
      if (entries.some((entry) => entry.name === name)) return fail(-2, 'already exists');
      entries.push({ name: name!, size: 0, isDir: true, modifiedAt: new Date().toISOString() });
      return ok(null);
    }),
    http.post(`${origin}/api/storage/media/files/delete`, async ({ request }) => {
      const body = await readJsonObject(request);
      if (protectedDelete(body)) return fail(-3, 'sensitive action verification required');
      if (body.type === 'partition' && !state.virtualDevice.isSdCardExist)
        return fail(-10, 'initialize the virtual disk first');
      if (current() === 'partition') return fail(-12, 'eject the shared disk first');
      const path = readString(body, 'path') || '';
      if (!path || path === '/' || !validPath(path)) return fail(-1, 'invalid path');
      const parts = path.replace(/^\//, '').split('/'),
        name = parts.pop();
      const key = directory(
        readString(body, 'type') || 'image',
        readString(body, 'rel') || '',
        parts.join('/') || '/'
      );
      files.set(
        key,
        (files.get(key) || []).filter((entry) => entry.name !== name)
      );
      return ok(null);
    }),
    http.get(`${origin}/api/storage/media/files/download`, ({ request }) => {
      const params = new URL(request.url).searchParams;
      if (params.get('type') === 'partition' && !state.virtualDevice.isSdCardExist)
        return fail(-10, 'initialize the virtual disk first');
      const path = (params.get('path') || '').replace(/^\//, '');
      if (
        current() === 'image' &&
        params.get('type') === 'partition' &&
        rel().toLowerCase() === path.toLowerCase()
      )
        return fail(-12, 'medium is in use');
      return new HttpResponse('hello world', {
        headers: {
          'Content-Type': 'application/octet-stream',
          'Content-Disposition': 'attachment; filename="hello.txt"'
        }
      });
    }),
    http.get(`${origin}/api/storage/media/download`, ({ request }) => {
      const medium = new URL(request.url).searchParams.get('rel') || '';
      if (current() === 'image' && rel().toLowerCase() === medium.toLowerCase())
        return fail(-12, 'medium is in use');
      return new HttpResponse('mock image', {
        headers: {
          'Content-Type': 'application/octet-stream',
          'Content-Disposition': 'attachment; filename="rescue.img"'
        }
      });
    })
  ];
}
