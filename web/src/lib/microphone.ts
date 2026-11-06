export type MicrophoneErrorTranslationKey =
  | 'microphone.noDevice'
  | 'microphone.permissionDenied'
  | 'microphone.notReadable'
  | 'microphone.overconstrained'
  | 'microphone.securityBlocked'
  | 'microphone.aborted'
  | 'microphone.unsupported'
  | 'microphone.enableFailed';

export const controlMicrophoneConstraints = {
  sampleRate: 48000,
  channelCount: 2,
  echoCancellation: true,
  noiseSuppression: true,
  autoGainControl: true
};

export class MicrophoneError extends Error {
  readonly translationKey: MicrophoneErrorTranslationKey;

  constructor(translationKey: MicrophoneErrorTranslationKey, message?: string) {
    super(message ?? translationKey);
    this.name = 'MicrophoneError';
    this.translationKey = translationKey;
  }
}

function errorName(error: unknown): string {
  if (typeof error !== 'object' || error === null || !('name' in error)) {
    return '';
  }

  const name = (error as { name?: unknown }).name;
  return typeof name === 'string' ? name : '';
}

export function getMicrophoneErrorTranslationKey(error: unknown): MicrophoneErrorTranslationKey {
  if (error instanceof MicrophoneError) {
    return error.translationKey;
  }

  switch (errorName(error)) {
    case 'NotFoundError':
    case 'DevicesNotFoundError':
      return 'microphone.noDevice';
    case 'NotAllowedError':
    case 'PermissionDeniedError':
      return 'microphone.permissionDenied';
    case 'NotReadableError':
    case 'TrackStartError':
      return 'microphone.notReadable';
    case 'OverconstrainedError':
    case 'ConstraintNotSatisfiedError':
      return 'microphone.overconstrained';
    case 'SecurityError':
      return 'microphone.securityBlocked';
    case 'AbortError':
      return 'microphone.aborted';
    case 'NotSupportedError':
      return 'microphone.unsupported';
    default:
      return 'microphone.enableFailed';
  }
}

export function getUsableAudioTrack(stream: Pick<MediaStream, 'getAudioTracks'>) {
  return stream.getAudioTracks().find((track) => track.readyState !== 'ended') ?? null;
}

export function stopMediaStream(stream: Pick<MediaStream, 'getTracks'> | null) {
  if (!stream) return;

  stream.getTracks().forEach((track) => {
    track.onended = null;
    track.stop();
  });
}

/**
 * Best-effort check for a local audio input before prompting getUserMedia.
 * Returns true when enumeration is unavailable/fails so the capture path can
 * surface a more specific error (permission, unsupported, etc.).
 */
export async function hasAudioInputDevice(): Promise<boolean> {
  if (
    typeof navigator === 'undefined' ||
    !navigator.mediaDevices ||
    typeof navigator.mediaDevices.enumerateDevices !== 'function'
  ) {
    return true;
  }

  try {
    const devices = await navigator.mediaDevices.enumerateDevices();
    return devices.some((device) => device.kind === 'audioinput');
  } catch {
    return true;
  }
}

export async function requestControlMicrophone() {
  if (typeof window !== 'undefined' && window.isSecureContext === false) {
    throw new MicrophoneError('microphone.securityBlocked');
  }

  if (
    typeof navigator === 'undefined' ||
    !navigator.mediaDevices ||
    typeof navigator.mediaDevices.getUserMedia !== 'function'
  ) {
    throw new MicrophoneError('microphone.unsupported');
  }

  const stream = await navigator.mediaDevices.getUserMedia({
    audio: controlMicrophoneConstraints
  });
  const track = getUsableAudioTrack(stream);

  if (!track) {
    stopMediaStream(stream);
    throw new MicrophoneError('microphone.noDevice');
  }

  return { stream, track };
}
