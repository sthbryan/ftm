import { describe, expect, it } from 'vitest';
import { connectionPath, connectionRoute } from './router';

describe('connectionRoute', () => {
  it('opens the new form on /new', () => {
    expect(connectionRoute('/new')).toEqual({ mode: 'new' });
  });

  it('opens the edit form for the id in the path', () => {
    expect(connectionRoute('/edit/abc-123')).toEqual({ mode: 'edit', id: 'abc-123' });
    expect(connectionRoute(connectionPath('a b'))).toEqual({ mode: 'edit', id: 'a b' });
  });

  it('shows no form anywhere else, so leaving edit never lands on new', () => {
    // Closing the edit page navigates home; the old modal reset its id first
    // and flashed the new form during the close animation.
    for (const path of ['/', '/settings', '/edit', '/edit/', '/news', '/mini']) {
      expect(connectionRoute(path)).toBeNull();
    }
    expect(connectionRoute('/edit/abc')).not.toEqual({ mode: 'new' });
  });
});
