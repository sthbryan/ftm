import { api } from '../client';
import type { UpdateInfo } from '../types';


export const updateApi = {
  get: (): Promise<UpdateInfo> => api.get('update').json(),
  apply: (): Promise<{ ok: boolean }> => api.post('update', { json: {} }).json(),
};
