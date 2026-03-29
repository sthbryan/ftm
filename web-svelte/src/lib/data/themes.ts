export interface ThemeGroup {
  name: string;
  themes: ThemeGroupItem[];
}

export interface ThemeGroupItem {
  id: string;
  color: string;
}

export const themeGroups: ThemeGroup[] = [
  {
    name: 'Dark',
    themes: [
      { id: 'dracula', color: '#bd93f9' },
      { id: 'nord', color: '#88c0d0' },
      { id: 'tokyo-night', color: '#7aa2f7' },
      { id: 'tokyo-night-storm', color: '#73daca' },
      { id: 'catppuccin-mocha', color: '#cba6f7' },
      { id: 'one-dark', color: '#61afef' },
      { id: 'gruvbox', color: '#fabd2f' },
      { id: 'solarized-dark', color: '#268bd2' },
      { id: 'rose-pine', color: '#ebbcba' },
      { id: 'red', color: '#ff5555' },
      { id: 'blue', color: '#8be9fd' },
      { id: 'purple', color: '#ff79c6' },
    ]
  },
  {
    name: 'Light',
    themes: [
      { id: 'nord-light', color: '#e5e9f0' },
      { id: 'tokyo-night-light', color: '#7aa2f7' },
      { id: 'catppuccin-latte', color: '#ca9ee6' },
      { id: 'gruvbox-light', color: '#98971a' },
      { id: 'solarized-light', color: '#2aa198' },
      { id: 'rose-pine-dawn', color: '#f2d6cd' },
      { id: 'red-light', color: '#ff6e6e' },
      { id: 'blue-light', color: '#8be9fd' },
      { id: 'purple-light', color: '#ff79c6' },
    ]
  }
];
