import { api } from '../client';

export const pipApi = {
  open: async (id: string): Promise<void> => {
    await api.post('pip', { json: { id } });
  }
};
