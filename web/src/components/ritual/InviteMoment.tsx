import { useEffect, useState } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { Tooltip } from '@zovaris/sephiro/tooltip';
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
import { DecodeText } from '../DecodeText';
import { attempt, copyInvite, copyLink, copyQr, displayUrl } from './shared';
import { Ticket } from './Ticket';
import { AsciiDie } from './AsciiDie';

/**
 * Moment three. The link and its QR at poster size, so the GM can read it out,
 * paste it or hold up a phone. Copying the invite message is one click.
 */
export function InviteMoment({ tunnel }: { tunnel: Tunnel }) {
  useEffect(() => useClock(), []);
  const [busy, setBusy] = useState(false);
  // The die from the opening lands on a 20 before the QR develops in its place.
  const [landed, setLanded] = useState(false);

  const url = tunnel.publicUrl ?? '';
  const remaining = tunnel.expiresAt ? tunnel.expiresAt - now.value : 0;
  const expiry = expiryLevel(remaining);

  async function stop() {
    setBusy(true);
    await attempt(() => stopTunnel(tunnel.id));
    setBusy(false);
  }

  return (
    <section class="ftm-moment" aria-labelledby="invite-title">
      <Ticket
        label={tunnel.name}
        state="open"
        stub={
          !landed ? (
            <AsciiDie mode="twenty" onSettled={() => setTimeout(() => setLanded(true), 260)} />
          ) : url ? (
            <>
              <Tooltip content={t('qr_copy')}>
                <Button
                  variant="quiet"
                  className="ftm-qr ftm-ticket__qr"
                  aria-label={t('qr_copy')}
                  onClick={() => void copyQr(url)}
                >
                  <QrImage value={url} size={176} fluid alt={t('qr_for_phones')} />
                </Button>
              </Tooltip>
              <span class="ftm-ticket__label">{t('qr_scan_short')}</span>
            </>
          ) : null
        }
      >
        <p class="ftm-ticket__eyebrow">
          <span class="ftm-dot" data-check="answers" aria-hidden="true" />
          {t('invite_title')}
        </p>
        <h1 id="invite-title" class="ftm-ticket__world">
          {tunnel.name}
        </h1>
        <p class="ftm-url" title={url}>
          <DecodeText text={displayUrl(url)} duration={760} />
        </p>
        <div class="ftm-ticket__actions">
          <Button variant="primary" onClick={() => void copyLink(tunnel)}>
            <Copy size={16} />
            {t('copy_link')}
          </Button>
          <Button variant="secondary" onClick={() => void copyInvite(tunnel)}>
            <ChatText size={16} />
            {t('copy_invite')}
          </Button>
        </div>
      </Ticket>

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
