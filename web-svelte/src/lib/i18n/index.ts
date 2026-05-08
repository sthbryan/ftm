import { writable, derived, get } from 'svelte/store';

export interface Translations {
  [key: string]: string;
}

interface I18nState {
  translations: Translations;
  language: string;
  available: string[];
  loading: boolean;
}

function createI18nStore() {
  const { subscribe, set, update } = writable<I18nState>({
    translations: {},
    language: 'en',
    available: ['en', 'es'],
    loading: true,
  });

  return {
    subscribe,
    
    async init() {
      try {
        const res = await fetch('/api/i18n');
        const data = await res.json();
        
        update(state => ({
          ...state,
          translations: data.translations || {},
          language: data.current,
          available: data.available,
          loading: false,
        }));
      } catch (e) {
        console.error('Failed to load i18n:', e);
        update(state => ({ ...state, loading: false }));
      }
    },

    async setLanguage(lang: string) {
      try {
        await fetch('/api/settings', {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ language: lang }),
        });

        const res = await fetch('/api/i18n?lang=' + lang);
        const data = await res.json();
        
        update(state => ({
          ...state,
          translations: data.translations || {},
          language: data.current,
          available: data.available,
        }));
      } catch (e) {
        console.error('Failed to set language:', e);
      }
    },

    refresh() {
      return this.init();
    },
  };
}

export const i18n = createI18nStore();


export function t(key: string, params?: Record<string, string>): string {
  const state = get(i18n);
  let text = state.translations[key] || key;
  
  if (params) {
    Object.entries(params).forEach(([k, v]) => {
      text = text.replace(new RegExp(`\\{${k}\\}`, 'g'), v);
    });
  }
  
  return text;
}


export const translations = derived(i18n, $i18n => $i18n.translations);
export const currentLanguage = derived(i18n, $i18n => $i18n.language);
export const availableLanguages = derived(i18n, $i18n => $i18n.available);


export const translate = derived(i18n, ($i18n) => {
  return (key: string, params?: Record<string, string>): string => {
    let text = $i18n.translations[key] || key;
    
    if (params) {
      Object.entries(params).forEach(([k, v]) => {
        text = text.replace(new RegExp(`\\{${k}\\}`, 'g'), v);
      });
    }
    
    return text;
  };
});