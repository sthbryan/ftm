import { tunnelsApi } from '$lib/api';
import { useNotifications } from './notification.svelte.js';
import { useExpirationMonitor } from './expiration.svelte.js';

let tunnelsById = $state({});
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
  
  const oldTunnel = tunnelsById[msg.id];
  if (!oldTunnel) return;
  
  const updates = {};
  if (msg.name !== undefined) updates.name = msg.name;
  if (msg.provider !== undefined) updates.provider = msg.provider;
  if (msg.port !== undefined) updates.port = msg.port;
  if (msg.state !== undefined) updates.state = msg.state;
  if (msg.publicUrl !== undefined) updates.publicUrl = msg.publicUrl;
  if (msg.errorMessage !== undefined) updates.errorMessage = msg.errorMessage;
  if (msg.expiresAt !== undefined) updates.expiresAt = msg.expiresAt;
  
  if (Object.keys(updates).length === 0) return;
  
  const newState = updates.state || 'stopped';
  
  if (oldTunnel.state === newState && 
      oldTunnel.publicUrl === (updates.publicUrl ?? oldTunnel.publicUrl) &&
      oldTunnel.errorMessage === (updates.errorMessage ?? oldTunnel.errorMessage)) {
    return;
  }
  
  tunnelsById = {
    ...tunnelsById,
    [msg.id]: { ...oldTunnel, ...updates }
  };
  
  const updatedTunnel = tunnelsById[msg.id];
  
  if (newState === 'online' && oldTunnel.state !== 'online') {
    notifications.notifyOnline(updatedTunnel.name, updatedTunnel.publicUrl);
    if (updatedTunnel.expiresAt) expirationMonitor.start(updatedTunnel);
    return;
  }
  
  if (newState === 'stopped' && oldTunnel.state === 'online') {
    expirationMonitor.stop(msg.id);
    notifications.notify('Tunnel Stopped', `${updatedTunnel.name} has been stopped`, 'info');
    return;
  }
  
  if (newState === 'error' && oldTunnel.state !== 'error') {
    notifications.notifyError(updatedTunnel.name, updatedTunnel.errorMessage);
    return;
  }
  
  if (newState === 'timeout' && oldTunnel.state !== 'timeout') {
    notifications.notify('Timeout', `${updatedTunnel.name} could not connect`, 'error');
    return;
  }
  
  if (newState === 'online') {
    const { [updatedTunnel.provider]: _, ...rest } = installProgress;
    installProgress = rest;
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
  const ws = new WebSocket(`${wsProtocol}//${window.location.host}/ws/events`);
  
  ws.onopen = () => {
    console.log('[WS] Connected');
  };
  
  ws.onmessage = (e) => {
    try {
      handleMessage(JSON.parse(e.data));
    } catch (err) {
      console.error('[WS] Parse error:', err);
    }
  };
  
  ws.onclose = () => {
    console.log('[WS] Disconnected');
    socket = null;
    setTimeout(connect, 3000);
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
    get tunnels() { return Object.values(tunnelsById); },
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
