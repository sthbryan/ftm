import { signal } from '@preact/signals';
import { play, unlock, type Emphasis, type SoundName } from 'cuelume';
import { prefersReducedMotion } from '@/lib/utils/motion';

/**
 * Every sound ftm makes goes through here. The spec maps four moments to
 * Cuelume cues: a connection online (success), a failure (error), an expiry
 * coming (attention) and a player joining (a subtle ready). Everything else is
 * silent. Sound follows the user's setting and stays off under reduced motion.
 */
export const soundOn = signal(true);

export type CueEvent = 'success' | 'error' | 'warning' | 'alert' | 'joined' | string;

export interface Cue {
  name: SoundName;
  emphasis?: Emphasis;
}

const CUES: Record<string, Cue> = {
  success: { name: 'success' },
  error: { name: 'error' },
  warning: { name: 'attention' },
  alert: { name: 'attention' },
  joined: { name: 'ready', emphasis: 'subtle' },
};

export function cueFor(event: CueEvent): Cue | null {
  return CUES[event] ?? null;
}

export function soundAllowed(): boolean {
  return soundOn.value && !prefersReducedMotion();
}

export function cue(event: CueEvent): void {
  const found = cueFor(event);
  if (!found || !soundAllowed()) return;
  play(found.name, found.emphasis ? { emphasis: found.emphasis } : undefined);
}

/** Call inside a click so the browser lets later cues play. */
export function unlockSound(): void {
  if (!soundAllowed()) return;
  void unlock().catch(() => false);
}
