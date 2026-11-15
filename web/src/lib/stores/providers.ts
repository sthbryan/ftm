import { signal } from '@preact/signals';
import { providersApi } from '@/lib/api';

interface Provider {
  id: string;
  name: string;
}

export const providers = signal<Provider[]>([]);
export const providersLoading = signal(false);
export const providersError = signal<string | null>(null);

export async function fetchProviders(): Promise<void> {
  providersLoading.value = true;
  providersError.value = null;
  try {
    providers.value = await providersApi.getAll();
  } catch (error) {
    providersError.value = (error as Error).message;
  } finally {
    providersLoading.value = false;
  }
}

export function providerLabel(id: string): string {
  return providers.value.find((provider) => provider.id === id)?.name ?? id;
}

export function detectPort(options?: { forceRefresh?: boolean }): Promise<number> {
  return providersApi.detectPort(options);
}
