import { getStatus } from './endpoints/status';

type WsHandler = (message: unknown) => void;

type WsPayload = Record<string, unknown>;

let socket: WebSocket | null = null;
let connecting: Promise<void> | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let statusPortPromise: Promise<number | null> | null = null;

const listeners = new Set<WsHandler>();
const pendingMessages: string[] = [];

function canUseWebSocket(): boolean {
  return typeof window !== 'undefined' && typeof WebSocket !== 'undefined';
}

function clearReconnectTimer() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
}

function scheduleReconnect() {
  if (!canUseWebSocket() || listeners.size === 0 || reconnectTimer) {
    return;
  }

  reconnectTimer = setTimeout(() => {
    reconnectTimer = null;
    void connectSharedWebSocket();
  }, 5000);
}

function notifyListeners(message: unknown) {
  listeners.forEach((listener) => {
    listener(message);
  });
}

async function resolveStatusPort(): Promise<number | null> {
  if (!statusPortPromise) {
    statusPortPromise = getStatus()
      .then((status) => status.port)
      .catch(() => null);
  }
  return statusPortPromise;
}

async function resolveWsUrl(): Promise<string> {
  const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const hostname = window.location.hostname.toLowerCase();
  const isWailsHost =
    hostname === 'wails' ||
    hostname === 'wails.localhost' ||
    hostname.endsWith('.wails.localhost');
  const statusPort = isWailsHost ? await resolveStatusPort() : null;
  const wsHost = isWailsHost && statusPort ? `127.0.0.1:${statusPort}` : window.location.host;

  return `${wsProtocol}//${wsHost}/ws/events`;
}

function flushPendingMessages() {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    return;
  }

  while (pendingMessages.length > 0) {
    const message = pendingMessages.shift();
    if (!message) {
      continue;
    }
    socket.send(message);
  }
}

export function connectSharedWebSocket(): Promise<void> {
  if (!canUseWebSocket()) {
    return Promise.resolve();
  }

  if (socket && socket.readyState === WebSocket.OPEN) {
    return Promise.resolve();
  }

  if (connecting) {
    return connecting;
  }

  connecting = resolveWsUrl().then(
    (url) =>
      new Promise<void>((resolve, reject) => {
        clearReconnectTimer();

        const ws = new WebSocket(url);
        socket = ws;
        let settled = false;

        ws.onopen = () => {
          settled = true;
          notifyListeners({ type: '__ws_open' });
          flushPendingMessages();
          resolve();
        };

        ws.onmessage = (event: MessageEvent) => {
          try {
            const message = JSON.parse(event.data) as unknown;
            notifyListeners(message);
          } catch {
            return;
          }
        };

        ws.onclose = () => {
          socket = null;
          if (!settled) {
            settled = true;
            reject(new Error('WebSocket closed'));
          }
          scheduleReconnect();
        };

        ws.onerror = () => {
          if (!settled) {
            settled = true;
            reject(new Error('WebSocket error'));
          }
        };
      }),
  );

  connecting = connecting.finally(() => {
    connecting = null;
  });

  return connecting;
}

export async function sendWsMessage(payload: WsPayload): Promise<void> {
  if (!canUseWebSocket()) {
    return;
  }

  const serialized = JSON.stringify(payload);

  if (socket && socket.readyState === WebSocket.OPEN) {
    socket.send(serialized);
    return;
  }

  pendingMessages.push(serialized);

  try {
    await connectSharedWebSocket();
  } catch {
    return;
  }
}

function closeSharedWebSocket() {
  clearReconnectTimer();
  if (socket) {
    socket.close();
    socket = null;
  }
}

export function subscribeWsMessages(handler: WsHandler): () => void {
  listeners.add(handler);
  void connectSharedWebSocket();

  return () => {
    listeners.delete(handler);
    if (listeners.size === 0) {
      closeSharedWebSocket();
    }
  };
}
