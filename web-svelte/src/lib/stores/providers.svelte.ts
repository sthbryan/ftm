import { providersApi } from '$lib/api';

interface Provider {
  id: string;
  name: string;
}

let providers: Provider[] = $state([]);
let loading = $state(false);
let error: string | null = $state(null);

export function useProviders() {
  return {
    get providers() { return providers; },
    get loading() { return loading; },
    get error() { return error; },
    
    async fetch() {
      loading = true;
      error = null;
      try {
        providers = await providersApi.getAll();
      } catch (e) {
        error = (e as Error).message;
      } finally {
        loading = false;
      }
    }
  };
}

export function detectPort(options?: { forceRefresh?: boolean }): Promise<number> {
  return providersApi.detectPort(options);
}
