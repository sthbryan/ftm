import { useState } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { IconAction } from '@/components/IconAction';
import { PencilSimple, Play, Plus, Trash } from '@phosphor-icons/react';
import { unlockSound } from '@/lib/sound';
import { t } from '@/lib/stores/i18n';
import { providerLabel } from '@/lib/stores/providers';
import { startTunnel } from '@/lib/stores/tunnels';
import type { Tunnel } from '@/lib/types';
import { attempt } from './shared';

/**
 * Saved connections as presets. On the ready screen a row picks the table to
 * open; once a session is running, each row starts its table directly.
 */
export function PresetList({
  presets,
  selectedId,
  mode,
  onSelect,
  onCreate,
  onEdit,
  onDelete,
}: {
  presets: Tunnel[];
  selectedId: string | null;
  mode: 'select' | 'start';
  onSelect: (id: string) => void;
  onCreate: () => void;
  onEdit: (id: string) => void;
  onDelete: (tunnel: Tunnel) => void;
}) {
  const title = mode === 'select' ? t('saved_tables') : t('session_others');

  return (
    <section class="flex flex-col" aria-label={title}>
      {presets.length > 0 && (
        <>
          <h2 class="m-0 mb-2 text-xs font-medium text-text-muted">{title}</h2>
          <ul class="m-0 list-none border-t border-border-light p-0">
            {presets.map((tunnel) => (
              <PresetRow
                key={tunnel.id}
                tunnel={tunnel}
                selected={mode === 'select' && tunnel.id === selectedId}
                mode={mode}
                onSelect={onSelect}
                onEdit={onEdit}
                onDelete={onDelete}
              />
            ))}
          </ul>
        </>
      )}

      <div class={presets.length > 0 ? 'mt-3' : ''}>
        <Button variant="quiet" size="sm" onClick={onCreate}>
          <Plus size={16} />
          {t('new_connection_action')}
        </Button>
      </div>
    </section>
  );
}

function PresetRow({
  tunnel,
  selected,
  mode,
  onSelect,
  onEdit,
  onDelete,
}: {
  tunnel: Tunnel;
  selected: boolean;
  mode: 'select' | 'start';
  onSelect: (id: string) => void;
  onEdit: (id: string) => void;
  onDelete: (tunnel: Tunnel) => void;
}) {
  const [busy, setBusy] = useState(false);
  const failed = tunnel.state === 'error' || tunnel.state === 'timeout';

  async function start() {
    unlockSound();
    setBusy(true);
    await attempt(() => startTunnel(tunnel.id));
    setBusy(false);
  }

  const body = (
    <span class="min-w-0 flex-1">
      <span class="block truncate text-sm font-medium text-text">{tunnel.name}</span>
      <span class="block truncate font-mono text-xs text-text-muted">
        {providerLabel(tunnel.provider)} · :{tunnel.port}
        {failed && <span class="text-text-error"> · {t(tunnel.state)}</span>}
      </span>
    </span>
  );

  return (
    <li
      class={`flex min-h-14 items-center gap-1 border-b border-border-light pe-1 ${
        selected ? 'bg-hover' : ''
      }`}
    >
      {mode === 'select' ? (
        <button
          type="button"
          class="flex min-w-0 flex-1 cursor-pointer items-center gap-3 self-stretch border-0 bg-transparent py-2 ps-3 text-start"
          aria-pressed={selected}
          onClick={() => onSelect(tunnel.id)}
        >
          <span
            class={`size-2 shrink-0 rounded-full ${selected ? 'bg-primary' : 'bg-border'}`}
            aria-hidden="true"
          />
          {body}
        </button>
      ) : (
        <div class="flex min-w-0 flex-1 items-center gap-3 py-2 ps-3">{body}</div>
      )}

      <IconAction
        label={t('edit')}
        size="sm"
        icon={<PencilSimple size={16} />}
        onClick={() => onEdit(tunnel.id)}
      />
      <IconAction
        label={t('delete')}
        size="sm"
        icon={<Trash size={16} />}
        onClick={() => onDelete(tunnel)}
      />
      {mode === 'start' && (
        <IconAction
          label={t('start')}
          size="sm"
          loading={busy}
          icon={<Play size={16} />}
          onClick={start}
        />
      )}
    </li>
  );
}
