import { useEffect, useState } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { EmptyState } from '@zovaris/sephiro/empty-state';
import { ArrowsClockwise, Plugs } from '@phosphor-icons/react';
import {
  deleteTunnel,
  retryTunnels,
  tunnels,
  tunnelsError,
  tunnelsLoading,
  tunnelsOnboarded,
} from '@/lib/stores/tunnels';
import { createIntent, selectIntent } from '@/lib/stores/ui';
import { ConnectionHero } from './ConnectionHero';
import { ConnectionList } from './ConnectionList';
import { Welcome } from './Welcome';
import { UpdateBanner } from './UpdateBanner';
import { NotificationPrompt } from './NotificationPrompt';
import { ConnectionDialog } from './ConnectionDialog';
import { ConfirmDialog } from './ConfirmDialog';
import { toast } from '@/lib/stores/toast';
import { t } from '@/lib/stores/i18n';
import type { Tunnel } from '@/lib/types';

/** Which connection the hero shows when the user has not picked one. */
const FOCUS_ORDER: Record<string, number> = {
  error: 0,
  timeout: 0,
  online: 1,
  connecting: 2,
  starting: 2,
  installing: 2,
  downloading: 2,
  stopping: 3,
};

export function Dashboard() {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [formOpen, setFormOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [pendingDelete, setPendingDelete] = useState<Tunnel | null>(null);

  const list = tunnels.value;
  const focused =
    list.find((tunnel) => tunnel.id === selectedId) ??
    list
      .slice()
      .sort((a, b) => (FOCUS_ORDER[a.state] ?? 9) - (FOCUS_ORDER[b.state] ?? 9))[0] ??
    null;
  const others = list.filter((tunnel) => tunnel.id !== focused?.id);
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
      if (event.metaKey || event.ctrlKey || event.altKey) return;
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

  function openEdit(id: string) {
    setEditingId(id);
    setFormOpen(true);
  }

  function select(id: string) {
    setSelectedId(id);
    window.scrollTo({ top: 0, behavior: 'smooth' });
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

  const dialogs = (
    <>
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
    </>
  );

  if (showWelcome) {
    return (
      <>
        <Welcome onCreate={openCreate} />
        {dialogs}
      </>
    );
  }

  return (
    <div class="flex flex-col">
      <UpdateBanner />

      {tunnelsLoading.value && list.length === 0 ? (
        <div class="flex justify-center py-24" aria-busy="true">
          <span class="sph-spinner sph-spinner--md" />
        </div>
      ) : tunnelsError.value && list.length === 0 ? (
        <EmptyState
          icon={<Plugs size={20} />}
          title={t('panel_unreachable')}
          description={t('panel_unreachable_desc')}
          action={
            <Button variant="primary" onClick={() => retryTunnels()}>
              <ArrowsClockwise size={16} />
              {t('retry')}
            </Button>
          }
        />
      ) : (
        <>
          {focused && (
            <ConnectionHero
              key={focused.id}
              tunnel={focused}
              onEdit={openEdit}
              onDelete={setPendingDelete}
            />
          )}
          <ConnectionList tunnels={others} onSelect={select} onCreate={openCreate} />
        </>
      )}

      {dialogs}
    </div>
  );
}
