function isAutoplayBlocked(error: unknown) {
  return (
    typeof error === 'object' &&
    error !== null &&
    'name' in error &&
    error.name === 'NotAllowedError'
  );
}

export type AudioPlaybackResult = 'playing' | 'muted' | 'not-ready' | 'blocked' | 'stale' | 'error';

export function resolveVolumeAfterAutoplayBlock(currentVolume: number, attemptedVolume: number) {
  return currentVolume === attemptedVolume ? 0 : currentVolume;
}

const synchronizationGenerations = new WeakMap<HTMLAudioElement, number>();

function startSynchronization(audioElement: HTMLAudioElement) {
  const generation = (synchronizationGenerations.get(audioElement) ?? 0) + 1;
  synchronizationGenerations.set(audioElement, generation);
  return generation;
}

export async function synchronizeAudioVolume(
  audioElement: HTMLAudioElement | null,
  volume: number
): Promise<AudioPlaybackResult> {
  if (!audioElement) return 'not-ready';

  const generation = startSynchronization(audioElement);
  audioElement.volume = volume / 100;

  if (volume === 0) {
    audioElement.muted = true;
    return 'muted';
  }

  if (!audioElement.srcObject) {
    return 'not-ready';
  }

  audioElement.muted = false;

  try {
    await audioElement.play();
    return synchronizationGenerations.get(audioElement) === generation ? 'playing' : 'stale';
  } catch (error) {
    if (synchronizationGenerations.get(audioElement) !== generation) {
      return 'stale';
    }

    if (!isAutoplayBlocked(error)) {
      console.error(error);
      return 'error';
    }

    audioElement.muted = true;
    return 'blocked';
  }
}

export async function attachAudioStream(
  audioElement: HTMLAudioElement | null,
  stream: MediaStream,
  volume: number
): Promise<AudioPlaybackResult> {
  if (!audioElement) return 'not-ready';

  audioElement.srcObject = stream;
  return synchronizeAudioVolume(audioElement, volume);
}

export function disposeAudioVolumeSynchronization(audioElement: HTMLAudioElement | null) {
  if (!audioElement) return;

  startSynchronization(audioElement);
}
