import { api } from '../client';
import type { StatusResponse } from '../types';


export const statusApi = {
  get: (): Promise<StatusResponse> => api.get('status').json(),
};
