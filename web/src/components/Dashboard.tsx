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
} from '@/lib/stores/tunnels';
import { createIntent, selectIntent } from '@/lib/stores/ui';
import { arrange } from '@/lib/ritual';
import { invited, selectedPreset, watchSession } from '@/lib/stores/ritual';
import { ReadyMoment } from './ritual/ReadyMoment';
import { OpeningMoment } from './ritual/OpeningMoment';
import { InviteMoment } from './ritual/InviteMoment';
import { FailedMoment } from './ritual/FailedMoment';
import { SessionStrip } from './ritual/SessionStrip';
import { PresetList } from './ritual/PresetList';
import { UpdateBanner } from './UpdateBanner';
import { NotificationPrompt } from './NotificationPrompt';
import { ConnectionDialog } from './ConnectionDialog';
import { ConfirmDialog } from './ConfirmDialog';
import { toast } from '@/lib/stores/toast';
import { t } from '@/lib/stores/i18n';
import type { Tunnel } from '@/lib/types';

/**
 * The ritual: one moment on the stage (ready, opening, invite, or what failed),
 * running tables stacked above it as strips, saved tables under it as presets.
 */
export function Dashboard() {
  const [formOpen, setFormOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [pendingDelete, setPendingDelete] = useState<Tunnel | null>(null);

  const list = tunnels.value;
  const view = arrange(list, selectedPreset.value, invited.value);

  useEffect(() => watchSession(), []);

  useEffect(() => {
    if (createIntent.value > 0) openCreate();
  }, [createIntent.value]);

  useEffect(() => {
    const id = selectIntent.value;
    if (!id) return;
    selectedPreset.value = id;
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

  const showPresets =
    list.length > 0 &&
    (view.stage === null || view.stage.moment === 'ready' || view.stage.moment === 'failed');

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
          {view.strips.length > 0 && (
            <div class="flex flex-col gap-2">
              {view.strips.map((tunnel) => (
                <SessionStrip key={tunnel.id} tunnel={tunnel} />
              ))}
            </div>
          )}

          {view.stage?.moment === 'ready' && (
            <ReadyMoment tunnel={view.stage.tunnel} onCreate={openCreate} />
          )}
          {view.stage?.moment === 'opening' && view.stage.tunnel && (
            <OpeningMoment key={view.stage.tunnel.id} tunnel={view.stage.tunnel} />
          )}
          {view.stage?.moment === 'invite' && view.stage.tunnel && (
            <InviteMoment key={view.stage.tunnel.id} tunnel={view.stage.tunnel} />
          )}
          {view.stage?.moment === 'failed' && view.stage.tunnel && (
            <FailedMoment key={view.stage.tunnel.id} tunnel={view.stage.tunnel} onEdit={openEdit} />
          )}

          {showPresets && (
            <PresetList
              presets={view.presets}
              selectedId={view.stage?.moment === 'ready' ? (view.stage.tunnel?.id ?? null) : null}
              mode={view.stage?.moment === 'ready' ? 'select' : 'start'}
              onSelect={(id) => {
                selectedPreset.value = id;
              }}
              onCreate={openCreate}
              onEdit={openEdit}
              onDelete={setPendingDelete}
            />
          )}
        </>
      )}

      {dialogs}
    </div>
  );
}
