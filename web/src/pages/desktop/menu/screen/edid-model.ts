import type { EdidErrorReason, EdidResponse } from '@/api/vm.ts';

export const EDID_MIN_FILE_SIZE = 128;
export const EDID_MAX_FILE_SIZE = 512;
export const EDID_BLOCK_SIZE = 128;

export type EdidErrorTranslationKey =
  | 'settings.screen.edid.errors.badRequest'
  | 'settings.screen.edid.errors.invalidFileName'
  | 'settings.screen.edid.errors.fileTooLarge'
  | 'settings.screen.edid.errors.invalidLength'
  | 'settings.screen.edid.errors.invalidHeader'
  | 'settings.screen.edid.errors.unsupportedVersion'
  | 'settings.screen.edid.errors.extensionCountMismatch'
  | 'settings.screen.edid.errors.invalidChecksum'
  | 'settings.screen.edid.errors.invalidCta'
  | 'settings.screen.edid.errors.currentCannotOverwrite'
  | 'settings.screen.edid.errors.saveFailed'
  | 'settings.screen.edid.errors.switchFailed'
  | 'settings.screen.edid.errors.uploadFailed'
  | 'settings.screen.edid.errors.deleteFailed'
  | 'settings.screen.edid.errors.networkFailed'
  | 'settings.screen.edid.monitorNameUnsupported'
  | 'settings.screen.edid.monitorNameLoadFailed'
  | 'settings.screen.edid.monitorNameUpdateFailed';

const reasonTranslationKeys: Record<EdidErrorReason, EdidErrorTranslationKey> = {
  bad_request: 'settings.screen.edid.errors.badRequest',
  invalid_file_name: 'settings.screen.edid.errors.invalidFileName',
  edid_too_large: 'settings.screen.edid.errors.fileTooLarge',
  invalid_edid_length: 'settings.screen.edid.errors.invalidLength',
  invalid_edid_header: 'settings.screen.edid.errors.invalidHeader',
  unsupported_edid_version: 'settings.screen.edid.errors.unsupportedVersion',
  edid_extension_count_mismatch: 'settings.screen.edid.errors.extensionCountMismatch',
  invalid_edid_checksum: 'settings.screen.edid.errors.invalidChecksum',
  invalid_cta_extension: 'settings.screen.edid.errors.invalidCta',
  current_edid_cannot_be_overwritten: 'settings.screen.edid.errors.currentCannotOverwrite',
  save_failed: 'settings.screen.edid.errors.saveFailed'
};

const knownEdidReasons = new Set<EdidErrorReason>(
  Object.keys(reasonTranslationKeys) as EdidErrorReason[]
);

const monitorNameUnsupportedMessages = new Set([
  'monitor name descriptor not found',
  'multiple monitor name descriptors found'
]);

export function getEdidUploadPreflightError(file: {
  name: string;
  size: number;
}): EdidErrorReason | undefined {
  if (!file.name.toLowerCase().endsWith('.bin')) {
    return 'invalid_file_name';
  }
  if (file.size > EDID_MAX_FILE_SIZE) {
    return 'edid_too_large';
  }
  if (file.size < EDID_MIN_FILE_SIZE || file.size % EDID_BLOCK_SIZE !== 0) {
    return 'invalid_edid_length';
  }

  return undefined;
}

export function getEdidResponseReason(response: EdidResponse): EdidErrorReason | undefined {
  const reason = response.data?.reason;
  return typeof reason === 'string' && knownEdidReasons.has(reason as EdidErrorReason)
    ? (reason as EdidErrorReason)
    : undefined;
}

export function getEdidErrorTranslationKey(
  reason: EdidErrorReason | undefined,
  fallback: EdidErrorTranslationKey
): EdidErrorTranslationKey {
  return reason ? reasonTranslationKeys[reason] : fallback;
}

export function getEdidResponseErrorTranslationKey(
  response: EdidResponse,
  fallback: EdidErrorTranslationKey
): EdidErrorTranslationKey {
  if (monitorNameUnsupportedMessages.has(response.msg)) {
    return 'settings.screen.edid.monitorNameUnsupported';
  }

  return getEdidErrorTranslationKey(getEdidResponseReason(response), fallback);
}

type EdidOperationOptions = {
  request: () => Promise<EdidResponse>;
  setLoading: (loading: boolean) => void;
  onError: (key: EdidErrorTranslationKey) => void;
  onSuccess?: () => void;
};

type EdidSwitchOptions = EdidOperationOptions & {
  refreshCurrent: () => void | Promise<void>;
};

export async function runEdidSwitch(options: EdidSwitchOptions): Promise<boolean> {
  options.setLoading(true);

  try {
    const response = await options.request();
    if (response.code !== 0) {
      options.onError(
        getEdidResponseErrorTranslationKey(response, 'settings.screen.edid.errors.switchFailed')
      );
      return false;
    }

    options.onSuccess?.();
    return true;
  } catch {
    options.onError('settings.screen.edid.errors.networkFailed');
    return false;
  } finally {
    try {
      await options.refreshCurrent();
    } catch {
      options.onError('settings.screen.edid.errors.networkFailed');
    } finally {
      options.setLoading(false);
    }
  }
}

type EdidUploadOptions = Omit<EdidOperationOptions, 'request'> & {
  file: { name: string; size: number };
  request: () => Promise<EdidResponse<{ file: string }>>;
  refreshCustomList: () => void | Promise<void>;
};

export async function runEdidUpload(options: EdidUploadOptions): Promise<boolean> {
  const preflightError = getEdidUploadPreflightError(options.file);
  if (preflightError) {
    options.onError(
      getEdidErrorTranslationKey(preflightError, 'settings.screen.edid.errors.uploadFailed')
    );
    return false;
  }

  options.setLoading(true);

  try {
    const response = await options.request();
    if (response.code !== 0) {
      options.onError(
        getEdidResponseErrorTranslationKey(response, 'settings.screen.edid.errors.uploadFailed')
      );
      return false;
    }

    await options.refreshCustomList();
    options.onSuccess?.();
    return true;
  } catch {
    options.onError('settings.screen.edid.errors.networkFailed');
    return false;
  } finally {
    options.setLoading(false);
  }
}
