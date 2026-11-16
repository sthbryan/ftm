export async function copyText(text: string): Promise<boolean> {
  if (!text) return false;

  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}

/**
 * Copies a PNG to the clipboard. The ClipboardItem gets a promise so Safari and
 * WebKit (the macOS window) keep the user gesture while the image is made.
 */
export async function copyImage(make: () => Promise<Blob>): Promise<boolean> {
  if (typeof ClipboardItem === 'undefined' || !navigator.clipboard?.write) return false;
  try {
    await navigator.clipboard.write([new ClipboardItem({ 'image/png': make() })]);
    return true;
  } catch {
    return false;
  }
}
