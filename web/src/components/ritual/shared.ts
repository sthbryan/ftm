import { t } from '@/lib/stores/i18n';
import { toast } from '@/lib/stores/toast';
import { copyText } from '@/lib/utils/clipboard';
import { inviteMessage } from '@/lib/ritual';
import type { Tunnel } from '@/lib/types';

export async function copyWithToast(text: string, doneKey: string): Promise<boolean> {
  if (!(await copyText(text))) {
    toast.error(t('copy_failed'));
    return false;
  }
  toast.success(t(doneKey));
  return true;
}

export function copyLink(tunnel: Tunnel): Promise<boolean> {
  return copyWithToast(tunnel.publicUrl ?? '', 'link_copied');
}

export function copyInvite(tunnel: Tunnel): Promise<boolean> {
  const text = inviteMessage(t('invite_message'), tunnel.name, tunnel.publicUrl ?? '');
  return copyWithToast(text, 'invite_copied');
}

/** Run a store action, surfacing its error as a toast. */
export async function attempt(action: () => Promise<unknown>): Promise<void> {
  try {
    await action();
  } catch (error) {
    toast.error((error as Error).message);
  }
}

export function displayUrl(url: string | undefined): string {
  return (url ?? '').replace(/^https?:\/\//, '');
}
