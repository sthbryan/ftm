import { providersApi, api } from '$lib/api';

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

export async function detectPort(): Promise<number> {
  try {
    const data = await api.get('detect-port').json<{ suggested?: number }>();
    return data.suggested || 30000;
  } catch {
    return 30000;
  }
}
