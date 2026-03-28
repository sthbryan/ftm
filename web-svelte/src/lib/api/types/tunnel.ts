export interface Tunnel {
  id: string;
  name: string;
  provider: string;
  providerName: string;
  port: number;
  state: TunnelState;
  publicUrl?: string;
  errorMessage?: string;
}

export type TunnelState = 'stopped' | 'starting' | 'installing' | 'online' | 'error';

export interface CreateTunnelInput {
  name: string;
  provider: string;
  localPort: number;
}

export interface UpdateTunnelInput {
  name?: string;
  provider?: string;
  localPort?: number;
}

export interface StartResponse {
  status: 'ok' | 'installing';
}
