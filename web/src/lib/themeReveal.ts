import { withViewTransition } from '@/router';

let origin: { x: number; y: number } | null = null;

/** Remember where the pointer went down, so the reveal starts there. */
export function rememberRevealOrigin(event: PointerEvent): void {
  origin = { x: event.clientX, y: event.clientY };
}

/**
 * Applies a theme change inside a view transition whose new picture grows as
 * a circle from the last press (or the middle of the window), so the new
 * colours spread from the thumbnail that was picked. Without view transitions,
 * or with reduced motion, the theme just changes.
 */
export function revealTheme(change: () => void): void {
  const root = document.documentElement;
  const at = origin ?? { x: window.innerWidth / 2, y: window.innerHeight / 2 };
  origin = null;
  const radius = Math.hypot(Math.max(at.x, window.innerWidth - at.x), Math.max(at.y, window.innerHeight - at.y));

  root.style.setProperty('--ftm-reveal-x', `${at.x}px`);
  root.style.setProperty('--ftm-reveal-y', `${at.y}px`);
  root.style.setProperty('--ftm-reveal-r', `${radius}px`);
  root.dataset.reveal = 'theme';
  withViewTransition(change);
  // The attribute only shapes this transition; page changes keep their fade.
  setTimeout(() => delete root.dataset.reveal, 700);
}
