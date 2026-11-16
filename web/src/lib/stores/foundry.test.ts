import { describe, expect, it } from 'vitest';
import { worldTitle } from './foundry';

describe('worldTitle', () => {
  it('turns a world id into a title, small words in lowercase', () => {
    expect(worldTitle('curse-of-strahd')).toBe('Curse of Strahd');
    expect(worldTitle('the_lost_mine')).toBe('The Lost Mine');
    expect(worldTitle('tumba-de-la-aniquilacion')).toBe('Tumba de la Aniquilacion');
  });
});
