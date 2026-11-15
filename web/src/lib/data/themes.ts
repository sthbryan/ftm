import type { Theme } from '@/lib/stores/theme';

export interface ThemeOption {
  id: Theme;
  label: string;
  /** Swatch colours, copied from the theme sheet so the picker needs no DOM probe. */
  bg: string;
  surface: string;
  accent: string;
}

export const themeOptions: ThemeOption[] = [
  { id: 'ftm', label: 'Foundry', bg: '#12100d', surface: '#191612', accent: '#b98c2f' },
  { id: 'dark', label: 'Sephiro', bg: '#151714', surface: '#1c201b', accent: '#4f7658' },
  { id: 'light', label: 'Sephiro Light', bg: '#f8f1f2', surface: '#fffafb', accent: '#a84f6d' },
  { id: 'asterism', label: 'Asterism', bg: '#111111', surface: '#1c1c1c', accent: '#d71921' },
  { id: 'curie', label: 'Curie', bg: '#0a0a0a', surface: '#131313', accent: '#d71921' },
  { id: 'fizza', label: 'Fizza', bg: '#000000', surface: '#111111', accent: '#7c3aed' },
  { id: 'pulso', label: 'Pulso', bg: '#f5f5f0', surface: '#fcfcf8', accent: '#3d7049' },
];
