import { signal } from '@preact/signals';

export interface Sample {
  at: number;
  state: string;
  sessions: number;
  visitors: number;
}

const MAX_SAMPLES = 60;
const PERIOD_MS = 15_000;

/**
 * The backend reports totals, not time series, so the dashboard samples what it
 * already receives. It is a session view: nothing is persisted, and a restart
 * begins a new history.
 */
export const samples = signal<Record<string, Sample[]>>({});

/** Fleet totals per tick, for the dashboard strip. */
export interface FleetSample {
  at: number;
  online: number;
  sessions: number;
}

export const fleetSamples = signal<FleetSample[]>([]);

let timer: ReturnType<typeof setInterval> | null = null;

export function recordSample(tunnel: { id: string; state: string; activeSessions?: number; visitors?: number }): void {
  const current = samples.value[tunnel.id] ?? [];
  const next: Sample = {
    at: Date.now(),
    state: tunnel.state,
    sessions: tunnel.activeSessions ?? 0,
    visitors: tunnel.visitors ?? 0,
  };

  samples.value = {
    ...samples.value,
    [tunnel.id]: [...current, next].slice(-MAX_SAMPLES),
  };
}

export function startHistorySampler(
  read: () => { id: string; state: string; activeSessions?: number; visitors?: number }[],
): void {
  if (timer) return;

  const tick = () => {
    const current = read();
    for (const tunnel of current) recordSample(tunnel);

    fleetSamples.value = [
      ...fleetSamples.value,
      {
        at: Date.now(),
        online: current.filter((tunnel) => tunnel.state === 'online').length,
        sessions: current.reduce((sum, tunnel) => sum + (tunnel.activeSessions ?? 0), 0),
      },
    ].slice(-MAX_SAMPLES);
  };

  tick();
  timer = setInterval(tick, PERIOD_MS);
}

export function stopHistorySampler(): void {
  if (!timer) return;
  clearInterval(timer);
  timer = null;
}

export function historyFor(id: string): Sample[] {
  return samples.value[id] ?? [];
}

export function forgetHistory(id: string): void {
  const { [id]: _removed, ...rest } = samples.value;
  samples.value = rest;
}
