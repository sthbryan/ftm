import { useState } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { Input } from '@zovaris/sephiro/input';
import { Field } from '@zovaris/sephiro/field';
import { EmptyState } from '@zovaris/sephiro/empty-state';
import { Plus, Plugs, ArrowsClockwise, MagnifyingGlass } from '@phosphor-icons/react';
import { installProgress, tunnels, tunnelsError, tunnelsLoading, retryTunnels } from '@/lib/stores/tunnels';
import { t } from '@/lib/stores/i18n';
import type { Tunnel } from '@/lib/types';
import { TunnelRow } from './TunnelRow';

const STATE_ORDER: Record<string, number> = {
  error: 0,
  timeout: 0,
  online: 1,
  starting: 2,
  connecting: 2,
  stopping: 2,
  installing: 3,
  downloading: 3,
};

const SEARCH_THRESHOLD = 6;

interface TunnelListProps {
  selectedId: string | null;
  onSelect: (id: string) => void;
  onCreate: () => void;
  onEdit: (id: string) => void;
  onDelete: (tunnel: Tunnel) => void;
}

export function TunnelList({ selectedId, onSelect, onCreate, onEdit, onDelete }: TunnelListProps) {
  const [query, setQuery] = useState('');

  const list = tunnels.value;
  const searchable = list.length >= SEARCH_THRESHOLD;
  const term = query.trim().toLowerCase();

  const visible = (term && searchable
    ? list.filter(
        (tunnel) =>
          tunnel.name.toLowerCase().includes(term) ||
          tunnel.provider.toLowerCase().includes(term) ||
          String(tunnel.port).includes(term),
      )
    : list
  )
    .slice()
    .sort((a, b) => (STATE_ORDER[a.state] ?? 9) - (STATE_ORDER[b.state] ?? 9));

  return (
    <section class="sph-card flex min-h-0 flex-col" aria-label={t('connections')}>
      <div class="flex shrink-0 items-center justify-between gap-2 border-b border-border px-3.5 py-2.5">
        <div class="flex min-w-0 items-center gap-2">
          <h2 class="m-0 truncate text-sm font-semibold text-text">{t('connections')}</h2>
          <span class="sph-badge font-mono tabular-nums" data-size="sm" data-variant="neutral">
            {list.length}
          </span>
        </div>
        <Button variant="primary" size="sm" onClick={onCreate}>
          <Plus size={14} weight="bold" />
          {t('new_connection_action')}
        </Button>
      </div>

      {searchable && (
        <div class="shrink-0 border-b border-border-light px-3 py-2">
          <Field>
            <div class="relative">
              <MagnifyingGlass
                size={14}
                className="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-text-muted"
              />
              <Input
                type="search"
                size="sm"
                className="sph-input--icon"
                value={query}
                placeholder={t('search_placeholder')}
                aria-label={t('search_placeholder')}
                onInput={(event) => setQuery((event.target as HTMLInputElement).value)}
              />
            </div>
          </Field>
        </div>
      )}

      <div class="min-h-0 flex-1 overflow-y-auto p-2">
        {tunnelsLoading.value ? (
          <div class="flex flex-col items-center justify-center gap-3 py-12">
            <span class="sph-spinner sph-spinner--md" />
            <span class="text-2xs text-text-muted">{t('loading')}</span>
          </div>
        ) : tunnelsError.value ? (
          <EmptyState
            compact
            icon={<Plugs size={20} />}
            title={t('panel_unreachable')}
            description={t('panel_unreachable_desc')}
            action={
              <div class="flex flex-col items-center gap-2">
                <p class="m-0 max-w-xs font-mono text-2xs break-words text-text-muted">
                  {tunnelsError.value}
                </p>
                <Button variant="primary" onClick={() => retryTunnels()}>
                  <ArrowsClockwise size={14} />
                  {t('retry')}
                </Button>
              </div>
            }
          />
        ) : list.length === 0 ? (
          <EmptyState
            compact
            icon={<Plugs size={20} />}
            title={t('no_tunnels')}
            action={
              <Button variant="primary" onClick={onCreate}>
                <Plus size={14} weight="bold" />
                {t('new_connection_action')}
              </Button>
            }
          />
        ) : visible.length === 0 ? (
          <p class="m-0 px-3 py-8 text-center text-2xs text-text-muted">{t('search_empty')}</p>
        ) : (
          <div class="flex flex-col gap-1.5">
            {visible.map((tunnel) => (
              <TunnelRow
                key={tunnel.id}
                tunnel={tunnel}
                selected={tunnel.id === selectedId}
                installProgress={installProgress.value[tunnel.provider] ?? null}
                onSelect={onSelect}
                onEdit={onEdit}
                onDelete={onDelete}
              />
            ))}
          </div>
        )}
      </div>

      {list.length > 0 && (
        <p class="m-0 shrink-0 border-t border-border-light px-3 py-1.5 text-center text-2xs text-text-muted">
          {t('list_hint_new')}
        </p>
      )}
    </section>
  );
}
