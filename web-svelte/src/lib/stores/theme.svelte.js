const THEMES = ['light', 'dark', 'sepia', 'contrast', 'red', 'blue', 'dracula'];
const STORAGE_KEY = 'ftm-theme';

let currentTheme = $state('light');

function getInitialTheme() {
  if (typeof window === 'undefined') return 'light';
  
  const saved = localStorage.getItem(STORAGE_KEY);
  if (saved && THEMES.includes(saved)) return saved;
  
  if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
    return 'dark';
  }
  
  return 'light';
}

export function useTheme() {
  return {
    get current() { return currentTheme; },
    get themes() { return THEMES; },
    
    init() {
      currentTheme = getInitialTheme();
      document.documentElement.setAttribute('data-theme', currentTheme);
    },
    
    set(theme) {
      if (!THEMES.includes(theme)) return;
      currentTheme = theme;
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
