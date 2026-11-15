import { signal } from '@preact/signals';

/** Bumped when something outside the dashboard asks for the create dialog. */
export const createIntent = signal(0);

/** Tunnel the dashboard should focus, raised by the command palette. */
export const selectIntent = signal<string | null>(null);

export function requestCreate(): void {
  createIntent.value += 1;
}

export function requestSelect(id: string): void {
  selectIntent.value = id;
}
