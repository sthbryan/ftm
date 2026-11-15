import { useEffect } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { Sparkline } from '@zovaris/sephiro/sparkline';
import { Check, Copy, Play, Stop, UsersThree, WarningCircle } from '@phosphor-icons/react';
import { useState } from 'preact/hooks';
import { now, useClock } from '@/lib/stores/clock';
import { historyFor } from '@/lib/stores/history';
import { providerLabel } from '@/lib/stores/providers';
import { startTunnel, stopTunnel, tunnelsConnected } from '@/lib/stores/tunnels';
import { toast } from '@/lib/stores/toast';
import { t } from '@/lib/stores/i18n';
import { copyText } from '@/lib/utils/clipboard';
import { formatDuration } from '@/lib/utils/duration';
import { isInstallingState, isRunningState } from '@/lib/utils/status';
import { providerErrorHint } from '@/lib/utils/providerError';
import type { Tunnel } from '@/lib/types';
import { StatusBadge, StatusMarkDot } from './StatusPill';

interface TunnelRowProps {
  tunnel: Tunnel;
  selected: boolean;
  installProgress: { percent: number; step: string } | null;
  onSelect: (id: string) => void;
  onEdit: (id: string) => void;
  onDelete: (tunnel: Tunnel) => void;
}

export function TunnelRow({ tunnel, selected, installProgress, onSelect }: TunnelRowProps) {
  useEffect(() => useClock(), []);

  const [copied, setCopied] = useState(false);

  const running = isRunningState(tunnel.state);
  const installing = isInstallingState(tunnel.state);
  const stale = running && !tunnelsConnected.value;
  const percent = tunnel.state === 'installing' && installProgress
    ? Math.trunc(installProgress.percent * 100) / 100
    : null;

  const history = historyFor(tunnel.id);
  const sessions = history.map((sample) => sample.sessions);

  const uptime =
    tunnel.startedAt && running && !stale
      ? formatDuration(now.value - tunnel.startedAt)
      : '';

  const remaining = tunnel.expiresAt && running ? tunnel.expiresAt - now.value : 0;
  const expiry = tunnel.expiresAt && running
    ? remaining > 0
      ? t('card_expires', { 0: formatDuration(remaining) })
      : t('card_expired')
    : '';

  const errorHint = providerErrorHint(tunnel.errorMessage);

  async function copy() {
    if (!tunnel.publicUrl) return;
    if (!(await copyText(tunnel.publicUrl))) {
      toast.error(t('copy_failed'));
      return;
    }
    toast.info(t('copied'));
    setCopied(true);
    setTimeout(() => setCopied(false), 3000);
  }

  async function action() {
    try {
      if (running) await stopTunnel(tunnel.id);
      else await startTunnel(tunnel.id);
    } catch (error) {
      toast.error((error as Error).message);
    }
  }

  return (
    <article class="sph-card" data-selected={selected ? 'true' : undefined}>
      <div class="flex items-start gap-2 p-3">
        <button
          type="button"
          aria-pressed={selected}
          onClick={() => onSelect(tunnel.id)}
          class="flex min-w-0 flex-1 cursor-pointer items-start gap-2.5 text-start"
        >
          <span class="mt-0.5 shrink-0">
            <StatusMarkDot state={tunnel.state} label={t(tunnel.state)} percent={percent} />
          </span>

          <span class="min-w-0 flex-1">
            <span class="flex min-w-0 items-baseline justify-between gap-2">
              <span class="truncate text-sm leading-snug font-semibold text-text">
                {tunnel.name}
              </span>
              <StatusBadge state={tunnel.state} />
            </span>

            <span class="mt-0.5 flex items-center gap-1.5 truncate text-2xs text-text-muted">
              <span class="truncate">{providerLabel(tunnel.provider)}</span>
              <span aria-hidden="true">·</span>
              <span class="font-mono">localhost:{tunnel.port}</span>
            </span>

            <span class="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-2xs text-text-muted">
              {uptime && <span class="font-mono tabular-nums">{uptime}</span>}
              {expiry && (
                <span class={remaining > 0 ? 'text-text-muted' : 'text-text-error'}>{expiry}</span>
              )}
              {stale && <span>{t('connection_lost')}</span>}
              {(tunnel.activeSessions ?? 0) > 0 && (
                <span class="flex items-center gap-1">
                  <UsersThree size={12} />
                  <span class="font-mono tabular-nums text-text">{tunnel.activeSessions}</span>
                </span>
              )}
              {sessions.length >= 4 && (
                <span class="ml-auto">
                  <Sparkline
                    values={sessions}
                    label={`${t('detail_sessions')} ${tunnel.name}`}
                    width={56}
                    height={16}
                    variant="line"
                    tone="accent"
                    animate={false}
                  />
                </span>
              )}
            </span>
          </span>
        </button>

        <Button
          variant={running ? 'secondary' : 'primary'}
          size="sm"
          onClick={action}
          disabled={installing || tunnel.state === 'stopping'}
        >
          {running ? <Stop size={14} weight="fill" /> : <Play size={14} weight="fill" />}
          <span class="max-sm:hidden">
            {installing ? t('wait') : tunnel.state === 'stopping' ? t('stopping') : running ? t('stop') : t('start')}
          </span>
        </Button>
      </div>

      {tunnel.state === 'installing' && installProgress && (
        <div class="px-3 pb-2.5">
          <div class="h-1 w-full overflow-hidden rounded-full bg-border-light">
            <div
              class="h-full rounded-full bg-status-installing transition-[width] duration-300"
              style={{ width: `${percent ?? 0}%` }}
            />
          </div>
          <div class="mt-1 truncate text-2xs text-text-muted">{installProgress.step}</div>
        </div>
      )}

      {tunnel.publicUrl && (
        <button
          type="button"
          class="flex w-full cursor-pointer items-center gap-2 border-t border-border-light px-3 py-1.5 text-start transition-colors hover:bg-hover"
          onClick={copy}
        >
          <span class="flex shrink-0 text-text-muted">
            {copied ? <Check size={13} /> : <Copy size={13} />}
          </span>
          <span class="min-w-0 flex-1 truncate font-mono text-2xs text-url-text">
            {tunnel.publicUrl}
          </span>
          <span class="shrink-0 text-2xs text-text-muted">
            {copied ? t('copied') : t('click_to_copy')}
          </span>
        </button>
      )}

      {tunnel.errorMessage && (
        <div class="flex items-start gap-2 border-t border-status-error/40 bg-status-error/10 px-3 py-2 text-text-error">
          <span class="mt-0.5 shrink-0">
            <WarningCircle size={14} />
          </span>
          <span class="min-w-0 flex-1">
            {errorHint && <span class="block text-2xs font-medium">{t(errorHint)}</span>}
            <span class="block font-mono text-2xs break-words opacity-80">
              {tunnel.errorMessage}
            </span>
          </span>
        </div>
      )}
    </article>
  );
}
