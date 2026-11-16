import { describe, expect, it } from 'vitest';
import { FALLBACK_FAVICON, faviconHref, markSvg, syncFavicon } from './mark';

describe('markSvg', () => {
  it('paints the ground and facets in the accent and the die in on-accent', () => {
    const svg = markSvg({ accent: '#3e63dd', onAccent: '#ffffff' });
    expect(svg).toContain('<rect width="40" height="40" rx="9" fill="#3e63dd"');
    expect(svg).toContain('fill="#ffffff"');
    expect(svg).toContain('stroke="#3e63dd"');
  });
});

describe('faviconHref', () => {
  it('encodes the themed mark as an SVG data URI', () => {
    const href = faviconHref({ accent: '#3e63dd', onAccent: '#ffffff' });
    expect(href.startsWith('data:image/svg+xml,')).toBe(true);
    expect(decodeURIComponent(href)).toContain('fill="#3e63dd"');
  });

  it('falls back to the shipped favicon when a colour is missing', () => {
    expect(faviconHref({ accent: '', onAccent: '#ffffff' })).toBe(FALLBACK_FAVICON);
    expect(faviconHref({ accent: '#3e63dd', onAccent: '  ' })).toBe(FALLBACK_FAVICON);
  });

  it('rejects values that are not plain colours', () => {
    expect(faviconHref({ accent: 'red"/><script>', onAccent: '#fff' })).toBe(FALLBACK_FAVICON);
  });
});

describe('syncFavicon', () => {
  function fakeDocument(colours: Record<string, string>) {
    const link = { rel: 'icon', href: '/favicon.svg', type: 'image/svg+xml' };
    const doc = {
      documentElement: {},
      querySelector: (selector: string) => (selector === 'link[rel="icon"]' ? link : null),
    };
    const read = () => ({ getPropertyValue: (name: string) => colours[name] ?? '' });
    return { link, doc, read };
  }

  it('points the icon link at the mark in the active theme colours', () => {
    const { link, doc, read } = fakeDocument({
      '--sph-accent': ' #3e63dd',
      '--sph-on-accent': '#ffffff ',
    });
    syncFavicon(doc as unknown as Document, read as unknown as typeof getComputedStyle);
    expect(decodeURIComponent(link.href)).toContain('fill="#3e63dd"');
  });

  it('keeps /favicon.svg when the theme tokens are not there', () => {
    const { link, doc, read } = fakeDocument({});
    syncFavicon(doc as unknown as Document, read as unknown as typeof getComputedStyle);
    expect(link.href).toBe(FALLBACK_FAVICON);
  });
});
