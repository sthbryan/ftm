import { useEffect, useState } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { IconButton } from '@zovaris/sephiro/icon-button';
import { Progress } from '@zovaris/sephiro/progress';
import {
  ArrowClockwise,
  CaretDown,
  CaretRight,
  Check,
  Copy,
  PencilSimple,
  PictureInPicture,
  Play,
  QrCode,
  Stop,
  Trash,
} from '@phosphor-icons/react';
import { pipApi, statusApi } from '@/lib/api';
import { now, useClock } from '@/lib/stores/clock';
import { t } from '@/lib/stores/i18n';
import { providerLabel } from '@/lib/stores/providers';
import { toast } from '@/lib/stores/toast';
import {
  installProgress,
  startTunnel,
  stopTunnel,
  tunnelsConnected,
} from '@/lib/stores/tunnels';
import { copyText } from '@/lib/utils/clipboard';
import { formatDuration } from '@/lib/utils/duration';
import { openDocumentPip, supportsDocumentPip } from '@/lib/utils/pip';
import { providerErrorHint } from '@/lib/utils/providerError';
import { isInstallingState, isRunningState } from '@/lib/utils/status';
import type { Tunnel } from '@/lib/types';
import { LogsPanel } from './LogsPanel';
import { QrDialog } from './QrDialog';
import { StatusMarkDot } from './StatusPill';

type Phase = 'idle' | 'installing' | 'connecting' | 'online' | 'failed';

function phaseOf(state: string): Phase {
  if (state === 'online') return 'online';
  if (state === 'error' || state === 'timeout') return 'failed';
  if (isInstallingState(state)) return 'installing';
  if (isRunningState(state)) return 'connecting';
  return 'idle';
}

/**
 * The connection the GM is working with. It always shows the next thing to do:
 * start it, wait for it, send the link, or fix what stopped it.
 */
export function ConnectionHero({
  tunnel,
  onEdit,
  onDelete,
}: {
  tunnel: Tunnel;
  onEdit: (id: string) => void;
  onDelete: (tunnel: Tunnel) => void;
}) {
  useEffect(() => useClock(), []);

  const [logsOpen, setLogsOpen] = useState(false);
  const [qrOpen, setQrOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const [busy, setBusy] = useState(false);
  const [nativePip, setNativePip] = useState(false);

  useEffect(() => {
    statusApi
      .get()
      .then((status) => setNativePip(status.nativePip === true))
      .catch(() => setNativePip(false));
  }, []);

  const phase = phaseOf(tunnel.state);
  const running = isRunningState(tunnel.state);
  const stale = running && !tunnelsConnected.value;
  const progress = installProgress.value[tunnel.provider];
  const provider = providerLabel(tunnel.provider);
  const hint = providerErrorHint(tunnel.errorMessage);
  const remaining = tunnel.expiresAt && running ? tunnel.expiresAt - now.value : 0;
  const uptime =
    tunnel.startedAt && running && !stale ? formatDuration(now.value - tunnel.startedAt) : '';
  const players = tunnel.activeSessions ?? 0;
  const pipAvailable = nativePip || supportsDocumentPip();

  async function run(action: () => Promise<unknown>) {
    setBusy(true);
    try {
      await action();
    } catch (error) {
      toast.error((error as Error).message);
    } finally {
      setBusy(false);
    }
  }

  async function copy() {
    if (!tunnel.publicUrl) return;
    if (!(await copyText(tunnel.publicUrl))) {
      toast.error(t('copy_failed'));
      return;
    }
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  async function openPip() {
    try {
      if (nativePip) await pipApi.open(tunnel.id);
      else await openDocumentPip(tunnel.id);
    } catch {
      toast.error(t('pip_failed'));
    }
  }

  return (
    <section
      class="overflow-hidden rounded-[12px] border border-border bg-card"
      aria-label={tunnel.name}
    >
      <header class="flex items-start gap-3 px-5 pt-5">
        <div class="min-w-0 flex-1">
          <h1 class="m-0 truncate font-serif text-[26px] leading-tight font-medium text-text">
            {tunnel.name}
          </h1>
          <p class="m-0 mt-1 truncate font-mono text-xs text-text-muted">
            {provider} · localhost:{tunnel.port}
          </p>
        </div>

        <div class="flex shrink-0 items-center gap-1">
          <span class="mr-1 flex items-center gap-2 text-[13px] text-text-muted">
            <StatusMarkDot state={tunnel.state} label={t(tunnel.state)} size="sm" />
            {stale ? t('connection_lost') : t(tunnel.state)}
          </span>
          <IconButton
            label={t('edit')}
            size="sm"
            icon={<PencilSimple size={16} />}
            disabled={running}
            onClick={() => onEdit(tunnel.id)}
          />
          <IconButton
            label={t('delete')}
            size="sm"
            icon={<Trash size={16} />}
            disabled={running}
            onClick={() => onDelete(tunnel)}
          />
        </div>
      </header>

      <div class="px-5 pt-4 pb-5">
        {phase === 'idle' && (
          <>
            <p class="m-0 mb-4 max-w-[58ch] text-sm text-text-muted">{t('hero_idle')}</p>
            <Button
              variant="primary"
              size="lg"
              loading={busy}
              onClick={() => run(() => startTunnel(tunnel.id))}
            >
              <Play size={16} weight="fill" />
              {t('hero_start')}
            </Button>
          </>
        )}

        {phase === 'installing' && (
          <div class="flex flex-col gap-3">
            <Progress value={progress?.percent} label={t(tunnel.state)} />
            <p class="m-0 truncate text-sm text-text-muted">
              {progress?.step || t('hero_installing', { 0: provider })}
            </p>
          </div>
        )}

        {phase === 'connecting' && (
          <div class="flex flex-col gap-3">
            <Progress label={t(tunnel.state)} />
            <div class="flex items-center gap-3">
              <p class="m-0 min-w-0 flex-1 truncate text-sm text-text-muted">
                {t('hero_connecting', { 0: provider })}
              </p>
              <Button
                variant="quiet"
                disabled={tunnel.state === 'stopping'}
                onClick={() => run(() => stopTunnel(tunnel.id))}
              >
                {t('cancel')}
              </Button>
            </div>
          </div>
        )}

        {phase === 'online' && (
          <>
            <p class="m-0 mb-2 text-xs text-text-muted">{t('overview_share')}</p>
            <div class="flex items-center gap-2 rounded-[10px] border border-border bg-bg py-1.5 pr-1.5 pl-4">
              <span
                class="min-w-0 flex-1 truncate font-mono text-[15px] text-url-text"
                title={tunnel.publicUrl}
              >
                {tunnel.publicUrl?.replace(/^https?:\/\//, '')}
              </span>
              <Button variant="primary" onClick={copy}>
                {copied ? <Check size={16} /> : <Copy size={16} />}
                {copied ? t('link_copied') : t('copy_link')}
              </Button>
              <IconButton
                label={t('qr_for_phones')}
                variant="outline"
                icon={<QrCode size={16} />}
                onClick={() => setQrOpen(true)}
              />
            </div>

            <div class="mt-4 flex flex-wrap items-center gap-x-5 gap-y-2 text-[13px] text-text-muted">
              <span>
                <span class="font-medium text-text tabular-nums">{players}</span>{' '}
                {t('overview_players')}
              </span>
              {uptime && <span class="font-mono text-xs tabular-nums">{uptime}</span>}
              {remaining > 0 && (
                <span class="font-mono text-xs text-status-starting tabular-nums">
                  {t('card_expires', { 0: formatDuration(remaining) })}
                </span>
              )}
              <span class="flex-1" />
              {pipAvailable && (
                <Button variant="quiet" size="sm" onClick={openPip}>
                  <PictureInPicture size={16} />
                  {t('pip_open')}
                </Button>
              )}
              <Button
                variant="quiet"
                size="sm"
                loading={busy}
                onClick={() => run(() => stopTunnel(tunnel.id))}
              >
                <Stop size={16} />
                {t('stop')}
              </Button>
            </div>
          </>
        )}

        {phase === 'failed' && (
          <>
            <div class="mb-4 border-l-2 border-status-error py-0.5 pl-3">
              <p class="m-0 text-sm font-medium text-text">
                {hint ? t(hint) : t('hero_failed')}
              </p>
              {tunnel.errorMessage && (
                <p class="m-0 mt-1 font-mono text-xs break-words text-text-muted">
                  {tunnel.errorMessage}
                </p>
              )}
            </div>
            <div class="flex flex-wrap gap-2">
              <Button
                variant="primary"
                loading={busy}
                onClick={() => run(() => startTunnel(tunnel.id))}
              >
                <ArrowClockwise size={16} />
                {t('retry')}
              </Button>
              <Button variant="quiet" onClick={() => onEdit(tunnel.id)}>
                <PencilSimple size={16} />
                {t('edit')}
              </Button>
            </div>
          </>
        )}
      </div>

      {(running || phase === 'failed') && (
        <div class="border-t border-border">
          <button
            type="button"
            class="flex w-full cursor-pointer items-center gap-2 border-0 bg-transparent px-5 py-3 text-start text-[13px] text-text-muted transition-colors hover:text-text"
            aria-expanded={logsOpen}
            onClick={() => setLogsOpen((open) => !open)}
          >
            {logsOpen ? <CaretDown size={14} /> : <CaretRight size={14} />}
            {t('detail_logs')}
          </button>
          {logsOpen && (
            <div class="px-5 pb-5">
              <LogsPanel tunnelId={tunnel.id} state={tunnel.state} />
            </div>
          )}
        </div>
      )}

      <QrDialog open={qrOpen} url={tunnel.publicUrl ?? ''} onClose={() => setQrOpen(false)} />
    </section>
  );
}
