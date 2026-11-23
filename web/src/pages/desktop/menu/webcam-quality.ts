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
  private beforeCut?: ReturnType<typeof delivery>;
  private beforeProbe?: Target;
  private cleanMs = 0;
  private nextProbeAt = 0;
  private probeFpsNext = true;

  private change(mbps: number, fps: number) {
    const next = {
      mbps: Math.max(minMbps, Math.min(maxMbps, Math.round(mbps * 100) / 100)),
      fps: Math.max(minFps, Math.min(this.hostFps, Math.floor(fps)))
    };
    if (next.mbps === this.target.mbps && next.fps === this.target.fps) return false;
    this.target = next;
    this.cleanMs = 0;
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
    this.beforeCut = undefined;
    this.beforeProbe = undefined;
    this.cleanMs = 0;
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
      this.cleanMs = 0;
      this.samples = [];
      this.beforeCut = undefined;
      this.beforeProbe = undefined;
      return;
    }
    this.cleanMs =
      s.missed === 0 && s.received > 0 && !s.queueGrew ? this.cleanMs + s.elapsedMs : 0;
    this.samples.push(s);
    if (this.samples.length > 10) this.samples.shift();
    if (this.samples.length >= 3) this.evaluate(this.samples, now);
  }

  private evaluate(samples: DeliverySample[], now: number) {
    const recent = delivery(samples.slice(-3));
    const severe = recent.loss > 0.25;
    if (!severe && samples.length < 10) return;
    const window = delivery(samples);
    const evidence = severe ? recent : window;
    const { mbps, fps } = this.target;

    if (evidence.loss > 0.1) {
      this.nextProbeAt = Math.max(this.nextProbeAt, now + (this.beforeProbe ? 20000 : 10000));
      if (this.beforeProbe) {
        const previous = this.beforeProbe;
        this.beforeProbe = undefined;
        this.beforeCut = undefined;
        this.change(previous.mbps, previous.fps);
        return;
      }
      const prior = this.beforeCut;
      // If smaller JPEGs still take just as long and deliver no more fps,
      // match capture pacing to the observed delivery ceiling.
      const flatCadence =
        prior &&
        prior.received >= 3 &&
        evidence.received >= 3 &&
        evidence.frameBytes <= prior.frameBytes * 0.95 &&
        // A bandwidth-limited transfer speeds up with smaller JPEGs. A
        // grant-limited one recovers little of that relative size reduction.
        evidence.grantMs >=
          prior.grantMs * Math.max(0.9, 1 - 0.4 * (1 - evidence.frameBytes / prior.frameBytes)) &&
        evidence.fps <= prior.fps * 1.1;
      const sustainableFps = Math.max(minFps, Math.floor(evidence.fps));
      if (evidence.received >= 3 && (flatCadence || mbps === minMbps) && sustainableFps < fps) {
        // Preserve the per-frame JPEG budget while changing motion cadence.
        this.change((mbps * sustainableFps) / fps, sustainableFps);
        this.beforeCut = undefined;
      } else {
        // Estimate what this JPEG size would cost at the requested cadence.
        // This also acts when simple scenes already undershoot the bitrate target.
        const used = (evidence.frameBytes * 8 * fps) / 1e6;
        const effective = used > 0 ? Math.min(mbps, used) : mbps;
        if (this.change(effective * (severe ? 0.8 : 0.9), fps)) this.beforeCut = evidence;
      }
      return;
    }
    if (samples.length < 10) return;
    this.beforeCut = undefined;
    this.beforeProbe = undefined;
    if (window.loss >= 0.03 || now < this.nextProbeAt || samples.some((s) => s.queueGrew)) return;
    const split = Math.floor(samples.length / 2);
    const early = delivery(samples.slice(0, split));
    const late = delivery(samples.slice(split));
    if (!early.received || !late.received || late.grantMs > early.grantMs * 1.15 + 3) return;

    const canRaiseRate = mbps < maxMbps && this.cleanMs >= 10000;
    const canRaiseFps = fps < this.hostFps && window.fps >= fps * 0.9;
    const previous = this.target;
    if (canRaiseFps && ((fps < 25 && this.probeFpsNext) || !canRaiseRate)) {
      this.change(mbps, fps + Math.max(1, Math.round(fps * 0.1)));
      this.probeFpsNext = false;
    } else if (canRaiseRate) {
      this.change(mbps * 1.1, fps);
      this.probeFpsNext = true;
    }
    if (previous !== this.target) this.beforeProbe = previous;
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
