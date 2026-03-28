export interface Tunnel {
  id: string;
  name: string;
  provider: string;
  port: number;
  state: string;
  publicUrl?: string;
  errorMessage?: string;
  expiresAt?: number;
}

export type TunnelState = 
  | 'online' 
  | 'starting' 
  | 'connecting' 
  | 'installing' 
  | 'downloading' 
  | 'stopping' 
  | 'stopped' 
  | 'offline' 
  | 'timeout' 
  | 'error';

export interface TunnelFormData {
  name: string;
  provider: string;
  localPort: number;
}

export interface DropdownOption {
  label: string;
  action?: string;
  icon?: unknown;
  disabled?: boolean;
  danger?: boolean;
  value?: string;
}

export interface LogStream {
  close: () => void;
}
