import { signal } from '@preact/signals';
import { HTTPError } from 'ky';
import { api, extractErrorMessage } from '@/lib/api/client';
import { subscribeEvents } from '@/lib/api/events';
import { t } from './i18n';
import { toast } from './toast';

export type ProviderSource = 'bundled' | 'path' | 'system' | 'none';

/** One provider binary as `GET /api/providers/installed` reports it. */
export interface InstalledProvider {
  id: string;
  name: string;
  installed: boolean;
  /** "" when unknown. */
  version: string;
  /** "" when unknown. */
  latest: string;
  /** `path` is user-installed and never updated by ftm. */
  source: ProviderSource;
  path: string;
  hasUpdate: boolean;
  updatable: boolean;
  updating: boolean;
  releaseUrl: string;
  /** Milliseconds; 0 means never checked. */
  checkedAt: number;
  error?: string;
}

interface ProviderUpdatedEvent {
  type: 'provider_updated';
  provider: string;
  ok: boolean;
  version?: string;
  error?: string;
  restarted?: string[];
  binary?: InstalledProvider;
}

/**
 * Installed provider binaries. Null while the server has no such endpoint
 * (older builds), so the settings section stays hidden there.
 */
export const installedProviders = signal<InstalledProvider[] | null>(null);
export const checkingProviders = signal(false);

let unsubscribe: (() => void) | null = null;

function replaceRow(row: InstalledProvider): void {
  const current = installedProviders.value;
  if (!current) return;
  installedProviders.value = current.map((p) => (p.id === row.id ? row : p));
}

function setUpdating(id: string, updating: boolean): void {
  const current = installedProviders.value;
  if (!current) return;
  installedProviders.value = current.map((p) => (p.id === id ? { ...p, updating } : p));
}

function nameOf(id: string): string {
  return installedProviders.value?.find((p) => p.id === id)?.name ?? id;
}

/** Apply a `provider_updated` event: swap in the fresh row and say how it went. */
export function applyProviderUpdated(event: ProviderUpdatedEvent): void {
  if (event.binary) replaceRow(event.binary);
  else setUpdating(event.provider, false);

  const name = event.binary?.name ?? nameOf(event.provider);
  if (event.ok) {
    toast.success(t('providers_updated', { 0: name, 1: event.version ?? event.binary?.version ?? '' }).replace(/^✓\s*/, ''));
  } else {
    toast.error(t('providers_update_failed', { 0: name, 1: event.error ?? '' }));
  }
}

function listen(): void {
  if (unsubscribe) return;
  unsubscribe = subscribeEvents((event) => {
    if (event.type === 'provider_updated') applyProviderUpdated(event as ProviderUpdatedEvent);
    if (event.type === 'provider_update_available' || event.type === 'resync') void loadInstalledProviders();
  });
}

export async function loadInstalledProviders(refresh = false): Promise<void> {
  if (refresh) checkingProviders.value = true;
  try {
    const data = await api
      .get(refresh ? 'providers/installed?refresh=1' : 'providers/installed')
      .json<{ providers?: InstalledProvider[] }>();
    installedProviders.value = Array.isArray(data.providers) ? data.providers : null;
  } catch {
    if (!refresh) installedProviders.value = null;
  } finally {
    checkingProviders.value = false;
  }

  if (installedProviders.value) listen();
}

/** Start an update (or a first install when the source is `none`). */
export async function updateProvider(id: string): Promise<void> {
  setUpdating(id, true);
  try {
    await api.post(`providers/${encodeURIComponent(id)}/update`);
  } catch (error) {
    setUpdating(id, false);
    let reason = (error as Error).message;
    if (error instanceof HTTPError) {
      reason = extractErrorMessage(await error.response.text().catch(() => '')) || reason;
    }
    toast.error(t('providers_update_failed', { 0: nameOf(id), 1: reason }));
  }
}

/** Whether ftm can act on this binary from the settings screen. */
export function canUpdate(row: InstalledProvider): boolean {
  if (row.updating || !row.updatable) return false;
  return row.source === 'none' || row.hasUpdate;
}
