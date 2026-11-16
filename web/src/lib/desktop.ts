/**
 * The desktop shell (Wails) loads this same web app from the local server, so
 * the page has to work out for itself that it is inside a desktop window.
 * Wails listens for a few plain messages on the webview's bridge even on pages
 * that do not load its runtime; `wails:drag` is the one that moves the window.
 */

type Bridge = { postMessage(message: string): void };

type DesktopWindow = Window & {
  webkit?: { messageHandlers?: { external?: Bridge } };
  chrome?: { webview?: Bridge };
};

export type DesktopPlatform = 'mac' | 'windows' | 'linux';

function bridge(): Bridge | null {
  if (typeof window === 'undefined') return null;
  const win = window as DesktopWindow;
  return win.chrome?.webview ?? win.webkit?.messageHandlers?.external ?? null;
}

function detectPlatform(): DesktopPlatform | null {
  if (!bridge()) return null;
  const agent = navigator.userAgent;
  if (/Mac OS X|Macintosh/.test(agent)) return 'mac';
  if (/Windows/.test(agent)) return 'windows';
  return 'linux';
}

/** The OS of the desktop window, or null in a browser. */
export const desktopPlatform: DesktopPlatform | null = detectPlatform();

export const isDesktop = desktopPlatform !== null;

/**
 * Marks <html> so CSS can make room for the macOS traffic lights and turn the
 * header into a title bar. Browsers get no attribute and keep the web layout.
 */
export function markDesktopShell(): void {
  if (typeof document === 'undefined' || !desktopPlatform) return;
  document.documentElement.dataset.shell = desktopPlatform;
}

const INTERACTIVE = 'button, a, input, textarea, select, label, [role="button"], [role="dialog"], [data-no-drag]';

function send(message: string): void {
  try {
    bridge()?.postMessage(message);
  } catch {
    /* the window went away; nothing to move */
  }
}

/**
 * Props for an element that should move the window, like a native title bar:
 * press and drag on empty space moves it, a double click zooms or maximises.
 * Controls inside it keep working because presses on them are ignored.
 */
export function dragRegion() {
  if (!isDesktop) return {};
  return {
    onMouseDown(event: MouseEvent) {
      if (event.button !== 0 || event.detail > 1) return;
      if ((event.target as Element | null)?.closest(INTERACTIVE)) return;
      event.preventDefault();
      send('wails:drag');
    },
    onDblClick(event: MouseEvent) {
      if ((event.target as Element | null)?.closest(INTERACTIVE)) return;
      send('wails:drag:doubleclick');
    },
  };
}
