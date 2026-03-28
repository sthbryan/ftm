import { useNotifications } from './notification.svelte';
import type { Tunnel } from '$lib/types';

const DEFAULT_THRESHOLDS = [30, 15, 10, 5, 1];
const timers = new Map<string, ReturnType<typeof setTimeout>>();

function getThresholds(): number[] {
  const saved = localStorage.getItem('ftm-expiration-thresholds');
  return saved ? JSON.parse(saved) : DEFAULT_THRESHOLDS;
}

function setThresholdsFn(thresholds: number[]) {
  localStorage.setItem('ftm-expiration-thresholds', JSON.stringify(thresholds));
}

function start(tunnel: Tunnel) {
  const notifications = useNotifications();
  if (!tunnel.expiresAt || !notifications.enabled) return;
  
  const thresholds = getThresholds();
  const expiresAt = new Date(tunnel.expiresAt).getTime();
  const now = Date.now();
  
  thresholds.forEach(mins => {
    const triggerAt = expiresAt - (mins * 60 * 1000);
    if (triggerAt <= now) return;
    
    const key = `${tunnel.id}-${mins}`;
    const delay = triggerAt - now;
    
    timers.set(key, setTimeout(() => {
      notifications.notifyExpiring(tunnel.name, mins);
      timers.delete(key);
    }, delay));
  });
}

function stop(tunnelId: string) {
  for (const [key, timer] of timers) {
    if (key.startsWith(tunnelId + '-')) {
      clearTimeout(timer);
      timers.delete(key);
    }
  }
}

function stopAll() {
  for (const timer of timers.values()) {
    clearTimeout(timer);
  }
  timers.clear();
}

export function useExpirationMonitor() {
  return { start, stop, stopAll, getThresholds, setThresholds: setThresholdsFn };
}
