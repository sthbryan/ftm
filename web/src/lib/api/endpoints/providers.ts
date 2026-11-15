import { api } from '../client';
import type { Provider } from '../types';

const DETECT_PORT_TTL_MS = 60_000;

let detectPortCache: { value: number; expiresAt: number } | null = null;
let detectPortRequest: Promise<number> | null = null;

async function fetchDetectPort(): Promise<number> {
  try {
    const data = await api.get('detect-port').json<{ suggested?: number }>();
    return data.suggested || 30000;
  } catch {
    return 30000;
  }
}

export const providersApi = {
  getAll: (): Promise<Provider[]> => api.get('providers').json<Provider[]>(),
  detectPort: ({ forceRefresh = false }: { forceRefresh?: boolean } = {}): Promise<number> => {
    const now = Date.now();
    if (!forceRefresh && detectPortCache && detectPortCache.expiresAt > now) {
      return Promise.resolve(detectPortCache.value);
    }

    if (detectPortRequest) {
      return detectPortRequest;
    }

    const request = fetchDetectPort()
      .then((port) => {
        detectPortCache = { value: port, expiresAt: Date.now() + DETECT_PORT_TTL_MS };
        return port;
      })
      .finally(() => {
        if (detectPortRequest === request) {
          detectPortRequest = null;
        }
      });

    detectPortRequest = request;
    return request;
  },
};
