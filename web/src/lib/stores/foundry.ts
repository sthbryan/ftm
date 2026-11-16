import { signal } from '@preact/signals';
import { api } from '@/lib/api/client';

/**
 * Which ports have something listening, from `/api/detect-port`: the usual
 * Foundry ports plus the one asked for. Null until the first answer, or when
 * the check failed. `scannedPorts` lists every port the server checked.
 */
export const busyPorts = signal<number[] | null>(null);
export const scannedPorts = signal<number[]>([]);
export const checkingPorts = signal(false);

export async function checkFoundry(port?: number): Promise<void> {
  checkingPorts.value = true;
  try {
    const searchParams = port ? { port } : undefined;
    const data = await api
      .get('detect-port', { searchParams })
      .json<{ ports?: number[]; scanned?: number[] }>();
    busyPorts.value = data.ports ?? [];
    scannedPorts.value = data.scanned ?? [];
  } catch {
    busyPorts.value = null;
    scannedPorts.value = [];
  } finally {
    checkingPorts.value = false;
  }
}
