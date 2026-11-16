import { useState } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { IconAction } from '@/components/IconAction';
import { Plus, WarningCircle } from '@phosphor-icons/react';
import { unlockSound } from '@/lib/sound';
import { t } from '@/lib/stores/i18n';
import { providerLabel } from '@/lib/stores/providers';
import { startTunnel } from '@/lib/stores/tunnels';
import type { Tunnel } from '@/lib/types';
import { ProviderIcon } from './ProviderIcon';
import { attempt } from './shared';

/**
 * Saved connections as presets: one chip each and a plus for a new one. On the
 * ready screen a chip picks the table the big action opens; mid-session a chip
 * opens that table straight away.
 */
export function PresetChips({
  presets,
  selectedId,
  mode,
  align = 'center',
  onSelect,
  onCreate,
}: {
  presets: Tunnel[];
  selectedId: string | null;
  mode: 'select' | 'start';
  align?: 'center' | 'start';
  onSelect: (id: string) => void;
  onCreate: () => void;
}) {
  return (
    <ul
      class={`m-0 flex list-none flex-wrap gap-2 p-0 ${align === 'center' ? 'justify-center' : ''}`}
      aria-label={mode === 'select' ? t('saved_tables') : t('session_others')}
    >
      {presets.map((tunnel) => (
        <li key={tunnel.id} class="min-w-0">
          <PresetChip
            tunnel={tunnel}
            selected={mode === 'select' && tunnel.id === selectedId}
            mode={mode}
            onSelect={onSelect}
          />
        </li>
      ))}
      <li>
        <IconAction
          label={t('new_connection_action')}
          variant="outline"
          size="sm"
          icon={<Plus size={16} />}
          onClick={onCreate}
        />
      </li>
    </ul>
  );
}

function PresetChip({
  tunnel,
  selected,
  mode,
  onSelect,
}: {
  tunnel: Tunnel;
  selected: boolean;
  mode: 'select' | 'start';
  onSelect: (id: string) => void;
}) {
  const [busy, setBusy] = useState(false);
  const failed = tunnel.state === 'error' || tunnel.state === 'timeout';

  async function start() {
    unlockSound();
    setBusy(true);
    await attempt(() => startTunnel(tunnel.id));
    setBusy(false);
  }

  return (
    <Button
      variant={selected ? 'secondary' : 'quiet'}
      size="sm"
      className={`ftm-chip max-w-[16rem] ${selected ? 'is-selected' : ''}`}
      aria-pressed={mode === 'select' ? selected : undefined}
      title={`${tunnel.name} · ${providerLabel(tunnel.provider)} · :${tunnel.port}`}
      loading={busy}
      onClick={() => (mode === 'select' ? onSelect(tunnel.id) : void start())}
    >
      {failed ? (
        <WarningCircle size={16} className="shrink-0 text-text-error" aria-label={t(tunnel.state)} />
      ) : (
        <ProviderIcon provider={tunnel.provider} />
      )}
      <span class="truncate">{tunnel.name}</span>
      <span class="ftm-chip__port">{tunnel.port}</span>
    </Button>
  );
}
