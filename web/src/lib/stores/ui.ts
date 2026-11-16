import { signal } from '@preact/signals';
import { connectionPath, navigate } from '@/router';

/** Tunnel the dashboard should focus, raised by the command palette. */
export const selectIntent = signal<string | null>(null);

export function requestCreate(): void {
  navigate(connectionPath());
}

export function requestSelect(id: string): void {
  selectIntent.value = id;
  navigate('/');
}
