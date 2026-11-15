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
