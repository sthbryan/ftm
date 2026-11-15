import { computed, signal } from '@preact/signals';

/**
 * Themes are Sephiro's, plus `ftm` from ./styles/ftm.css. Each one is a
 * `data-sephiro-theme` value on <html>; the palette itself lives only in CSS.
 */
export const THEMES = ['ftm', 'dark', 'light', 'asterism', 'curie', 'fizza', 'pulso'] as const;

export type Theme = (typeof THEMES)[number];

const STORAGE_KEY = 'ftm-theme';
const DEFAULT_THEME: Theme = 'ftm';

type SystemScheme = 'dark' | 'light';

type Preference = { kind: 'system' } | { kind: 'manual'; theme: Theme };

export const preference = signal<Preference>({ kind: 'manual', theme: DEFAULT_THEME });
export const systemScheme = signal<SystemScheme>('dark');
export const currentTheme = signal<Theme>(DEFAULT_THEME);

export const isAutoTheme = computed(() => preference.value.kind === 'system');

const themeSet = new Set<string>(THEMES);

function isTheme(value: string): value is Theme {
  return themeSet.has(value);
}

function getSystemScheme(): SystemScheme {
  if (typeof window === 'undefined') return 'dark';
  return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
}

function resolveTheme(pref: Preference, scheme: SystemScheme): Theme {
  if (pref.kind === 'manual') return pref.theme;
  return scheme === 'light' ? 'light' : 'dark';
}

function loadPreference(): Preference {
  if (typeof window === 'undefined') return { kind: 'manual', theme: DEFAULT_THEME };

  const raw = localStorage.getItem(STORAGE_KEY);
  if (!raw) return { kind: 'manual', theme: DEFAULT_THEME };

  if (raw.startsWith('{')) {
    try {
      const parsed = JSON.parse(raw) as { mode?: string; theme?: string };
      if (parsed.mode === 'system') return { kind: 'system' };
      if (parsed.mode === 'manual' && parsed.theme && isTheme(parsed.theme)) {
        return { kind: 'manual', theme: parsed.theme };
      }
    } catch {
      /* fall through to the default */
    }
  }

  // Values written by the old palette (nord, gruvbox, …) no longer exist.
  if (isTheme(raw)) return { kind: 'manual', theme: raw };

  return { kind: 'manual', theme: DEFAULT_THEME };
}

function savePreference(pref: Preference): void {
  if (typeof window === 'undefined') return;
  localStorage.setItem(STORAGE_KEY, JSON.stringify(pref));
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
  preference.value = loadPreference();
  currentTheme.value = resolveTheme(preference.value, systemScheme.value);

  window.matchMedia('(prefers-color-scheme: light)').addEventListener('change', (event) => {
    systemScheme.value = event.matches ? 'light' : 'dark';
    if (preference.value.kind === 'system') {
      currentTheme.value = resolveTheme(preference.value, systemScheme.value);
      applyTheme(currentTheme.value);
    }
  });

  applyTheme(currentTheme.value);
}

export function setTheme(theme: string): void {
  if (!isTheme(theme)) return;
  preference.value = { kind: 'manual', theme };
  currentTheme.value = theme;
  savePreference(preference.value);
  applyTheme(theme);
}

/** Follow the OS: Sephiro's own light/dark pair. */
export function followSystem(): void {
  preference.value = { kind: 'system' };
  currentTheme.value = resolveTheme(preference.value, systemScheme.value);
  savePreference(preference.value);
  applyTheme(currentTheme.value);
}

/** Step to the next theme; used by the command palette and Shift+T. */
export function cycleTheme(): void {
  const index = THEMES.indexOf(currentTheme.value);
  setTheme(THEMES[(index + 1) % THEMES.length]);
}
