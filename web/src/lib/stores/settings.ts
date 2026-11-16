import { signal } from '@preact/signals';
import { settingsApi, type Settings, type SettingsPatch } from '@/lib/api';
import { applyNotificationSettings } from './notification';

export const settings = signal<Settings>({
  notifications_enabled: 'pending',
  notification_sound: true,
  language: 'en',
  language_resolved: 'en',
  onboarded: true,
  autostart_supported: false,
  autostart_enabled: false,
});

export const settingsLoaded = signal(false);

export async function loadSettings(): Promise<void> {
  settings.value = await settingsApi.get();
  applyNotificationSettings(settings.value);
  settingsLoaded.value = true;
}

export async function updateSettings(partial: SettingsPatch): Promise<Settings> {
  const previous = { ...settings.value };
  settings.value = { ...settings.value, ...partial };

  try {
    settings.value = await settingsApi.update(partial);
    applyNotificationSettings(settings.value);
    return settings.value;
  } catch (error) {
    settings.value = previous;
    throw error;
  }
}
