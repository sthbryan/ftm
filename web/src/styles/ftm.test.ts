import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { FAMILIES, htmlAttributes, type Scheme } from '@/lib/stores/theme';

/**
 * Every family ships a dark and a light variant, and every variant has to stay
 * readable. The colours are resolved from the sheets the app actually loads,
 * in the order it loads them (sephiro.css and its theme imports), so the test
 * cannot drift from what ships, Sephiro's preset sheets included.
 */

const AA_NORMAL_TEXT = 4.5;

type Rule = { selectors: string[]; declarations: Record<string, string> };

const require = createRequire(import.meta.url);

/** The theme rules of a sheet and of the theme sheets it imports, in cascade order. */
function loadRules(file: string): Rule[] {
  const source = readFileSync(file, 'utf8').replace(/\/\*[\s\S]*?\*\//g, '');
  const rules: Rule[] = [];

  for (const [, target] of source.matchAll(/@import\s+["']([^"']+)["']\s*;/g)) {
    if (target.startsWith('.')) rules.push(...loadRules(resolve(dirname(file), target)));
    else if (/\/theme-[\w-]+\.css$/.test(target)) rules.push(...loadRules(require.resolve(target)));
  }

  const body = source.replace(/@import[^;]+;/g, '');
  for (const [, selector, block] of body.matchAll(/([^{}]+)\{([^{}]*)\}/g)) {
    const declarations: Record<string, string> = {};
    for (const [, name, value] of block.matchAll(/(--sph-[\w-]+)\s*:\s*([^;]+?)\s*(?:;|$)/g)) {
      declarations[name] = value;
    }
    rules.push({ selectors: splitTopLevel(selector.trim()), declarations });
  }

  return rules;
}

function splitTopLevel(list: string): string[] {
  const parts: string[] = [];
  let depth = 0;
  let start = 0;
  for (let index = 0; index < list.length; index += 1) {
    const char = list[index];
    if (char === '(') depth += 1;
    else if (char === ')') depth -= 1;
    else if (char === ',' && depth === 0) {
      parts.push(list.slice(start, index).trim());
      start = index + 1;
    }
  }
  parts.push(list.slice(start).trim());
  return parts.filter(Boolean);
}

type Root = Record<string, string>;

/**
 * Specificity of `selector` when it matches <html> with these attributes, or
 * null. Handles what theme sheets use: `:root`, attribute selectors and
 * `:where()`; a combinator needs an ancestor, which <html> does not have.
 */
function matchRoot(selector: string, root: Root): number | null {
  let rest = selector.trim();
  let specificity = 0;

  while (rest) {
    if (rest.startsWith(':root')) {
      specificity += 1;
      rest = rest.slice(5);
      continue;
    }
    const attribute = rest.match(/^\[([\w-]+)(?:=(?:"([^"]*)"|'([^']*)'|([\w-]+)))?\]/);
    if (attribute) {
      const [whole, name, quoted, single, bare] = attribute;
      const expected = quoted ?? single ?? bare;
      if (!(name in root) || (expected !== undefined && root[name] !== expected)) return null;
      specificity += 1;
      rest = rest.slice(whole.length);
      continue;
    }
    if (rest.startsWith(':where(')) {
      let depth = 0;
      let end = 0;
      for (let index = 6; index < rest.length; index += 1) {
        if (rest[index] === '(') depth += 1;
        else if (rest[index] === ')' && --depth === 0) {
          end = index;
          break;
        }
      }
      const inner = splitTopLevel(rest.slice(7, end));
      if (!inner.some((part) => matchRoot(part, root) !== null)) return null;
      rest = rest.slice(end + 1);
      continue;
    }
    return null;
  }

  return specificity;
}

const rules = loadRules(fileURLToPath(new URL('./sephiro.css', import.meta.url)));

function resolveTokens(root: Root): Record<string, string> {
  const matched: Array<{ specificity: number; order: number; rule: Rule }> = [];
  rules.forEach((rule, order) => {
    const scores = rule.selectors
      .map((selector) => matchRoot(selector, root))
      .filter((score): score is number => score !== null);
    if (scores.length) matched.push({ specificity: Math.max(...scores), order, rule });
  });
  matched.sort((a, b) => a.specificity - b.specificity || a.order - b.order);
  return Object.assign({}, ...matched.map(({ rule }) => rule.declarations));
}

function toRgb(hex: string): [number, number, number] {
  const normalized = hex.replace('#', '');
  const full =
    normalized.length === 3
      ? normalized
          .split('')
          .map((channel) => channel + channel)
          .join('')
      : normalized;

  return [
    parseInt(full.slice(0, 2), 16),
    parseInt(full.slice(2, 4), 16),
    parseInt(full.slice(4, 6), 16),
  ];
}

function luminance(hex: string): number {
  const [r, g, b] = toRgb(hex).map((channel) => {
    const c = channel / 255;
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function ratio(foreground: string, background: string): number {
  const a = luminance(foreground);
  const b = luminance(background);
  const [light, dark] = a > b ? [a, b] : [b, a];
  return (light + 0.05) / (dark + 0.05);
}

/** url-bg is the accent at 12% over the page background. */
function tint(background: string, over: string, alpha: number): string {
  const front = toRgb(background);
  const back = toRgb(over);
  const blended = front.map((channel, index) =>
    Math.round(channel * alpha + back[index] * (1 - alpha)),
  );
  return `#${blended.map((channel) => channel.toString(16).padStart(2, '0')).join('')}`;
}

const PAIRS: Array<[string, string, string]> = [
  ['--sph-text', '--sph-bg', 'body text on the page'],
  ['--sph-text', '--sph-surface', 'body text on a card'],
  ['--sph-text', '--sph-surface-elevated', 'input text'],
  ['--sph-text-muted', '--sph-bg', 'muted text on the page'],
  ['--sph-text-muted', '--sph-surface', 'muted text on a card'],
  ['--sph-text-muted', '--sph-surface-elevated', 'hints on inputs and wells'],
  ['--sph-on-accent', '--sph-accent', 'button label on the accent'],
  ['--sph-success', '--sph-bg', 'running state on the page'],
  ['--sph-success', '--sph-surface', 'running state on a card'],
  ['--sph-warning', '--sph-bg', 'warning state on the page'],
  ['--sph-warning', '--sph-surface', 'warning state on a card'],
  ['--sph-danger', '--sph-bg', 'error state on the page'],
  ['--sph-danger', '--sph-surface', 'error state on a card'],
];

const VARIANTS = FAMILIES.flatMap((family) =>
  (['dark', 'light'] as Scheme[]).map((scheme) => [family.id, scheme] as const),
);

describe.each(VARIANTS)('%s %s', (family, scheme) => {
  const { sephiroTheme, theme } = htmlAttributes(family, scheme);
  const tokens = resolveTokens({ 'data-sephiro-theme': sephiroTheme, 'data-theme': theme });

  function token(name: string): string {
    const value = tokens[name];
    if (!value || !/^#[0-9a-f]{3}([0-9a-f]{3})?$/i.test(value)) {
      throw new Error(`${name} is ${value ?? 'missing'} for ${family} ${scheme}; expected a hex`);
    }
    return value;
  }

  it('resolves to a variant of the right brightness', () => {
    const background = luminance(token('--sph-bg'));
    if (scheme === 'dark') expect(background).toBeLessThan(0.05);
    else expect(background).toBeGreaterThan(0.6);
  });

  it.each(PAIRS)('%s on %s clears AA (%s)', (foreground, background) => {
    const value = ratio(token(foreground), token(background));
    expect(Number(value.toFixed(2))).toBeGreaterThanOrEqual(AA_NORMAL_TEXT);
  });

  it('the shared URL colour clears AA on its tinted well', () => {
    const well = tint(token('--sph-accent'), token('--sph-bg'), 0.12);
    expect(Number(ratio(token('--sph-accent-strong'), well).toFixed(2))).toBeGreaterThanOrEqual(
      AA_NORMAL_TEXT,
    );
  });
});

describe('the cascade model', () => {
  it('lets the variant beat the family default, as the browser does', () => {
    const dark = resolveTokens({ 'data-sephiro-theme': 'pulso', 'data-theme': 'dark' });
    const light = resolveTokens({ 'data-sephiro-theme': 'pulso', 'data-theme': 'light' });
    expect(dark['--sph-bg']).not.toBe(light['--sph-bg']);
  });
});
