import { useEffect, useState } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { ArrowRight, ChatText, Copy, Stop } from '@phosphor-icons/react';
import { now, useClock } from '@/lib/stores/clock';
import { t } from '@/lib/stores/i18n';
import { markInvited } from '@/lib/stores/ritual';
import { stopTunnel } from '@/lib/stores/tunnels';
import { expiryLevel } from '@/lib/ritual';
import { formatDuration } from '@/lib/utils/duration';
import type { Tunnel } from '@/lib/types';
import { QrImage } from '../QrImage';
import { attempt, copyInvite, copyLink, displayUrl } from './shared';

/**
 * Moment three. The link and its QR at poster size, so the GM can read it out,
 * paste it or hold up a phone. Copying the invite message is one click.
 */
export function InviteMoment({ tunnel }: { tunnel: Tunnel }) {
  useEffect(() => useClock(), []);
  const [busy, setBusy] = useState(false);

  const url = tunnel.publicUrl ?? '';
  const remaining = tunnel.expiresAt ? tunnel.expiresAt - now.value : 0;
  const expiry = expiryLevel(remaining);

  async function stop() {
    setBusy(true);
    await attempt(() => stopTunnel(tunnel.id));
    setBusy(false);
  }

  return (
    <section class="flex flex-col items-center py-8 text-center" aria-labelledby="invite-title">
      <p class="m-0 max-w-full truncate text-[13px] text-text-muted">{tunnel.name}</p>
      <h1
        id="invite-title"
        class="m-0 mt-1 font-serif text-[clamp(28px,8vw,40px)] leading-tight font-medium text-text"
      >
        {t('invite_title')}
      </h1>
      <p class="m-0 mt-2 max-w-[44ch] text-sm text-text-muted">{t('invite_lead')}</p>

      {url && (
        <div class="mt-7 rounded-[14px] bg-white p-3 shadow-sm">
          <QrImage
            value={url}
            size={248}
            alt={t('qr_for_phones')}
            class="block"
          />
        </div>
      )}

      <p
        class="m-0 mt-6 max-w-full font-mono text-[clamp(16px,4.6vw,22px)] break-all text-url-text"
        title={url}
      >
        {displayUrl(url)}
      </p>

      <div class="mt-6 flex w-full flex-wrap justify-center gap-2">
        <Button variant="primary" size="lg" onClick={() => void copyLink(tunnel)}>
          <Copy size={16} />
          {t('copy_link')}
        </Button>
        <Button variant="secondary" size="lg" onClick={() => void copyInvite(tunnel)}>
          <ChatText size={16} />
          {t('copy_invite')}
        </Button>
      </div>

      <div class="mt-6 flex flex-wrap items-center justify-center gap-x-4 gap-y-2 text-[13px] text-text-muted">
        {expiry !== 'none' && (
          <span
            class={`font-mono text-xs tabular-nums ${expiry === 'soon' ? 'text-status-starting' : ''}`}
          >
            {t('card_expires', { 0: formatDuration(remaining) })}
          </span>
        )}
        <Button variant="quiet" size="sm" onClick={() => markInvited(tunnel.id)}>
          {t('invite_done')}
          <ArrowRight size={14} />
        </Button>
        <Button variant="quiet" size="sm" loading={busy} onClick={stop}>
          <Stop size={14} />
          {t('stop')}
        </Button>
      </div>
    </section>
  );
}
