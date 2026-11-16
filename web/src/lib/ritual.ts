import { isRunningState } from '@/lib/utils/status';
import type { Tunnel } from '@/lib/types';

/**
 * The four moments of opening a table (ready, opening, invite, in session),
 * plus the one nobody wants (failed). Each running connection is in exactly one.
 */
export type Moment = 'ready' | 'opening' | 'invite' | 'session' | 'failed';

/** Ports `/api/detect-port` scans for a running Foundry. */
export const FOUNDRY_PORTS = [
  30000, 30001, 30002, 30003, 30004, 30005, 30006, 30007, 30008, 30009,
];

/** Below this many milliseconds left, the expiry turns into a warning. */
export const EXPIRY_WARNING_MS = 10 * 60_000;

/**
 * A connection is in the invite moment until a player joins or the GM moves on;
 * after that it lives as a strip.
 */
export function momentOf(tunnel: Tunnel, invited: ReadonlySet<string>): Moment {
  if (tunnel.state === 'error' || tunnel.state === 'timeout') return 'failed';
  if (tunnel.state === 'online') {
    return (tunnel.activeSessions ?? 0) > 0 || invited.has(tunnel.id) ? 'session' : 'invite';
  }
  if (tunnel.state === 'stopping') return 'session';
  if (isRunningState(tunnel.state)) return 'opening';
  return 'ready';
}

export interface Stage {
  moment: Moment;
  tunnel: Tunnel | null;
}

export interface Arrangement {
  /** The one moment that takes the screen, or null when only strips are left. */
  stage: Stage | null;
  /** Running connections not on the stage, stacked as slim strips. */
  strips: Tunnel[];
  /** Saved connections that are not running. */
  presets: Tunnel[];
}

const STAGE_ORDER: Moment[] = ['opening', 'invite', 'failed'];

/** Decide what takes the screen and what becomes a strip. */
export function arrange(
  list: Tunnel[],
  selectedId: string | null,
  invited: ReadonlySet<string>,
): Arrangement {
  const withMoment = list.map((tunnel) => ({ tunnel, moment: momentOf(tunnel, invited) }));

  let stage: Stage | null = null;
  for (const moment of STAGE_ORDER) {
    const found = withMoment.find((item) => item.moment === moment);
    if (found) {
      stage = { moment, tunnel: found.tunnel };
      break;
    }
  }

  const strips = withMoment
    .filter(
      (item) =>
        item.tunnel !== stage?.tunnel && item.moment !== 'ready' && item.moment !== 'failed',
    )
    .map((item) => item.tunnel);

  // Failed connections the stage cannot show stay reachable as presets, to retry.
  const presets = withMoment
    .filter(
      (item) =>
        item.moment === 'ready' || (item.moment === 'failed' && item.tunnel !== stage?.tunnel),
    )
    .map((item) => item.tunnel);

  if (!stage && strips.length === 0) {
    const selected = presets.find((tunnel) => tunnel.id === selectedId) ?? presets[0] ?? null;
    stage = { moment: 'ready', tunnel: selected };
  }

  return { stage, strips, presets };
}

export type ExpiryLevel = 'none' | 'later' | 'soon';

export function expiryLevel(remainingMs: number): ExpiryLevel {
  if (remainingMs <= 0) return 'none';
  return remainingMs <= EXPIRY_WARNING_MS ? 'soon' : 'later';
}

export type FoundryCheck = 'answers' | 'silent' | 'unknown';

/**
 * Whether Foundry answers on a port, from the busy ports detect-port reported.
 * `scanned` is every port the server checked; without it, the usual ports.
 */
export function foundryCheck(
  port: number,
  busy: number[] | null,
  scanned: readonly number[] = FOUNDRY_PORTS,
): FoundryCheck {
  if (!busy || !scanned.includes(port)) return 'unknown';
  return busy.includes(port) ? 'answers' : 'silent';
}

export function inviteMessage(template: string, name: string, url: string): string {
  return template.replaceAll('{name}', name).replaceAll('{url}', url);
}

/** New players across connections that were already known in `before`. */
export function playersJoined(
  before: Record<string, number>,
  after: Record<string, number>,
): number {
  let joined = 0;
  for (const [id, count] of Object.entries(after)) {
    const previous = before[id];
    if (previous !== undefined && count > previous) joined += count - previous;
  }
  return joined;
}
