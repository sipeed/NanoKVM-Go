# Storage and file sharing

The desktop storage menu opens `#/storage` in a separate tab. The page lists
existing ISO/IMG files anywhere below `/data` and the shared folder backed by
`/exfat.img`. Existing media stay at their current paths.

The new media API is under `/api/storage/media`:

| Endpoint | Method | Purpose |
| --- | --- | --- |
| `/` | GET | Media and loop capability |
| `/state` | GET | Current medium, mode, and busy images |
| `/mount`, `/eject` | POST | Attach or release a medium |
| `/delete` | POST | Delete an image |
| `/download` | GET | Download a whole image, including HTTP ranges |
| `/files` | GET | List a folder inside a medium |
| `/files/upload`, `/files/delete` | POST | Upload a file or delete an entry |
| `/files/upload/cancel` | POST | Cancel an incomplete file upload |
| `/files/download` | GET | Download a file, including HTTP ranges |
| `/status` | GET | File upload progress |
| `/transfer` | GET | Image download progress and final result |

Media use `type: image | partition` and a path `rel` relative to `/data`.
File operations additionally use `path`, relative to the medium root.
Deletion accepts the existing `delete_image` protection challenge. Recursive
directory deletion uses the same protection because directories may contain
stored images. All routes retain the storage router's authentication.

`disk0` serves individual images. Disk mode supports read-only or read-write
mounts through `readOnly`; omitting it keeps the read-only default. CD-ROM
mode always serves images read-only. The media state reports `readOnly`, and
both mounting controls display the attached image's actual access mode.
The legacy image API retains its existing `readOnly` parameter. `disk1`
serves `/exfat.img`, making the local `/data` mount read-only. Ejecting in
the storage page leaves the LUN empty and freshly mounts `/data` writable.
The disk1 preference is retained for the next explicit mount or USB rebuild.
After the target machine ejects the shared disk, opening or polling storage
automatically returns local ownership under the storage lock. A write request
also performs this recovery before accepting data. Recovery requires the USB
ownership marker and empty disk0/disk1 LUNs; it does not remount an unrelated
read-only filesystem. The media state reports `localWritable` separately from
USB attachment, and write controls wait for confirmed local write access.
Recovery failures are reported through `localStorageError`.
Deploy the backend together with the updated kvmcomm
`/kvmcomm/scripts/mount.py`; shared-file refresh uses its new
`refresh-readonly` action.

File streams and image loop mounts hold `storageguard.Local` until closed.
USB transitions use the existing exclusive lock. Image replacement, deletion,
and file-manager opening are serialized; attached images, mounted images,
images being downloaded, and unpublished upload targets cannot be replaced
or deleted through the file manager.

The shared folder is refreshed on demand, at most once a second, by recreating
its verified read-only loop mount under the exclusive storage lock. This
handles the extra block cache between `/exfat.img` and `/data`; dropping VFS
dentries alone does not. The page polls shared files every three seconds.
An active download retains its current view until it closes. Host writeback
timing and concurrent host-side writes still affect what a listing can show.

Image uploads retain sequential 8 MiB chunks and cancellation. URL downloads
retain pause, resume, cancellation, and URL policy checks. Both accept an
optional SHA-256 checksum and verify it with bounded memory before publishing
the completed image. Existing image and download routes remain available.

File sharing also uploads sequential 8 MiB raw chunks through `/files/upload`.
Each request supplies `uploadId` (32 hex digits), `filename`, the exact file
`size`, `chunkSize=8388608`, `chunkIndex`, `totalChunks`, and the medium and
directory parameters. Empty files use one empty chunk. Each chunk streams
through a 64 KiB buffer, is synced, and requests file-cache release before the
next chunk. File and image chunks share one receive slot. The file remains
hidden until the final chunk publishes it; existing files are never replaced.
The response reports `written` (total accepted bytes) and `complete`.

The file upload retains its job slot, storage lock, and destination reservation
between chunks. Cancellation takes `{"uploadId":"..."}` and removes the
temporary file. Sessions abandoned between chunks expire after two minutes.
Closing either storage modal keeps browser uploads and progress alive;
refreshing or closing the browser page stops browser uploads. Legacy multipart
file upload requests remain accepted for clients opened before an update.

Filesystem browsing supports FAT12/16/32 and exFAT volumes starting at offset
zero. ISO 9660 and UDF are recognized as optical media for CD-ROM export;
their contents are not browsable. Loop mounts use `noexec,nosuid,nodev`, and
paths reject traversal and symlinks without requiring a newer kernel's
`MS_NOSYMFOLLOW` support.

Downloading an image through the shared-file endpoint uses the same attachment
check and read pin as a whole-image download. Hidden storage windows stop
directory and device-state polling while their browser uploads keep running;
an active URL download continues polling its transfer result. Refreshing the
file view also refreshes expanded folders without collapsing them.

Before device acceptance, exercise a real target-machine copy and safe eject,
verify fresh listings and free-space accounting after ownership switches,
check host-locked and forced ejects, and confirm HID/network/audio recovery
after USB re-enumeration.
