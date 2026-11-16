import { signal } from '@preact/signals';
import { api } from '@/lib/api/client';

export const LANGUAGE_AUTO = 'auto';

export const translations = signal<Record<string, string>>({});
export const language = signal('en');
export const availableLanguages = signal<string[]>(['en', 'es', 'pt']);
export const i18nReady = signal(false);

/**
 * Translate a key, substituting {name} placeholders. Reading the signal during
 * render is what re-renders the tree after a language switch; an unknown key
 * returns the key itself, the same as the Go side.
 */
export function t(key: string, params?: Record<string, string | number>): string {
  let text = translations.value[key] ?? key;

  if (params) {
    for (const [name, value] of Object.entries(params)) {
      text = text.replaceAll(`{${name}}`, String(value));
    }
  }

  return text;
}

interface I18nResponse {
  translations?: Record<string, string>;
  current?: string;
  available?: string[];
}

async function fetchCatalogue(lang?: string): Promise<void> {
  const path = lang ? `i18n?lang=${encodeURIComponent(lang)}` : 'i18n';
  const data = await api.get(path).json<I18nResponse>();

  translations.value = data.translations ?? {};
  if (data.current) language.value = data.current;
  if (data.available) availableLanguages.value = data.available;
}

/** Ready is set even on failure, so an unreachable API shows keys, not a blank page. */
export async function initI18n(): Promise<void> {
  if (i18nReady.value) return;

  try {
    await fetchCatalogue();
  } catch (error) {
    console.error('i18n: failed to load translations', error);
  } finally {
    i18nReady.value = true;
  }
}

export async function setLanguage(lang: string): Promise<void> {
  await fetchCatalogue(lang === LANGUAGE_AUTO ? undefined : lang);
}
