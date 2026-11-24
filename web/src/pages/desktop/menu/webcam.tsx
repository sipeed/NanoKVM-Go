import { useCallback, useEffect, useRef, useState } from 'react';
import { Button } from 'antd';
import { VideoIcon, VideoOffIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { getVirtualDevice, WEBCAM_CHANGED } from '@/api/virtual-device';
import { getBaseUrl } from '@/lib/service.ts';
import { client } from '@/lib/websocket';
import { MenuItem } from '@/components/menu-item.tsx';

import { cameraFrameStream, webcamFrames } from './webcam-frames';
import { sendWebcam, type WebcamReport } from './webcam-sender';

type Status = 'off' | 'starting' | 'calibrating' | 'waiting' | 'sharing';

function cameraError(err: unknown): string {
  if (err instanceof DOMException && err.name === 'NotAllowedError')
    return 'Allow Camera for this site and enable camera access for your browser in system settings.';
  if (err instanceof DOMException && err.name === 'NotFoundError')
    return 'The browser could not find a connected camera.';
  return err instanceof Error ? err.message : 'Could not access the camera.';
}

export function useWebcamShare() {
  const [status, setStatus] = useState<Status>('off');
  const [diagnostics, setDiagnostics] = useState('');
  const [enabled, setEnabled] = useState(false);
  const [error, setError] = useState('');
  const finish = useRef<((reason?: string) => void) | null>(null);
  const stop = useCallback(() => finish.current?.(), []);

  useEffect(() => {
    let current = true;
    let version = 0;
    const refresh = async () => {
      const request = ++version;
      try {
        const rsp = await getVirtualDevice();
        if (current && request === version)
          setEnabled(rsp.code === 0 && !!rsp.data?.isWebcamEnabled);
      } catch {
        if (current && request === version) setEnabled(false);
      }
    };
    void refresh();
    const unsubscribe = client.onConnectionState((state) => {
      if (state === 'closed') {
        ++version;
        setEnabled(false);
        stop();
      }
      if (state === 'open') void refresh();
    });
    window.addEventListener(WEBCAM_CHANGED, refresh);
    window.addEventListener('focus', refresh);
    window.addEventListener('offline', stop);
    window.addEventListener('pagehide', stop);
    return () => {
      current = false;
      unsubscribe();
      window.removeEventListener(WEBCAM_CHANGED, refresh);
      window.removeEventListener('focus', refresh);
      window.removeEventListener('offline', stop);
      window.removeEventListener('pagehide', stop);
      stop();
    };
  }, [stop]);

  useEffect(() => {
    if (!enabled) stop();
  }, [enabled, stop]);

  async function start(deviceId: string, onError?: (reason: string) => void) {
    if (!enabled || finish.current) return;
    let frames: ReturnType<typeof webcamFrames> | undefined;
    let media: MediaStream | undefined;
    let stopSender: (() => void) | undefined;
    function end(reason = '') {
      if (finish.current !== end) return;
      finish.current = null;
      stopSender?.();
      media?.getTracks().forEach((track) => track.stop());
      frames?.close();
      setStatus('off');
      setError(reason);
      if (reason) onError?.(reason);
    }

    finish.current = end;
    setDiagnostics('');
    setStatus('starting');
    setError('');
    try {
      if (!navigator.mediaDevices?.getUserMedia)
        throw new Error('Open the Go+ over HTTPS to share a camera.');
      media = await navigator.mediaDevices.getUserMedia({
        audio: false,
        video: {
          width: { ideal: 1280 },
          height: { ideal: 720 },
          // Cameras commonly expose 30 fps. Pace the JPEG output independently.
          frameRate: { ideal: 30 },
          ...(deviceId ? { deviceId: { exact: deviceId } } : {})
        }
      });
      if (finish.current !== end) {
        media.getTracks().forEach((track) => track.stop());
        return;
      }
      const track = media.getVideoTracks()[0];
      track.onended = () => end('The camera disconnected.');
      const stream = cameraFrameStream(track);
      const url = getBaseUrl('ws') + '/api/stream/webcam';
      const report = (message: WebcamReport) => {
        if (finish.current !== end) return;
        if ('error' in message) end(message.error);
        else {
          setDiagnostics(message.sample);
          setStatus(message.status);
        }
      };
      if (stream && typeof Worker !== 'undefined' && typeof OffscreenCanvas !== 'undefined') {
        const worker = new Worker(new URL('./webcam.worker.ts', import.meta.url), {
          type: 'module'
        });
        worker.onmessage = (event: MessageEvent<WebcamReport>) => report(event.data);
        worker.onerror = () => end('The webcam worker stopped.');
        try {
          worker.postMessage({ stream, url }, [stream]);
          stopSender = () => worker.terminate();
          return;
        } catch (err) {
          worker.terminate();
          if (stream.locked) throw err;
        }
      }
      frames = webcamFrames(stream ?? track);
      await frames.prepare();
      if (finish.current !== end) return;
      const sender = sendWebcam(url, frames, report);
      stopSender = sender.stop;
    } catch (err) {
      end(cameraError(err));
    }
  }
  return {
    status,
    error,
    diagnostics,
    enabled,
    start,
    stop,
    clearError: () => setError('')
  };
}

export function Webcam({ share }: { share: ReturnType<typeof useWebcamShare> }) {
  const { t } = useTranslation();
  const [devices, setDevices] = useState<MediaDeviceInfo[]>([]);
  const [deviceId, selectDevice] = useState('');
  const active = share.status !== 'off';
  const labels = {
    off: '',
    starting: '',
    calibrating: 'Calibrating',
    waiting: 'Ready',
    sharing: 'Streaming'
  };
  const hasCapture =
    share.status === 'calibrating' || share.status === 'waiting' || share.status === 'sharing';
  const statusText = hasCapture
    ? [labels[share.status], share.diagnostics].filter(Boolean).join(' · ')
    : '';
  const [open, setOpen] = useState(false);
  const [listing, setListing] = useState(false);
  const [listError, setListError] = useState('');
  const generation = useRef(0);
  const refresh = useCallback(async () => {
    const request = ++generation.current;
    const all = await navigator.mediaDevices?.enumerateDevices();
    if (request !== generation.current || !all) return;
    const cameras = all.filter((d) => d.kind === 'videoinput' && d.deviceId);
    setDevices(cameras);
    selectDevice((id) => (cameras.some((d) => d.deviceId === id) ? id : ''));
    return cameras;
  }, []);

  useEffect(() => {
    if (!open) return;
    const update = () => {
      void refresh().catch((err) => setListError(cameraError(err)));
    };
    update();
    navigator.mediaDevices?.addEventListener('devicechange', update);
    return () => {
      generation.current++;
      navigator.mediaDevices?.removeEventListener('devicechange', update);
    };
  }, [open, refresh, hasCapture]);

  async function listCameras() {
    setListing(true);
    setListError('');
    share.clearError();
    let stream: MediaStream | undefined;
    try {
      if (!navigator.mediaDevices?.getUserMedia)
        throw new Error('Open the Go+ over HTTPS to list cameras.');
      stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: false });
      const cameras = await refresh();
      if (cameras?.length === 0)
        setListError(
          'The browser did not expose a camera. Check the camera connection and site permission.'
        );
    } catch (err) {
      setListError(cameraError(err));
    } finally {
      stream?.getTracks().forEach((track) => track.stop());
      setListing(false);
    }
  }
  return (
    <MenuItem
      title={t('webcam.title', 'Webcam')}
      icon={
        active ? <VideoIcon size={18} className="text-green-400" /> : <VideoOffIcon size={18} />
      }
      action={{
        pressed: active,
        disabled: listing,
        toggle: () => {
          setListError('');
          if (active) share.stop();
          else void share.start(deviceId);
        }
      }}
      onOpenChange={setOpen}
      content={
        <div className="flex w-60 flex-col gap-2">
          <div className="flex items-center justify-between">
            <span>{t('webcam.source', 'Camera')}</span>
            <Button
              size="small"
              type="text"
              disabled={active}
              loading={listing}
              onClick={() => void listCameras()}
            >
              {t('webcam.refresh', 'Refresh')}
            </Button>
          </div>
          <div className="flex items-center gap-2">
            <select
              className="min-w-0 flex-1 rounded bg-neutral-800 p-1"
              aria-label={t('webcam.source', 'Camera')}
              value={deviceId}
              disabled={active || listing}
              onChange={(e) => selectDevice(e.target.value)}
            >
              <option value="">{t('webcam.default', 'Browser default')}</option>
              {devices.map((d, i) => (
                <option key={d.deviceId} value={d.deviceId}>
                  {d.label || 'Camera ' + (i + 1)}
                </option>
              ))}
            </select>
          </div>
          <div aria-live="polite" className="text-xs text-neutral-400 empty:hidden">
            {listError || share.error || statusText}
          </div>
        </div>
      }
    />
  );
}
