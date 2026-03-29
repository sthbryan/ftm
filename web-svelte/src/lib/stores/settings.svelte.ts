import { getSettings, updateSettings, type Settings } from '$lib/api';

let settings = $state<Settings>({
  notifications_enabled: false,
  notification_sound: true
});

let loaded = $state(false);

async function load() {
  settings = await getSettings();
  loaded = true;
}

async function update(partial: Partial<Settings>) {
  const old = { ...settings };
  settings = { ...settings, ...partial };
  try {
    settings = await updateSettings(partial);
  } catch {
    settings = old;
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
