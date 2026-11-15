import { describe, it, expect } from 'vitest';
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const AA_NORMAL_TEXT = 4.5;

const themesDir = fileURLToPath(new URL('.', import.meta.url));

type Palette = Record<string, string>;

function parseThemes(): Record<string, Palette> {
  const themes: Record<string, Palette> = {};

  for (const file of readdirSync(themesDir).filter((name) => name.endsWith('.css'))) {
    const css = readFileSync(join(themesDir, file), 'utf8');
    const blocks = css.matchAll(/\[data-theme="([^"]+)"\]\s*\{([^}]*)\}/g);

    for (const [, name, body] of blocks) {
      const palette: Palette = {};
      for (const [, prop, value] of body.matchAll(/(--color-[\w-]+)\s*:\s*([^;]+);/g)) {
        palette[prop] = value.trim();
      }
      themes[name] = palette;
    }
  }

  return themes;
}

function resolve(palette: Palette, prop: string, seen = new Set<string>()): string {
  const value = palette[prop];
  if (!value) throw new Error(`missing ${prop}`);

  const reference = value.match(/^var\((--color-[\w-]+)\)$/);
  if (!reference) return value;

  if (seen.has(prop)) throw new Error(`circular ${prop}`);
  seen.add(prop);
  return resolve(palette, reference[1], seen);
}

function toRgb(hex: string): [number, number, number] {
  const normalized = hex.trim().replace('#', '');
  const full =
    normalized.length === 3
      ? normalized
          .split('')
          .map((c) => c + c)
          .join('')
      : normalized;

  if (!/^[0-9a-f]{6}$/i.test(full)) throw new Error(`unsupported color ${hex}`);

  return [
    parseInt(full.slice(0, 2), 16),
    parseInt(full.slice(2, 4), 16),
    parseInt(full.slice(4, 6), 16),
  ];
}

function luminance(rgb: [number, number, number]): number {
  const [r, g, b] = rgb.map((channel) => {
    const c = channel / 255;
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function ratio(foreground: string, background: string): number {
  const a = luminance(toRgb(foreground));
  const b = luminance(toRgb(background));
  const [light, dark] = a > b ? [a, b] : [b, a];
  return (light + 0.05) / (dark + 0.05);
}

function flatten(tint: string, surface: string, alpha: number): string {
  const front = toRgb(tint);
  const back = toRgb(surface);
  const blended = front.map((channel, index) =>
    Math.round(channel * alpha + back[index] * (1 - alpha)),
  );
  return `#${blended.map((channel) => channel.toString(16).padStart(2, '0')).join('')}`;
}

const TEXT_ON_SURFACE: Array<[string, string[]]> = [
  ['--color-muted', ['--color-card', '--color-bg', '--color-url-bg']],
  ['--color-text', ['--color-card', '--color-bg']],
  ['--color-heading', ['--color-card', '--color-bg']],
  ['--color-input-placeholder', ['--color-input-bg']],
  ['--color-input-text', ['--color-input-bg']],
  ['--color-url-text', ['--color-url-bg']],
  ['--color-logs-text', ['--color-logs-bg']],
  ['--color-secondary-btn-text', ['--color-secondary-btn']],
];

const BADGE_TOKENS = [
  '--color-status-running',
  '--color-status-starting',
  '--color-status-installing',
  '--color-status-error',
];

const TEXT_ON_FILL: Array<[string, string[]]> = [
  [
    '--color-btn-text',
    [
      '--color-primary',
      '--color-btn-primary',
      '--color-btn-primary-hover',
      '--color-btn-start',
      '--color-btn-start-hover',
      '--color-btn-stop',
      '--color-btn-stop-hover',
      '--color-btn-danger',
      '--color-btn-danger-hover',
      ...BADGE_TOKENS,
    ],
  ],
];

const TINT_ALPHAS = [0, 0.1, 0.15, 0.2];

const themes = parseThemes();
const themeNames = Object.keys(themes).sort();

describe('theme contrast', () => {
  it('finds every theme declared in themes.css', () => {
    const index = readFileSync(join(themesDir, '..', 'themes.css'), 'utf8');
    const imported = [...index.matchAll(/\.\/themes\/(_[\w-]+)\.css/g)].length;
    expect(imported).toBeGreaterThan(0);
    expect(themeNames.length).toBeGreaterThanOrEqual(imported);
  });

  it.each(themeNames)('%s defines the full token set', (name) => {
    const reference = new Set(Object.keys(themes['foundry-dark']));
    const missing = [...reference].filter((prop) => !(prop in themes[name]));
    expect(missing).toEqual([]);
  });

  describe.each(themeNames)('%s', (name) => {
    const palette = themes[name];

    for (const [foreground, backgrounds] of [...TEXT_ON_SURFACE, ...TEXT_ON_FILL]) {
      for (const background of backgrounds) {
        it(`${foreground} on ${background} clears AA`, () => {
          const value = ratio(resolve(palette, foreground), resolve(palette, background));
          expect(Number(value.toFixed(2))).toBeGreaterThanOrEqual(AA_NORMAL_TEXT);
        });
      }
    }

    for (const alpha of TINT_ALPHAS) {
      it(`--color-error-text on a ${alpha} error tint over --color-card clears AA`, () => {
        const tint = flatten(
          resolve(palette, '--color-status-error'),
          resolve(palette, '--color-card'),
          alpha,
        );
        const value = ratio(resolve(palette, '--color-error-text'), tint);
        expect(Number(value.toFixed(2))).toBeGreaterThanOrEqual(AA_NORMAL_TEXT);
      });
    }
  });
});
