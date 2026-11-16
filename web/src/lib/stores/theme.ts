import { computed, signal } from '@preact/signals';
import { syncFavicon } from '@/lib/mark';

/**
 * A theme is a family with a dark and a light variant. <html> carries
 * `data-sephiro-theme="<family>"` and `data-theme="dark|light"`, the contract
 * Sephiro's own presets use, so a family is pure CSS: one Sephiro preset sheet
 * per family, imported in styles/sephiro.css. The mode picks the variant: Auto follows the OS,
 * Dark and Light force it. Switching mode keeps the family.
 */
export const FAMILIES = [
  { id: 'flexoki', name: 'Flexoki', source: 'Flexoki · Steph Ango' },
  { id: 'tinta', name: 'Tinta', source: 'Radix Sand · Prusia' },
  { id: 'sephiro', name: 'Sephiro', source: 'Sephiro' },
  { id: 'rose-pine', name: 'Rosé Pine', source: 'Rosé Pine' },
  { id: 'orquidea', name: 'Orquídea', source: 'Sephiro' },
  { id: 'pizarra', name: 'Pizarra', source: 'Radix Slate · Teal' },
  { id: 'asterism', name: 'Asterism', source: 'Sephiro' },
  { id: 'curie', name: 'Curie', source: 'Sephiro' },
  { id: 'fizza', name: 'Fizza', source: 'Sephiro' },
  { id: 'pulso', name: 'Pulso', source: 'Sephiro' },
] as const;

export type Family = (typeof FAMILIES)[number]['id'];

export const THEME_MODES = ['system', 'dark', 'light'] as const;

export type ThemeMode = (typeof THEME_MODES)[number];

export type Scheme = 'dark' | 'light';

export type ThemePreference = { family: Family; mode: ThemeMode };

export const DEFAULT_PREFERENCE: ThemePreference = { family: 'flexoki', mode: 'system' };

const STORAGE_KEY = 'ftm-theme';

/**
 * Sephiro's default pair answers to `dark` (and `dark` + `data-theme="light"`
 * for its light variant), so the Sephiro family is stored as `sephiro` but
 * written to <html> under Sephiro's own name.
 */
const ATTRIBUTE: Partial<Record<Family, string>> = { sephiro: 'dark' };

/** Palettes the first theme picker stored, mapped to the family and variant they looked like. */
const LEGACY_THEMES: Record<string, ThemePreference> = {
  ftm: { family: 'tinta', mode: 'dark' },
  tinta: { family: 'tinta', mode: 'dark' },
  'tinta-light': { family: 'tinta', mode: 'light' },
  dark: { family: 'sephiro', mode: 'dark' },
  light: { family: 'sephiro', mode: 'light' },
  asterism: { family: 'asterism', mode: 'dark' },
  curie: { family: 'curie', mode: 'dark' },
  fizza: { family: 'fizza', mode: 'dark' },
  pulso: { family: 'pulso', mode: 'light' },
};

function isFamily(value: unknown): value is Family {
  return typeof value === 'string' && FAMILIES.some((family) => family.id === value);
}

function isMode(value: unknown): value is ThemeMode {
  return typeof value === 'string' && (THEME_MODES as readonly string[]).includes(value);
}

/** Read whatever an earlier build stored under `ftm-theme`. */
export function parsePreference(raw: string | null): ThemePreference {
  if (!raw) return DEFAULT_PREFERENCE;
  // The Tinta-only build stored a bare mode; those users keep the look they had.
  if (isMode(raw)) return { family: 'tinta', mode: raw };
  if (Object.hasOwn(LEGACY_THEMES, raw)) return LEGACY_THEMES[raw];

  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return DEFAULT_PREFERENCE;
  }
  if (!parsed || typeof parsed !== 'object') return DEFAULT_PREFERENCE;

  const stored = parsed as { family?: unknown; mode?: unknown; theme?: unknown };
  if ('family' in stored) {
    return {
      family: isFamily(stored.family) ? stored.family : DEFAULT_PREFERENCE.family,
      mode: isMode(stored.mode) ? stored.mode : DEFAULT_PREFERENCE.mode,
    };
  }
  if (stored.mode === 'system') return { family: 'tinta', mode: 'system' };
  if (typeof stored.theme === 'string' && Object.hasOwn(LEGACY_THEMES, stored.theme)) {
    return LEGACY_THEMES[stored.theme];
  }
  return DEFAULT_PREFERENCE;
}

export function resolveScheme(mode: ThemeMode, system: Scheme): Scheme {
  return mode === 'system' ? system : mode;
}

export function htmlAttributes(family: Family, scheme: Scheme) {
  return { sephiroTheme: ATTRIBUTE[family] ?? family, theme: scheme };
}

export function nextFamily(family: Family): Family {
  const index = FAMILIES.findIndex((entry) => entry.id === family);
  return FAMILIES[(index + 1) % FAMILIES.length].id;
}

export const themeFamily = signal<Family>(DEFAULT_PREFERENCE.family);
export const themeMode = signal<ThemeMode>(DEFAULT_PREFERENCE.mode);
export const systemScheme = signal<Scheme>('dark');
export const currentScheme = computed<Scheme>(() =>
  resolveScheme(themeMode.value, systemScheme.value),
);
export const isAutoTheme = computed(() => themeMode.value === 'system');

function getSystemScheme(): Scheme {
  if (typeof window === 'undefined') return 'dark';
  return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
}

function loadPreference(): ThemePreference {
  try {
    return parsePreference(localStorage.getItem(STORAGE_KEY));
  } catch {
    return DEFAULT_PREFERENCE;
  }
}

function savePreference(): void {
  try {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ family: themeFamily.value, mode: themeMode.value }),
    );
  } catch {
    /* storage blocked; the choice lasts for this session */
  }
}

function applyTheme(): void {
  if (typeof document === 'undefined') return;
  const { sephiroTheme, theme } = htmlAttributes(themeFamily.value, currentScheme.value);
  const root = document.documentElement;
  root.setAttribute('data-sephiro-theme', sephiroTheme);
  root.setAttribute('data-theme', theme);
  syncFavicon();
}

let initialized = false;

export function initTheme(): void {
  if (initialized) return;
  initialized = true;

  systemScheme.value = getSystemScheme();
  const preference = loadPreference();
  themeFamily.value = preference.family;
  themeMode.value = preference.mode;

  window.matchMedia('(prefers-color-scheme: light)').addEventListener('change', (event) => {
    systemScheme.value = event.matches ? 'light' : 'dark';
    applyTheme();
  });

  applyTheme();
}

export function setThemeMode(mode: string): void {
  if (!isMode(mode)) return;
  themeMode.value = mode;
  savePreference();
  applyTheme();
}

export function setThemeFamily(family: string): void {
  if (!isFamily(family)) return;
  themeFamily.value = family;
  savePreference();
  applyTheme();
}

/** Follow the OS for the variant, keeping the family. */
export function followSystem(): void {
  setThemeMode('system');
}

/** Step to the next family in the current mode; used by the command palette and ⌘⇧T. */
export function cycleTheme(): void {
  setThemeFamily(nextFamily(themeFamily.value));
}
