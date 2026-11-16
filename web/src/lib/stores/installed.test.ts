import { beforeEach, describe, expect, it, vi } from 'vitest';

const success = vi.fn();
const error = vi.fn();
vi.mock('./toast', () => ({ toast: { success, error } }));
vi.mock('./i18n', () => ({ t: (key: string, args?: Record<string, unknown>) => `${key}:${JSON.stringify(args ?? {})}` }));
vi.mock('@/lib/api/ws', () => ({ subscribeWsMessages: vi.fn(() => () => undefined) }));

class FakeHTTPError extends Error {
  constructor(public response: { text: () => Promise<string> }) {
    super('Request failed with status code 409');
  }
}
vi.mock('ky', () => ({ HTTPError: FakeHTTPError }));
const post = vi.fn();
vi.mock('@/lib/api/client', () => ({
  api: { post: (...a: unknown[]) => post(...a), get: vi.fn() },
  extractErrorMessage: (body: string) => {
    try {
      return (JSON.parse(body) as { message: string }).message;
    } catch {
      return body.trim();
    }
  },
}));

const { applyProviderUpdated, canUpdate, installedProviders, updateProvider } = await import('./installed');
type Row = import('./installed').InstalledProvider;

function row(over: Partial<Row> = {}): Row {
  return {
    id: 'cloudflared', name: 'Cloudflare Tunnel', installed: true, version: '1', latest: '2',
    source: 'bundled', path: '/x', hasUpdate: true, updatable: true, updating: false,
    releaseUrl: '', checkedAt: 0, ...over,
  };
}

beforeEach(() => {
  success.mockClear();
  error.mockClear();
  post.mockReset();
  installedProviders.value = [row()];
});

describe('canUpdate', () => {
  it('offers install when not installed and update when behind', () => {
    expect(canUpdate(row({ source: 'none', installed: false, hasUpdate: false }))).toBe(true);
    expect(canUpdate(row())).toBe(true);
  });
  it('offers nothing otherwise', () => {
    expect(canUpdate(row({ hasUpdate: false }))).toBe(false);
    expect(canUpdate(row({ source: 'path', updatable: false }))).toBe(false);
    expect(canUpdate(row({ source: 'system', updatable: false, hasUpdate: false }))).toBe(false);
    expect(canUpdate(row({ updating: true }))).toBe(false);
  });
});

describe('applyProviderUpdated', () => {
  it('swaps in the fresh row and toasts success', () => {
    applyProviderUpdated({ type: 'provider_updated', provider: 'cloudflared', ok: true, version: '2', binary: row({ version: '2', hasUpdate: false }) });
    expect(installedProviders.value?.[0].version).toBe('2');
    expect(success).toHaveBeenCalledOnce();
  });
  it('clears updating and toasts the error on failure', () => {
    installedProviders.value = [row({ updating: true })];
    applyProviderUpdated({ type: 'provider_updated', provider: 'cloudflared', ok: false, error: 'checksum mismatch' });
    expect(installedProviders.value?.[0].updating).toBe(false);
    expect(error.mock.calls[0][0]).toContain('checksum mismatch');
  });
});

describe('updateProvider', () => {
  it('shows the server text on 4xx', async () => {
    post.mockRejectedValue(new FakeHTTPError({ text: async () => 'already updating\n' }));
    await updateProvider('cloudflared');
    expect(error.mock.calls[0][0]).toContain('already updating');
    expect(installedProviders.value?.[0].updating).toBe(false);
  });

  it('shows the message of a JSON error body', async () => {
    error.mockClear();
    post.mockRejectedValue(
      new FakeHTTPError({ text: async () => '{"code":"update_in_progress","message":"already updating"}' }),
    );
    await updateProvider('cloudflared');
    expect(error.mock.calls[0][0]).toContain('already updating');
    expect(error.mock.calls[0][0]).not.toContain('update_in_progress');
  });
});
