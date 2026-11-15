import { computed, signal } from '@preact/signals';

/**
 * ftm ships one theme, Tinta, in a dark and a light mode (./styles/ftm.css).
 * Each mode is a `data-sephiro-theme` value on <html>; the palette lives only
 * in CSS. The user picks a mode, or lets the OS pick it.
 */
export const THEMES = ['tinta', 'tinta-light'] as const;

export type Theme = (typeof THEMES)[number];

export const THEME_MODES = ['system', 'dark', 'light'] as const;

export type ThemeMode = (typeof THEME_MODES)[number];

const STORAGE_KEY = 'ftm-theme';
const DEFAULT_MODE: ThemeMode = 'system';

type SystemScheme = 'dark' | 'light';

export const themeMode = signal<ThemeMode>(DEFAULT_MODE);
export const systemScheme = signal<SystemScheme>('dark');
export const currentTheme = computed<Theme>(() => resolveTheme(themeMode.value, systemScheme.value));

export const isAutoTheme = computed(() => themeMode.value === 'system');

function isMode(value: unknown): value is ThemeMode {
  return typeof value === 'string' && (THEME_MODES as readonly string[]).includes(value);
}

function getSystemScheme(): SystemScheme {
  if (typeof window === 'undefined') return 'dark';
  return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
}

function resolveTheme(mode: ThemeMode, scheme: SystemScheme): Theme {
  const effective = mode === 'system' ? scheme : mode;
  return effective === 'light' ? 'tinta-light' : 'tinta';
}

function loadMode(): ThemeMode {
  if (typeof window === 'undefined') return DEFAULT_MODE;

  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return DEFAULT_MODE;
    if (isMode(raw)) return raw;

    // Earlier builds stored {mode, theme} with a palette name (ftm, nord, pulso…).
    // Those palettes are gone, so the choice falls back to following the OS.
    const parsed = JSON.parse(raw) as { mode?: unknown };
    if (isMode(parsed.mode)) return parsed.mode;
  } catch {
    /* unreadable or blocked storage: use the default */
  }

  return DEFAULT_MODE;
}

function saveMode(mode: ThemeMode): void {
  try {
    localStorage.setItem(STORAGE_KEY, mode);
  } catch {
    /* storage blocked; the choice lasts for this session */
  }
}

function applyTheme(theme: Theme): void {
  if (typeof document === 'undefined') return;
  document.documentElement.setAttribute('data-sephiro-theme', theme);
}

let initialized = false;

export function initTheme(): void {
  if (initialized) return;
  initialized = true;

  systemScheme.value = getSystemScheme();
  themeMode.value = loadMode();

  window.matchMedia('(prefers-color-scheme: light)').addEventListener('change', (event) => {
    systemScheme.value = event.matches ? 'light' : 'dark';
    applyTheme(currentTheme.value);
  });

  applyTheme(currentTheme.value);
}

export function setThemeMode(mode: string): void {
  if (!isMode(mode)) return;
  themeMode.value = mode;
  saveMode(mode);
  applyTheme(currentTheme.value);
}

/** Follow the OS between Tinta dark and light. */
export function followSystem(): void {
  setThemeMode('system');
}

/** Step system → dark → light; used by the command palette and ⌘⇧T. */
export function cycleTheme(): void {
  const index = THEME_MODES.indexOf(themeMode.value);
  setThemeMode(THEME_MODES[(index + 1) % THEME_MODES.length]);
}
