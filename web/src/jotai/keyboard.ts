import { atom } from 'jotai';

import { getKeyboardSwapCommandCtrl } from '@/lib/localstorage.ts';

// is the keyboard enabled (Disable keyboard events when input is required)
export const isKeyboardEnableAtom = atom(true);

// is the virtual keyboard opened
export const isKeyboardOpenAtom = atom(false);

// leader key code for bypassing browser shortcuts (empty string means disabled)
export const leaderKeyAtom = atom('');

// swap Command/Meta and Ctrl modifiers for physical keyboard input
export const swapCommandCtrlAtom = atom(getKeyboardSwapCommandCtrl());
