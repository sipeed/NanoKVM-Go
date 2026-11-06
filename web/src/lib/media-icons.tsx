import clsx from 'clsx';
import {
  DiscIcon,
  FileArchiveIcon,
  FileAudioIcon,
  FileIcon,
  FileImageIcon,
  FileTextIcon,
  FileVideoIcon,
  FolderSyncIcon,
  HardDriveIcon,
  type LucideIcon
} from 'lucide-react';

import { isOpticalImage, type MediaItem } from '@/api/media.ts';
import { extension } from '@/lib/file-names.ts';

// The glyphs the storage page and the desktop menu draw for media and for the
// files inside them. They live together so one thing cannot look like two: the
// shared folder is deliberately not a disk, because it is a directory this
// device exchanges with the target machine while an .img is a disk image the
// emulated USB disk carries, and the two used to share one icon.
export function MediaIcon({
  item,
  className,
  size = 14
}: {
  item: MediaItem;
  className?: string;
  size?: number;
}) {
  if (item.type === 'partition') {
    return <FolderSyncIcon size={size} className={className} />;
  }
  if (isOpticalImage(item)) {
    return <DiscIcon size={size} className={className} />;
  }
  return <HardDriveIcon size={size} className={className} />;
}

// The shared folder carries whatever the two machines exchange, and a listing
// is read at a glance: an archive must not look like a movie or a picture.
// These are the kinds a shared folder mostly holds, and a file whose kind is
// not here is a payload rather than a mistake, so it keeps the plain file
// glyph. A .ts file is read as a document, not as an MPEG transport stream.
const archiveExtensions = new Set([
  '7z',
  'bz2',
  'cab',
  'deb',
  'gz',
  'jar',
  'rar',
  'rpm',
  'tar',
  'tgz',
  'txz',
  'xz',
  'zip',
  'zst'
]);
const documentExtensions = new Set([
  'c',
  'cfg',
  'conf',
  'cpp',
  'css',
  'csv',
  'doc',
  'docx',
  'go',
  'h',
  'hpp',
  'html',
  'ini',
  'java',
  'js',
  'json',
  'jsx',
  'log',
  'markdown',
  'md',
  'odt',
  'pdf',
  'php',
  'ppt',
  'pptx',
  'py',
  'rb',
  'rs',
  'sh',
  'sql',
  'toml',
  'ts',
  'tsx',
  'txt',
  'xls',
  'xlsx',
  'xml',
  'yaml',
  'yml'
]);
const videoExtensions = new Set([
  'avi',
  'flv',
  'm4v',
  'mkv',
  'mov',
  'mp4',
  'mpeg',
  'mpg',
  'ogv',
  'webm',
  'wmv'
]);
const imageExtensions = new Set([
  'avif',
  'bmp',
  'gif',
  'heic',
  'heif',
  'ico',
  'jpeg',
  'jpg',
  'png',
  'svg',
  'tif',
  'tiff',
  'webp'
]);
const audioExtensions = new Set(['aac', 'flac', 'm4a', 'mp3', 'oga', 'ogg', 'opus', 'wav', 'wma']);

// The colour each kind carries in a listing: at a glance an archive must not
// read like a movie or a picture. The tones stay muted so a folder full of
// mixed files does not turn into a rainbow.
type FileGlyph = { Icon: LucideIcon; tone: string };

function fileGlyph(name: string): FileGlyph {
  const ext = extension(name);
  if (ext === 'img') return { Icon: HardDriveIcon, tone: 'text-neutral-400' };
  if (ext === 'iso') return { Icon: DiscIcon, tone: 'text-neutral-400' };
  if (archiveExtensions.has(ext)) return { Icon: FileArchiveIcon, tone: 'text-amber-400' };
  if (videoExtensions.has(ext)) return { Icon: FileVideoIcon, tone: 'text-violet-400' };
  if (imageExtensions.has(ext)) return { Icon: FileImageIcon, tone: 'text-emerald-400' };
  if (audioExtensions.has(ext)) return { Icon: FileAudioIcon, tone: 'text-rose-400' };
  if (documentExtensions.has(ext)) return { Icon: FileTextIcon, tone: 'text-sky-400' };
  return { Icon: FileIcon, tone: 'text-neutral-400' };
}

// FileTypeIcon draws one entry of a listing, at the size and colour its rows
// use. Directories are not its business: the caller draws the folder glyph.
export function FileTypeIcon({
  name,
  size = 14,
  className
}: {
  name: string;
  size?: number;
  className?: string;
}) {
  const { Icon, tone } = fileGlyph(name);
  return <Icon size={size} className={clsx('shrink-0', className ?? tone)} />;
}
