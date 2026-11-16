import { signal } from '@preact/signals';
import { api } from '@/lib/api/client';

/**
 * What `/api/detect-port` saw: ports something listens on (`busyPorts`), the
 * ones whose front page is Foundry VTT (`foundryPorts`), and every port it
 * checked (`scannedPorts`): the usual Foundry ports plus the one asked for.
 * The busy and Foundry lists are null until the first answer or after a failure.
 */
export const busyPorts = signal<number[] | null>(null);
export const foundryPorts = signal<number[] | null>(null);
export const scannedPorts = signal<number[]>([]);
export const checkingPorts = signal(false);

/** What a running Foundry says about itself (its /api/status). */
export type FoundryWorld = {
  port: number;
  active: boolean;
  version?: string;
  world?: string;
  system?: string;
  users: number;
  uptimeMs?: number;
};

export const foundryWorlds = signal<FoundryWorld[]>([]);

/** Short words that stay lowercase inside a title, in the app's languages. */
const MINOR = new Set(['a', 'an', 'and', 'of', 'the', 'in', 'on', 'to', 'de', 'del', 'la', 'el', 'los', 'las', 'y', 'do', 'da', 'e']);

/** "curse-of-strahd" → "Curse of Strahd": Foundry reports the world's id. */
export function worldTitle(id: string): string {
  return id
    .split(/[-_\s]+/)
    .filter(Boolean)
    .map((word, index) =>
      index > 0 && MINOR.has(word.toLowerCase()) ? word.toLowerCase() : word.charAt(0).toUpperCase() + word.slice(1),
    )
    .join(' ');
}

export function worldOn(port: number): FoundryWorld | undefined {
  return foundryWorlds.value.find((entry) => entry.port === port);
}

type DetectAnswer = {
  ports?: number[];
  scanned?: number[];
  foundry?: number[];
  worlds?: FoundryWorld[] | null;
};

let watchedPort: number | undefined;

export async function checkFoundry(port: number | undefined = watchedPort): Promise<void> {
  checkingPorts.value = true;
  try {
    const searchParams = port ? { port } : undefined;
    const data = await api.get('detect-port', { searchParams, retry: 0 }).json<DetectAnswer>();
    busyPorts.value = data.ports ?? [];
    foundryPorts.value = data.foundry ?? [];
    scannedPorts.value = data.scanned ?? [];
    foundryWorlds.value = data.worlds ?? [];
  } catch {
    busyPorts.value = null;
    foundryPorts.value = null;
    scannedPorts.value = [];
    foundryWorlds.value = [];
  } finally {
    checkingPorts.value = false;
  }
}

/** Also check this port on every poll, e.g. the connection on screen. */
export function watchPort(port: number | undefined): void {
  if (port === watchedPort) return;
  watchedPort = port;
  void checkFoundry(port);
}

/** How often the header looks for Foundry while the window is visible. */
const POLL_MS = 5000;

/**
 * Keeps the Foundry lists fresh: a check now, then one every few seconds while
 * the page is visible, and one as soon as it becomes visible again. Returns the
 * cleanup.
 */
export function watchFoundry(): () => void {
  let timer: ReturnType<typeof setInterval> | undefined;

  const start = () => {
    stop();
    void checkFoundry();
    timer = setInterval(() => void checkFoundry(), POLL_MS);
  };
  const stop = () => {
    if (timer) clearInterval(timer);
    timer = undefined;
  };
  const onVisibility = () => (document.visibilityState === 'visible' ? start() : stop());

  start();
  document.addEventListener('visibilitychange', onVisibility);
  return () => {
    stop();
    document.removeEventListener('visibilitychange', onVisibility);
  };
}
