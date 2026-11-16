import type { EventData, EventFrame, EventName } from './types';

/** A server event's data; its `type` equals the event name. */
export type ServerEvent = {
  [N in EventName]: N extends 'tunnel_state' ? TunnelStateEvent : EventData<N>;
}[EventName];

/**
 * While a provider installs, tunnel_state carries only id, state and
 * provider; every other time it is the full tunnel.
 */
export type TunnelStateEvent = Pick<EventData<'tunnel_state'>, 'type' | 'id' | 'state'> &
  Partial<EventData<'tunnel_state'>>;

/** The stream opened (again), or dropped and is reconnecting. */
export type ConnectionEvent = { type: 'connected' } | { type: 'disconnected' };

export type StreamEvent = ServerEvent | ConnectionEvent;

type Handler = (event: StreamEvent) => void;

/** Every named event of /api/events. A new server event fails the typecheck until it is listed. */
const EVENT_NAMES: Record<EventFrame['event'], true> = {
  tunnel_state: true,
  tunnel_deleted: true,
  tunnel_stats: true,
  install: true,
  notification: true,
  update_available: true,
  provider_update_available: true,
  provider_updated: true,
  resync: true,
};

export const EVENTS_URL = '/api/events';
const RECONNECT_MIN_MS = 1000;
const RECONNECT_MAX_MS = 30000;

const listeners = new Set<Handler>();
let source: EventSource | null = null;
let connected = false;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let reconnectDelay = RECONNECT_MIN_MS;
let reopened = false;

function emit(event: StreamEvent): void {
  listeners.forEach((listener) => listener(event));
}

function setConnected(next: boolean): void {
  if (connected === next) return;
  connected = next;
  emit({ type: next ? 'connected' : 'disconnected' });
}

function onFrame(raw: MessageEvent): void {
  let data: unknown;
  try {
    data = JSON.parse(raw.data as string);
  } catch {
    return;
  }
  if (typeof data !== 'object' || data === null) return;
  emit({ ...(data as object), type: raw.type } as ServerEvent);
}

function open(): void {
  if (typeof EventSource === 'undefined' || source) return;

  const es = new EventSource(EVENTS_URL);
  source = es;

  es.onopen = () => {
    reconnectDelay = RECONNECT_MIN_MS;
    setConnected(true);
    if (reopened) {
      // A new EventSource cannot resume with Last-Event-ID, so whatever
      // happened while it was closed is unknown: reload.
      reopened = false;
      emit({ type: 'resync', reason: 'reconnected' });
    }
  };

  es.onerror = () => {
    setConnected(false);
    // CONNECTING means the browser retries on its own and resumes with
    // Last-Event-ID; CLOSED means it gave up (a non-stream answer).
    if (es.readyState === EventSource.CLOSED) {
      es.close();
      if (source === es) source = null;
      scheduleReconnect();
    }
  };

  for (const name of Object.keys(EVENT_NAMES)) es.addEventListener(name, onFrame);
}

function scheduleReconnect(): void {
  if (reconnectTimer || listeners.size === 0) return;
  const delay = reconnectDelay;
  reconnectDelay = Math.min(reconnectDelay * 2, RECONNECT_MAX_MS);
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null;
    if (listeners.size === 0) return;
    reopened = true;
    open();
  }, delay);
}

function close(): void {
  if (reconnectTimer) clearTimeout(reconnectTimer);
  reconnectTimer = null;
  source?.close();
  source = null;
  connected = false;
  reopened = false;
  reconnectDelay = RECONNECT_MIN_MS;
}

/**
 * Listen to /api/events over one shared EventSource. The handler gets every
 * server event plus `connected` / `disconnected`; on `resync` (the server
 * lost events this client missed, or the stream had to be reopened) a store
 * reloads its state over REST. Returns the unsubscribe function; the stream
 * closes with the last listener.
 */
export function subscribeEvents(handler: Handler): () => void {
  listeners.add(handler);
  if (connected) handler({ type: 'connected' });
  open();

  return () => {
    listeners.delete(handler);
    if (listeners.size === 0) close();
  };
}
