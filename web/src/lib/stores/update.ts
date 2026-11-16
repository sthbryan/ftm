import { signal } from '@preact/signals';
import { updateApi, type UpdateInfo } from '@/lib/api';
import { subscribeEvents } from '@/lib/api/events';

export const updateInfo = signal<UpdateInfo | null>(null);
export const updateApplying = signal(false);
export const updateError = signal<string | null>(null);

let unsubscribe: (() => void) | null = null;

function listen(): void {
  if (unsubscribe) return;
  unsubscribe = subscribeEvents((event) => {
    if (event.type === 'update_available' || event.type === 'resync') void checkForUpdate();
  });
}

/** Load the update state, then keep it current from update_available events. */
export async function checkForUpdate(): Promise<void> {
  listen();
  try {
    updateInfo.value = await updateApi.get();
  } catch {
    // Offline or a build without the update endpoint: no update to show.
    updateInfo.value = null;
  }
}

export async function applyUpdate(): Promise<void> {
  updateApplying.value = true;
  updateError.value = null;

  try {
    await updateApi.apply();
  } catch (error) {
    updateApplying.value = false;
    updateError.value = error instanceof Error ? error.message : String(error);
  }
}
