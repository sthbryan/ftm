import { signal } from '@preact/signals';

function readPath(): string {
  return typeof window === 'undefined' ? '/' : window.location.pathname;
}

function readSearch(): string {
  return typeof window === 'undefined' ? '' : window.location.search;
}

export const route = signal(readPath());
export const search = signal(readSearch());

if (typeof window !== 'undefined') {
  window.addEventListener('popstate', () => {
    route.value = readPath();
    search.value = readSearch();
  });
}

export function navigate(to: string, options?: { replace?: boolean }): void {
  if (options?.replace) {
    window.history.replaceState({}, '', to);
  } else {
    window.history.pushState({}, '', to);
  }

  route.value = readPath();
  search.value = readSearch();
}

export function queryParam(name: string): string {
  return new URLSearchParams(search.value).get(name) ?? '';
}

/** Which connection form a path opens: a new one, or an existing one by id. */
export type ConnectionRoute = { mode: 'new' } | { mode: 'edit'; id: string };

/**
 * The connection form is a page. Its mode comes from the path alone, so leaving
 * the edit page goes straight to wherever the path points and never passes
 * through the new form on the way.
 */
export function connectionRoute(path: string): ConnectionRoute | null {
  if (path === '/new') return { mode: 'new' };
  const match = /^\/edit\/([^/]+)$/.exec(path);
  return match ? { mode: 'edit', id: decodeURIComponent(match[1]) } : null;
}

export function connectionPath(id?: string): string {
  return id ? `/edit/${encodeURIComponent(id)}` : '/new';
}
