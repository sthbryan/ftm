import { useEffect, useState } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { Surface } from '@zovaris/sephiro/surface';
import { IconAction } from '@/components/IconAction';
import { Copy, PictureInPicture, QrCode, Stop, Terminal, User } from '@phosphor-icons/react';
import { pipApi, statusApi } from '@/lib/api';
import { expiryLevel } from '@/lib/ritual';
import { now, useClock } from '@/lib/stores/clock';
import { t } from '@/lib/stores/i18n';
import { toast } from '@/lib/stores/toast';
import { stopTunnel, tunnelsConnected } from '@/lib/stores/tunnels';
import { formatClock, formatDuration } from '@/lib/utils/duration';
import { openDocumentPip, supportsDocumentPip } from '@/lib/utils/pip';
import type { Tunnel } from '@/lib/types';
import { LogsPanel } from '../LogsPanel';
import { QrDialog } from '../QrDialog';
import { StatusMarkDot } from '../StatusPill';
import { attempt, copyLink } from './shared';

let nativePipCache: Promise<boolean> | null = null;

function nativePip(): Promise<boolean> {
  nativePipCache ??= statusApi
    .get()
    .then((status) => status.nativePip === true)
    .catch(() => false);
  return nativePipCache;
}

/** Seats drawn before the rest collapse into the count. */
const MAX_SEATS = 5;

export function playersLabel(count: number): string {
  if (count === 0) return t('session_waiting');
  return count === 1 ? t('session_players_one') : t('session_players', { count });
}

/**
 * Moment four. One slim strip per running table: who is in, how long it has
 * run, when it closes, and the few things the GM reaches for mid-session.
 * Logs open under the strip on demand, on the page, not in a box inside it.
 */
export function SessionStrip({ tunnel }: { tunnel: Tunnel }) {
  useEffect(() => useClock(), []);
  const [logsOpen, setLogsOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [native, setNative] = useState(false);
  const [qrOpen, setQrOpen] = useState(false);

  useEffect(() => {
    void nativePip().then(setNative);
  }, []);

  const online = tunnel.state === 'online';
  const stale = online && !tunnelsConnected.value;
  const players = tunnel.activeSessions ?? 0;
  const remaining = tunnel.expiresAt && online ? tunnel.expiresAt - now.value : 0;
  const expiry = expiryLevel(remaining);
  const elapsed = tunnel.startedAt && online && !stale ? now.value - tunnel.startedAt : 0;
  const pipAvailable = native || supportsDocumentPip();

  async function stop() {
    setBusy(true);
    await attempt(() => stopTunnel(tunnel.id));
    setBusy(false);
  }

  async function openPip() {
    try {
      if (native) await pipApi.open(tunnel.id);
      else await openDocumentPip(tunnel.id);
    } catch {
      toast.error(t('pip_failed'));
    }
  }

  return (
    <div class="flex flex-col gap-2">
      <Surface className="ftm-strip" role="group" aria-label={tunnel.name}>
        <div class="flex min-w-0 items-center gap-2.5">
          <StatusMarkDot state={tunnel.state} label={t(tunnel.state)} size="sm" />
          <p class="m-0 truncate text-sm font-medium text-text" title={tunnel.publicUrl}>
            {tunnel.name}
          </p>
        </div>

        <div class="flex items-center gap-3 text-xs text-text-muted">
          {stale ? (
            <span>{t('connection_lost')}</span>
          ) : online ? (
            <>
              {players > 0 && (
                <span class="ftm-seats" aria-hidden="true">
                  {Array.from({ length: Math.min(players, MAX_SEATS) }, (_, index) => (
                    <span key={index} class="ftm-seat">
                      <User size={14} />
                    </span>
                  ))}
                </span>
              )}
              <span class="whitespace-nowrap">{playersLabel(players)}</span>
            </>
          ) : (
            <span>{t(tunnel.state)}</span>
          )}
          {elapsed > 0 && (
            <span class="font-mono text-2xs tabular-nums" title={t('session_time')}>
              <span class="sr-only">{t('session_time')} </span>
              {formatClock(elapsed)}
            </span>
          )}
          {expiry !== 'none' && (
            <span
              class={`font-mono text-2xs whitespace-nowrap tabular-nums ${expiry === 'soon' ? 'text-status-starting' : ''}`}
            >
              {t('card_expires', { 0: formatDuration(remaining) })}
            </span>
          )}
        </div>

        <div class="ms-auto flex items-center gap-1">
          <Button
            variant="secondary"
            size="sm"
            disabled={!online}
            onClick={() => void copyLink(tunnel)}
          >
            <Copy size={16} />
            {t('session_link')}
          </Button>
          <IconAction
            label={t('qr_for_phones')}
            size="sm"
            variant="outline"
            disabled={!online}
            icon={<QrCode size={16} />}
            onClick={() => setQrOpen(true)}
          />
          {pipAvailable && (
            <IconAction
              label={t('pip_open')}
              size="sm"
              disabled={!online}
              icon={<PictureInPicture size={16} />}
              onClick={openPip}
            />
          )}
          <IconAction
            label={t('session_logs')}
            size="sm"
            aria-expanded={logsOpen}
            aria-pressed={logsOpen}
            icon={<Terminal size={16} />}
            onClick={() => setLogsOpen((open) => !open)}
          />
          <IconAction
            label={t('stop')}
            size="sm"
            loading={busy}
            disabled={tunnel.state === 'stopping'}
            icon={<Stop size={16} />}
            onClick={stop}
          />
        </div>
      </Surface>

      {logsOpen && <LogsPanel tunnelId={tunnel.id} state={tunnel.state} />}

      <QrDialog open={qrOpen} url={tunnel.publicUrl ?? ''} onClose={() => setQrOpen(false)} />
    </div>
  );
}
