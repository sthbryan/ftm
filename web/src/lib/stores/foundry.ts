import { signal } from '@preact/signals';
import { api } from '@/lib/api/client';

/**
 * Which of the usual Foundry ports have something listening, from
 * `/api/detect-port`. Null until the first answer, or when the check failed.
 */
export const busyPorts = signal<number[] | null>(null);
export const checkingPorts = signal(false);

export async function checkFoundry(): Promise<void> {
  checkingPorts.value = true;
  try {
    const data = await api.get('detect-port').json<{ ports?: number[] }>();
    busyPorts.value = data.ports ?? [];
  } catch {
    busyPorts.value = null;
  } finally {
    checkingPorts.value = false;
  }
}
