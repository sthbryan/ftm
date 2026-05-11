import { api } from '../client';

export interface Settings {
  notifications_enabled: "granted" | "pending" | "rejected";
  notification_sound: boolean;
  language: string;
}

export const settingsApi = {
  get: (): Promise<Settings> => api.get('settings').json(),
  update: (settings: Partial<Settings>): Promise<Settings> =>
    api.patch('settings', { json: settings }).json(),
};
