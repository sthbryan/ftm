import { api } from '../client';
import type { Tunnel, CreateTunnelInput, UpdateTunnelInput, StartResponse } from '../types';

export const tunnelsApi = {
  getAll: (): Promise<Tunnel[]> => api.get('tunnels').json<Tunnel[]>(),

  getById: (id: string): Promise<Tunnel> =>
    api.get(`tunnels/${id}`).json<Tunnel>(),

  create: (data: CreateTunnelInput): Promise<Tunnel> =>
    api.post('tunnels', { json: data }).json<Tunnel>(),

  update: (id: string, data: UpdateTunnelInput): Promise<Tunnel> =>
    api.put(`tunnels/${id}`, { json: data }).json<Tunnel>(),

  start: (id: string): Promise<StartResponse> =>
    api.post(`tunnels/${id}/start`).json<StartResponse>(),

  stop: async (id: string): Promise<void> => {
    await api.post(`tunnels/${id}/stop`);
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`tunnels/${id}`);
  },

  reorder: async (ids: string[]): Promise<void> => {
    await api.post('tunnels/reorder', { json: ids });
  },
};
