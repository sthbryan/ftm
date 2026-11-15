import { useEffect, useState } from 'preact/hooks';
import {
  deleteTunnel,
  tunnels,
  tunnelsError,
  tunnelsOnboarded,
} from '@/lib/stores/tunnels';
import { createIntent, selectIntent } from '@/lib/stores/ui';
import { StatStrip } from './StatStrip';
import { TunnelList } from './TunnelList';
import { TunnelDetail } from './TunnelDetail';
import { Welcome } from './Welcome';
import { UpdateBanner } from './UpdateBanner';
import { NotificationPrompt } from './NotificationPrompt';
import { ConnectionDialog } from './ConnectionDialog';
import { ConfirmDialog } from './ConfirmDialog';
import { toast } from '@/lib/stores/toast';
import { t } from '@/lib/stores/i18n';
import type { Tunnel } from '@/lib/types';

export function Dashboard() {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [pendingDelete, setPendingDelete] = useState<Tunnel | null>(null);

  const list = tunnels.value;
  const selected = list.find((tunnel) => tunnel.id === selectedId) ?? list[0] ?? null;
  const showWelcome = list.length === 0 && !tunnelsOnboarded.value && !tunnelsError.value;

  useEffect(() => {
    if (selectedId && !list.some((tunnel) => tunnel.id === selectedId)) {
      setSelectedId(null);
    }
  }, [list, selectedId]);

  useEffect(() => {
    if (createIntent.value > 0) openCreate();
  }, [createIntent.value]);

  useEffect(() => {
    const id = selectIntent.value;
    if (!id) return;
    setSelectedId(id);
    selectIntent.value = null;
  }, [selectIntent.value]);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (formOpen || pendingDelete) return;
      const target = event.target as HTMLElement | null;
      if (target && /^(INPUT|TEXTAREA|SELECT)$/.test(target.tagName)) return;

      if (event.key === 'n') {
        event.preventDefault();
        openCreate();
      }
    };

    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [formOpen, pendingDelete]);

  function openCreate() {
    setEditingId(null);
    setFormOpen(true);
  }

  async function confirmDelete() {
    if (!pendingDelete) return;

    const { id, name } = pendingDelete;

    try {
      await deleteTunnel(id);
      toast.success(t('connection_deleted', { name }));
      setPendingDelete(null);
    } catch (error) {
      toast.error(t('connection_delete_failed', { error: (error as Error).message }));
    }
  }

  if (showWelcome) {
    return <Welcome onCreate={openCreate} />;
  }

  return (
    <div class="flex h-full min-h-0 flex-col gap-3">
      <UpdateBanner />
      <StatStrip />

      <div class="grid min-h-0 flex-1 gap-3 overflow-y-auto lg:grid-cols-[minmax(0,1fr)_23rem] lg:overflow-hidden">
        <TunnelList
          selectedId={selected?.id ?? null}
          onSelect={setSelectedId}
          onCreate={openCreate}
          onEdit={(id) => {
            setEditingId(id);
            setFormOpen(true);
          }}
          onDelete={setPendingDelete}
        />

        <TunnelDetail
          tunnel={selected}
          onEdit={(id) => {
            setEditingId(id);
            setFormOpen(true);
          }}
          onDelete={setPendingDelete}
        />
      </div>

      <NotificationPrompt />

      <ConnectionDialog
        open={formOpen}
        tunnelId={editingId}
        onClose={() => {
          setFormOpen(false);
          setEditingId(null);
        }}
      />

      <ConfirmDialog
        open={pendingDelete !== null}
        title={t('confirm_delete_title')}
        body={t('confirm_delete_body', { 0: pendingDelete?.name ?? '' })}
        confirmLabel={t('delete')}
        danger
        onConfirm={confirmDelete}
        onCancel={() => setPendingDelete(null)}
      />
    </div>
  );
}
