import { useEffect, useState } from 'react';
import { Button, Modal, Tooltip } from 'antd';
import { InfoIcon, LogInIcon, MaximizeIcon, MinimizeIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { feedback } from '@/lib/feedback.tsx';
import { ButtonBase } from '@/components/ui/button-base.tsx';

import {
  enterNativeFullscreen,
  exitNativeFullscreen,
  getFullscreenCapability
} from './platform.ts';

type GuideStepProps = {
  index: number;
  children: string;
};

const GuideStep = ({ index, children }: GuideStepProps) => (
  <li className="flex items-start gap-3">
    <span
      aria-hidden="true"
      className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-sky-500/15 text-xs font-semibold text-sky-300"
    >
      {index}
    </span>
    <span className="pt-0.5 text-sm leading-5 text-neutral-200">{children}</span>
  </li>
);

export const Fullscreen = () => {
  const { t } = useTranslation();
  const capability = getFullscreenCapability();

  const [isFullscreen, setIsFullscreen] = useState(false);
  const [isGuideOpen, setIsGuideOpen] = useState(false);
  const [isChanging, setIsChanging] = useState(false);
  const [isTooltipOpen, setIsTooltipOpen] = useState(false);

  useEffect(() => {
    const onFullscreenChange = () => {
      setIsFullscreen(!!document.fullscreenElement);
    };

    onFullscreenChange();
    document.addEventListener('fullscreenchange', onFullscreenChange);

    return () => {
      document.removeEventListener('fullscreenchange', onFullscreenChange);
    };
  }, []);

  if (capability.mode === 'hidden') return null;

  async function handleFullscreen() {
    setIsTooltipOpen(false);

    if (capability.mode === 'guide') {
      setIsGuideOpen(true);
      return;
    }
    if (isChanging) return;

    setIsChanging(true);
    try {
      if (document.fullscreenElement) {
        await exitNativeFullscreen();
      } else {
        await enterNativeFullscreen();
      }
    } catch {
      feedback.error(t('fullscreen.error'));
    } finally {
      setIsChanging(false);
    }
  }

  const guideSteps = [
    t('fullscreen.guide.shareMenu'),
    t('fullscreen.guide.addToHomeScreen'),
    t('fullscreen.guide.add')
  ];

  return (
    <>
      <Tooltip
        title={t('fullscreen.toggle')}
        placement="bottom"
        mouseEnterDelay={0.6}
        open={isTooltipOpen && !isGuideOpen}
        destroyOnHidden
        onOpenChange={(open) => setIsTooltipOpen(open && !isGuideOpen)}
      >
        <ButtonBase
          className="hidden h-[30px] w-[30px] items-center justify-center rounded text-neutral-300 hover:bg-neutral-700/80 hover:text-white sm:flex"
          aria-label={t('fullscreen.toggle')}
          aria-haspopup={capability.mode === 'guide' ? 'dialog' : undefined}
          aria-pressed={capability.mode === 'native' ? isFullscreen : undefined}
          aria-busy={isChanging || undefined}
          disabled={isChanging}
          onClick={() => void handleFullscreen()}
        >
          {isFullscreen ? <MinimizeIcon size={18} /> : <MaximizeIcon size={18} />}
        </ButtonBase>
      </Tooltip>

      {capability.mode === 'guide' && (
        <Modal
          open={isGuideOpen}
          title={t('fullscreen.guide.title')}
          width={560}
          centered
          onCancel={() => setIsGuideOpen(false)}
          footer={
            <Button type="primary" onClick={() => setIsGuideOpen(false)}>
              {t('auth.ok')}
            </Button>
          }
        >
          <div className="space-y-4 pt-2 text-neutral-200">
            <div className="inline-flex rounded-full border border-neutral-700 bg-neutral-900 px-3 py-1 text-xs font-medium text-neutral-300">
              iOS
            </div>

            <ol className="space-y-3">
              {guideSteps.map((step, index) => (
                <GuideStep key={step} index={index + 1}>
                  {step}
                </GuideStep>
              ))}
            </ol>

            <div className="space-y-2 rounded border border-neutral-800 bg-neutral-950/70 px-3 py-3 text-sm leading-5 text-neutral-400">
              <div className="flex items-start gap-2">
                <InfoIcon className="mt-0.5 shrink-0 text-sky-400" size={16} aria-hidden="true" />
                <span>{t('fullscreen.guide.webAppNote')}</span>
              </div>
              <div className="flex items-start gap-2">
                <LogInIcon
                  className="mt-0.5 shrink-0 text-neutral-500"
                  size={16}
                  aria-hidden="true"
                />
                <span>{t('fullscreen.guide.loginNotice')}</span>
              </div>
            </div>
          </div>
        </Modal>
      )}
    </>
  );
};
