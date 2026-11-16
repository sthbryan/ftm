import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { FakeEventSource } from '../test-eventsource';
import { logsApi } from './logs';

describe('logsApi.createStream', () => {
  beforeEach(() => {
    FakeEventSource.instances = [];
    vi.stubGlobal('EventSource', FakeEventSource);
  });
  afterEach(() => vi.unstubAllGlobals());

  it('follows the tunnel logs stream', () => {
    logsApi.createStream('a b/c', { onSnapshot: () => undefined, onLine: () => undefined });
    expect(FakeEventSource.latest().url).toBe('/api/tunnels/a%20b%2Fc/logs?follow=1');
  });

  it('replaces lines on every snapshot and appends each new line', () => {
    const snapshots: string[][] = [];
    const lines: string[] = [];
    logsApi.createStream('t1', { onSnapshot: (s) => snapshots.push(s), onLine: (l) => lines.push(l) });
    const es = FakeEventSource.latest();

    es.send('log_snapshot', { type: 'log_snapshot', id: 't1', lines: ['one', 'two'] });
    es.send('log', { type: 'log', id: 't1', line: 'three' });
    es.send('log_snapshot', { type: 'log_snapshot', id: 't1', lines: null });
    es.send('log', 'garbage');

    expect(snapshots).toEqual([['one', 'two'], []]);
    expect(lines).toEqual(['three']);
  });

  it('closes the stream', () => {
    const stream = logsApi.createStream('t1', { onSnapshot: () => undefined, onLine: () => undefined });
    stream.close();
    expect(FakeEventSource.latest().closed).toBe(true);
  });
});
