import { render } from 'preact';
import { App } from './app';
import { initTheme } from './lib/stores/theme';
import './styles/app.css';

initTheme();

const root = document.getElementById('app');
if (root) render(<App />, root);
