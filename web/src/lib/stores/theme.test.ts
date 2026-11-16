import { describe, expect, it } from 'vitest';
import {
  DEFAULT_PREFERENCE,
  FAMILIES,
  type Family,
  htmlAttributes,
  nextFamily,
  parsePreference,
  resolveScheme,
} from './theme';

describe('parsePreference', () => {
  it('starts on Tinta following the OS when nothing is stored', () => {
    expect(parsePreference(null)).toEqual({ family: 'tinta', mode: 'system' });
    expect(DEFAULT_PREFERENCE).toEqual({ family: 'tinta', mode: 'system' });
  });

  it('reads the current {family, mode} shape', () => {
    expect(parsePreference('{"family":"flexoki","mode":"light"}')).toEqual({
      family: 'flexoki',
      mode: 'light',
    });
  });

  it('keeps the mode when the stored family no longer exists', () => {
    expect(parsePreference('{"family":"nord","mode":"dark"}')).toEqual({
      family: 'tinta',
      mode: 'dark',
    });
  });

  it('migrates the bare mode the Tinta-only build stored', () => {
    expect(parsePreference('system')).toEqual({ family: 'tinta', mode: 'system' });
    expect(parsePreference('light')).toEqual({ family: 'tinta', mode: 'light' });
    expect(parsePreference('dark')).toEqual({ family: 'tinta', mode: 'dark' });
  });

  it('migrates the old per-variant theme names', () => {
    expect(parsePreference('tinta-light')).toEqual({ family: 'tinta', mode: 'light' });
    expect(parsePreference('tinta')).toEqual({ family: 'tinta', mode: 'dark' });
  });

  it('migrates the old {mode, theme} JSON, keeping the look it had', () => {
    expect(parsePreference('{"mode":"system"}')).toEqual({ family: 'tinta', mode: 'system' });
    expect(parsePreference('{"mode":"manual","theme":"ftm"}')).toEqual({
      family: 'tinta',
      mode: 'dark',
    });
    expect(parsePreference('{"mode":"manual","theme":"light"}')).toEqual({
      family: 'sephiro',
      mode: 'light',
    });
    expect(parsePreference('{"mode":"manual","theme":"dark"}')).toEqual({
      family: 'sephiro',
      mode: 'dark',
    });
    expect(parsePreference('{"mode":"manual","theme":"asterism"}')).toEqual({
      family: 'asterism',
      mode: 'dark',
    });
    expect(parsePreference('{"mode":"manual","theme":"pulso"}')).toEqual({
      family: 'pulso',
      mode: 'light',
    });
    expect(parsePreference('{"mode":"manual","theme":"gruvbox"}')).toEqual(DEFAULT_PREFERENCE);
  });

  it('falls back to the default on anything unreadable', () => {
    expect(parsePreference('{not json')).toEqual(DEFAULT_PREFERENCE);
    expect(parsePreference('42')).toEqual(DEFAULT_PREFERENCE);
    expect(parsePreference('null')).toEqual(DEFAULT_PREFERENCE);
    expect(parsePreference('constructor')).toEqual(DEFAULT_PREFERENCE);
    expect(parsePreference('{"mode":"manual","theme":"toString"}')).toEqual(DEFAULT_PREFERENCE);
  });
});

describe('resolveScheme', () => {
  it('follows the OS on Auto and forces the variant otherwise', () => {
    expect(resolveScheme('system', 'light')).toBe('light');
    expect(resolveScheme('system', 'dark')).toBe('dark');
    expect(resolveScheme('dark', 'light')).toBe('dark');
    expect(resolveScheme('light', 'dark')).toBe('light');
  });
});

describe('htmlAttributes', () => {
  it('names the family and the variant on <html>', () => {
    expect(htmlAttributes('flexoki', 'light')).toEqual({ sephiroTheme: 'flexoki', theme: 'light' });
    expect(htmlAttributes('pulso', 'dark')).toEqual({ sephiroTheme: 'pulso', theme: 'dark' });
  });

  it("uses Sephiro's own attribute value for its default pair", () => {
    expect(htmlAttributes('sephiro', 'light')).toEqual({ sephiroTheme: 'dark', theme: 'light' });
  });
});

describe('nextFamily', () => {
  it('steps through every family and wraps, leaving the mode alone', () => {
    const seen: Family[] = [FAMILIES[0].id];
    for (let i = 1; i < FAMILIES.length; i += 1) seen.push(nextFamily(seen[i - 1]));
    expect(seen).toEqual(FAMILIES.map((family) => family.id));
    expect(nextFamily(seen[seen.length - 1])).toBe('tinta');
  });

  it('ships Tinta first and the eight agreed families', () => {
    expect(FAMILIES.map((family) => family.id)).toEqual([
      'tinta',
      'sephiro',
      'asterism',
      'curie',
      'fizza',
      'pulso',
      'flexoki',
      'pizarra',
    ]);
  });
});
