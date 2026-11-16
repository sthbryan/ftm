import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { FakeEventSource } from './test-eventsource';
import type { StreamEvent } from './events';

async function load() {
  vi.resetModules();
  return import('./events');
}

describe('subscribeEvents', () => {
  beforeEach(() => {
    FakeEventSource.instances = [];
    vi.stubGlobal('EventSource', FakeEventSource);
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it('opens one shared stream on /api/events', async () => {
    const { subscribeEvents } = await load();
    subscribeEvents(() => undefined);
    subscribeEvents(() => undefined);

    expect(FakeEventSource.instances).toHaveLength(1);
    expect(FakeEventSource.latest().url).toBe('/api/events');
  });

  it('hands named events to every listener with type set to the event name', async () => {
    const { subscribeEvents } = await load();
    const a: StreamEvent[] = [];
    const b: StreamEvent[] = [];
    subscribeEvents((e) => a.push(e));
    subscribeEvents((e) => b.push(e));

    FakeEventSource.latest().send('tunnel_deleted', { type: 'tunnel_deleted', id: 't1' });
    FakeEventSource.latest().send('tunnel_stats', { id: 't1', visitors: 2, activeSessions: 1 });

    expect(a).toEqual([
      { type: 'tunnel_deleted', id: 't1' },
      { type: 'tunnel_stats', id: 't1', visitors: 2, activeSessions: 1 },
    ]);
    expect(b).toEqual(a);
  });

  it('ignores data that is not a JSON object', async () => {
    const { subscribeEvents } = await load();
    const got: StreamEvent[] = [];
    subscribeEvents((e) => got.push(e));

    FakeEventSource.latest().send('tunnel_deleted', 'not json');
    FakeEventSource.latest().send('tunnel_deleted', 'null');

    expect(got).toEqual([]);
  });

  it('reports connected and disconnected once per change', async () => {
    const { subscribeEvents } = await load();
    const got: string[] = [];
    subscribeEvents((e) => got.push(e.type));
    const es = FakeEventSource.latest();

    es.open();
    es.fail(FakeEventSource.CONNECTING);
    es.fail(FakeEventSource.CONNECTING);
    es.open();

    expect(got).toEqual(['connected', 'disconnected', 'connected']);
  });

  it('tells a late listener the stream is already connected', async () => {
    const { subscribeEvents } = await load();
    subscribeEvents(() => undefined);
    FakeEventSource.latest().open();

    const got: string[] = [];
    subscribeEvents((e) => got.push(e.type));
    expect(got).toEqual(['connected']);
  });

  it('leaves retries to the browser while it is reconnecting on its own', async () => {
    const { subscribeEvents } = await load();
    subscribeEvents(() => undefined);
    FakeEventSource.latest().fail(FakeEventSource.CONNECTING);

    await vi.advanceTimersByTimeAsync(60_000);
    expect(FakeEventSource.instances).toHaveLength(1);
  });

  it('reopens a stream the browser gave up on, with backoff, and asks for a resync', async () => {
    const { subscribeEvents } = await load();
    const got: StreamEvent[] = [];
    subscribeEvents((e) => got.push(e));
    FakeEventSource.latest().open();

    FakeEventSource.latest().fail(FakeEventSource.CLOSED);
    expect(FakeEventSource.instances[0].closed).toBe(true);
    await vi.advanceTimersByTimeAsync(999);
    expect(FakeEventSource.instances).toHaveLength(1);
    await vi.advanceTimersByTimeAsync(1);
    expect(FakeEventSource.instances).toHaveLength(2);

    FakeEventSource.latest().fail(FakeEventSource.CLOSED);
    await vi.advanceTimersByTimeAsync(1999);
    expect(FakeEventSource.instances).toHaveLength(2);
    await vi.advanceTimersByTimeAsync(1);
    expect(FakeEventSource.instances).toHaveLength(3);

    got.length = 0;
    FakeEventSource.latest().open();
    expect(got).toEqual([{ type: 'connected' }, { type: 'resync', reason: 'reconnected' }]);
  });

  it('forwards the server resync event', async () => {
    const { subscribeEvents } = await load();
    const got: StreamEvent[] = [];
    subscribeEvents((e) => got.push(e));

    FakeEventSource.latest().send('resync', { type: 'resync', reason: 'missed events are no longer kept' });
    expect(got).toEqual([{ type: 'resync', reason: 'missed events are no longer kept' }]);
  });

  it('closes the stream with the last listener and stops reconnecting', async () => {
    const { subscribeEvents } = await load();
    const off1 = subscribeEvents(() => undefined);
    const off2 = subscribeEvents(() => undefined);

    off1();
    expect(FakeEventSource.latest().closed).toBe(false);
    FakeEventSource.latest().fail(FakeEventSource.CLOSED);
    off2();

    await vi.advanceTimersByTimeAsync(60_000);
    expect(FakeEventSource.instances).toHaveLength(1);
  });
});
