import { useEffect, useRef, useState } from 'react';
import { Tooltip, type TooltipProps } from 'antd';
import clsx from 'clsx';
import { useAtomValue, useStore } from 'jotai';
import { PictureInPicture2Icon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { feedback } from '@/lib/feedback.tsx';
import {
  canEnterPictureInPicture,
  enterPictureInPicture,
  exitPictureInPicture,
  isPictureInPictureActive
} from '@/lib/picture-in-picture.ts';
import { isWebrtcMode } from '@/lib/video.ts';
import { effectiveScreenAreaAtom, videoModeAtom } from '@/jotai/screen.ts';
import { ButtonBase } from '@/components/ui/button-base.tsx';

type PictureInPictureProps = {
  tooltipPlacement?: TooltipProps['placement'];
};

export const PictureInPicture = ({ tooltipPlacement = 'bottom' }: PictureInPictureProps) => {
  const { t } = useTranslation();
  const store = useStore();
  const videoMode = useAtomValue(videoModeAtom);
  const [isActive, setIsActive] = useState(isPictureInPictureActive);
  const [isChanging, setIsChanging] = useState(false);
  const previousVideoMode = useRef(videoMode);

  useEffect(() => {
    const sync = () => setIsActive(isPictureInPictureActive());

    // The events fire on the video element; listen in the capture phase so
    // both the WebRTC <video> and the direct-mode proxy are covered.
    document.addEventListener('enterpictureinpicture', sync, true);
    document.addEventListener('leavepictureinpicture', sync, true);
    return () => {
      document.removeEventListener('enterpictureinpicture', sync, true);
      document.removeEventListener('leavepictureinpicture', sync, true);
    };
  }, []);

  // A video mode switch replaces the screen element. The proxy session exits
  // itself; the uncropped fallback PiPs the page's own <video>, whose leave
  // event then fires while detached and never reaches the listeners above.
  useEffect(() => {
    if (videoMode === previousVideoMode.current) return;
    previousVideoMode.current = videoMode;
    void exitPictureInPicture()
      .catch(() => undefined)
      .finally(() => setIsActive(isPictureInPictureActive()));
  }, [videoMode]);

  if (!canEnterPictureInPicture(isWebrtcMode(videoMode))) return null;

  async function togglePictureInPicture() {
    if (isChanging) return;

    setIsChanging(true);
    try {
      if (isPictureInPictureActive()) {
        await exitPictureInPicture();
      } else {
        await enterPictureInPicture({
          // Read per frame so the PiP window follows the same crop as the page.
          getCrop: () => store.get(effectiveScreenAreaAtom),
          subscribeScreenChange: (onChange) => store.sub(videoModeAtom, onChange)
        });
      }
    } catch {
      feedback.error(t('pictureInPicture.error'));
    } finally {
      setIsChanging(false);
      setIsActive(isPictureInPictureActive());
    }
  }

  const label = t(isActive ? 'pictureInPicture.exit' : 'pictureInPicture.enter');

  return (
    <Tooltip title={label} placement={tooltipPlacement} mouseEnterDelay={0.6}>
      <ButtonBase
        className="flex h-[30px] w-[30px] items-center justify-center rounded text-neutral-300 hover:bg-neutral-700/80 hover:text-white"
        aria-label={label}
        aria-pressed={isActive}
        aria-busy={isChanging || undefined}
        disabled={isChanging}
        onClick={() => void togglePictureInPicture()}
      >
        <PictureInPicture2Icon size={18} className={clsx(isActive && 'text-sky-400')} />
      </ButtonBase>
    </Tooltip>
  );
};
