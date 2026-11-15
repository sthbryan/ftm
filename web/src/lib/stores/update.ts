import { signal } from '@preact/signals';
import { updateApi, type UpdateInfo } from '@/lib/api';

export const updateInfo = signal<UpdateInfo | null>(null);
export const updateApplying = signal(false);
export const updateError = signal<string | null>(null);

export async function checkForUpdate(): Promise<void> {
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
