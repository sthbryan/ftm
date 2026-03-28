const THEMES = [
  'nord',
  'nord-light',
  'rose-pine',
  'rose-pine-dawn',
  'tokyo-night',
  'tokyo-night-storm',
  'tokyo-night-light',
  'catppuccin-mocha',
  'catppuccin-latte',
  'one-dark',
  'gruvbox',
  'gruvbox-light',
  'solarized-dark',
  'solarized-light',
  'dracula',
  'red',
  'blue',
  'purple'
] as const;

type Theme = typeof THEMES[number];

const STORAGE_KEY = 'ftm-theme';

let currentTheme = $state<Theme>('dracula');

function getInitialTheme(): Theme {
  if (typeof window === 'undefined') return 'dracula';
  
  const saved = localStorage.getItem(STORAGE_KEY);
  if (saved && THEMES.includes(saved as Theme)) return saved as Theme;
  
  return 'dracula';
}

export function useTheme() {
  return {
    get current() { return currentTheme; },
    get themes() { return [...THEMES]; },
    
    init() {
      currentTheme = getInitialTheme();
      document.documentElement.setAttribute('data-theme', currentTheme);
    },
    
    set(theme: string) {
      if (!THEMES.includes(theme as Theme)) return;
      currentTheme = theme as Theme;
      document.documentElement.setAttribute('data-theme', theme);
      localStorage.setItem(STORAGE_KEY, theme);
    },
    
    toggle() {
      const idx = THEMES.indexOf(currentTheme);
      const next = THEMES[(idx + 1) % THEMES.length];
      this.set(next);
    }
  };
}
