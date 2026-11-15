/**
 * The floating panel is the same SPA on the /mini route. Under Wails the native
 * window is used instead; in a browser we open a small popup window, which works
 * in every engine and keeps the panel a real, resizable window.
 */
export function supportsDocumentPip(): boolean {
  return typeof window !== 'undefined' && typeof window.open === 'function';
}

export function miniUrl(tunnelId: string): string {
  const url = new URL('/mini', window.location.origin);
  url.searchParams.set('id', tunnelId);
  return url.toString();
}

export async function openDocumentPip(tunnelId: string): Promise<void> {
  const handle = window.open(
    miniUrl(tunnelId),
    `ftm-mini-${tunnelId}`,
    'popup=yes,width=384,height=620,menubar=no,toolbar=no,location=no,status=no',
  );

  if (!handle) {
    throw new Error('popup blocked');
  }

  handle.focus();
}
