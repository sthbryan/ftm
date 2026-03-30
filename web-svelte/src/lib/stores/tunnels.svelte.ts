import { statusApi, tunnelsApi } from '$lib/api';
import { subscribeWsMessages } from '$lib/api/ws';
import { useNotifications } from './notification.svelte';
import { useExpirationMonitor } from './expiration.svelte';
import type { Tunnel, TunnelState } from '$lib/types';

interface TunnelMap {
  [id: string]: Tunnel;
}

interface TunnelMessage {
  type?: string;
  id?: string;
  state?: TunnelState;
  name?: string;
  provider?: string;
  port?: number;
  publicUrl?: string;
  errorMessage?: string;
  expiresAt?: number;
  install?: { provider: string; percent: number; step: string };
}

interface InstallProgress {
  [provider: string]: { provider: string; percent: number; step: string };
}

let tunnelsById: TunnelMap = $state({});
let loading = $state(true);
let error: string | null = $state(null);
let unsubscribeWs: (() => void) | null = $state(null);
let installProgress: InstallProgress = $state({});

const previousStates: Record<string, { state: string; publicUrl: string; errorMessage: string }> = {};

const notifications = useNotifications();
const expirationMonitor = useExpirationMonitor();

const tunnels = $derived.by(() => Object.values(tunnelsById));

function processStateMessage(msg: TunnelMessage) {
  if (msg.type === 'install' && msg.install) {
    installProgress = { ...installProgress, [msg.install.provider]: msg.install };
    return;
  }

  if (msg.type === 'tunnel_deleted' && msg.id) {
    expirationMonitor.stop(msg.id);
    const { [msg.id]: _, ...rest } = tunnelsById;
    tunnelsById = rest as TunnelMap;
    return;
  }

  const isTunnelStateMessage = msg.type === 'tunnel_state' || msg.type === undefined;
  if (!isTunnelStateMessage || !msg.id) return;

  const tunnel = tunnelsById[msg.id];
  if (!tunnel) return;

  const newState = msg.state ?? tunnel.state;
  const newUrl = msg.publicUrl ?? tunnel.publicUrl ?? '';
  const newError = msg.errorMessage ?? tunnel.errorMessage ?? '';

  if (tunnel.state === newState && tunnel.publicUrl === newUrl && tunnel.errorMessage === newError) {
    return;
  }

  const updated: Tunnel = { ...tunnel, state: newState, publicUrl: newUrl, errorMessage: newError };
  if (msg.name !== undefined) updated.name = msg.name;
  if (msg.provider !== undefined) updated.provider = msg.provider;
  if (msg.port !== undefined) updated.port = msg.port;
  if (msg.expiresAt !== undefined) updated.expiresAt = msg.expiresAt;

  tunnelsById = { ...tunnelsById, [msg.id]: updated };

  const prevState = previousStates[msg.id] ?? { state: '', publicUrl: '', errorMessage: '' };
  if (prevState.state !== newState || prevState.publicUrl !== newUrl || prevState.errorMessage !== newError) {
    previousStates[msg.id] = { state: newState, publicUrl: newUrl, errorMessage: newError };

    if (newState === 'online' && updated.publicUrl) {
      notifications.notifyOnline(updated.name, updated.publicUrl);
      if (updated.expiresAt) expirationMonitor.start(updated);
      if (updated.provider) {
        const { [updated.provider]: _, ...rest } = installProgress;
        installProgress = rest as InstallProgress;
      }
    } else if (newState === 'stopped') {
      notifications.notify('Tunnel Stopped', `${updated.name} has been stopped`, 'info');
      expirationMonitor.stop(msg.id);
    } else if (newState === 'error') {
      notifications.notifyError(updated.name, newError);
    } else if (newState === 'timeout') {
      notifications.notify('Timeout', `${updated.name} could not connect`, 'error');
    } else if (newState === 'installing') {
      notifications.notify('Installing', `Installing tunnel for ${updated.provider}...`, 'info');
    }
  }
}

function connect() {
  if (unsubscribeWs) return;

  loading = true;

  unsubscribeWs = subscribeWsMessages((message) => {
    if (typeof message !== 'object' || message === null) {
      return;
    }
    const msg = message as TunnelMessage;
    if (msg.type === '__ws_open') {
      return;
    }
    processStateMessage(msg);
  });

  statusApi.get()
    .then((status) => {
      notifications.setStatus(status.notificationsStatus);
    })
    .catch(() => {
      notifications.setStatus('pending');
    });

  tunnelsApi.getAll()
    .then((data: Tunnel[]) => {
      const map: TunnelMap = {};
      data.forEach(t => { map[t.id] = t; });
      tunnelsById = map;
      data.forEach(t => {
        previousStates[t.id] = { state: t.state, publicUrl: t.publicUrl ?? '', errorMessage: t.errorMessage ?? '' };
        if (t.state === 'online' && t.expiresAt) {
          expirationMonitor.start(t);
        }
      });
      loading = false;
    })
    .catch((e: Error) => {
      error = e.message;
      loading = false;
    });
}

function disconnect() {
  if (unsubscribeWs) {
    unsubscribeWs();
    unsubscribeWs = null;
  }
  expirationMonitor.stopAll();
}

async function start(id: string) {
  const current = tunnelsById[id];
  if (!current) {
    throw new Error(`Tunnel ${id} not found`);
  }

  tunnelsById = {
    ...tunnelsById,
    [id]: { ...current, state: 'starting' }
  };

  try {
    return await tunnelsApi.start(id);
  } catch (e) {
    const message = e instanceof Error ? e.message : 'Failed to start tunnel';
    const latest = tunnelsById[id] ?? current;
    tunnelsById = {
      ...tunnelsById,
      [id]: { ...latest, state: 'error', errorMessage: message }
    };
    throw e;
  }
}

async function stop(id: string) {
  const current = tunnelsById[id];
  if (!current) {
    throw new Error(`Tunnel ${id} not found`);
  }

  const previous = {
    state: current.state,
    publicUrl: current.publicUrl,
    errorMessage: current.errorMessage
  };

  tunnelsById = {
    ...tunnelsById,
    [id]: { ...current, state: 'stopping' }
  };

  try {
    await tunnelsApi.stop(id);
  } catch (e) {
    const latest = tunnelsById[id] ?? current;
    tunnelsById = {
      ...tunnelsById,
      [id]: {
        ...latest,
        state: previous.state,
        publicUrl: previous.publicUrl,
        errorMessage: previous.errorMessage
      }
    };
    throw e;
  }
}

async function remove(id: string) {
  const current = tunnelsById[id];
  if (!current) {
    throw new Error(`Tunnel ${id} not found`);
  }

  expirationMonitor.stop(id);
  const { [id]: _, ...rest } = tunnelsById;
  tunnelsById = rest as TunnelMap;

  try {
    await tunnelsApi.delete(id);
  } catch (e) {
    tunnelsById = { ...tunnelsById, [id]: current };
    if (current.state === 'online' && current.expiresAt) {
      expirationMonitor.start(current);
    }
    throw e;
  }
}

async function add(data: { name: string; provider: string; localPort: number }) {
  const newTunnel = await tunnelsApi.create(data);
  tunnelsById = {
    ...tunnelsById,
    [newTunnel.id]: newTunnel
  };
  return newTunnel;
}

async function update(id: string, data: Partial<Tunnel> & { localPort?: number }) {
  const payload: { name?: string; provider?: string; localPort?: number } = {};

  if (data.name !== undefined) payload.name = data.name;
  if (data.provider !== undefined) payload.provider = data.provider;

  const localPort = data.localPort ?? data.port;
  if (localPort !== undefined) payload.localPort = localPort;

  const updated = await tunnelsApi.update(id, payload);
  tunnelsById = {
    ...tunnelsById,
    [id]: updated
  };
  return updated;
}

export function useTunnels() {
  return {
    get tunnels() { return tunnels; },
    get loading() { return loading; },
    get error() { return error; },
    get installProgress() { return installProgress; },
    connect,
    disconnect,
    start,
    stop,
    delete: remove,
    create: add,
    update,
    getById: (id: string) => tunnelsById[id]
  };
}
