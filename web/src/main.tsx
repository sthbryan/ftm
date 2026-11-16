import { render } from 'preact';
import { App } from './app';
import { initTheme } from './lib/stores/theme';
import { markDesktopShell } from './lib/desktop';
import './styles/app.css';

markDesktopShell();
initTheme();

const root = document.getElementById('app');
if (root) render(<App />, root);
