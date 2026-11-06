import { atom } from 'jotai';

export type MicrophoneStatus = 'disabled' | 'enabling' | 'enabled';

export const microphoneStatusAtom = atom<MicrophoneStatus>('disabled');
