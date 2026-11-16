import { effect, signal } from '@preact/signals';
import { playersJoined } from '@/lib/ritual';
import { cue } from '@/lib/sound';
import { tunnels } from './tunnels';

/** Online connections the GM has already invited to; they live as strips. */
export const invited = signal<ReadonlySet<string>>(new Set());

/** The preset the ready moment opens. */
export const selectedPreset = signal<string | null>(null);

export function markInvited(id: string): void {
  invited.value = new Set([...invited.value, id]);
}

let stopWatching: (() => void) | null = null;

/**
 * Play the subtle "ready" cue when a player joins, and forget an invite once its
 * connection stops so the next session starts with the invite again.
 */
export function watchSession(): () => void {
  if (stopWatching) return stopWatching;

  let before: Record<string, number> = {};
  const dispose = effect(() => {
    const after: Record<string, number> = {};
    const stillOnline = new Set<string>();
    for (const tunnel of tunnels.value) {
      if (tunnel.state !== 'online') continue;
      after[tunnel.id] = tunnel.activeSessions ?? 0;
      stillOnline.add(tunnel.id);
    }

    if (playersJoined(before, after) > 0) cue('joined');
    before = after;

    const current = invited.peek();
    if ([...current].some((id) => !stillOnline.has(id))) {
      invited.value = new Set([...current].filter((id) => stillOnline.has(id)));
    }
  });

  stopWatching = () => {
    dispose();
    stopWatching = null;
  };
  return stopWatching;
}
