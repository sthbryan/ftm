import { signal } from '@preact/signals';

function readPath(): string {
  return typeof window === 'undefined' ? '/' : window.location.pathname;
}

function readSearch(): string {
  return typeof window === 'undefined' ? '' : window.location.search;
}

export const route = signal(readPath());
export const search = signal(readSearch());

interface TransitionDocument {
  startViewTransition?: (
    update: () => Promise<void> | void,
  ) => { ready?: Promise<unknown> } | void;
}

/**
 * Swap pages inside a view transition when the browser has them and the user
 * has not asked for reduced motion; otherwise swap at once. The update waits a
 * task so Preact has rendered the new page before the browser snapshots it.
 */
export function withViewTransition(
  update: () => void,
  doc: TransitionDocument | undefined = typeof document === 'undefined' ? undefined : document,
  reducedMotion = typeof window !== 'undefined' &&
    window.matchMedia?.('(prefers-reduced-motion: reduce)').matches === true,
): void {
  if (!doc?.startViewTransition || reducedMotion) {
    update();
    return;
  }
  const transition = doc.startViewTransition(() => {
    update();
    return new Promise<void>((resolve) => setTimeout(resolve, 0));
  });
  // A skipped transition (hidden tab, a newer navigation) still applies the
  // update; only the animation is dropped, so its rejection is not an error.
  transition?.ready?.catch(() => undefined);
}

function sync(): void {
  route.value = readPath();
  search.value = readSearch();
}

if (typeof window !== 'undefined') {
  window.addEventListener('popstate', () => withViewTransition(sync));
}

export function navigate(to: string, options?: { replace?: boolean }): void {
  if (options?.replace) {
    window.history.replaceState({}, '', to);
  } else {
    window.history.pushState({}, '', to);
  }

  if (readPath() === route.value) {
    sync();
    return;
  }
  withViewTransition(sync);
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
