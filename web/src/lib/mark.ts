/**
 * The d20 mark, drawn once. The in-app <Mark> and the favicon both read these
 * paths, and both take their colours from the active theme: the ground and the
 * facet strokes are the accent, the die is the colour that sits on the accent.
 */
export const MARK_DIE = 'M3.34 7 L12 2 L20.66 7 L20.66 17 L12 22 L3.34 17 Z';
export const MARK_FACE = 'M12 5.82 L17.352 15.09 L6.648 15.09 Z';
export const MARK_EDGES =
  'M12 2 L12 5.82 M3.34 7 L12 5.82 M20.66 7 L12 5.82 M3.34 7 L6.648 15.09 M20.66 7 L17.352 15.09 M3.34 17 L6.648 15.09 M20.66 17 L17.352 15.09 M12 22 L6.648 15.09 M12 22 L17.352 15.09';

export const FALLBACK_FAVICON = '/favicon.svg';

export type MarkColours = { accent: string; onAccent: string };

const COLOUR = /^(#[0-9a-f]{3,8}|(rgb|hsl|oklch|oklab|lab|lch)a?\([\d\s.,%/+-]+\))$/i;

export function markSvg({ accent, onAccent }: MarkColours): string {
  return (
    `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 40" width="40" height="40">` +
    `<rect width="40" height="40" rx="9" fill="${accent}"/>` +
    `<g transform="translate(8 8)">` +
    `<path d="${MARK_DIE}" fill="${onAccent}"/>` +
    `<g fill="none" stroke="${accent}" stroke-width="1.15" stroke-linejoin="round" stroke-linecap="round">` +
    `<path d="${MARK_FACE}"/><path d="${MARK_EDGES}"/>` +
    `</g></g></svg>`
  );
}

/** The favicon for these colours, or the shipped file when a colour is unusable. */
export function faviconHref(colours: MarkColours): string {
  const accent = colours.accent.trim();
  const onAccent = colours.onAccent.trim();
  if (!COLOUR.test(accent) || !COLOUR.test(onAccent)) return FALLBACK_FAVICON;
  return `data:image/svg+xml,${encodeURIComponent(markSvg({ accent, onAccent }))}`;
}

/** Repaint <link rel="icon"> from the theme tokens currently on <html>. */
export function syncFavicon(
  doc: Document = document,
  read: typeof getComputedStyle = getComputedStyle,
): void {
  const link = doc.querySelector<HTMLLinkElement>('link[rel="icon"]');
  if (!link) return;
  const style = read(doc.documentElement);
  link.href = faviconHref({
    accent: style.getPropertyValue('--sph-accent'),
    onAccent: style.getPropertyValue('--sph-on-accent'),
  });
}
