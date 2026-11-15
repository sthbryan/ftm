import { computed, signal } from '@preact/signals';

const THEME_IDS = [
  'foundry-dark',
  'foundry-light',
  'nord',
  'nord-light',
  'catppuccin-mocha',
  'catppuccin-latte',
  'gruvbox',
  'gruvbox-light',
  'tokyo-night',
  'tokyo-night-day',
  'dracula',
  'alucard',
  'kanagawa',
  'kanagawa-lotus',
  'one-dark',
  'one-light',
  'ayu',
  'ayu-light',
  'everforest',
  'everforest-light',
  'solarized',
  'solarized-light',
  'rose-pine',
  'rose-pine-dawn',
] as const;

export type Theme = (typeof THEME_IDS)[number];

const FAMILY_THEMES: Record<string, { dark: Theme; light: Theme }> = {
  foundry: { dark: 'foundry-dark', light: 'foundry-light' },
  nord: { dark: 'nord', light: 'nord-light' },
  catppuccin: { dark: 'catppuccin-mocha', light: 'catppuccin-latte' },
  'rose-pine': { dark: 'rose-pine', light: 'rose-pine-dawn' },
  gruvbox: { dark: 'gruvbox', light: 'gruvbox-light' },
  'tokyo-night': { dark: 'tokyo-night', light: 'tokyo-night-day' },
  dracula: { dark: 'dracula', light: 'alucard' },
  kanagawa: { dark: 'kanagawa', light: 'kanagawa-lotus' },
  one: { dark: 'one-dark', light: 'one-light' },
  ayu: { dark: 'ayu', light: 'ayu-light' },
  everforest: { dark: 'everforest', light: 'everforest-light' },
  solarized: { dark: 'solarized', light: 'solarized-light' },
};

export const THEME_FAMILIES = FAMILY_THEMES;

const STORAGE_KEY = 'ftm-theme';
const DEFAULT_THEME: Theme = 'foundry-dark';
const DEFAULT_FAMILY = 'foundry';

type Preference =
  | { kind: 'auto'; family: string }
  | { kind: 'manual'; theme: Theme };

export const preference = signal<Preference>({ kind: 'auto', family: DEFAULT_FAMILY });
export const systemScheme = signal<'dark' | 'light'>('dark');
export const currentTheme = signal<Theme>(DEFAULT_THEME);

export const isAutoTheme = computed(() => preference.value.kind === 'auto');

const themeSet = new Set<string>(THEME_IDS);

function isTheme(value: string): value is Theme {
  return themeSet.has(value);
}

function isFamily(value: string): boolean {
  return value in FAMILY_THEMES;
}

function getSystemScheme(): 'dark' | 'light' {
  if (typeof window === 'undefined') return 'dark';
  return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
}

function resolveTheme(pref: Preference, scheme: 'dark' | 'light'): Theme {
  if (pref.kind === 'manual') return pref.theme;
  const family = FAMILY_THEMES[pref.family];
  if (!family) return DEFAULT_THEME;
  return scheme === 'light' ? family.light : family.dark;
}

function loadPreference(): Preference {
  if (typeof window === 'undefined') return { kind: 'auto', family: DEFAULT_FAMILY };

  const raw = localStorage.getItem(STORAGE_KEY);
  if (!raw) return { kind: 'auto', family: DEFAULT_FAMILY };

  if (raw.startsWith('{')) {
    try {
      const parsed = JSON.parse(raw) as { mode?: string; family?: string; theme?: string };
      if (parsed.mode === 'auto' && parsed.family && isFamily(parsed.family)) {
        return { kind: 'auto', family: parsed.family };
      }
      if (parsed.mode === 'manual' && parsed.theme && isTheme(parsed.theme)) {
        return { kind: 'manual', theme: parsed.theme };
      }
    } catch {
      /* fall through to the default */
    }
  }

  if (isTheme(raw)) return { kind: 'manual', theme: raw };

  return { kind: 'auto', family: DEFAULT_FAMILY };
}

function savePreference(pref: Preference): void {
  if (typeof window === 'undefined') return;

  const payload =
    pref.kind === 'auto'
      ? { mode: 'auto', family: pref.family }
      : { mode: 'manual', theme: pref.theme };

  localStorage.setItem(STORAGE_KEY, JSON.stringify(payload));
}

function applyTheme(theme: Theme): void {
  if (typeof document === 'undefined') return;
  document.documentElement.setAttribute('data-theme', theme);
}

let initialized = false;
let mediaQuery: MediaQueryList | null = null;

export function initTheme(): void {
  if (initialized) return;
  initialized = true;

  systemScheme.value = getSystemScheme();
  preference.value = loadPreference();
  currentTheme.value = resolveTheme(preference.value, systemScheme.value);

  mediaQuery = window.matchMedia('(prefers-color-scheme: light)');
  mediaQuery.addEventListener('change', (event) => {
    systemScheme.value = event.matches ? 'light' : 'dark';
    if (preference.value.kind === 'auto') {
      currentTheme.value = resolveTheme(preference.value, systemScheme.value);
      applyTheme(currentTheme.value);
    }
  });

  applyTheme(currentTheme.value);
}

export function setThemeFamily(family: string): void {
  if (!isFamily(family)) return;
  preference.value = { kind: 'auto', family };
  currentTheme.value = resolveTheme(preference.value, systemScheme.value);
  savePreference(preference.value);
  applyTheme(currentTheme.value);
}

export function setTheme(theme: string): void {
  if (!isTheme(theme)) return;
  preference.value = { kind: 'manual', theme };
  currentTheme.value = theme;
  savePreference(preference.value);
  applyTheme(currentTheme.value);
}

/** Step to the next theme; used by the command palette and Shift+T. */
export function cycleTheme(): void {
  const index = THEME_IDS.indexOf(currentTheme.value);
  setTheme(THEME_IDS[(index + 1) % THEME_IDS.length]);
}

export const themeIds = [...THEME_IDS] as Theme[];
