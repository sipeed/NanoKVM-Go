const minMbps = 2;
const maxMbps = 10;
const minFps = 15;
const maxFps = 30;
const startMbps = 8;
const startFps = 25;
const warmupSeconds = 1;
const evidenceSeconds = 2;
export const startupSeconds = warmupSeconds + 5 * evidenceSeconds;

type Target = { mbps: number; fps: number };

export type DeliverySample = {
  elapsedMs: number;
  eligible: number;
  missed: number;
  received: number;
  bytes: number;
  grantMs: number; // Sum of send-to-next-grant times for acknowledged JPEGs.
  queueGrew: boolean;
};

function delivery(samples: DeliverySample[]) {
  let eligible = 0,
    missed = 0,
    received = 0,
    bytes = 0,
    ms = 0,
    grantMs = 0;
  for (const s of samples) {
    eligible += s.eligible;
    missed += s.missed;
    received += s.received;
    bytes += s.bytes;
    ms += s.elapsedMs;
    grantMs += s.grantMs;
  }
  return {
    loss: eligible ? missed / eligible : NaN,
    fps: (received * 1000) / ms,
    frameBytes: received ? bytes / received : 0,
    grantMs: received ? grantMs / received : 0,
    received
  };
}

export class WebcamQuality {
  target: Target = { mbps: startMbps, fps: startFps };
  private hostFps = maxFps;
  private samples: DeliverySample[] = [];
  private startupSamples: DeliverySample[] = [];
  private lastStartupSecond = 0;
  private good?: Target;
  private badMbps?: number;
  private beforeProbe?: Target;
  private nextProbeAt = 0;

  private change(mbps: number, fps: number) {
    const next = {
      mbps: Math.max(minMbps, Math.min(maxMbps, Math.round(mbps * 100) / 100)),
      fps: Math.max(minFps, Math.min(this.hostFps, Math.floor(fps)))
    };
    if (next.mbps === this.target.mbps && next.fps === this.target.fps) return false;
    this.target = next;
    this.samples = [];
    return true;
  }

  setHostFps(fps: number) {
    this.hostFps = fps;
    if (this.target.fps > fps) {
      this.change(this.target.mbps, fps);
      this.resumeAfterIdle();
    }
  }

  resumeAfterIdle() {
    this.startupSamples = [];
    this.samples = [];
    this.beforeProbe = undefined;
  }

  observeStartup(s: DeliverySample, second: number) {
    if (second <= warmupSeconds || second > startupSeconds) return;
    const slot = (second - warmupSeconds - 1) % evidenceSeconds;
    if (slot === 0 || second !== this.lastStartupSecond + 1) this.startupSamples = [];
    this.lastStartupSecond = second;
    if (
      slot !== this.startupSamples.length ||
      s.elapsedMs < 900 ||
      s.elapsedMs > 1500 ||
      !s.eligible
    ) {
      this.startupSamples = [];
      return;
    }
    this.startupSamples.push(s);
    if (slot + 1 < evidenceSeconds && second < startupSeconds) return;

    const samples = this.startupSamples;
    this.startupSamples = [];
    const window = delivery(samples);
    const queued = samples.filter((sample) => sample.queueGrew).length;
    const split = Math.floor(samples.length / 2);
    const early = delivery(samples.slice(0, split));
    const late = delivery(samples.slice(split));
    const risingDelay =
      early.received > 0 && late.received > 0 && late.grantMs > early.grantMs * 1.15 + 3;
    const { mbps } = this.target;
    const congested = window.loss > 0.1 || queued === samples.length || (queued > 0 && risingDelay);
    // A quiet scene can deliver every frame without exercising this bitrate.
    const loaded = (window.frameBytes * window.fps * 8) / 1e6 >= mbps * 0.85;
    const healthy =
      loaded &&
      window.loss < 0.03 &&
      queued === 0 &&
      !risingDelay &&
      samples.every((sample) => sample.received > 0);

    if (congested) {
      this.badMbps = Math.min(this.badMbps ?? maxMbps, mbps);
      if (this.good && this.good.mbps >= mbps) this.good = undefined;
    } else if (healthy) {
      this.good = this.target;
    } else return;

    if (second === startupSeconds) return;
    let next: number;
    if (healthy && this.badMbps === undefined) next = maxMbps;
    else if (!this.good && startupSeconds - second <= evidenceSeconds * 2) next = minMbps;
    else {
      // Targets are stored to hundredths of a Mbps. Bisect on that same grid.
      const lower = Math.round((this.good?.mbps ?? minMbps) * 100);
      const upper = Math.round((this.badMbps ?? maxMbps) * 100);
      next = Math.round((lower + upper) / 2) / 100;
    }
    // Keep JPEGs larger on constrained links by lowering cadence more gently than bitrate.
    const fps =
      next <= startMbps
        ? startFps * Math.sqrt(next / startMbps)
        : startFps + ((maxFps - startFps) * (next - startMbps)) / (maxMbps - startMbps);
    this.change(next, Math.round(fps));
  }

  finishStartup() {
    if (this.good) this.change(this.good.mbps, this.good.fps);
    else if (this.badMbps !== undefined && this.target.mbps >= this.badMbps)
      this.change(minMbps, minFps);
    this.good = undefined;
    this.badMbps = undefined;
    this.startupSamples = [];
    this.lastStartupSecond = 0;
  }

  observe(s: DeliverySample, now: number) {
    if (s.elapsedMs > 1500 || s.elapsedMs < 900 || s.eligible === 0) {
      this.samples = [];
      this.beforeProbe = undefined;
      return;
    }
    this.samples.push(s);
    if (this.samples.length > 5) this.samples.shift();
    if (this.samples.length < 3) return;

    const recent = delivery(this.samples.slice(-3));
    const severe = recent.loss > 0.25;
    if (!severe && this.samples.length < 5) return;
    const window = severe ? recent : delivery(this.samples);
    const { mbps, fps } = this.target;
    if (window.loss > 0.1) {
      const previous = this.beforeProbe;
      this.beforeProbe = undefined;
      this.nextProbeAt = Math.max(this.nextProbeAt, now + (previous ? 20000 : 10000));
      if (previous) {
        this.change(previous.mbps, previous.fps);
      } else {
        // Base the cut on the JPEG budget actually used at the requested cadence.
        const usedMbps = (window.frameBytes * 8 * fps) / 1e6;
        const effective = usedMbps > 0 ? Math.min(mbps, usedMbps) : mbps;
        const factor = severe ? 0.8 : 0.9;
        this.change(effective * factor, Math.round(fps * factor));
      }
      return;
    }

    if (window.loss >= 0.03 || this.samples.some((sample) => sample.queueGrew)) return;
    const early = delivery(this.samples.slice(0, 2));
    const late = delivery(this.samples.slice(2));
    if (!early.received || !late.received || late.grantMs > early.grantMs * 1.15 + 3) return;
    this.beforeProbe = undefined;
    if (now < this.nextProbeAt) return;

    const previous = this.target;
    const nextFps = window.fps >= fps * 0.9 ? Math.round(fps * 1.1) : fps;
    if (this.change(mbps * 1.15, nextFps)) this.beforeProbe = previous;
  }
}

export class FramePacer {
  private previous?: number;
  private credit = 1.2;

  accept(timeMs: number, fps: number) {
    // Keep up to one spare frame of credit across irregular camera arrivals.
    // The Go+ still grants at most 30 uploads per second. Credit lets an early
    // real frame use a slot left empty by a late or absent camera frame.
    const elapsed = this.previous === undefined ? 0 : timeMs - this.previous;
    this.previous = timeMs;
    this.credit = elapsed < 0 ? 1.2 : Math.min(2, this.credit + (elapsed * fps) / 1000);
    if (this.credit < 1 - 1e-8) return false;
    this.credit -= 1;
    return true;
  }
}
