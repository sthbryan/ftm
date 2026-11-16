import { useEffect, useState } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { Surface } from '@zovaris/sephiro/surface';
import { ArrowRight, ChatText, Copy, Stop } from '@phosphor-icons/react';
import { now, useClock } from '@/lib/stores/clock';
import { t } from '@/lib/stores/i18n';
import { markInvited } from '@/lib/stores/ritual';
import { stopTunnel } from '@/lib/stores/tunnels';
import { expiryLevel } from '@/lib/ritual';
import { formatDuration } from '@/lib/utils/duration';
import type { Tunnel } from '@/lib/types';
import { NotificationNudge } from '../NotificationNudge';
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
    <section class="mx-auto flex w-full max-w-2xl flex-col gap-6" aria-labelledby="invite-title">
      <Surface padding="lg" className="ftm-invite">
        {url && (
          <div class="ftm-qr">
            <QrImage value={url} size={176} alt={t('qr_for_phones')} class="block" />
          </div>
        )}
        <div class="min-w-0">
          <p class="m-0 text-xs text-text-muted">{t('invite_title')}</p>
          <h1 id="invite-title" class="m-0 mt-1.5 truncate font-serif text-2xl font-medium text-text">
            {tunnel.name}
          </h1>
          <p class="ftm-url" title={url}>
            {displayUrl(url)}
          </p>
          <div class="flex flex-wrap gap-2">
            <Button variant="primary" onClick={() => void copyLink(tunnel)}>
              <Copy size={16} />
              {t('copy_link')}
            </Button>
            <Button variant="secondary" onClick={() => void copyInvite(tunnel)}>
              <ChatText size={16} />
              {t('copy_invite')}
            </Button>
          </div>
        </div>
      </Surface>

      <div class="flex flex-wrap items-center justify-center gap-x-3 gap-y-2 text-xs text-text-muted">
        {expiry !== 'none' && (
          <span
            class={`font-mono text-2xs tabular-nums ${expiry === 'soon' ? 'text-status-starting' : ''}`}
          >
            {t('card_expires', { 0: formatDuration(remaining) })}
          </span>
        )}
        <Button variant="quiet" size="sm" loading={busy} onClick={stop}>
          <Stop size={16} />
          {t('stop')}
        </Button>
        <Button variant="quiet" size="sm" onClick={() => markInvited(tunnel.id)}>
          {t('invite_done')}
          <ArrowRight size={16} />
        </Button>
      </div>

      <div class="flex justify-center">
        <NotificationNudge />
      </div>
    </section>
  );
}
