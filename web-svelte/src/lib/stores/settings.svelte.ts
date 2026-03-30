import { settingsApi, type Settings } from '$lib/api';

let settings = $state<Settings>({
  notifications_enabled: false,
  notification_sound: true
});

let loaded = $state(false);

async function load() {
  settings = await settingsApi.get();
  loaded = true;
}

async function update(partial: Partial<Settings>) {
  const old = { ...settings };
  settings = { ...settings, ...partial };
  try {
    settings = await settingsApi.update(partial);
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
