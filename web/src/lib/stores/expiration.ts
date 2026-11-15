import { notificationsEnabled, notifyExpiring } from './notification';
import type { Tunnel } from '@/lib/types';

const DEFAULT_THRESHOLDS = [30, 15, 10, 5, 1];
const STORAGE_KEY = 'ftm-expiration-thresholds';

const timers = new Map<string, ReturnType<typeof setTimeout>>();

export function getThresholds(): number[] {
  const saved = localStorage.getItem(STORAGE_KEY);
  if (!saved) return DEFAULT_THRESHOLDS;
  try {
    return JSON.parse(saved) as number[];
  } catch {
    return DEFAULT_THRESHOLDS;
  }
}

export function setThresholds(thresholds: number[]): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(thresholds));
}

export function startExpiration(tunnel: Tunnel): void {
  if (!tunnel.expiresAt || !notificationsEnabled.value) return;

  const expiresAt = tunnel.expiresAt;
  const current = Date.now();

  for (const minutes of getThresholds()) {
    const triggerAt = expiresAt - minutes * 60 * 1000;
    if (triggerAt <= current) continue;

    const key = `${tunnel.id}-${minutes}`;
    timers.set(
      key,
      setTimeout(() => {
        notifyExpiring(tunnel.name, minutes);
        timers.delete(key);
      }, triggerAt - current),
    );
  }
}

export function stopExpiration(tunnelId: string): void {
  for (const [key, timer] of timers) {
    if (key.startsWith(`${tunnelId}-`)) {
      clearTimeout(timer);
      timers.delete(key);
    }
  }
}

export function stopAllExpirations(): void {
  for (const timer of timers.values()) clearTimeout(timer);
  timers.clear();
}
