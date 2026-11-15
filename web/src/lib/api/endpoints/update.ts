import { api } from '../client';

export type UpdateMethod = 'self' | 'store' | 'homebrew' | 'download';

export interface UpdateInfo {
  current: string;
  latest: string;
  tag: string;
  assetName: string;
  releaseUrl: string;
  hasUpdate: boolean;
  method: UpdateMethod;
}

export const updateApi = {
  get: (): Promise<UpdateInfo> => api.get('update').json(),
  apply: (): Promise<{ ok: boolean }> => api.post('update', { json: {} }).json(),
};
