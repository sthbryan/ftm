import { sendWsMessage, subscribeWsMessages } from '../ws';
import { api } from '../client';

export interface LogStreamOptions {
  onLine: (line: string) => void;
  onClose?: () => void;
}

interface LogWsMessage {
  type?: string;
  id?: string;
  line?: string;
}

function createStream(
  tunnelId: string,
  options: LogStreamOptions
): { close: () => void } {
  let closed = false;

  const unsubscribe = subscribeWsMessages((message) => {
    if (closed || typeof message !== 'object' || message === null) {
      return;
    }

    const payload = message as LogWsMessage;

    if (payload.type === '__ws_open') {
      void sendWsMessage({ type: 'logs_subscribe', id: tunnelId });
      return;
    }

    if (payload.type === 'log' && payload.id === tunnelId && typeof payload.line === 'string') {
      options.onLine(payload.line);
    }
  });

  void sendWsMessage({ type: 'logs_subscribe', id: tunnelId });

  return {
    close: () => {
      if (closed) {
        return;
      }
      closed = true;
      unsubscribe();
      void sendWsMessage({ type: 'logs_unsubscribe', id: tunnelId });
      if (options.onClose) {
        options.onClose();
      }
    },
  };
}

function get(tunnelId: string): Promise<string> {
  return api.get(`logs/${tunnelId}`).text();
}

export const logsApi = {
  createStream,
  get,
};
