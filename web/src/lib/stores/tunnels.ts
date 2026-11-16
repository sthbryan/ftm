import { computed, signal } from '@preact/signals';
import { settingsApi, tunnelsApi, type Settings } from '@/lib/api';
import { subscribeEvents, type ServerEvent } from '@/lib/api/events';
import { renderNotificationEvent, syncNotificationSettings } from './notification';
import { startExpiration, stopAllExpirations, stopExpiration } from './expiration';
import type { Tunnel, TunnelState } from '@/lib/types';

export interface InstallProgress {
  provider: string;
  percent: number;
  step: string;
}

const tunnelsById = signal<Record<string, Tunnel>>({});

export const tunnels = computed(() => Object.values(tunnelsById.value));
export const tunnelsLoading = signal(true);
export const tunnelsError = signal<string | null>(null);
export const tunnelsConnected = signal(false);
export const tunnelsOnboarded = signal(true);
export const installProgress = signal<Record<string, InstallProgress>>({});

let unsubscribeEvents: (() => void) | null = null;
let resolvedOnce = false;

export function getTunnel(id: string): Tunnel | undefined {
  return tunnelsById.value[id];
}

export function patchTunnel(id: string, patch: Partial<Tunnel>): void {
  const current = tunnelsById.value[id];
  if (!current) return;
  tunnelsById.value = { ...tunnelsById.value, [id]: { ...current, ...patch } };
}

function processMessage(msg: ServerEvent): void {
  if (msg.type === 'resync') {
    void syncTunnels();
    return;
  }

  if (msg.type === 'install' && msg.install) {
    installProgress.value = { ...installProgress.value, [msg.install.provider]: msg.install };
    if (msg.done) void syncTunnels();
    return;
  }

  if (msg.type === 'notification' && msg.notification) {
    renderNotificationEvent(msg.notification);
    return;
  }

  if (msg.type === 'tunnel_deleted' && msg.id) {
    stopExpiration(msg.id);
    const { [msg.id]: _removed, ...rest } = tunnelsById.value;
    tunnelsById.value = rest;
    return;
  }

  if (msg.type === 'tunnel_stats' && msg.id) {
    const current = tunnelsById.value[msg.id];
    if (!current) return;
    if (
      current.visitors === msg.visitors &&
      current.activeSessions === msg.activeSessions
    ) {
      return;
    }

    tunnelsById.value = {
      ...tunnelsById.value,
      [msg.id]: { ...current, visitors: msg.visitors, activeSessions: msg.activeSessions },
    };
    return;
  }

  if (msg.type !== 'tunnel_state' || !msg.id) return;

  const tunnel = tunnelsById.value[msg.id];
  if (!tunnel) return;

  const newState = msg.state ?? tunnel.state;
  const newUrl = msg.publicUrl ?? tunnel.publicUrl ?? '';
  const newError = msg.errorMessage ?? tunnel.errorMessage ?? '';

  if (tunnel.state === newState && tunnel.publicUrl === newUrl && tunnel.errorMessage === newError) {
    return;
  }

  const updated: Tunnel = {
    ...tunnel,
    state: newState,
    publicUrl: newUrl,
    errorMessage: newError,
  };

  if (msg.name !== undefined) updated.name = msg.name;
  if (msg.provider !== undefined) updated.provider = msg.provider;
  if (msg.port !== undefined) updated.port = msg.port;
  if (msg.expiresAt !== undefined) updated.expiresAt = msg.expiresAt;
  if (msg.startedAt !== undefined) updated.startedAt = msg.startedAt;
  if (newState !== 'online') {
    updated.visitors = 0;
    updated.activeSessions = 0;
  }

  tunnelsById.value = { ...tunnelsById.value, [msg.id]: updated };

  if (newState === 'online' && updated.expiresAt) startExpiration(updated);
  if (newState === 'stopped') stopExpiration(msg.id);

  if (newState === 'online' && updated.provider) {
    const { [updated.provider]: _cleared, ...rest } = installProgress.value;
    installProgress.value = rest;
  }
}

async function syncTunnels(): Promise<void> {
  try {
    const data = (await tunnelsApi.getAll()) as Tunnel[];
    tunnelsById.value = Object.fromEntries(data.map((tunnel) => [tunnel.id, tunnel]));

    stopAllExpirations();
    for (const tunnel of data) {
      if (tunnel.state === 'online' && tunnel.expiresAt) startExpiration(tunnel);
      if (tunnel.state === 'online' && tunnel.provider) {
        const { [tunnel.provider]: _cleared, ...rest } = installProgress.value;
        installProgress.value = rest;
      }
    }
  } catch {
    /* the next connected or resync event retries */
  }
}

async function loadTunnels(): Promise<void> {
  tunnelsLoading.value = !resolvedOnce;
  tunnelsError.value = null;

  try {
    const [data, settings] = await Promise.all([
      tunnelsApi.getAll(),
      settingsApi.get().catch(() => null),
    ]);

    tunnelsById.value = Object.fromEntries(
      (data as Tunnel[]).map((tunnel) => [tunnel.id, tunnel]),
    );

    for (const tunnel of data as Tunnel[]) {
      if (tunnel.state === 'online' && tunnel.expiresAt) startExpiration(tunnel);
    }

    tunnelsOnboarded.value = (settings as Settings | null)?.onboarded ?? true;
    resolvedOnce = true;
    tunnelsLoading.value = false;
  } catch (error) {
    tunnelsError.value = (error as Error).message;
    tunnelsLoading.value = false;
  }
}

export function connectTunnels(): void {
  if (unsubscribeEvents) return;

  unsubscribeEvents = subscribeEvents((event) => {
    if (event.type === 'connected') {
      tunnelsConnected.value = true;
      tunnelsError.value = null;
      void syncTunnels();
      return;
    }
    if (event.type === 'disconnected') {
      tunnelsConnected.value = false;
      return;
    }

    processMessage(event);
  });

  void syncNotificationSettings();
  void loadTunnels();
}

export function disconnectTunnels(): void {
  unsubscribeEvents?.();
  unsubscribeEvents = null;
  tunnelsConnected.value = false;
  stopAllExpirations();
}

export function retryTunnels(): void {
  void loadTunnels();
  connectTunnels();
}

export async function startTunnel(id: string): Promise<unknown> {
  const current = tunnelsById.value[id];
  if (!current) throw new Error(`Tunnel ${id} not found`);

  patchTunnel(id, { state: 'starting' });

  try {
    return await tunnelsApi.start(id);
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Failed to start tunnel';
    const latest = tunnelsById.value[id] ?? current;
    patchTunnel(id, { ...latest, state: 'error', errorMessage: message });
    throw error;
  }
}

export async function stopTunnel(id: string): Promise<void> {
  const current = tunnelsById.value[id];
  if (!current) throw new Error(`Tunnel ${id} not found`);

  const previous = {
    state: current.state,
    publicUrl: current.publicUrl,
    errorMessage: current.errorMessage,
  };

  patchTunnel(id, { state: 'stopping' });

  try {
    await tunnelsApi.stop(id);
  } catch (error) {
    patchTunnel(id, previous);
    throw error;
  }
}

export async function deleteTunnel(id: string): Promise<void> {
  const current = tunnelsById.value[id];
  if (!current) throw new Error(`Tunnel ${id} not found`);

  stopExpiration(id);
  const { [id]: _removed, ...rest } = tunnelsById.value;
  tunnelsById.value = rest;

  try {
    await tunnelsApi.delete(id);
  } catch (error) {
    tunnelsById.value = { ...tunnelsById.value, [id]: current };
    if (current.state === 'online' && current.expiresAt) startExpiration(current);
    throw error;
  }
}

export async function clearAllTunnels(): Promise<{ deleted: number }> {
  const previous = tunnelsById.value;

  stopAllExpirations();
  tunnelsById.value = {};

  try {
    return await tunnelsApi.deleteAll();
  } catch (error) {
    tunnelsById.value = previous;
    for (const tunnel of Object.values(previous)) {
      if (tunnel.state === 'online' && tunnel.expiresAt) startExpiration(tunnel);
    }
    throw error;
  }
}

export async function createTunnel(data: {
  name: string;
  provider: string;
  localPort: number;
}): Promise<Tunnel> {
  const tunnel = (await tunnelsApi.create(data)) as Tunnel;
  tunnelsById.value = { ...tunnelsById.value, [tunnel.id]: tunnel };
  tunnelsOnboarded.value = true;
  return tunnel;
}

export async function updateTunnel(
  id: string,
  data: { name?: string; provider?: string; localPort?: number },
): Promise<Tunnel> {
  const updated = (await tunnelsApi.update(id, data)) as Tunnel;
  tunnelsById.value = { ...tunnelsById.value, [id]: updated };
  return updated;
}
