import { describe, expect, it } from 'vitest';
import { arrange, expiryLevel, foundryCheck, inviteMessage, momentOf, playersJoined } from './ritual';
import type { Tunnel } from './types';

function tunnel(id: string, state: string, extra: Partial<Tunnel> = {}): Tunnel {
  return { id, name: id, provider: 'cloudflare', port: 30000, state, ...extra };
}

describe('momentOf', () => {
  it('is ready when nothing runs', () => {
    expect(momentOf(tunnel('a', 'stopped'), new Set())).toBe('ready');
    expect(momentOf(tunnel('a', 'need_installing'), new Set())).toBe('ready');
  });

  it('is opening while it downloads, starts or connects', () => {
    for (const state of ['starting', 'installing', 'downloading', 'connecting']) {
      expect(momentOf(tunnel('a', state), new Set())).toBe('opening');
    }
  });

  it('keeps a stopping connection as a strip until it is down', () => {
    expect(momentOf(tunnel('a', 'stopping'), new Set())).toBe('session');
  });

  it('is invite when online and nobody has joined yet', () => {
    expect(momentOf(tunnel('a', 'online'), new Set())).toBe('invite');
  });

  it('is session once a player joins or the invite was dismissed', () => {
    expect(momentOf(tunnel('a', 'online', { activeSessions: 1 }), new Set())).toBe('session');
    expect(momentOf(tunnel('a', 'online'), new Set(['a']))).toBe('session');
  });

  it('is failed after an error or timeout', () => {
    expect(momentOf(tunnel('a', 'error'), new Set())).toBe('failed');
    expect(momentOf(tunnel('a', 'timeout'), new Set())).toBe('failed');
  });
});

describe('arrange', () => {
  it('puts the selected preset on the ready stage when nothing runs', () => {
    const list = [tunnel('a', 'stopped'), tunnel('b', 'stopped')];
    const view = arrange(list, 'b', new Set());
    expect(view.stage).toEqual({ moment: 'ready', tunnel: list[1] });
    expect(view.strips).toEqual([]);
    expect(view.presets.map((t) => t.id)).toEqual(['a', 'b']);
  });

  it('falls back to the first preset when the selection is gone', () => {
    const list = [tunnel('a', 'stopped')];
    expect(arrange(list, 'zzz', new Set()).stage).toEqual({ moment: 'ready', tunnel: list[0] });
  });

  it('has an empty ready stage without connections', () => {
    expect(arrange([], null, new Set()).stage).toEqual({ moment: 'ready', tunnel: null });
  });

  it('gives the stage to an opening connection over an invite', () => {
    const list = [tunnel('a', 'online'), tunnel('b', 'connecting')];
    const view = arrange(list, null, new Set());
    expect(view.stage).toEqual({ moment: 'opening', tunnel: list[1] });
    expect(view.strips.map((t) => t.id)).toEqual(['a']);
  });

  it('stacks every connection in session as a strip and hides the ready stage', () => {
    const list = [
      tunnel('a', 'online', { activeSessions: 2 }),
      tunnel('b', 'online', { activeSessions: 1 }),
      tunnel('c', 'stopped'),
    ];
    const view = arrange(list, 'c', new Set());
    expect(view.stage).toBeNull();
    expect(view.strips.map((t) => t.id)).toEqual(['a', 'b']);
    expect(view.presets.map((t) => t.id)).toEqual(['c']);
  });

  it('keeps a second failed connection reachable as a preset', () => {
    const list = [tunnel('a', 'error'), tunnel('b', 'timeout'), tunnel('c', 'stopped')];
    const view = arrange(list, null, new Set());
    expect(view.stage).toEqual({ moment: 'failed', tunnel: list[0] });
    expect(view.presets.map((t) => t.id)).toEqual(['b', 'c']);
  });

  it('shows a failed connection on the stage', () => {
    const list = [tunnel('a', 'stopped'), tunnel('b', 'error')];
    expect(arrange(list, 'a', new Set()).stage).toEqual({ moment: 'failed', tunnel: list[1] });
  });
});

describe('expiryLevel', () => {
  it('is none without an expiry', () => {
    expect(expiryLevel(0)).toBe('none');
  });
  it('warns in the last ten minutes', () => {
    expect(expiryLevel(30 * 60_000)).toBe('later');
    expect(expiryLevel(10 * 60_000)).toBe('soon');
    expect(expiryLevel(60_000)).toBe('soon');
  });
});

describe('foundryCheck', () => {
  it('answers when the port is busy among the common Foundry ports', () => {
    expect(foundryCheck(30000, [30000])).toBe('answers');
  });
  it('is silent when a common port is free', () => {
    expect(foundryCheck(30001, [30000])).toBe('silent');
  });
  it('is unknown for ports outside the scanned range or before a scan', () => {
    expect(foundryCheck(8080, [30000])).toBe('unknown');
    expect(foundryCheck(30000, null)).toBe('unknown');
  });
  it('checks a custom port when the server scanned it', () => {
    expect(foundryCheck(8080, [8080], [30000, 8080])).toBe('answers');
    expect(foundryCheck(8080, [], [30000, 8080])).toBe('silent');
  });
});

describe('inviteMessage', () => {
  it('fills the world and link into the template', () => {
    expect(inviteMessage('Join {name}: {url}', 'Curse', 'https://x.test')).toBe(
      'Join Curse: https://x.test',
    );
  });
});

describe('playersJoined', () => {
  it('counts only new players on connections already known', () => {
    const before = { a: 1, b: 0 };
    expect(playersJoined(before, { a: 2, b: 0 })).toBe(1);
    expect(playersJoined(before, { a: 0, b: 0 })).toBe(0);
    expect(playersJoined(before, { a: 1, b: 0, c: 3 })).toBe(0);
  });
});
