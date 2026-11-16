/** A stand-in for the browser's EventSource that tests drive by hand. */
export class FakeEventSource {
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSED = 2;
  static instances: FakeEventSource[] = [];

  readyState = FakeEventSource.CONNECTING;
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;
  closed = false;
  private listeners = new Map<string, Set<(event: MessageEvent) => void>>();

  constructor(public url: string) {
    FakeEventSource.instances.push(this);
  }

  static latest(): FakeEventSource {
    const last = FakeEventSource.instances.at(-1);
    if (!last) throw new Error('no EventSource opened');
    return last;
  }

  addEventListener(name: string, fn: (event: MessageEvent) => void): void {
    if (!this.listeners.has(name)) this.listeners.set(name, new Set());
    this.listeners.get(name)!.add(fn);
  }

  close(): void {
    this.closed = true;
    this.readyState = FakeEventSource.CLOSED;
  }

  open(): void {
    this.readyState = FakeEventSource.OPEN;
    this.onopen?.();
  }

  fail(state: number): void {
    this.readyState = state;
    this.onerror?.();
  }

  send(name: string, data: unknown): void {
    const event = { type: name, data: typeof data === 'string' ? data : JSON.stringify(data) } as MessageEvent;
    this.listeners.get(name)?.forEach((fn) => fn(event));
  }
}
