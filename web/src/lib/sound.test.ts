import { beforeEach, describe, expect, it, vi } from 'vitest';

const play = vi.fn();
vi.mock('cuelume', () => ({ play, unlock: vi.fn(async () => true), setEnabled: vi.fn() }));

const reduced = { value: false };
vi.mock('@/lib/utils/motion', () => ({ prefersReducedMotion: () => reduced.value }));

const { cue, cueFor, soundOn } = await import('./sound');

describe('cueFor', () => {
  it('maps the spec moments to Cuelume cues', () => {
    expect(cueFor('success')).toEqual({ name: 'success' });
    expect(cueFor('error')).toEqual({ name: 'error' });
    expect(cueFor('warning')).toEqual({ name: 'attention' });
    expect(cueFor('alert')).toEqual({ name: 'attention' });
    expect(cueFor('joined')).toEqual({ name: 'ready', emphasis: 'subtle' });
  });

  it('stays silent for routine events', () => {
    expect(cueFor('info')).toBeNull();
    expect(cueFor('anything')).toBeNull();
  });
});

describe('cue', () => {
  beforeEach(() => {
    play.mockClear();
    soundOn.value = true;
    reduced.value = false;
  });

  it('plays the mapped cue when sound is on', () => {
    cue('success');
    expect(play).toHaveBeenCalledWith('success', undefined);
    cue('joined');
    expect(play).toHaveBeenCalledWith('ready', { emphasis: 'subtle' });
  });

  it('is silent when the sound setting is off', () => {
    soundOn.value = false;
    cue('success');
    expect(play).not.toHaveBeenCalled();
  });

  it('is silent when the user prefers reduced motion', () => {
    reduced.value = true;
    cue('error');
    expect(play).not.toHaveBeenCalled();
  });
});
