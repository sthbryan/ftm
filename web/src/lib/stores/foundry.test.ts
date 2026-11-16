import { describe, expect, it } from 'vitest';
import { keepThroughOneMiss, type FoundryWorld, worldTitle } from './foundry';

describe('worldTitle', () => {
  it('turns a world id into a title, small words in lowercase', () => {
    expect(worldTitle('curse-of-strahd')).toBe('Curse of Strahd');
    expect(worldTitle('the_lost_mine')).toBe('The Lost Mine');
    expect(worldTitle('tumba-de-la-aniquilacion')).toBe('Tumba de la Aniquilacion');
  });
});

describe('keepThroughOneMiss', () => {
  const test: FoundryWorld = { port: 30000, active: true, world: 'test', users: 2 };

  it('keeps a Foundry that misses one check and drops it on the second', () => {
    const first = keepThroughOneMiss([test], [], new Set());
    expect(first.worlds).toEqual([test]);

    const second = keepThroughOneMiss(first.worlds, [], first.missed);
    expect(second.worlds).toEqual([]);
  });

  it('takes the fresh answer when Foundry is back', () => {
    const missed = keepThroughOneMiss([test], [], new Set());
    const back = { ...test, users: 3 };
    const next = keepThroughOneMiss(missed.worlds, [back], missed.missed);
    expect(next.worlds).toEqual([back]);
    expect(next.missed.size).toBe(0);
  });
});
