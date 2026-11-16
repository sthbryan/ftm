import { describe, expect, it } from 'vitest';
import { FACES, facing, render } from './AsciiDie';

describe('AsciiDie', () => {
  it('builds a d20: twenty faces', () => {
    expect(FACES).toHaveLength(20);
  });

  it('lands showing 20, or 1 when the opening failed', () => {
    const twenty = render(facing(FACES.length - 1), '20');
    const one = render(facing(0), '1');
    expect(twenty).toContain('20');
    expect(one).toMatch(/1/);
    // A solid die fills a good part of the stub.
    expect(twenty.replace(/\s/g, '').length).toBeGreaterThan(150);
  });
});
