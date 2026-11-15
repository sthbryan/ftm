export function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined') return false;
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

export function revealDuration(ms = 200): number {
  return prefersReducedMotion() ? 0 : ms;
}
