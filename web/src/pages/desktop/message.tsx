import { ReactNode, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { message } from 'antd';
import { useAtomValue } from 'jotai';
import { MonitorPauseIcon, MonitorXIcon, VideoOffIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { VideoStatus } from '@/types';
import { LAYER_VIDEO_STATUS_MESSAGE } from '@/lib/layers.ts';
import {
  videoBandwidthWarningAtom,
  videoStatusAtom,
  webrtcConnectionFailedAtom
} from '@/jotai/screen.ts';

const videoMessageKey = 'video';

const isFullyInsideViewport = (notice: Element) => {
  const bounds = notice.getBoundingClientRect();
  const viewport = window.visualViewport;
  const left = viewport?.offsetLeft ?? 0;
  const top = viewport?.offsetTop ?? 0;
  const right = left + (viewport?.width ?? window.innerWidth);
  const bottom = top + (viewport?.height ?? window.innerHeight);

  return (
    bounds.left >= left && bounds.top >= top && bounds.right <= right && bounds.bottom <= bottom
  );
};

export const Message = () => {
  const { t } = useTranslation();
  const videoStatus = useAtomValue(videoStatusAtom);
  const bandwidthWarning = useAtomValue(videoBandwidthWarningAtom);
  const webrtcConnectionFailed = useAtomValue(webrtcConnectionFailedAtom);

  const messageContainerRef = useRef<HTMLDivElement>(null);
  const [messageApi, contextHolder] = message.useMessage({
    getContainer: () => messageContainerRef.current ?? document.body
  });
  const [messageVisible, setMessageVisible] = useState(false);
  const contentRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    setMessageVisible(false);
  }, [t, videoStatus, webrtcConnectionFailed]);

  useEffect(() => {
    if (videoStatus === VideoStatus.Normal && !bandwidthWarning && !webrtcConnectionFailed) {
      messageApi.destroy(videoMessageKey);
      return;
    }

    let icon: ReactNode;
    let content: ReactNode;

    if (webrtcConnectionFailed) {
      icon = <MonitorXIcon size={16} className="text-red-500/80" />;
      content = <div className="pl-2 text-sm">{t('screen.webrtcConnectionFailed')}</div>;
    } else if (bandwidthWarning && videoStatus === VideoStatus.Normal) {
      icon = <MonitorPauseIcon size={16} className="text-yellow-500/80" />;
      content = <div className="pl-2 text-sm">{t('screen.bandwidthWarning')}</div>;
    } else if (videoStatus === VideoStatus.NoImage) {
      icon = <MonitorXIcon size={16} className="text-red-500/80" />;
      content = <div className="pl-2 text-sm">{t('screen.noSignal')}</div>;
    } else if (videoStatus === VideoStatus.CaptureDisabled) {
      icon = <VideoOffIcon size={16} className="text-yellow-500/80" />;
      content = <div className="pl-2 text-sm">{t('screen.captureDisabled')}</div>;
    } else if (videoStatus === VideoStatus.InconsistentVideoMode) {
      icon = <MonitorPauseIcon size={16} className="text-yellow-500/80" />;
      content = <div className="pl-2 text-sm">{t('screen.inconsistentVideoMode')}</div>;
    } else {
      return;
    }

    let frame: number | undefined;
    let observer: ResizeObserver | undefined;
    const viewport = window.visualViewport;

    const measure = () => {
      frame = window.requestAnimationFrame(() => {
        const notice = contentRef.current?.closest('.ant-message-notice');
        if (!notice) {
          return;
        }

        observer?.observe(notice);
        setMessageVisible(isFullyInsideViewport(notice));
      });
    };

    const scheduleMeasure = () => {
      if (frame !== undefined) {
        window.cancelAnimationFrame(frame);
      }
      frame = window.requestAnimationFrame(measure);
    };

    messageApi.open({
      key: videoMessageKey,
      type: 'warning',
      icon: icon,
      content: <div ref={contentRef}>{content}</div>,
      // The bandwidth hint is a transient nudge: show it for 2s, then stop nagging.
      duration:
        !webrtcConnectionFailed && bandwidthWarning && videoStatus === VideoStatus.Normal ? 2 : 0,
      className: 'mt-16',
      style: { visibility: messageVisible ? 'visible' : 'hidden' }
    });

    if (typeof ResizeObserver !== 'undefined') {
      observer = new ResizeObserver(scheduleMeasure);
    }

    window.addEventListener('resize', scheduleMeasure);
    viewport?.addEventListener('resize', scheduleMeasure);
    viewport?.addEventListener('scroll', scheduleMeasure);
    scheduleMeasure();

    return () => {
      if (frame !== undefined) {
        window.cancelAnimationFrame(frame);
      }
      observer?.disconnect();
      window.removeEventListener('resize', scheduleMeasure);
      viewport?.removeEventListener('resize', scheduleMeasure);
      viewport?.removeEventListener('scroll', scheduleMeasure);
    };
  }, [bandwidthWarning, messageApi, messageVisible, t, videoStatus, webrtcConnectionFailed]);

  return (
    <div
      ref={messageContainerRef}
      className="relative"
      style={{ zIndex: LAYER_VIDEO_STATUS_MESSAGE }}
    >
      {contextHolder}
    </div>
  );
};
