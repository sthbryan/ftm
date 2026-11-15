import { api } from '../client';

export interface StatusResponse {
  port: number;
  version: string;
  nativePip?: boolean;
}

export const statusApi = {
  get: (): Promise<StatusResponse> => api.get('status').json(),
};
