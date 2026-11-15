import ky from 'ky';
import type { HTTPError } from 'ky';

export function extractErrorMessage(body: unknown): string {
  if (body && typeof body === 'object') {
    const record = body as Record<string, unknown>;
    const value = record.error ?? record.message;
    return typeof value === 'string' ? value.trim() : '';
  }

  if (typeof body !== 'string') return '';

  const text = body.trim();
  if (!text) return '';

  try {
    return extractErrorMessage(JSON.parse(text)) || text;
  } catch {
    return text;
  }
}

export const api = ky.create({
  prefix: '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
  retry: {
    limit: 2,
    methods: ['get'],
    statusCodes: [408, 413, 429, 500, 502, 503, 504],
    backoffLimit: 10000,
  },
  hooks: {
    beforeError: [
      async ({ error }) => {
        const { data, response } = error as HTTPError;

        let message = extractErrorMessage(data);

        if (!message && response && !response.bodyUsed) {
          try {
            message = extractErrorMessage(await response.clone().text());
          } catch {
            message = '';
          }
        }

        if (message) error.message = message;

        return error;
      },
    ],
  },
});
