import { describe, expect, it } from 'vitest';
import { formatClock } from './duration';

describe('formatClock', () => {
  it('reads like a stopwatch', () => {
    expect(formatClock(0)).toBe('0:00');
    expect(formatClock(65_000)).toBe('1:05');
    expect(formatClock(4_328_000)).toBe('1:12:08');
    expect(formatClock(-5)).toBe('0:00');
  });
});
