import { signal } from '@preact/signals';
import { api } from '@/lib/api/client';
import { subscribeWsMessages } from '@/lib/api/ws';

/** One provider binary as `/api/providers/installed` reports it. */
export interface InstalledProvider {
  id: string;
  name: string;
  installed: boolean;
  version: string;
  latest: string;
  source: string;
  hasUpdate: boolean;
  updatable: boolean;
  updating: boolean;
  error?: string;
}

/**
 * Installed provider binaries. Null while the server has no such endpoint
 * (older builds), so the settings section stays hidden until the API exists.
 */
export const installedProviders = signal<InstalledProvider[] | null>(null);

let unsubscribe: (() => void) | null = null;

export async function loadInstalledProviders(refresh = false): Promise<void> {
  try {
    const data = await api
      .get(refresh ? 'providers/installed?refresh=1' : 'providers/installed')
      .json<{ providers?: InstalledProvider[] }>();
    installedProviders.value = Array.isArray(data.providers) ? data.providers : null;
  } catch {
    installedProviders.value = null;
  }

  if (!unsubscribe && installedProviders.value) {
    unsubscribe = subscribeWsMessages((message) => {
      const type = (message as { type?: string } | null)?.type;
      if (type === 'provider_updated') void loadInstalledProviders();
    });
  }
}

export async function updateProvider(id: string): Promise<void> {
  const current = installedProviders.value;
  if (current) {
    installedProviders.value = current.map((p) => (p.id === id ? { ...p, updating: true } : p));
  }
  try {
    await api.post(`providers/${encodeURIComponent(id)}/update`);
  } catch (error) {
    if (current) installedProviders.value = current;
    throw error;
  }
}
