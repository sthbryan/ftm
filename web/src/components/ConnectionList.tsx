import { useEffect, useState } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { IconButton } from '@zovaris/sephiro/icon-button';
import { Play, Plus, Stop } from '@phosphor-icons/react';
import { now, useClock } from '@/lib/stores/clock';
import { t } from '@/lib/stores/i18n';
import { providerLabel } from '@/lib/stores/providers';
import { toast } from '@/lib/stores/toast';
import { startTunnel, stopTunnel } from '@/lib/stores/tunnels';
import { formatDuration } from '@/lib/utils/duration';
import { isInstallingState, isRunningState } from '@/lib/utils/status';
import type { Tunnel } from '@/lib/types';
import { StatusMarkDot } from './StatusPill';

/** Every connection except the one in the hero, as quiet rows. */
export function ConnectionList({
  tunnels,
  onSelect,
  onCreate,
}: {
  tunnels: Tunnel[];
  onSelect: (id: string) => void;
  onCreate: () => void;
}) {
  useEffect(() => useClock(), []);

  return (
    <section class="mt-8" aria-label={t('other_connections')}>
      <div class="mb-2 flex items-center gap-3">
        <h2 class="m-0 flex-1 text-[13px] font-normal text-text-muted">
          {tunnels.length > 0 ? t('other_connections') : ''}
        </h2>
        <Button variant="secondary" size="sm" onClick={onCreate}>
          <Plus size={14} />
          {t('new_connection_action')}
        </Button>
      </div>

      {tunnels.length > 0 && (
        <ul class="m-0 list-none border-t border-border p-0">
          {tunnels.map((tunnel) => (
            <ConnectionRow key={tunnel.id} tunnel={tunnel} onSelect={onSelect} />
          ))}
        </ul>
      )}
    </section>
  );
}

function ConnectionRow({ tunnel, onSelect }: { tunnel: Tunnel; onSelect: (id: string) => void }) {
  const [busy, setBusy] = useState(false);

  const running = isRunningState(tunnel.state);
  const failed = tunnel.state === 'error' || tunnel.state === 'timeout';
  const remaining = tunnel.expiresAt && running ? tunnel.expiresAt - now.value : 0;

  async function toggle() {
    setBusy(true);
    try {
      if (running) await stopTunnel(tunnel.id);
      else await startTunnel(tunnel.id);
    } catch (error) {
      toast.error((error as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <li class="flex min-h-[52px] items-center gap-3 border-b border-border">
      <button
        type="button"
        class="flex min-w-0 flex-1 cursor-pointer items-center gap-3 border-0 bg-transparent py-2 pl-1 text-start transition-colors hover:text-text"
        onClick={() => onSelect(tunnel.id)}
      >
        <StatusMarkDot state={tunnel.state} label={t(tunnel.state)} size="sm" />
        <span class="min-w-0 flex-1">
          <span class="block truncate text-sm font-medium text-text">{tunnel.name}</span>
          <span class="block truncate font-mono text-xs text-text-muted">
            {providerLabel(tunnel.provider)} · :{tunnel.port}
          </span>
        </span>
        {remaining > 0 ? (
          <span class="shrink-0 font-mono text-xs text-status-starting tabular-nums">
            {t('card_expires', { 0: formatDuration(remaining) })}
          </span>
        ) : failed || running ? (
          <span class={`shrink-0 text-xs ${failed ? 'text-text-error' : 'text-text-muted'}`}>
            {t(tunnel.state)}
          </span>
        ) : null}
      </button>

      <IconButton
        label={running ? t('stop') : t('start')}
        size="sm"
        icon={running ? <Stop size={16} /> : <Play size={16} />}
        loading={busy}
        disabled={isInstallingState(tunnel.state) || tunnel.state === 'stopping'}
        onClick={toggle}
      />
    </li>
  );
}
