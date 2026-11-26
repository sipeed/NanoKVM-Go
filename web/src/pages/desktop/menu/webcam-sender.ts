import { releaseFrame, type CameraFrame, type webcamFrames } from './webcam-frames.ts';
import { FramePacer, startupSeconds, WebcamQuality } from './webcam-quality.ts';

type ActiveStatus = 'waiting' | 'calibrating' | 'sharing';
export type WebcamReport = { status: ActiveStatus; sample: string } | { error: string };

export function sendWebcam(
  url: string,
  source: ReturnType<typeof webcamFrames>,
  report: (message: WebcamReport) => void
) {
  const endpoint = new URL(url);
  endpoint.searchParams.set('maxFps', '30');
  endpoint.searchParams.set('calibrate', '1');
  const ws = new WebSocket(endpoint.toString());
  const control = new WebcamQuality();
  let pacer = new FramePacer();
  let quality = 0.48;
  let jpegBytes = 0;
  let jpegCount = 0;
  let demand = false;
  let grant = false;
  let probeGrant = false;
  let startupPending = true;
  let startupAt: number | undefined;
  let finishSample: (() => void) | undefined;
  let calibratedTarget = control.target;
  let receivedAny = false;
  let inFlight: { bytes: number; sentAt: number; generation: number } | undefined;
  let pending: CameraFrame | undefined;
  let encoding = false;
  let ready: { jpeg: Blob; capturedAt: number } | undefined;
  let pendingAt = 0;
  let generation = 0;
  let stopped = false;
  let status: ActiveStatus | undefined;
  const noSample = '— Mbps · — fps · — ms';
  let liveSample = noSample;
  let poll: ReturnType<typeof setTimeout>;
  let watchdog: ReturnType<typeof setTimeout>;
  let sample: ReturnType<typeof setTimeout>;
  let eligible = 0,
    missed = 0,
    received = 0,
    bytes = 0,
    grantMs = 0;

  function clearPending() {
    releaseFrame(pending);
    pending = undefined;
  }
  function stop() {
    if (stopped) return;
    stopped = true;
    clearTimeout(poll);
    clearTimeout(watchdog);
    clearTimeout(sample);
    clearPending();
    ready = undefined;
    source.close();
    ws.close();
  }
  function fail(error: string) {
    if (stopped) return;
    stop();
    report({ error });
  }
  function noFrames() {
    fail('No camera frames reached the Go+. Check the camera and start again.');
  }
  function watch() {
    clearTimeout(watchdog);
    watchdog = setTimeout(() => fail('The Go+ stopped responding.'), 12000);
  }
  function setStatus(next: ActiveStatus) {
    if (next === status) return;
    status = next;
    const sample =
      next === 'waiting'
        ? `${calibratedTarget.mbps.toFixed(2)} Mbps · ${calibratedTarget.fps} fps`
        : liveSample;
    report({ status: next, sample });
  }
  function finishStartup() {
    control.finishStartup();
    calibratedTarget = control.target;
    startupAt = undefined;
  }
  function measure(epoch: number, second: number) {
    if (stopped || !demand) return;
    eligible = missed = received = bytes = grantMs = 0;
    const start = performance.now();
    const queue = ws.bufferedAmount;
    finishSample = () => {
      finishSample = undefined;
      clearTimeout(sample);
      const now = performance.now();
      const elapsed = now - start;
      const previous = control.target;
      const observation = {
        elapsedMs: elapsed,
        eligible,
        missed,
        received,
        bytes,
        grantMs,
        // A freshly queued JPEG is normal with one upload in flight.
        queueGrew:
          ws.bufferedAmount > queue &&
          !!inFlight &&
          now - inFlight.sentAt > 2000 / control.target.fps
      };
      // Anchor the sample deadlines so timer drift cannot drop the fifth window.
      const completed = Math.max(second, Math.round((now - epoch) / 1000));
      if (
        startupAt !== undefined &&
        completed >= startupSeconds &&
        !receivedAny &&
        !inFlight &&
        !probeGrant
      ) {
        noFrames();
        return;
      }
      if (startupAt !== undefined && completed <= startupSeconds) {
        control.observeStartup(observation, completed);
        if (completed === startupSeconds) {
          finishStartup();
        }
      } else {
        if (startupAt !== undefined) finishStartup();
        control.observe(observation, now);
      }
      if (previous !== control.target) resetEncoder();
      const fps = ((received * 1000) / elapsed).toFixed(1);
      const mbps = ((bytes * 0.008) / elapsed).toFixed(2);
      const delay = received ? Math.round(grantMs / received) : '—';
      liveSample = `${mbps} Mbps · ${fps} fps · ${delay} ms`;
      if (status === 'calibrating' || status === 'sharing') report({ status, sample: liveSample });
      measure(epoch, completed + 1);
    };
    sample = setTimeout(finishSample, Math.max(0, epoch + second * 1000 - start));
  }
  function resetEncoder() {
    generation++;
    ready = undefined;
    clearPending();
    pacer = new FramePacer();
    jpegBytes = jpegCount = 0;
  }
  function startWindows() {
    clearTimeout(sample);
    if (!demand) return;
    liveSample = noSample;
    measure(performance.now(), 1);
  }

  // One grant permits one send. While it is outstanding we can prepare the
  // next JPEG. Keep one prepared image and only the most recent raw capture.
  function pump() {
    if (stopped || !demand || ws.readyState !== WebSocket.OPEN) return;
    if (ready && grant) {
      const { jpeg, capturedAt } = ready;
      ready = undefined;
      eligible++;
      if (pending && performance.now() - capturedAt > Math.max(80, 2000 / control.target.fps)) {
        // A stalled grant made this JPEG stale. Use the newest capture.
        missed++;
      } else {
        clearTimeout(poll);
        grant = false;
        ws.send(jpeg);
        inFlight = { bytes: jpeg.size, sentAt: performance.now(), generation };
        setStatus(probeGrant ? 'calibrating' : 'sharing');
      }
    }
    if (pending && !ready && !encoding) {
      const frame = pending;
      pending = undefined;
      void prepare(frame, pendingAt);
    }
  }

  async function prepare(frame: CameraFrame, capturedAt: number) {
    encoding = true;
    const version = generation;
    try {
      const jpeg = await source.encode(frame, quality);
      if (stopped || !demand || version !== generation) return;
      if (!jpeg || jpeg.size > 131072) {
        eligible++;
        if (jpeg) quality = Math.max(0.08, quality * 0.82);
        return;
      }
      ready = { jpeg, capturedAt };
      jpegBytes += jpeg.size;
      const { mbps, fps } = control.target;
      if (++jpegCount >= Math.ceil(fps / 2)) {
        const budget = (mbps * 1e6) / (8 * fps);
        const step = Math.max(
          -0.12,
          Math.min(0.12, 0.35 * Math.log((budget * jpegCount) / jpegBytes))
        );
        quality = Math.max(0.08, Math.min(0.86, quality + step));
        jpegBytes = jpegCount = 0;
      }
    } catch (err) {
      fail(err instanceof Error ? err.message : 'Camera capture failed.');
    } finally {
      releaseFrame(frame);
      encoding = false;
      pump();
    }
  }

  void source
    .listen((frame, timeMs) => {
      if (!demand || stopped) {
        releaseFrame(frame);
        return;
      }
      if (!pacer.accept(timeMs, control.target.fps)) {
        releaseFrame(frame);
        return;
      }
      if (pending) {
        eligible++;
        if (inFlight && ready) missed++;
        clearPending();
      }
      pending = frame;
      pendingAt = performance.now();
      pump();
    })
    .catch((err) => fail(err instanceof Error ? err.message : 'Camera capture failed.'));
  watch();
  ws.onclose = (e) => fail(e.reason || 'The Go+ closed the camera connection.');
  ws.onerror = () => fail('Could not connect to the Go+ camera service.');
  ws.onmessage = ({ data }) => {
    if (stopped) return;
    watch();
    clearTimeout(poll);
    if (inFlight) receivedAny = true;
    if (inFlight && inFlight.generation === generation) {
      received++;
      bytes += inFlight.bytes;
      grantMs += performance.now() - inFlight.sentAt;
    }
    inFlight = undefined;
    if (data !== 'calibrate' && data !== 'idle' && data !== '720p25' && data !== '720p30')
      return fail('Unexpected camera service response.');
    control.setHostFps(data === '720p25' ? 25 : 30);
    // An idle grant ends the server's complete startup probe. It also
    // acknowledges its final upload, including one from a previous target.
    if (data === 'idle' && probeGrant && !receivedAny) {
      noFrames();
      return;
    }
    probeGrant = data === 'calibrate';
    if (data === 'idle') {
      if (demand) {
        // The Go+ can send idle just before the local final sample timer fires.
        if (
          startupAt !== undefined &&
          performance.now() - startupAt >= (startupSeconds - 0.1) * 1000
        ) {
          finishSample?.();
          if (stopped) return;
        }
        finishSample = undefined;
        clearTimeout(sample);
        resetEncoder();
        if (startupAt !== undefined) finishStartup();
        control.resumeAfterIdle();
      }
      demand = grant = false;
      setStatus('waiting');
      poll = setTimeout(() => {
        if (!stopped) ws.send('poll');
      }, 400);
      return;
    }
    if (!demand) {
      demand = true;
      if (startupPending) {
        startupPending = false;
        startupAt = performance.now();
      }
      startWindows();
    }
    if (probeGrant) setStatus('calibrating');
    grant = true;
    // Keep checking demand if the source stalls or becomes unavailable.
    poll = setTimeout(() => {
      if (!grant || stopped) return;
      grant = false;
      ws.send('poll');
    }, 250);
    pump();
  };
  return { stop };
}
