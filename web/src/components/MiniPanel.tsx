import { useEffect, useState } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { Check, Copy, Plugs, QrCode as QrIcon } from '@phosphor-icons/react';
import { now, useClock } from '@/lib/stores/clock';
import { providerLabel } from '@/lib/stores/providers';
import { tunnels } from '@/lib/stores/tunnels';
import { startTunnel, stopTunnel } from '@/lib/stores/tunnels';
import { t } from '@/lib/stores/i18n';
import { toast } from '@/lib/stores/toast';
import { copyText } from '@/lib/utils/clipboard';
import { formatDuration } from '@/lib/utils/duration';
import { isRunningState } from '@/lib/utils/status';
import { providerErrorHint } from '@/lib/utils/providerError';
import { queryParam } from '@/router';
import { QrImage } from './QrImage';
import { StatusBadge } from './StatusPill';

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div class="min-w-0 px-2 py-1.5">
      <dt class="m-0 truncate text-2xs text-text-muted">{label}</dt>
      <dd class="m-0 truncate font-mono text-xs text-text">{value}</dd>
    </div>
  );
}

export function MiniPanel() {
  useEffect(() => useClock(), []);

  const [copied, setCopied] = useState(false);
  const id = queryParam('id');
  const tunnel = tunnels.value.find((candidate) => candidate.id === id) ?? null;

  const running = tunnel ? isRunningState(tunnel.state) : false;
  const uptime =
    tunnel?.startedAt && running ? formatDuration(now.value - tunnel.startedAt) : '';
  const remaining = tunnel?.expiresAt && running ? tunnel.expiresAt - now.value : 0;
  const errorHint = providerErrorHint(tunnel?.errorMessage);

  async function copy() {
    if (!tunnel?.publicUrl) return;
    if (!(await copyText(tunnel.publicUrl))) {
      toast.error(t('copy_failed'));
      return;
    }
    toast.success(t('overview_copied'));
    setCopied(true);
    setTimeout(() => setCopied(false), 3000);
  }

  async function toggle() {
    if (!tunnel) return;
    try {
      if (running) await stopTunnel(tunnel.id);
      else await startTunnel(tunnel.id);
    } catch (error) {
      toast.error((error as Error).message);
    }
  }

  return (
    <div class="flex h-dvh min-h-0 flex-col gap-2.5 bg-bg p-3 text-text">
      {!tunnel ? (
        <div class="flex h-full flex-col items-center justify-center gap-2 text-center text-text-muted">
          <Plugs size={22} />
          <p class="m-0 text-2xs leading-relaxed">{t('pip_gone')}</p>
        </div>
      ) : (
        <>
          <div class="flex shrink-0 items-start justify-between gap-2">
            <div class="min-w-0">
              <h1 class="m-0 truncate text-sm font-semibold text-text" title={tunnel.name}>
                {tunnel.name}
              </h1>
              <p class="m-0 truncate text-2xs text-text-muted">
                {providerLabel(tunnel.provider)} ·{' '}
                <span class="font-mono">localhost:{tunnel.port}</span>
              </p>
            </div>
            <span class="shrink-0">
              <StatusBadge state={tunnel.state} />
            </span>
          </div>

          <dl class="m-0 grid shrink-0 grid-cols-3 divide-x divide-border-light overflow-hidden rounded-[var(--sph-radius-sm)] border border-border-light bg-card">
            <Stat label={t('detail_uptime')} value={uptime || '—'} />
            <Stat
              label={t('detail_sessions')}
              value={tunnel.state === 'online' ? String(tunnel.activeSessions ?? 0) : '—'}
            />
            <Stat
              label={remaining > 0 ? t('detail_expires') : t('detail_visitors')}
              value={
                remaining > 0
                  ? formatDuration(remaining)
                  : tunnel.state === 'online'
                    ? String(tunnel.visitors ?? 0)
                    : '—'
              }
            />
          </dl>

          {tunnel.publicUrl && (
            <button
              type="button"
              onClick={copy}
              class="flex w-full shrink-0 cursor-pointer items-center gap-2 rounded-[var(--sph-radius-sm)] bg-url-bg px-2.5 py-2 text-start transition-colors hover:bg-hover"
            >
              <span class="min-w-0 flex-1 truncate font-mono text-2xs text-url-text">
                {tunnel.publicUrl}
              </span>
              {copied ? (
                <Check size={12} className="shrink-0 text-status-running" />
              ) : (
                <Copy size={12} className="shrink-0 text-text-muted" />
              )}
            </button>
          )}

          {tunnel.errorMessage && (
            <p
              class="m-0 shrink-0 rounded-[var(--sph-radius-sm)] border border-status-error/40 bg-status-error/10 px-2 py-1 text-2xs break-words text-text-error"
              title={tunnel.errorMessage}
            >
              {errorHint ? t(errorHint) : tunnel.errorMessage}
            </p>
          )}

          <div class="flex min-h-0 flex-1 flex-col items-center justify-center gap-1.5">
            {tunnel.publicUrl ? (
              <>
                <QrImage
                  value={tunnel.publicUrl}
                  size={176}
                  alt={t('qr_for_phones')}
                  class="max-h-full min-h-0 w-auto max-w-full object-contain"
                />
                <p class="m-0 shrink-0 text-center text-2xs text-text-muted">
                  {t('overview_share')}
                </p>
              </>
            ) : (
              <>
                <div class="flex aspect-square h-full min-h-0 items-center justify-center rounded-[var(--sph-radius-sm)] border border-dashed border-border text-text-muted/50">
                  <QrIcon size={32} />
                </div>
                <p class="m-0 max-w-[24ch] shrink-0 text-center text-2xs leading-relaxed text-text-muted">
                  {t('pip_idle')}
                </p>
              </>
            )}
          </div>

          <Button
            variant={running ? 'secondary' : 'primary'}
            className="w-full shrink-0"
            disabled={tunnel.state === 'installing' || tunnel.state === 'stopping'}
            onClick={toggle}
          >
            {running ? t('stop') : t('start')}
          </Button>
        </>
      )}
    </div>
  );
}
