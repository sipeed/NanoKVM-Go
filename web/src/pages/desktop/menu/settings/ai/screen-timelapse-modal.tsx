import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type {
  ChangeEvent,
  ReactNode,
  PointerEvent as ReactPointerEvent,
  SyntheticEvent
} from 'react';
import { Button, Divider, Empty, Input, Modal, Spin } from 'antd';
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  FilmIcon,
  HistoryIcon,
  Loader2Icon,
  MaximizeIcon,
  MinimizeIcon,
  PauseIcon,
  PlayIcon,
  SearchIcon,
  SearchXIcon,
  XIcon
} from 'lucide-react';
import mpegts from 'mpegts.js';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/ai.ts';
import type {
  ScreenTimelapseDay,
  ScreenTimelapseFrame,
  ScreenTimelapseSearchResult
} from '@/api/ai.ts';
import { getIntlLocale } from '@/i18n/locale.ts';
import { feedback } from '@/lib/feedback.tsx';

import {
  resolveScreenTimelapseDeleteOutcome,
  selectAdjacentDayAfterDeletion,
  type ScreenTimelapseDeleteOutcome
} from './screen-timelapse-delete-model.ts';

type ScreenTimelapseDaysResponse = {
  days?: ScreenTimelapseDay[];
};

type ScreenTimelapseFramesResponse = {
  date?: string;
  durationMs?: number;
  frameCount?: number;
  frames?: ScreenTimelapseFrame[];
};

type ScreenTimelapseSearchResultsResponse = {
  results?: ScreenTimelapseSearchResult[];
};

type TimelapsePlayer = ReturnType<typeof mpegts.createPlayer>;

type ScreenTimelapseModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  authorizationVersion: number;
  onAuthorizationRequired: (onAuthorized?: () => void) => void;
};

const MPEG_TS_CLOCK_HZ = 90000;
const PROGRESS_END_SNAP_SECONDS = 0.05;
const DAYS_REFRESH_INTERVAL_MS = 60_000;
const DELETE_BUSY_RETRY_DELAY_MS = 300;

function formatClock(seconds: number) {
  const normalizedSeconds = Math.max(0, Math.floor(seconds));
  const hours = Math.floor(normalizedSeconds / 3600);
  const minutes = Math.floor((normalizedSeconds % 3600) / 60);
  const secs = normalizedSeconds % 60;

  if (hours > 0) {
    return `${hours}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
  }

  return `${minutes}:${String(secs).padStart(2, '0')}`;
}

function snapToFrame(frames: ScreenTimelapseFrame[], seconds: number) {
  if (frames.length === 0) return Math.max(0, seconds);

  let left = 0;
  let right = frames.length - 1;

  while (left < right) {
    const mid = Math.floor((left + right) / 2);
    if (frames[mid].currentTime < seconds) {
      left = mid + 1;
    } else {
      right = mid;
    }
  }

  const current = frames[left];
  const previous = frames[left - 1];
  if (!previous) return current.currentTime;

  return Math.abs(previous.currentTime - seconds) <= Math.abs(current.currentTime - seconds)
    ? previous.currentTime
    : current.currentTime;
}

function getResultSeconds(result: ScreenTimelapseSearchResult) {
  if (typeof result.currentTime === 'number') return result.currentTime;
  if (typeof result.pts === 'number') return result.pts / MPEG_TS_CLOCK_HZ;
  return 0;
}

function getSearchResultSortTime(result: ScreenTimelapseSearchResult) {
  const matchedFrameTime = Date.parse(result.matchedFrameTime || '');
  if (Number.isFinite(matchedFrameTime)) return matchedFrameTime;

  const capturedTime = Date.parse(result.capturedAt);
  if (Number.isFinite(capturedTime)) return capturedTime;

  const dateTime = Date.parse(`${result.date}T00:00:00`);
  if (Number.isFinite(dateTime)) return dateTime + getResultSeconds(result) * 1000;

  return 0;
}

function sortSearchResultsByTimeDesc(results: ScreenTimelapseSearchResult[]) {
  return [...results].sort((a, b) => getSearchResultSortTime(b) - getSearchResultSortTime(a));
}

function buildSearchContext(text: string, query: string, locale: string) {
  const content = text.trim();
  const keyword = query.trim();
  if (!content || !keyword) return content;

  const lowerKeyword = keyword.toLocaleLowerCase(locale);
  const matchedLine = content
    .split(/\r?\n/)
    .map((line) => line.trim())
    .find((line) => line.toLocaleLowerCase(locale).includes(lowerKeyword));

  return matchedLine || content;
}

function renderHighlightedText(text: string, query: string, locale: string): ReactNode {
  const keyword = query.trim();
  if (!keyword) return text;

  const lowerText = text.toLocaleLowerCase(locale);
  const lowerKeyword = keyword.toLocaleLowerCase(locale);
  const parts: ReactNode[] = [];
  let cursor = 0;

  while (cursor < text.length) {
    const matchIndex = lowerText.indexOf(lowerKeyword, cursor);
    if (matchIndex < 0) {
      parts.push(text.slice(cursor));
      break;
    }

    if (matchIndex > cursor) {
      parts.push(text.slice(cursor, matchIndex));
    }

    const matchEnd = matchIndex + keyword.length;
    parts.push(
      <mark key={matchIndex} className="rounded bg-transparent px-0.5 font-semibold text-red-300">
        {text.slice(matchIndex, matchEnd)}
      </mark>
    );
    cursor = matchEnd;
  }

  return parts.length > 0 ? parts : text;
}

function formatDayLabel(date: string, locale: string) {
  const value = new Date(`${date}T00:00:00`);
  if (Number.isNaN(value.getTime())) return date;

  return value.toLocaleDateString(locale, {
    month: 'short',
    day: 'numeric'
  });
}

function formatBytes(bytes: number, locale: string) {
  const normalizedBytes = Number.isFinite(bytes) ? Math.max(0, bytes) : 0;
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let value = normalizedBytes;
  let unitIndex = 0;

  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }

  return `${new Intl.NumberFormat(locale, {
    maximumFractionDigits: unitIndex === 0 ? 0 : 1
  }).format(value)} ${units[unitIndex]}`;
}

function toAbsoluteURL(url: string) {
  return new URL(url, window.location.origin).toString();
}

function buildPlaybackURL(url: string, startSeconds: number) {
  const targetURL = new URL(toAbsoluteURL(url));
  if (startSeconds > 0) {
    targetURL.searchParams.set('start', startSeconds.toFixed(3));
  }
  return targetURL.toString();
}

function isMediaTimeBuffered(video: HTMLVideoElement, seconds: number) {
  if (!Number.isFinite(seconds) || seconds < 0) return false;

  for (let index = 0; index < video.buffered.length; index += 1) {
    if (seconds >= video.buffered.start(index) && seconds <= video.buffered.end(index)) {
      return true;
    }
  }
  return false;
}

function normalizeProgressTime(seconds: number, durationSeconds: number) {
  if (!Number.isFinite(seconds) || durationSeconds <= 0) return 0;

  const clampedSeconds = Math.min(Math.max(0, seconds), durationSeconds);
  if (durationSeconds - clampedSeconds <= PROGRESS_END_SNAP_SECONDS) {
    return durationSeconds;
  }

  return clampedSeconds;
}

export const ScreenTimelapseModal = ({
  open,
  onOpenChange,
  authorizationVersion,
  onAuthorizationRequired
}: ScreenTimelapseModalProps) => {
  const { t, i18n } = useTranslation();
  const [modal, contextHolder] = Modal.useModal();
  const intlLocale = getIntlLocale(i18n.language);
  const videoRef = useRef<HTMLVideoElement>(null);
  const playerShellRef = useRef<HTMLDivElement>(null);
  const playerRef = useRef<TimelapsePlayer>();
  const timelineRef = useRef<HTMLDivElement>(null);
  const scrubTimeRef = useRef<number>();
  const isScrubbingRef = useRef(false);
  const isSeekReloadingRef = useRef(false);
  const activeSeekReloadIDRef = useRef<number>();
  const pendingPlayerSeekRef = useRef<{ seconds: number; startedAt: number }>();
  const pendingAutoplayRef = useRef<number>();
  const shouldAutoplayRef = useRef(false);
  const suppressNextDateAutoplayRef = useRef(false);
  const playbackReloadIDRef = useRef(0);
  const playerSourceDateRef = useRef('');
  const latestFramesDateRef = useRef('');
  const latestDaysRequestRef = useRef(0);
  const latestFramesRequestRef = useRef(0);
  const latestSearchRequestRef = useRef(0);
  const isModalOpenRef = useRef(open);
  const deletingDateRef = useRef('');
  const daysRef = useRef<ScreenTimelapseDay[]>([]);
  const selectedDateRef = useRef('');

  const isModalOpen = open;
  const [isLoadingDays, setIsLoadingDays] = useState(false);
  const [isLoadingFrames, setIsLoadingFrames] = useState(false);
  const [isSearching, setIsSearching] = useState(false);
  const [deletingDate, setDeletingDate] = useState('');
  const [isPlaying, setIsPlaying] = useState(false);
  const [isPlayerReady, setIsPlayerReady] = useState(false);
  const [isPlayerSupported, setIsPlayerSupported] = useState(true);
  const [hasPlayerError, setHasPlayerError] = useState(false);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [days, setDays] = useState<ScreenTimelapseDay[]>([]);
  const [frames, setFrames] = useState<ScreenTimelapseFrame[]>([]);
  const [framesDate, setFramesDate] = useState('');
  const [framesDurationMs, setFramesDurationMs] = useState(0);
  const [selectedDate, setSelectedDate] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<ScreenTimelapseSearchResult[]>([]);
  const [hasSearched, setHasSearched] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [scrubTime, setScrubTime] = useState<number>();
  const [isSeekReloading, setIsSeekReloading] = useState(false);
  const [playbackStartDate, setPlaybackStartDate] = useState('');
  const [playbackStartSeconds, setPlaybackStartSeconds] = useState(0);
  const [playbackReloadID, setPlaybackReloadID] = useState(0);
  const [pendingSeek, setPendingSeek] = useState<{ date: string; seconds: number }>();

  const replaceDays = useCallback((nextDays: ScreenTimelapseDay[]) => {
    daysRef.current = nextDays;
    setDays(nextDays);
  }, []);

  const selectDate = useCallback((date: string) => {
    selectedDateRef.current = date;
    setSelectedDate(date);
  }, []);

  const selectedDay = useMemo(() => {
    return days.find((day) => day.date === selectedDate);
  }, [days, selectedDate]);

  const playbackDay = useMemo<ScreenTimelapseDay | undefined>(() => {
    if (selectedDay) {
      return {
        ...selectedDay,
        durationMs:
          framesDate === selectedDate && framesDurationMs > 0
            ? framesDurationMs
            : selectedDay.durationMs
      };
    }
    if (!selectedDate || framesDate !== selectedDate || frames.length === 0) return undefined;

    return {
      date: selectedDate,
      videoUrl: `/api/ai/screen-timelapse/video?date=${encodeURIComponent(selectedDate)}`,
      startTime: frames[0].capturedAt,
      endTime: frames[frames.length - 1].capturedAt,
      durationMs: framesDurationMs,
      frameCount: frames.length,
      filesize: 0,
      deletable: false
    };
  }, [frames, framesDate, framesDurationMs, selectedDate, selectedDay]);

  const timelineDays = useMemo(() => {
    if (!playbackDay || days.some((day) => day.date === playbackDay.date)) return days;
    return [playbackDay, ...days].sort((a, b) => {
      if (a.date === b.date) return 0;
      return a.date < b.date ? 1 : -1;
    });
  }, [days, playbackDay]);

  const selectedTimelineIndex = useMemo(() => {
    return timelineDays.findIndex((day) => day.date === selectedDate);
  }, [selectedDate, timelineDays]);

  const hasPreviousTimelineDay = selectedTimelineIndex > 0;
  const hasNextTimelineDay =
    selectedTimelineIndex >= 0 && selectedTimelineIndex < timelineDays.length - 1;

  const durationSeconds = useMemo(() => {
    if (!playbackDay) return 0;
    const durationMs =
      framesDate === selectedDate && framesDurationMs > 0
        ? framesDurationMs
        : playbackDay.durationMs;
    return Math.max(0, durationMs / 1000);
  }, [framesDate, framesDurationMs, playbackDay, selectedDate]);

  const snapPlaybackTime = useCallback(
    (seconds: number) => {
      const normalizedSeconds = Number.isFinite(seconds) ? Math.max(0, seconds) : 0;
      return Math.min(durationSeconds, snapToFrame(frames, normalizedSeconds));
    },
    [durationSeconds, frames]
  );

  const updateSeekReloading = useCallback((reloading: boolean) => {
    isSeekReloadingRef.current = reloading;
    setIsSeekReloading(reloading);
  }, []);

  const seekToSeconds = useCallback(
    (seconds: number) => {
      const snappedSeconds = snapPlaybackTime(seconds);
      const currentSourceOffset = playbackStartDate === selectedDate ? playbackStartSeconds : 0;
      const targetMediaSeconds = snappedSeconds - currentSourceOffset;
      const video = videoRef.current;

      if (
        video &&
        playerSourceDateRef.current === selectedDate &&
        isMediaTimeBuffered(video, targetMediaSeconds)
      ) {
        activeSeekReloadIDRef.current = undefined;
        pendingAutoplayRef.current = undefined;
        pendingPlayerSeekRef.current = {
          seconds: snappedSeconds,
          startedAt: Date.now()
        };
        updateSeekReloading(false);
        setCurrentTime(snappedSeconds);

        try {
          video.currentTime = targetMediaSeconds;
          if (isPlaying && video.paused) {
            const playPromise = playerRef.current?.play();
            void Promise.resolve(playPromise).catch(() => {
              if (isModalOpenRef.current) {
                setIsPlaying(false);
              }
            });
          }
          return;
        } catch {
          pendingPlayerSeekRef.current = undefined;
        }
      }

      const nextReloadID = playbackReloadIDRef.current + 1;
      playbackReloadIDRef.current = nextReloadID;
      pendingAutoplayRef.current = isPlaying ? nextReloadID : undefined;
      activeSeekReloadIDRef.current = nextReloadID;
      updateSeekReloading(true);
      pendingPlayerSeekRef.current = {
        seconds: snappedSeconds,
        startedAt: Date.now()
      };
      if (!isPlaying) {
        setIsPlaying(false);
      }
      setIsPlayerReady(false);
      setCurrentTime(snappedSeconds);
      setPlaybackStartDate(selectedDate);
      setPlaybackStartSeconds(snappedSeconds);
      setPlaybackReloadID(nextReloadID);
    },
    [
      isPlaying,
      playbackStartDate,
      playbackStartSeconds,
      selectedDate,
      snapPlaybackTime,
      updateSeekReloading
    ]
  );

  const displayedProgressTime =
    typeof scrubTime === 'number'
      ? Math.min(Math.max(0, scrubTime), durationSeconds)
      : normalizeProgressTime(currentTime, durationSeconds);
  const playbackStartOffsetSeconds = playbackStartDate === selectedDate ? playbackStartSeconds : 0;

  useEffect(() => {
    isModalOpenRef.current = open;
    if (!open) return;

    shouldAutoplayRef.current = true;
    pendingAutoplayRef.current = playbackReloadIDRef.current;
  }, [open]);

  const stopPlaybackForRemovedDay = useCallback(() => {
    shouldAutoplayRef.current = false;
    suppressNextDateAutoplayRef.current = true;
    pendingAutoplayRef.current = undefined;
    activeSeekReloadIDRef.current = undefined;
    pendingPlayerSeekRef.current = undefined;
    playerRef.current?.pause();
    latestFramesRequestRef.current += 1;
    latestFramesDateRef.current = '';
    updateSeekReloading(false);
    setIsPlaying(false);
    setIsPlayerReady(false);
    setHasPlayerError(false);
    setPendingSeek(undefined);
    setFrames([]);
    setFramesDate('');
    setFramesDurationMs(0);
    setCurrentTime(0);
    setScrubTime(undefined);
    setPlaybackStartDate('');
    setPlaybackStartSeconds(0);
  }, [updateSeekReloading]);

  const loadDays = useCallback(
    async (silent = false) => {
      const requestID = latestDaysRequestRef.current + 1;
      latestDaysRequestRef.current = requestID;
      if (!silent) setIsLoadingDays(true);

      try {
        const rsp = await api.getScreenTimelapseDays();
        if (!isModalOpenRef.current || latestDaysRequestRef.current !== requestID) return;

        if (rsp.code === -3) {
          onAuthorizationRequired();
          return;
        }
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          if (!silent) feedback.error(t('settings.ai.errors.loadScreenTimelapseRecordings'));
          return;
        }

        const data = (rsp.data ?? {}) as ScreenTimelapseDaysResponse;
        const nextDays = Array.isArray(data.days) ? data.days : [];
        const previousDays = daysRef.current;
        const removedDates = new Set(
          previousDays
            .filter((day) => !nextDays.some((nextDay) => nextDay.date === day.date))
            .map((day) => day.date)
        );
        const currentDate = selectedDateRef.current;
        const selectedDayRemoved = Boolean(currentDate) && removedDates.has(currentDate);

        replaceDays(nextDays);
        if (removedDates.size > 0) {
          setSearchResults((currentResults) =>
            currentResults.map((result) =>
              result.available && removedDates.has(result.date)
                ? { ...result, available: false, unavailableReason: 'missing_video' }
                : result
            )
          );
        }

        if (selectedDayRemoved) {
          const removedIndex = previousDays.findIndex((day) => day.date === currentDate);
          const replacementDate =
            nextDays[Math.min(Math.max(0, removedIndex), Math.max(0, nextDays.length - 1))]?.date ||
            '';
          stopPlaybackForRemovedDay();
          selectDate(replacementDate);
          feedback.warning(t('settings.ai.screenTimelapseRecordingCleaned'));
        } else if (!currentDate) {
          selectDate(nextDays[0]?.date || '');
        }
      } catch {
        if (!isModalOpenRef.current || latestDaysRequestRef.current !== requestID) return;
        if (!silent) feedback.error(t('settings.ai.errors.loadScreenTimelapseRecordings'));
      } finally {
        if (isModalOpenRef.current && latestDaysRequestRef.current === requestID) {
          setIsLoadingDays(false);
        }
      }
    },
    [onAuthorizationRequired, replaceDays, selectDate, stopPlaybackForRemovedDay, t]
  );

  const performDeleteDay = useCallback(
    async (day: ScreenTimelapseDay) => {
      if (deletingDateRef.current) return;

      deletingDateRef.current = day.date;
      setDeletingDate(day.date);
      let outcome: ScreenTimelapseDeleteOutcome | undefined;

      try {
        let rsp: Awaited<ReturnType<typeof api.deleteScreenTimelapseDay>>;
        try {
          rsp = await api.deleteScreenTimelapseDay(day.date);
        } catch (error) {
          console.error(error);
          feedback.error(t('settings.ai.screenTimelapseDeleteFailed', { date: day.date }));
          return;
        }

        outcome = resolveScreenTimelapseDeleteOutcome(rsp.code);

        let leaseReleased = false;
        let resumePlaybackAfterRebuild = false;

        // A busy day whose stream is playing locally holds the server-side
        // playback lease; destroying the player releases it, then the deletion
        // is retried once while the single-flight lock is still held.
        if (outcome === 'busy' && playerRef.current && playerSourceDateRef.current === day.date) {
          resumePlaybackAfterRebuild = isPlaying;
          const player = playerRef.current;
          playerRef.current = undefined;
          playerSourceDateRef.current = '';
          shouldAutoplayRef.current = false;
          pendingAutoplayRef.current = undefined;
          setIsPlaying(false);
          setIsPlayerReady(false);
          try {
            player.pause();
            player.unload();
            player.detachMediaElement();
            player.destroy();
          } catch {
            // best effort: the lease is released once the stream aborts
          }
          leaseReleased = true;

          await new Promise<void>((resolve) => {
            window.setTimeout(resolve, DELETE_BUSY_RETRY_DELAY_MS);
          });

          try {
            rsp = await api.deleteScreenTimelapseDay(day.date);
            outcome = resolveScreenTimelapseDeleteOutcome(rsp.code);
          } catch (error) {
            console.error(error);
            outcome = 'failed';
          }
        }

        if (leaseReleased && outcome !== 'success') {
          // The deletion did not happen; rebuild the player to restore playback.
          const nextReloadID = playbackReloadIDRef.current + 1;
          playbackReloadIDRef.current = nextReloadID;
          pendingAutoplayRef.current = resumePlaybackAfterRebuild ? nextReloadID : undefined;
          setPlaybackReloadID(nextReloadID);
        }

        if (outcome === 'success') {
          const currentDays = daysRef.current;
          const currentSelectedDate = selectedDateRef.current;
          const deletingSelectedDay = currentSelectedDate === day.date;
          const remainingDays = currentDays.filter((candidate) => candidate.date !== day.date);
          const nextSelectedDate = selectAdjacentDayAfterDeletion(
            currentDays,
            day.date,
            currentSelectedDate
          );

          if (deletingSelectedDay) {
            latestFramesRequestRef.current += 1;
            latestFramesDateRef.current = '';
            setPendingSeek(undefined);
          } else {
            setPendingSeek((currentSeek) =>
              currentSeek?.date === day.date ? undefined : currentSeek
            );
          }
          setSearchResults((currentResults) =>
            currentResults.map((result) =>
              result.date === day.date
                ? { ...result, available: false, unavailableReason: 'missing_video' }
                : result
            )
          );
          replaceDays(remainingDays);
          selectDate(nextSelectedDate);
          if (deletingSelectedDay) {
            stopPlaybackForRemovedDay();
          }
          feedback.success(t('settings.ai.screenTimelapseDeleteSuccess', { date: day.date }));
          void loadDays(true);
        } else if (outcome !== 'authorization_required') {
          console.log(rsp.msg);
          feedback.error(t('settings.ai.screenTimelapseDeleteFailed', { date: day.date }));
        }
      } finally {
        deletingDateRef.current = '';
        setDeletingDate('');
      }

      // The gate is entered only after the single-flight lock is released, so
      // a retry is never blocked by ourselves. Reached by falling out of the
      // try block, not by returning early. The continuation lives only inside
      // the gate's pending execute closure: cancelling the 2FA modal drops it.
      if (outcome === 'authorization_required') {
        onAuthorizationRequired(() => {
          if (!isModalOpenRef.current) return;
          const retryDay = daysRef.current.find((candidate) => candidate.date === day.date);
          if (!retryDay) return;
          if (!retryDay.deletable) {
            feedback.error(t('settings.ai.screenTimelapseDeleteFailed', { date: day.date }));
            return;
          }
          // The user already confirmed this deletion; retry it directly.
          void performDeleteDay(retryDay);
        });
      }
    },
    [
      isPlaying,
      loadDays,
      onAuthorizationRequired,
      replaceDays,
      selectDate,
      stopPlaybackForRemovedDay,
      t
    ]
  );

  const loadFrames = useCallback(
    (date: string) => {
      const requestID = latestFramesRequestRef.current + 1;
      latestFramesRequestRef.current = requestID;

      if (!date) {
        latestFramesDateRef.current = '';
        setFrames([]);
        setFramesDate('');
        setFramesDurationMs(0);
        setIsLoadingFrames(false);
        return;
      }

      latestFramesDateRef.current = date;
      setIsLoadingFrames(true);
      setFrames([]);
      setFramesDate('');
      setFramesDurationMs(0);
      api
        .getScreenTimelapseFrames(date)
        .then((rsp) => {
          if (
            !isModalOpenRef.current ||
            latestFramesRequestRef.current !== requestID ||
            latestFramesDateRef.current !== date
          ) {
            return;
          }

          if (rsp.code === -3) {
            onAuthorizationRequired();
            return;
          }
          if (rsp.code !== 0) {
            console.log(rsp.msg);
            feedback.error(t('settings.ai.errors.loadScreenTimelapseRecordings'));
            return;
          }

          const data = (rsp.data ?? {}) as ScreenTimelapseFramesResponse;
          setFrames(Array.isArray(data.frames) ? data.frames : []);
          setFramesDate(data.date || date);
          setFramesDurationMs(data.durationMs ?? 0);
        })
        .catch(() => {
          if (
            !isModalOpenRef.current ||
            latestFramesRequestRef.current !== requestID ||
            latestFramesDateRef.current !== date
          ) {
            return;
          }
          feedback.error(t('settings.ai.errors.loadScreenTimelapseRecordings'));
        })
        .finally(() => {
          if (
            !isModalOpenRef.current ||
            latestFramesRequestRef.current !== requestID ||
            latestFramesDateRef.current !== date
          ) {
            return;
          }
          setIsLoadingFrames(false);
        });
    },
    [onAuthorizationRequired, t]
  );

  useEffect(() => {
    if (!isModalOpen) return;
    void loadDays();
  }, [authorizationVersion, isModalOpen, loadDays]);

  useEffect(() => {
    if (!isModalOpen) return;

    const refreshDays = () => void loadDays(true);
    const refreshDaysWhenVisible = () => {
      if (document.visibilityState === 'visible') refreshDays();
    };
    const timer = window.setInterval(refreshDays, DAYS_REFRESH_INTERVAL_MS);
    window.addEventListener('focus', refreshDays);
    document.addEventListener('visibilitychange', refreshDaysWhenVisible);

    return () => {
      window.clearInterval(timer);
      window.removeEventListener('focus', refreshDays);
      document.removeEventListener('visibilitychange', refreshDaysWhenVisible);
    };
  }, [authorizationVersion, isModalOpen, loadDays]);

  useEffect(() => {
    if (!isModalOpen || !selectedDate) return;

    timelineRef.current
      ?.querySelector<HTMLElement>('[data-selected="true"]')
      ?.scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'center' });
  }, [isModalOpen, selectedDate, timelineDays.length]);

  useEffect(() => {
    const syncFullscreenState = () => {
      setIsFullscreen(document.fullscreenElement === playerShellRef.current);
    };

    document.addEventListener('fullscreenchange', syncFullscreenState);
    return () => {
      document.removeEventListener('fullscreenchange', syncFullscreenState);
    };
  }, []);

  useEffect(() => {
    if (!isModalOpen) return;
    const suppressAutoplay = suppressNextDateAutoplayRef.current;
    suppressNextDateAutoplayRef.current = false;
    setIsPlaying(false);
    setIsPlayerReady(false);
    scrubTimeRef.current = undefined;
    isScrubbingRef.current = false;
    activeSeekReloadIDRef.current = undefined;
    pendingPlayerSeekRef.current = undefined;
    playbackReloadIDRef.current = 0;
    shouldAutoplayRef.current = Boolean(selectedDate) && !suppressAutoplay;
    pendingAutoplayRef.current = shouldAutoplayRef.current
      ? playbackReloadIDRef.current
      : undefined;
    playerSourceDateRef.current = '';
    updateSeekReloading(false);
    setScrubTime(undefined);
    setCurrentTime(0);
    setPlaybackStartDate('');
    setPlaybackStartSeconds(0);
    setPlaybackReloadID(0);
    loadFrames(selectedDate);
  }, [authorizationVersion, isModalOpen, loadFrames, selectedDate, updateSeekReloading]);

  useEffect(() => {
    const videoElement = videoRef.current;
    const playback = playbackDay;
    if (!isModalOpen || !playback || !videoElement) return;
    const activePlayback: ScreenTimelapseDay = playback;
    const activeVideo: HTMLVideoElement = videoElement;

    if (activeSeekReloadIDRef.current !== playbackReloadID) {
      setIsPlaying(false);
    }
    setIsPlayerReady(false);
    setHasPlayerError(false);

    if (!mpegts.isSupported()) {
      setIsPlayerSupported(false);
      return;
    }

    setIsPlayerSupported(true);

    let disposed = false;
    let reportedError = false;
    let player: TimelapsePlayer | undefined;

    const isCurrentPlayer = () => {
      return !disposed && isModalOpenRef.current && (!player || playerRef.current === player);
    };

    const finishSeekReload = () => {
      if (activeSeekReloadIDRef.current !== playbackReloadID) return;

      activeSeekReloadIDRef.current = undefined;
      pendingPlayerSeekRef.current = undefined;
      updateSeekReloading(false);
    };

    const markPlayerReady = () => {
      if (!isCurrentPlayer()) return;
      setHasPlayerError(false);
      setIsPlayerReady(true);

      const shouldAutoplay =
        shouldAutoplayRef.current || pendingAutoplayRef.current === playbackReloadID;
      if (!shouldAutoplay || !player) {
        finishSeekReload();
        return;
      }

      shouldAutoplayRef.current = false;
      pendingAutoplayRef.current = undefined;
      Promise.resolve(player.play())
        .then(() => {
          if (isCurrentPlayer()) {
            setIsPlaying(true);
            finishSeekReload();
          }
        })
        .catch(() => {
          if (isCurrentPlayer()) {
            setIsPlaying(false);
            finishSeekReload();
          }
        });
    };

    const reportPlayerError = () => {
      if (!isCurrentPlayer()) return;
      pendingAutoplayRef.current = undefined;
      shouldAutoplayRef.current = false;
      activeSeekReloadIDRef.current = undefined;
      updateSeekReloading(false);
      setHasPlayerError(true);
      setIsPlayerReady(false);
      setIsPlaying(false);
      if (!reportedError) {
        reportedError = true;
        feedback.error(t('settings.ai.errors.playScreenTimelapse'));
      }
    };

    async function createPlayer(playback: ScreenTimelapseDay, video: HTMLVideoElement) {
      let accessResponse;
      try {
        accessResponse = await api.authorizeScreenTimelapsePlayback();
      } catch {
        reportPlayerError();
        return;
      }
      if (!isCurrentPlayer()) return;
      if (accessResponse.code === -3) {
        onAuthorizationRequired();
        return;
      }
      if (accessResponse.code !== 0) {
        reportPlayerError();
        return;
      }

      try {
        playerSourceDateRef.current = playback.date;
        player = mpegts.createPlayer(
          {
            type: 'mpegts',
            isLive: false,
            hasAudio: false,
            hasVideo: true,
            duration: Math.max(0, playback.durationMs - playbackStartOffsetSeconds * 1000),
            filesize: playbackStartOffsetSeconds > 0 ? undefined : playback.filesize || undefined,
            url: buildPlaybackURL(playback.videoUrl, playbackStartOffsetSeconds),
            withCredentials: true
          },
          {
            enableWorker: true,
            lazyLoad: true,
            lazyLoadMaxDuration: 10,
            lazyLoadRecoverDuration: 3,
            seekType: 'range'
          }
        );

        playerRef.current = player;
        player.on(mpegts.Events.ERROR, reportPlayerError);
        video.addEventListener('canplay', markPlayerReady);
        video.addEventListener('loadeddata', finishSeekReload);
        video.addEventListener('error', reportPlayerError);
        player.attachMediaElement(video);
        player.load();
      } catch {
        reportPlayerError();
        if (player) {
          player.off(mpegts.Events.ERROR, reportPlayerError);
          player.destroy();
        }
        if (playerRef.current === player) {
          playerRef.current = undefined;
        }
      }
    }

    void createPlayer(activePlayback, activeVideo);

    return () => {
      disposed = true;
      activeVideo.removeEventListener('canplay', markPlayerReady);
      activeVideo.removeEventListener('loadeddata', finishSeekReload);
      activeVideo.removeEventListener('error', reportPlayerError);
      if (!isSeekReloadingRef.current) {
        setIsPlaying(false);
      }
      setIsPlayerReady(false);
      // The instance may already be destroyed (delete-day lease release or a
      // failed attach); mpegts facade calls throw on a destroyed player, so
      // the teardown stays best effort.
      try {
        player?.off(mpegts.Events.ERROR, reportPlayerError);
        player?.pause();
        player?.unload();
        player?.detachMediaElement();
        player?.destroy();
      } catch {
        // ignore: the player is gone either way
      }
      if (playerRef.current === player) {
        playerRef.current = undefined;
        playerSourceDateRef.current = '';
      }
    };
  }, [
    authorizationVersion,
    isModalOpen,
    onAuthorizationRequired,
    playbackDay,
    playbackReloadID,
    playbackStartOffsetSeconds,
    t,
    updateSeekReloading
  ]);

  useEffect(() => {
    if (
      !pendingSeek ||
      pendingSeek.date !== selectedDate ||
      pendingSeek.date !== framesDate ||
      frames.length === 0 ||
      isLoadingFrames
    ) {
      return;
    }
    seekToSeconds(pendingSeek.seconds);
    setPendingSeek(undefined);
  }, [frames.length, framesDate, isLoadingFrames, pendingSeek, seekToSeconds, selectedDate]);

  function closeModal() {
    if (document.fullscreenElement === playerShellRef.current) {
      void document.exitFullscreen().catch(() => undefined);
    }

    isModalOpenRef.current = false;
    latestDaysRequestRef.current += 1;
    latestFramesRequestRef.current += 1;
    latestSearchRequestRef.current += 1;
    latestFramesDateRef.current = '';
    onOpenChange(false);
    setIsLoadingDays(false);
    setIsLoadingFrames(false);
    setSearchQuery('');
    setSearchResults([]);
    setHasSearched(false);
    setIsSearching(false);
    setPendingSeek(undefined);
    setIsPlaying(false);
    setIsPlayerReady(false);
    setHasPlayerError(false);
    setIsFullscreen(false);
    scrubTimeRef.current = undefined;
    isScrubbingRef.current = false;
    activeSeekReloadIDRef.current = undefined;
    pendingPlayerSeekRef.current = undefined;
    pendingAutoplayRef.current = undefined;
    shouldAutoplayRef.current = false;
    suppressNextDateAutoplayRef.current = false;
    playbackReloadIDRef.current = 0;
    playerSourceDateRef.current = '';
    updateSeekReloading(false);
    setScrubTime(undefined);
    setCurrentTime(0);
    setPlaybackStartDate('');
    setPlaybackStartSeconds(0);
    setPlaybackReloadID(0);
  }

  function shouldIgnorePendingSeekUpdate(nextTime: number) {
    const pendingSeek = pendingPlayerSeekRef.current;

    if (!pendingSeek) return false;

    const reachedSeekTarget = Math.abs(nextTime - pendingSeek.seconds) <= 0.25;
    const seekTimedOut = Date.now() - pendingSeek.startedAt > 5000;
    if (!reachedSeekTarget && !seekTimedOut) return true;

    pendingPlayerSeekRef.current = undefined;
    return false;
  }

  function handleVideoTimeUpdate(event: SyntheticEvent<HTMLVideoElement>) {
    const nextTime = Math.min(
      durationSeconds,
      playbackStartOffsetSeconds + event.currentTarget.currentTime
    );
    if (shouldIgnorePendingSeekUpdate(nextTime)) return;

    if (!isScrubbingRef.current) {
      setCurrentTime(nextTime);
    }
  }

  function handleVideoSeeked(event: SyntheticEvent<HTMLVideoElement>) {
    const nextTime = Math.min(
      durationSeconds,
      playbackStartOffsetSeconds + event.currentTarget.currentTime
    );
    if (shouldIgnorePendingSeekUpdate(nextTime)) return;

    if (!isScrubbingRef.current) {
      setCurrentTime(nextTime);
    }
  }

  function handleVideoEnded() {
    activeSeekReloadIDRef.current = undefined;
    updateSeekReloading(false);
    setIsPlaying(false);
    setCurrentTime(durationSeconds);
  }

  function handleVideoPause() {
    if (!isSeekReloadingRef.current) {
      setIsPlaying(false);
    }
  }

  function handleProgressPointerDown(event: ReactPointerEvent<HTMLInputElement>) {
    isScrubbingRef.current = true;
    event.currentTarget.setPointerCapture(event.pointerId);
    const snappedSeconds = snapPlaybackTime(Number(event.currentTarget.value));
    scrubTimeRef.current = snappedSeconds;
    setScrubTime(snappedSeconds);
  }

  function releaseProgressPointerCapture(event?: ReactPointerEvent<HTMLInputElement>) {
    if (event?.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
  }

  function clearProgressScrub() {
    isScrubbingRef.current = false;
    scrubTimeRef.current = undefined;
    setScrubTime(undefined);
  }

  function finishProgressScrub(event?: ReactPointerEvent<HTMLInputElement>) {
    const targetSeconds = scrubTimeRef.current;
    clearProgressScrub();
    releaseProgressPointerCapture(event);

    if (typeof targetSeconds === 'number') {
      seekToSeconds(targetSeconds);
    }
  }

  function cancelProgressScrub(event?: ReactPointerEvent<HTMLInputElement>) {
    if (!isScrubbingRef.current) return;
    clearProgressScrub();
    releaseProgressPointerCapture(event);
  }

  function handleProgressChange(event: ChangeEvent<HTMLInputElement>) {
    const snappedSeconds = snapPlaybackTime(Number(event.target.value));
    scrubTimeRef.current = snappedSeconds;
    setScrubTime(snappedSeconds);

    if (!isScrubbingRef.current) {
      scrubTimeRef.current = undefined;
      setScrubTime(undefined);
      seekToSeconds(snappedSeconds);
    }
  }

  function jumpToResult(result: ScreenTimelapseSearchResult) {
    if (!result.available) return;

    const seconds = getResultSeconds(result);
    selectDate(result.date);
    setPendingSeek({ date: result.date, seconds });

    if (
      result.date === selectedDate &&
      result.date === framesDate &&
      frames.length > 0 &&
      !isLoadingFrames
    ) {
      seekToSeconds(seconds);
      setPendingSeek(undefined);
    }
  }

  function search() {
    const query = searchQuery.trim();
    if (!query || isSearching) return;

    setIsSearching(true);
    setHasSearched(true);
    setSearchResults([]);
    setPendingSeek(undefined);
    const requestID = latestSearchRequestRef.current + 1;
    latestSearchRequestRef.current = requestID;
    api
      .searchScreenTimelapse(query)
      .then((rsp) => {
        if (!isModalOpenRef.current || latestSearchRequestRef.current !== requestID) return;

        if (rsp.code === -3) {
          onAuthorizationRequired();
          return;
        }
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          feedback.error(t('settings.ai.screenTimelapseSearchFailed'));
          return;
        }

        const data = (rsp.data ?? {}) as ScreenTimelapseSearchResultsResponse;
        const results = sortSearchResultsByTimeDesc(
          Array.isArray(data.results) ? data.results : []
        );
        setSearchResults(results);

        if (results.length === 1 && results[0].available) {
          const result = results[0];
          selectDate(result.date);
          setPendingSeek({ date: result.date, seconds: getResultSeconds(result) });
        }
      })
      .catch(() => {
        if (!isModalOpenRef.current || latestSearchRequestRef.current !== requestID) return;
        feedback.error(t('settings.ai.screenTimelapseSearchFailed'));
      })
      .finally(() => {
        if (!isModalOpenRef.current || latestSearchRequestRef.current !== requestID) return;
        setIsSearching(false);
      });
  }

  function closeSearchResults() {
    latestSearchRequestRef.current += 1;
    setHasSearched(false);
    setSearchResults([]);
    setPendingSeek(undefined);
    setIsSearching(false);
  }

  function selectAdjacentTimelineDay(direction: -1 | 1) {
    if (selectedTimelineIndex < 0) return;

    const nextDay = timelineDays[selectedTimelineIndex + direction];
    if (!nextDay) return;

    selectDate(nextDay.date);
  }

  function deleteDay(day: ScreenTimelapseDay) {
    if (!day.deletable || deletingDateRef.current) return;

    modal.confirm({
      title: t('settings.ai.screenTimelapseDeleteConfirmTitle'),
      content: (
        <span className="text-xs text-neutral-500">
          {t('settings.ai.screenTimelapseDeleteConfirmDesc', {
            date: day.date,
            size: formatBytes(day.filesize, intlLocale)
          })}
        </span>
      ),
      okText: t('settings.ai.screenTimelapseDelete'),
      okButtonProps: { danger: true },
      cancelText: t('settings.ai.cancelBtn'),
      onOk: () => performDeleteDay(day)
    });
  }

  async function togglePlayback() {
    if (!playbackDay || !isPlayerSupported) {
      feedback.error(t('settings.ai.screenTimelapsePlayerUnsupported'));
      return;
    }

    if (!playerRef.current || hasPlayerError) {
      feedback.error(t('settings.ai.errors.playScreenTimelapse'));
      return;
    }

    const player = playerRef.current;
    if (isPlaying) {
      shouldAutoplayRef.current = false;
      pendingAutoplayRef.current = undefined;
      activeSeekReloadIDRef.current = undefined;
      updateSeekReloading(false);
      player.pause();
      setIsPlaying(false);
      return;
    }

    try {
      await player.play();
      if (playerRef.current === player && isModalOpenRef.current) {
        setIsPlaying(true);
      }
    } catch {
      if (playerRef.current === player && isModalOpenRef.current) {
        feedback.error(t('settings.ai.errors.playScreenTimelapse'));
        setIsPlaying(false);
      }
    }
  }

  async function toggleFullscreen() {
    const playerShell = playerShellRef.current;
    if (!playbackDay || !playerShell) return;

    if (!document.fullscreenEnabled || !playerShell.requestFullscreen) {
      feedback.error(t('settings.ai.screenTimelapseFullscreenUnsupported'));
      return;
    }

    try {
      if (document.fullscreenElement === playerShell) {
        await document.exitFullscreen();
        return;
      }

      await playerShell.requestFullscreen();
    } catch {
      feedback.error(t('settings.ai.errors.screenTimelapseFullscreen'));
    }
  }

  return (
    <Modal
      width="min(1040px, calc(100vw - 32px))"
      centered
      footer={null}
      closeIcon={null}
      open={isModalOpen}
      onCancel={closeModal}
      className="[&_.ant-modal-content]:overflow-hidden [&_.ant-modal-content]:rounded-xl [&_.ant-modal-content]:border [&_.ant-modal-content]:border-white/10 [&_.ant-modal-content]:bg-[#1c1b1d] [&_.ant-modal-content]:p-0 [&_.ant-modal-content]:shadow-2xl"
    >
      {contextHolder}
      <div className="flex h-[min(720px,calc(100dvh-48px))] flex-col overflow-hidden text-neutral-100">
        <div className="flex items-center justify-between px-4 py-3">
          <div className="flex min-w-0 items-center gap-2">
            <HistoryIcon size={20} className="shrink-0 text-indigo-300" />
            <span className="truncate text-base font-semibold">
              {t('settings.ai.screenTimelapseService')}
            </span>
          </div>
          <Button
            type="text"
            className="text-neutral-400 hover:bg-white/5 hover:text-white"
            icon={<XIcon size={18} />}
            aria-label={t('settings.close')}
            onClick={closeModal}
          />
        </div>
        <Divider className="opacity-30" style={{ margin: 0 }} />

        <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-4">
          <div className="flex justify-end">
            <Input
              allowClear
              className="w-[280px] rounded-lg border-white/10 bg-[#201f22] text-sm hover:border-white/20 focus:border-white/20"
              prefix={<SearchIcon size={16} className="mr-1 text-neutral-500" />}
              suffix={
                isSearching ? (
                  <Loader2Icon size={16} className="animate-spin text-neutral-500" />
                ) : null
              }
              placeholder={t('settings.ai.screenTimelapseSearchPlaceholder')}
              value={searchQuery}
              onChange={(event) => setSearchQuery(event.target.value)}
              onPressEnter={search}
            />
          </div>

          <div
            className={`grid min-h-0 flex-1 grid-cols-1 gap-4 ${
              hasSearched &&
              !isSearching &&
              (searchResults.length !== 1 || !searchResults[0]?.available)
                ? 'lg:grid-cols-[minmax(0,1fr)_260px] landscape:grid-cols-[minmax(0,1fr)_minmax(200px,260px)]'
                : ''
            }`}
          >
            <div
              ref={playerShellRef}
              className="flex flex-col gap-3 bg-[#1c1b1d] lg:min-h-0 [&:fullscreen]:h-screen [&:fullscreen]:w-screen [&:fullscreen]:p-4"
            >
              <div
                className="relative flex min-h-[300px] flex-1 overflow-hidden rounded-lg border border-white/10 bg-black"
                aria-busy={isSeekReloading || undefined}
              >
                {playbackDay && isPlayerSupported ? (
                  <video
                    ref={videoRef}
                    className="h-full w-full cursor-pointer object-contain"
                    muted
                    playsInline
                    onClick={togglePlayback}
                    onDoubleClick={() => void toggleFullscreen()}
                    onPlay={() => setIsPlaying(true)}
                    onPause={handleVideoPause}
                    onEnded={handleVideoEnded}
                    onSeeked={handleVideoSeeked}
                    onTimeUpdate={handleVideoTimeUpdate}
                  />
                ) : playbackDay && !isPlayerSupported ? (
                  <div className="flex h-full w-full items-center justify-center px-4 text-center text-xs text-neutral-500">
                    {t('settings.ai.screenTimelapsePlayerUnsupported')}
                  </div>
                ) : isLoadingDays ? (
                  <div className="flex h-full w-full items-center justify-center px-4 text-center text-xs text-neutral-500">
                    {t('settings.ai.screenTimelapseLoading')}
                  </div>
                ) : (
                  <Empty
                    className="m-auto px-4"
                    image={
                      <div className="flex h-full items-center justify-center">
                        <FilmIcon size={40} strokeWidth={1.2} className="text-neutral-600" />
                      </div>
                    }
                    description={t('settings.ai.screenTimelapseNoVideos')}
                  />
                )}

                {(isLoadingDays || isLoadingFrames || isSeekReloading) && (
                  <div className="absolute inset-0 flex items-center justify-center bg-black/45">
                    <Spin />
                  </div>
                )}
                {hasPlayerError && playbackDay && isPlayerSupported && (
                  <div className="absolute inset-0 flex items-center justify-center bg-black/60 px-4 text-center text-sm text-neutral-300">
                    {t('settings.ai.errors.playScreenTimelapse')}
                  </div>
                )}
              </div>

              <div className="flex items-center gap-3 px-2 py-1">
                <Button
                  type="text"
                  className="shrink-0 text-neutral-400 hover:bg-white/5 hover:text-white disabled:text-neutral-600"
                  disabled={!playbackDay || !isPlayerSupported || !isPlayerReady || hasPlayerError}
                  icon={isPlaying ? <PauseIcon size={16} /> : <PlayIcon size={16} />}
                  aria-label={
                    isPlaying
                      ? t('settings.ai.screenTimelapsePause')
                      : t('settings.ai.screenTimelapsePlay')
                  }
                  onClick={togglePlayback}
                />
                <span className="w-12 shrink-0 font-mono text-sm text-neutral-300">
                  {formatClock(displayedProgressTime)}
                </span>
                <div className="relative flex h-4 min-w-0 flex-1 items-center">
                  <input
                    className="absolute inset-0 h-1.5 w-full cursor-pointer appearance-none rounded-full bg-white/10 outline-none [&::-webkit-slider-thumb]:h-3 [&::-webkit-slider-thumb]:w-3 [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:rounded-full [&::-webkit-slider-thumb]:bg-indigo-300"
                    style={{
                      background:
                        durationSeconds > 0
                          ? `linear-gradient(to right, #a5b4fc ${(displayedProgressTime / durationSeconds) * 100}%, rgba(255,255,255,0.1) ${(displayedProgressTime / durationSeconds) * 100}%)`
                          : undefined
                    }}
                    type="range"
                    min={0}
                    max={durationSeconds}
                    step="any"
                    value={displayedProgressTime}
                    disabled={!playbackDay || frames.length === 0 || hasPlayerError}
                    onChange={handleProgressChange}
                    onLostPointerCapture={cancelProgressScrub}
                    onPointerCancel={cancelProgressScrub}
                    onPointerDown={handleProgressPointerDown}
                    onPointerUp={finishProgressScrub}
                  />
                </div>
                <span className="w-12 shrink-0 text-right font-mono text-sm text-neutral-300">
                  {formatClock(durationSeconds)}
                </span>
                <Button
                  type="text"
                  className="shrink-0 text-neutral-400 hover:bg-white/5 hover:text-white disabled:text-neutral-600"
                  disabled={!playbackDay || !isPlayerSupported || hasPlayerError}
                  icon={isFullscreen ? <MinimizeIcon size={16} /> : <MaximizeIcon size={16} />}
                  aria-label={
                    isFullscreen
                      ? t('settings.ai.screenTimelapseExitFullscreen')
                      : t('settings.ai.screenTimelapseEnterFullscreen')
                  }
                  title={
                    isFullscreen
                      ? t('settings.ai.screenTimelapseExitFullscreen')
                      : t('settings.ai.screenTimelapseEnterFullscreen')
                  }
                  onClick={() => void toggleFullscreen()}
                />
              </div>
            </div>

            {hasSearched &&
              !isSearching &&
              (searchResults.length !== 1 || !searchResults[0]?.available) && (
                <div className="flex min-h-0 flex-col overflow-hidden rounded-lg border border-white/10 bg-[#201f22]">
                  <div className="flex items-center justify-between gap-2 border-b border-white/5 px-3 py-2">
                    <span className="min-w-0 truncate text-xs font-medium text-neutral-300">
                      {t('settings.ai.screenTimelapseMultipleResults')}
                    </span>
                    <Button
                      type="text"
                      size="small"
                      className="shrink-0 text-neutral-500 hover:bg-white/5 hover:text-white"
                      icon={<XIcon size={14} />}
                      aria-label={t('settings.close')}
                      onClick={closeSearchResults}
                    />
                  </div>

                  <div className="min-h-0 flex-1 space-y-2 overflow-y-auto p-2">
                    {searchResults.map((result) => {
                      const resultText = buildSearchContext(
                        result.snippet || '',
                        searchQuery,
                        intlLocale
                      );

                      return (
                        <button
                          key={`${result.date}-${result.capturedAt}-${result.currentTime ?? result.pts ?? 0}`}
                          type="button"
                          disabled={isSearching || !result.available}
                          className="flex w-full min-w-0 flex-col rounded-lg border border-white/5 bg-white/[0.03] px-3 py-2 text-left transition-colors hover:bg-white/[0.06] disabled:cursor-not-allowed disabled:opacity-45"
                          onClick={() => jumpToResult(result)}
                        >
                          <span className="min-w-0 truncate font-mono text-xs text-neutral-300">
                            {result.matchedFrameTime || result.capturedAt}
                          </span>
                          <span className="mt-1 line-clamp-2 text-xs text-neutral-500">
                            {resultText
                              ? renderHighlightedText(resultText, searchQuery, intlLocale)
                              : null}
                          </span>
                          {!result.available && (
                            <span className="mt-1 text-xs text-amber-400">
                              {result.unavailableReason === 'missing_video'
                                ? t('settings.ai.screenTimelapseRecordingCleaned')
                                : t('settings.ai.screenTimelapseUnavailable')}
                            </span>
                          )}
                        </button>
                      );
                    })}

                    {searchResults.length === 0 && (
                      <Empty
                        className="px-3 py-8"
                        image={
                          <div className="flex h-full items-center justify-center">
                            <SearchXIcon size={28} strokeWidth={1.2} className="text-neutral-600" />
                          </div>
                        }
                        imageStyle={{ height: 56 }}
                        description={t('settings.ai.screenTimelapseNoResults')}
                      />
                    )}
                  </div>
                </div>
              )}
          </div>
        </div>

        <Divider className="opacity-30" style={{ margin: 0 }} />
        <div className="flex items-center justify-between bg-[#1c1b1d] px-6 py-4">
          <Button
            type="text"
            className="shrink-0 text-neutral-400 hover:bg-white/5 hover:text-white"
            disabled={!hasPreviousTimelineDay}
            icon={<ChevronLeftIcon size={18} />}
            aria-label={t('settings.ai.screenTimelapsePreviousDay')}
            onClick={() => selectAdjacentTimelineDay(-1)}
          />
          <div className="flex min-w-0 flex-1 justify-start px-4">
            <div
              ref={timelineRef}
              className="flex max-w-full items-center gap-2 overflow-x-auto scroll-smooth [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
            >
              {timelineDays.map((day) => {
                const canDelete = day.deletable;

                return (
                  <div
                    key={day.date}
                    data-selected={day.date === selectedDate || undefined}
                    className={`flex shrink-0 items-center rounded-lg border text-sm transition-all duration-200 ${
                      day.date === selectedDate
                        ? 'border-white/5 bg-white/5 font-medium text-neutral-200 shadow-sm'
                        : 'border-transparent bg-transparent text-neutral-500 hover:border-white/[0.04] hover:bg-white/[0.02] hover:text-neutral-400'
                    }`}
                  >
                    <button
                      type="button"
                      className={`appearance-none border-0 bg-transparent py-1.5 text-inherit ${canDelete ? 'pl-3 pr-1.5' : 'px-3'}`}
                      onClick={() => selectDate(day.date)}
                    >
                      <span className="block max-w-24 truncate">
                        {formatDayLabel(day.date, intlLocale)}
                      </span>
                    </button>
                    {canDelete && (
                      <button
                        type="button"
                        disabled={Boolean(deletingDate)}
                        className="mr-1 flex h-6 w-6 shrink-0 appearance-none items-center justify-center rounded border-0 bg-transparent p-0 text-neutral-400 transition-colors hover:bg-red-500/10 hover:text-red-300 disabled:cursor-not-allowed disabled:opacity-45"
                        aria-label={t('settings.ai.screenTimelapseDeleteDay', { date: day.date })}
                        title={t('settings.ai.screenTimelapseDeleteDay', { date: day.date })}
                        onClick={() => deleteDay(day)}
                      >
                        {deletingDate === day.date ? (
                          <Loader2Icon size={15} className="animate-spin" />
                        ) : (
                          <XIcon size={15} strokeWidth={2.25} />
                        )}
                      </button>
                    )}
                  </div>
                );
              })}

              {timelineDays.length === 0 && (
                <span className="text-xs text-neutral-500">
                  {isLoadingDays
                    ? t('settings.ai.screenTimelapseLoading')
                    : t('settings.ai.screenTimelapseNoVideos')}
                </span>
              )}
            </div>
          </div>
          <Button
            type="text"
            className="shrink-0 text-neutral-400 hover:bg-white/5 hover:text-white"
            disabled={!hasNextTimelineDay}
            icon={<ChevronRightIcon size={18} />}
            aria-label={t('settings.ai.screenTimelapseNextDay')}
            onClick={() => selectAdjacentTimelineDay(1)}
          />
        </div>
      </div>
    </Modal>
  );
};
