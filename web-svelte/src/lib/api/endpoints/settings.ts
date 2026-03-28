import { api } from '../client';

export interface Settings {
  notifications_enabled: boolean;
  notification_sound: boolean;
  theme: 'light' | 'dark' | 'system';
}

export async function getSettings(): Promise<Settings> {
  return api.get('settings').json();
}

export async function updateSettings(settings: Partial<Settings>): Promise<Settings> {
  return api.patch('settings', { json: settings }).json();
}
