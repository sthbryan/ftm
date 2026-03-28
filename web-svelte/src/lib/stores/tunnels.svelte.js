import { tunnelsApi } from '$lib/api';
import { useNotifications } from './notification.svelte.js';
import { useExpirationMonitor } from './expiration.svelte.js';

let tunnelsById = $state({});
let loading = $state(true);
let error = $state(null);
let socket = $state(null);
let installProgress = $state({});

let previousStates = {};

const notifications = useNotifications();
const expirationMonitor = useExpirationMonitor();

const tunnels = $derived.by(() => Object.values(tunnelsById));

function processStateMessage(msg) {
  if (msg.type === 'install') {
    installProgress = { ...installProgress, [msg.provider]: msg };
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

  const updated = { ...tunnel, state: newState, publicUrl: newUrl, errorMessage: newError };
  if (msg.name !== undefined) updated.name = msg.name;
  if (msg.provider !== undefined) updated.provider = msg.provider;
  if (msg.port !== undefined) updated.port = msg.port;
  if (msg.expiresAt !== undefined) updated.expiresAt = msg.expiresAt;

  tunnelsById = { ...tunnelsById, [msg.id]: updated };

  const prevState = previousStates[msg.id] ?? {};
  if (prevState.state !== newState || prevState.publicUrl !== newUrl || prevState.errorMessage !== newError) {
    previousStates[msg.id] = { state: newState, publicUrl: newUrl, errorMessage: newError };

    if (newState === 'online') {
      notifications.notifyOnline(updated.name, updated.publicUrl);
      if (updated.expiresAt) expirationMonitor.start(updated);
      const { [updated.provider]: _, ...rest } = installProgress;
      installProgress = rest;
    } else if (newState === 'stopped') {
      expirationMonitor.stop(msg.id);
      notifications.notify('Tunnel Stopped', `${updated.name} has been stopped`, 'info');
    } else if (newState === 'error') {
      notifications.notifyError(updated.name, updated.errorMessage);
    } else if (newState === 'timeout') {
      notifications.notify('Timeout', `${updated.name} could not connect`, 'error');
    } else if (newState === 'installing') {
      notifications.notify('Installing', `Installing tunnel for ${updated.provider}...`, 'info');
    }
  }
}

function connect() {
  if (socket && socket.readyState === WebSocket.OPEN) return;

  loading = true;
  notifications.init();

  tunnelsApi.getAll()
    .then(data => {
      const map = {};
      data.forEach(t => { map[t.id] = t; });
      tunnelsById = map;
      data.forEach(t => {
        previousStates[t.id] = { state: t.state, publicUrl: t.publicUrl, errorMessage: t.errorMessage };
        if (t.state === 'online' && t.expiresAt) {
          expirationMonitor.start(t);
        }
      });
      loading = false;
    })
    .catch(e => {
      error = e.message;
      loading = false;
    });

  const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const ws = new WebSocket(`${wsProtocol}//${window.location.host}/api/tunnels/ws`);

  ws.onopen = () => {
    console.log('[WS] Connected');
    notifications.notify('Connected', 'Welcome back!', 'success');
  };

  ws.onmessage = (e) => {
    try {
      const msg = JSON.parse(e.data);
      processStateMessage(msg);
    } catch (err) {
      console.error('[WS] Parse error:', err);
    }
  };

  ws.onclose = () => {
    console.log('[WS] Disconnected');
    notifications.notify('Disconnected', 'Catch you later!', 'warning');
    socket = null;
  };

  ws.onerror = (e) => {
    console.error('[WS] Error:', e);
    error = 'Connection error';
  };

  socket = ws;
}

function disconnect() {
  if (socket) {
    socket.close();
    socket = null;
  }
  expirationMonitor.stopAll();
}

function start(id) {
  tunnelsById = {
    ...tunnelsById,
    [id]: { ...tunnelsById[id], state: 'starting' }
  };
  tunnelsApi.start(id).catch(e => {
    tunnelsById = {
      ...tunnelsById,
      [id]: { ...tunnelsById[id], state: 'error', errorMessage: e.message }
    };
  });
}

function stop(id) {
  tunnelsApi.stop(id).catch(() => {});
}

function remove(id) {
  expirationMonitor.stop(id);
  const { [id]: _, ...rest } = tunnelsById;
  tunnelsById = rest;
  tunnelsApi.delete(id).catch(() => {});
}

function add(data) {
  tunnelsApi.create(data).then(newTunnel => {
    tunnelsById = {
      ...tunnelsById,
      [newTunnel.id]: newTunnel
    };
  });
}

function update(id, data) {
  return tunnelsApi.update(id, data).then(updated => {
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
    update
  };
}
