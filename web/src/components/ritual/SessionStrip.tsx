import { useEffect, useState } from 'preact/hooks';
import { IconButton } from '@zovaris/sephiro/icon-button';
import {
  Clock,
  Copy,
  PictureInPicture,
  QrCode,
  Stop,
  Terminal,
  UsersThree,
} from '@phosphor-icons/react';
import { pipApi, statusApi } from '@/lib/api';
import { expiryLevel } from '@/lib/ritual';
import { now, useClock } from '@/lib/stores/clock';
import { t } from '@/lib/stores/i18n';
import { toast } from '@/lib/stores/toast';
import { stopTunnel, tunnelsConnected } from '@/lib/stores/tunnels';
import { formatDuration } from '@/lib/utils/duration';
import { openDocumentPip, supportsDocumentPip } from '@/lib/utils/pip';
import type { Tunnel } from '@/lib/types';
import { LogsPanel } from '../LogsPanel';
import { QrDialog } from '../QrDialog';
import { StatusMarkDot } from '../StatusPill';
import { attempt, copyLink, displayUrl } from './shared';

let nativePipCache: Promise<boolean> | null = null;

function nativePip(): Promise<boolean> {
  nativePipCache ??= statusApi
    .get()
    .then((status) => status.nativePip === true)
    .catch(() => false);
  return nativePipCache;
}

/**
 * Moment four. A slim strip per running table: status, players connected,
 * session time, expiry, and the few things the GM reaches for mid-session.
 * Logs open under it on demand.
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
    <article
      class="overflow-hidden rounded-[12px] border border-border bg-card"
      aria-label={tunnel.name}
    >
      <div class="flex flex-wrap items-center gap-x-4 gap-y-2 px-4 py-3">
        <div class="flex min-w-0 flex-[1_1_14rem] items-center gap-3">
          <StatusMarkDot state={tunnel.state} label={t(tunnel.state)} size="sm" />
          <div class="min-w-0 flex-1">
            <p class="m-0 truncate text-sm font-medium text-text">{tunnel.name}</p>
            <p class="m-0 truncate font-mono text-xs text-text-muted" title={tunnel.publicUrl}>
              {stale
                ? t('connection_lost')
                : online
                  ? displayUrl(tunnel.publicUrl)
                  : t(tunnel.state)}
            </p>
          </div>
        </div>

        <p class="m-0 flex flex-wrap items-center gap-x-4 gap-y-1 text-[13px] text-text-muted">
          <span class="flex items-center gap-1.5" title={t('overview_players')}>
            <UsersThree size={15} aria-hidden="true" />
            <span class="font-medium text-text tabular-nums">{players}</span>
            <span class="sr-only">{t('overview_players')}</span>
          </span>
          {elapsed > 0 && (
            <span class="flex items-center gap-1.5" title={t('session_time')}>
              <Clock size={15} aria-hidden="true" />
              <span class="sr-only">{t('session_time')}</span>
              <span class="font-mono text-xs tabular-nums">{formatDuration(elapsed)}</span>
            </span>
          )}
          {expiry !== 'none' && (
            <span
              class={`font-mono text-xs tabular-nums ${expiry === 'soon' ? 'text-status-starting' : ''}`}
            >
              {t('card_expires', { 0: formatDuration(remaining) })}
            </span>
          )}
        </p>

        <div class="ms-auto flex items-center gap-0.5">
          <IconButton
            label={t('copy_link')}
            size="sm"
            disabled={!online}
            icon={<Copy size={16} />}
            onClick={() => void copyLink(tunnel)}
          />
          <IconButton
            label={t('qr_for_phones')}
            size="sm"
            disabled={!online}
            icon={<QrCode size={16} />}
            onClick={() => setQrOpen(true)}
          />
          {pipAvailable && (
            <IconButton
              label={t('pip_open')}
              size="sm"
              disabled={!online}
              icon={<PictureInPicture size={16} />}
              onClick={openPip}
            />
          )}
          <IconButton
            label={t('session_logs')}
            size="sm"
            aria-expanded={logsOpen}
            icon={<Terminal size={16} />}
            onClick={() => setLogsOpen((open) => !open)}
          />
          <IconButton
            label={t('stop')}
            size="sm"
            loading={busy}
            disabled={tunnel.state === 'stopping'}
            icon={<Stop size={16} />}
            onClick={stop}
          />
        </div>
      </div>

      {logsOpen && (
        <div class="border-t border-border px-4 py-3">
          <LogsPanel tunnelId={tunnel.id} state={tunnel.state} />
        </div>
      )}

      <QrDialog open={qrOpen} url={tunnel.publicUrl ?? ''} onClose={() => setQrOpen(false)} />
    </article>
  );
}
