import { tunnelsApi } from '$lib/api';
import { useNotifications } from './notification.svelte.js';
import { useExpirationMonitor } from './expiration.svelte.js';

let tunnels = $state([]);
let loading = $state(true);
let error = $state(null);
let socket = $state(null);
let installProgress = $state({});

const notifications = useNotifications();
const expirationMonitor = useExpirationMonitor();

function handleMessage(msg) {
  if (msg.type === 'install') {
    installProgress = { ...installProgress, [msg.provider]: msg };
    return;
  }
  
  if (!msg.id) return;
  
  const idx = tunnels.findIndex(t => t.id === msg.id);
  if (idx === -1) return;
  
  const oldTunnel = tunnels[idx];
  const newState = msg.state || 'stopped';
  
  if (oldTunnel.state === newState && 
      oldTunnel.publicUrl === (msg.publicUrl ?? oldTunnel.publicUrl) &&
      oldTunnel.errorMessage === (msg.errorMessage ?? oldTunnel.errorMessage)) {
    return;
  }
  
  const updated = [...tunnels];
  updated[idx] = {
    ...updated[idx],
    name: msg.name ?? updated[idx].name,
    provider: msg.provider ?? updated[idx].provider,
    port: msg.port ?? updated[idx].port,
    state: newState,
    publicUrl: msg.publicUrl ?? updated[idx].publicUrl,
    errorMessage: msg.errorMessage ?? updated[idx].errorMessage,
    expiresAt: msg.expiresAt ?? updated[idx].expiresAt
  };
  tunnels = updated;
  
  if (!notifications.enabled) return;
  
  if (newState === 'online' && oldTunnel.state !== 'online') {
    notifications.notifyOnline(updated[idx].name, msg.publicUrl);
    if (msg.expiresAt) expirationMonitor.start(updated[idx]);
  }
  
  if (newState === 'error' && oldTunnel.state !== 'error') {
    notifications.notifyError(updated[idx].name, msg.errorMessage);
  }
  
  if (newState === 'timeout' && oldTunnel.state !== 'timeout') {
    notifications.notify('Timeout', `${updated[idx].name} could not connect`);
  }
  
  if (newState === 'stopped' && oldTunnel.state === 'online') {
    expirationMonitor.stop(msg.id);
  }
  
  if (newState === 'online') {
    const updatedProgress = { ...installProgress };
    delete updatedProgress[updated[idx].provider];
    installProgress = updatedProgress;
  }
}

function connect() {
  if (socket) return;
  
  loading = true;
  notifications.init();
  
  tunnelsApi.getAll()
    .then(data => {
      tunnels = data;
      data.forEach(t => {
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
  
  const ws = new WebSocket(`ws://${window.location.host}/ws/events`);
  
  ws.onmessage = (e) => {
    try {
      handleMessage(JSON.parse(e.data));
    } catch {}
  };
  
  ws.onclose = () => {
    error = 'Connection closed. Reconnecting...';
    socket = null;
    setTimeout(connect, 3000);
  };
  
  ws.onerror = () => {
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
  tunnels = tunnels.map(t => t.id === id ? { ...t, state: 'starting' } : t);
  tunnelsApi.start(id).catch(e => {
    tunnels = tunnels.map(t => t.id === id ? { ...t, state: 'error', errorMessage: e.message } : t);
  });
}

function stop(id) {
  tunnels = tunnels.map(t => t.id === id ? { ...t, state: 'stopped', publicUrl: null } : t);
  expirationMonitor.stop(id);
  tunnelsApi.stop(id).catch(() => {});
}

function remove(id) {
  expirationMonitor.stop(id);
  tunnelsApi.delete(id).catch(() => {});
  tunnels = tunnels.filter(t => t.id !== id);
}

function add(data) {
  tunnelsApi.create(data).then(newTunnel => {
    tunnels = [...tunnels, newTunnel];
  });
}

function update(id, data) {
  return tunnelsApi.update(id, data).then(updated => {
    tunnels = tunnels.map(t => t.id === id ? updated : t);
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
