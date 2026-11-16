import { api } from '../client';
import type { Settings, SettingsPatch } from '../types';


export const settingsApi = {
  get: (): Promise<Settings> => api.get('settings').json(),
  update: (settings: SettingsPatch): Promise<Settings> =>
    api.patch('settings', { json: settings }).json(),
};
