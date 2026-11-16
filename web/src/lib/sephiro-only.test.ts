import { describe, expect, it } from 'vitest';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

/**
 * Every control in the Preact app is a Sephiro component. Native form controls
 * and markup that borrows Sephiro's classes by hand are what this guards
 * against: they drift from the library and skip its states and accessibility.
 */

const root = fileURLToPath(new URL('../components', import.meta.url));

function sources(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name);
    if (statSync(path).isDirectory()) return sources(path);
    return /\.tsx?$/.test(name) && !/\.test\.tsx?$/.test(name) ? [path] : [];
  });
}

const NATIVE_CONTROL = /<(button|input|select|textarea|a)\b/g;
const HAND_SPH_CLASS = /\b(?:class|className)\s*=\s*[{"'`][^>]*?\bsph-/g;

export function offences(source: string): string[] {
  const found: string[] = [];
  for (const match of source.matchAll(NATIVE_CONTROL)) found.push(`native <${match[1]}>`);
  for (const match of source.matchAll(HAND_SPH_CLASS)) found.push(`hand-written ${match[0]}`);
  return found;
}

describe('Sephiro-only components', () => {
  it('flags native controls and hand-written sph- classes', () => {
    expect(offences('<button type="button">x</button>')).toEqual(['native <button>']);
    expect(offences('<span class="sph-spinner" />')).toHaveLength(1);
    expect(offences('<Button variant="primary" />')).toEqual([]);
    expect(offences("style={{ fill: 'var(--sph-accent)' }}")).toEqual([]);
  });

  it('finds none in web/src/components', () => {
    const report = sources(root).flatMap((file) =>
      offences(readFileSync(file, 'utf8')).map((what) => `${file.slice(root.length + 1)}: ${what}`),
    );
    expect(report).toEqual([]);
  });
});
