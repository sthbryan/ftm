import type { Theme } from "@/lib/stores/theme";

export interface ThemeVariant {
  id: Theme;
  color: string;
}

export interface ThemeFamily {
  id: string;
  name: string;
  dark: ThemeVariant;
  light: ThemeVariant;
}

export const themeFamilies: ThemeFamily[] = [
  {
    id: 'foundry',
    name: 'Foundry',
    dark: { id: 'foundry-dark', color: '#c9a227' },
    light: { id: 'foundry-light', color: '#8a6d1f' },
  },
  {
    id: 'nord',
    name: 'Nord',
    dark: { id: 'nord', color: '#88c0d0' },
    light: { id: 'nord-light', color: '#5e81ac' },
  },
  {
    id: 'catppuccin',
    name: 'Catppuccin',
    dark: { id: 'catppuccin-mocha', color: '#cba6f7' },
    light: { id: 'catppuccin-latte', color: '#8839ef' },
  },
  {
    id: 'rose-pine',
    name: 'Rose Pine',
    dark: { id: 'rose-pine', color: '#c4a7e7' },
    light: { id: 'rose-pine-dawn', color: '#907aa9' },
  },
  {
    id: 'gruvbox',
    name: 'Gruvbox',
    dark: { id: 'gruvbox', color: '#fe8019' },
    light: { id: 'gruvbox-light', color: '#af3a03' },
  },
  {
    id: 'tokyo-night',
    name: 'Tokyo Night',
    dark: { id: 'tokyo-night', color: '#82aaff' },
    light: { id: 'tokyo-night-day', color: '#2e7de9' },
  },
  {
    id: 'dracula',
    name: 'Dracula',
    dark: { id: 'dracula', color: '#bd93f9' },
    light: { id: 'alucard', color: '#644ac9' },
  },
  {
    id: 'kanagawa',
    name: 'Kanagawa',
    dark: { id: 'kanagawa', color: '#957fb8' },
    light: { id: 'kanagawa-lotus', color: '#624c83' },
  },
  {
    id: 'one',
    name: 'One',
    dark: { id: 'one-dark', color: '#61afef' },
    light: { id: 'one-light', color: '#4078f2' },
  },
  {
    id: 'ayu',
    name: 'Ayu',
    dark: { id: 'ayu', color: '#ff8f40' },
    light: { id: 'ayu-light', color: '#fa8d3e' },
  },
  {
    id: 'everforest',
    name: 'Everforest',
    dark: { id: 'everforest', color: '#a7c080' },
    light: { id: 'everforest-light', color: '#8da101' },
  },
  {
    id: 'solarized',
    name: 'Solarized',
    dark: { id: 'solarized', color: '#268bd2' },
    light: { id: 'solarized-light', color: '#268bd2' },
  },
];