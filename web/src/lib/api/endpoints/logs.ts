import { api } from '../client';
import type { LogFrame } from '../types';

export interface LogStreamOptions {
  /** The recent lines; sent first and again after every reconnect, so it replaces what is shown. */
  onSnapshot: (lines: string[]) => void;
  onLine: (line: string) => void;
}

type LogData = LogFrame['data'];

function streamUrl(tunnelId: string): string {
  return `/api/tunnels/${encodeURIComponent(tunnelId)}/logs?follow=1`;
}

/** Follow a tunnel's output over Server-Sent Events. */
function createStream(tunnelId: string, options: LogStreamOptions): { close: () => void } {
  if (typeof EventSource === 'undefined') return { close: () => undefined };

  const es = new EventSource(streamUrl(tunnelId));

  const read = (raw: MessageEvent): LogData | null => {
    try {
      return JSON.parse(raw.data as string) as LogData;
    } catch {
      return null;
    }
  };

  es.addEventListener('log_snapshot', (raw) => {
    const data = read(raw as MessageEvent);
    if (data && 'lines' in data) options.onSnapshot(data.lines ?? []);
  });
  es.addEventListener('log', (raw) => {
    const data = read(raw as MessageEvent);
    if (data && 'line' in data && typeof data.line === 'string') options.onLine(data.line);
  });

  return { close: () => es.close() };
}

function get(tunnelId: string): Promise<string> {
  return api.get(`logs/${tunnelId}`).text();
}

export const logsApi = {
  createStream,
  get,
  streamUrl,
};
