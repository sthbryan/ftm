import { tunnelsApi, getStatus } from '$lib/api';
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

  if (!msg.id) return;

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
    processStateMessage(message as TunnelMessage);
  });

  getStatus()
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

function start(id: string) {
  tunnelsById = {
    ...tunnelsById,
    [id]: { ...tunnelsById[id], state: 'starting' }
  };
  tunnelsApi.start(id).catch((e: Error) => {
    tunnelsById = {
      ...tunnelsById,
      [id]: { ...tunnelsById[id], state: 'error', errorMessage: e.message }
    };
  });
}

function stop(id: string) {
  tunnelsApi.stop(id).catch(() => {});
}

function remove(id: string) {
  expirationMonitor.stop(id);
  const { [id]: _, ...rest } = tunnelsById;
  tunnelsById = rest as TunnelMap;
  tunnelsApi.delete(id).catch(() => {});
}

function add(data: { name: string; provider: string; localPort: number }) {
  tunnelsApi.create(data).then((newTunnel: Tunnel) => {
    tunnelsById = {
      ...tunnelsById,
      [newTunnel.id]: newTunnel
    };
  });
}

function update(id: string, data: Partial<Tunnel>) {
  return tunnelsApi.update(id, data).then((updated: Tunnel) => {
    tunnelsById = {
      ...tunnelsById,
      [id]: updated
    };
    return updated;
  });
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
