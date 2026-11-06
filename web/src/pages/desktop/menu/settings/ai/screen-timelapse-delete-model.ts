import type { ScreenTimelapseDay } from '@/api/ai.ts';

export type ScreenTimelapseDeleteOutcome = 'success' | 'authorization_required' | 'busy' | 'failed';

/**
 * Maps the delete-day response code to the handling branch. Only -3 means
 * playback authorization is required; -5 means the day is busy (a playback
 * lease or another mutation holds it); every other non-zero code, including
 * the generic -6 deletion failure, is a plain failure.
 */
export function resolveScreenTimelapseDeleteOutcome(code: number): ScreenTimelapseDeleteOutcome {
  if (code === 0) return 'success';
  if (code === -3) return 'authorization_required';
  if (code === -5) return 'busy';
  return 'failed';
}

/**
 * Picks the day to select after a deletion. Keeps the current selection when
 * another day was deleted; otherwise falls to the day that slides into the
 * deleted position, or the last remaining day when the tail was removed.
 */
export function selectAdjacentDayAfterDeletion(
  days: ScreenTimelapseDay[],
  deletedDate: string,
  selectedDate: string
): string {
  if (selectedDate !== deletedDate) return selectedDate;

  const deletedIndex = days.findIndex((day) => day.date === deletedDate);
  const remainingDays = days.filter((day) => day.date !== deletedDate);
  const adjacentIndex = deletedIndex >= 0 ? Math.min(deletedIndex, remainingDays.length - 1) : 0;
  return remainingDays[adjacentIndex]?.date ?? '';
}
