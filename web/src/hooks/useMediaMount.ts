import { useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/media.ts';

type Options = {
  onError: (message: string, description?: string) => void;
  // Disk initialization and HID-only mode are configured in Settings > Device.
  onNeedEnableDisk?: () => void;
  // The target machine is holding the medium, so taking it back means
  // re-enumerating the device under a machine that is using it. Only the
  // caller can ask whether that is acceptable.
  onNeedForce?: (retry: () => void) => void;
  onChanged?: () => void | Promise<void>;
};

type Action = {
  // Retry as a forced operation, which the target machine cannot refuse.
  force?: boolean;
  // False once the user has already been asked.
  prompt?: boolean;
};

type MountOptions = Action & { readOnly?: boolean };

// Mounting and ejecting is the one media action both the desktop menu and the
// storage page offer.
export function useMediaMount({ onError, onNeedEnableDisk, onNeedForce, onChanged }: Options) {
  const { t } = useTranslation();
  const [busy, setBusy] = useState('');
  const running = useRef(false);

  // Every action is retried the same way: the first attempt is the polite one,
  // and a refusal the user can act on is handed to the caller to confirm.
  async function run(
    key: string,
    action: (force: boolean) => Promise<{ code: number; msg: string }>,
    opts: Action,
    retry: (opts: Action) => void
  ) {
    if (running.current) return;
    running.current = true;
    setBusy(key);

    try {
      let rsp;
      try {
        rsp = await action(opts.force ?? false);
      } catch {
        onError(t('storage.failed'));
        return;
      }

      if (rsp.code === 0) {
        await onChanged?.();
        return;
      }

      const mayPrompt = opts.prompt !== false;
      if (rsp.code === api.codeStorageDisabled && mayPrompt && onNeedEnableDisk) {
        onNeedEnableDisk();
        return;
      }
      if (rsp.code === api.codeMediumLocked && mayPrompt && onNeedForce) {
        onNeedForce(() => retry({ ...opts, force: true, prompt: false }));
        return;
      }
      if (rsp.code === api.codeNotOptical) {
        // The list the page was showing is older than the medium it names.
        onError(t('storage.add.cdromOnlyIso'));
        return;
      }

      onError(rsp.msg);
    } finally {
      running.current = false;
      setBusy('');
    }
  }

  function mount(item: api.MediaItem, cdrom = false, opts: MountOptions = {}) {
    return run(
      `${item.type}:${item.rel}`,
      (force) =>
        api.mountMedia({
          type: item.type,
          rel: item.rel,
          cdrom,
          readOnly: cdrom || (opts.readOnly ?? true),
          force
        }),
      opts,
      (next) => void mount(item, cdrom, { ...opts, ...next })
    );
  }

  function eject(opts: Action = {}) {
    return run(
      'eject',
      (force) => api.ejectMedia(force),
      opts,
      (next) => void eject(next)
    );
  }

  return { mount, eject, busy };
}
