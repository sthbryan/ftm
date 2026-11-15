import type { TunnelState } from '@/lib/types';

export type StatusKey = 'running' | 'starting' | 'installing' | 'error' | 'stopped';

export interface StatusInfo {
  key: StatusKey;
  textKey: string;
}

export const STATUS_FILL: Record<StatusKey, string> = {
  running: 'bg-status-running',
  starting: 'bg-status-starting',
  installing: 'bg-status-installing',
  error: 'bg-status-error',
  stopped: ''
};

const STATUS_MAP: Record<TunnelState, StatusInfo> = {
  online: { key: 'running', textKey: 'online' },
  starting: { key: 'starting', textKey: 'starting' },
  connecting: { key: 'starting', textKey: 'connecting' },
  installing: { key: 'installing', textKey: 'installing' },
  downloading: { key: 'installing', textKey: 'downloading' },
  need_installing: { key: 'stopped', textKey: 'need_installing' },
  stopping: { key: 'starting', textKey: 'stopping' },
  stopped: { key: 'stopped', textKey: 'stopped' },
  offline: { key: 'stopped', textKey: 'offline' },
  timeout: { key: 'error', textKey: 'timeout' },
  error: { key: 'error', textKey: 'error' }
};

const RUNNING_STATES: TunnelState[] = [
  'online',
  'starting',
  'connecting',
  'installing',
  'downloading',
  'stopping'
];

export function statusInfo(state: TunnelState | string): StatusInfo {
  return STATUS_MAP[state as TunnelState] ?? STATUS_MAP.error;
}

export function isRunningState(state: TunnelState | string): boolean {
  return RUNNING_STATES.includes(state as TunnelState);
}

export function isInstallingState(state: TunnelState | string): boolean {
  return state === 'installing' || state === 'downloading';
}
