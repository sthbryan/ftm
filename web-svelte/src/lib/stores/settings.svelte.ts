import type { Settings } from '../api';
import { settingsApi } from '../api';
import { useNotifications } from './notification.svelte';

let settings = $state<Settings>({
  notifications_enabled: "pending",
  notification_sound: true,
  language: "en"
});

let loaded = $state(false);

const notifications = useNotifications();

async function load() {
  settings = await settingsApi.get();
  notifications.applySettings(settings);
  loaded = true;
}

async function update(partial: Partial<Settings>) {
  const old = { ...settings };
  settings = { ...settings, ...partial };
  try {
    settings = await settingsApi.update(partial);
    notifications.applySettings(settings);
    return settings;
  } catch (e) {
    settings = old;
    throw e;
  }
}

export function useSettings() {
  return {
    get settings() { return settings; },
    get loaded() { return loaded; },
    load,
    update
  };
}
