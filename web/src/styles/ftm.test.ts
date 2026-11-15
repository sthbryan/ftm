import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

/**
 * The Sephiro themes are tested where they are made; `ftm` is ours, so it is the
 * one palette this repo has to keep readable. Colours are read from the sheet
 * itself, so the test cannot drift from what ships.
 */

const AA_NORMAL_TEXT = 4.5;

const css = readFileSync(fileURLToPath(new URL('./ftm.css', import.meta.url)), 'utf8');
const block = css.match(/\[data-sephiro-theme="ftm"\]\s*\{([\s\S]*?)\}/)?.[1] ?? '';

const tokens: Record<string, string> = Object.fromEntries(
  [...block.matchAll(/(--sph-[\w-]+)\s*:\s*(#[0-9a-fA-F]{3,8})\s*;/g)].map((match) => [
    match[1],
    match[2],
  ]),
);

function token(name: string): string {
  const value = tokens[name];
  if (!value) throw new Error(`missing ${name} in ftm.css`);
  return value;
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
  ['--sph-text-muted', '--sph-bg', 'muted text on the page'],
  ['--sph-text-muted', '--sph-surface', 'muted text on a card'],
  ['--sph-text-muted', '--sph-surface-elevated', 'hints on inputs and wells'],
  ['--sph-on-accent', '--sph-accent', 'button label on the accent'],
  ['--sph-success', '--sph-surface', 'running state as text'],
  ['--sph-warning', '--sph-surface', 'warning state as text'],
  ['--sph-danger', '--sph-surface', 'error state as text'],
];

describe('ftm theme', () => {
  it('defines every colour it is checked against', () => {
    expect(Object.keys(tokens)).toContain('--sph-accent-strong');
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
