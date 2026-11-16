import { useEffect } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { EmptyState } from '@zovaris/sephiro/empty-state';
import { Reveal } from '@zovaris/sephiro/reveal';
import { Spinner } from '@zovaris/sephiro/spinner';
import { ArrowsClockwise, Plugs } from '@phosphor-icons/react';
import { retryTunnels, tunnels, tunnelsError, tunnelsLoading } from '@/lib/stores/tunnels';
import { selectIntent } from '@/lib/stores/ui';
import { arrange } from '@/lib/ritual';
import { invited, selectedPreset, watchSession } from '@/lib/stores/ritual';
import { t } from '@/lib/stores/i18n';
import { connectionPath, navigate } from '@/router';
import { ReadyMoment } from './ritual/ReadyMoment';
import { OpeningMoment } from './ritual/OpeningMoment';
import { InviteMoment } from './ritual/InviteMoment';
import { FailedMoment } from './ritual/FailedMoment';
import { SessionStrip } from './ritual/SessionStrip';
import { PresetChips } from './ritual/PresetChips';
import { NotificationNudge } from './NotificationNudge';
import { UpdateBanner } from './UpdateBanner';

function openCreate() {
  navigate(connectionPath());
}

function openEdit(id: string) {
  navigate(connectionPath(id));
}

function selectPreset(id: string) {
  selectedPreset.value = id;
}

/**
 * The ritual: one moment on the stage (ready, opening, invite, or what failed),
 * running tables as slim strips above it, saved tables as preset chips.
 */
export function Dashboard() {
  const list = tunnels.value;
  const view = arrange(list, selectedPreset.value, invited.value);
  const stage = view.stage;

  useEffect(() => watchSession(), []);

  useEffect(() => {
    const id = selectIntent.value;
    if (!id) return;
    selectedPreset.value = id;
    selectIntent.value = null;
  }, [selectIntent.value]);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.metaKey || event.ctrlKey || event.altKey || event.defaultPrevented) return;
      const target = event.target as HTMLElement | null;
      if (target?.closest('input, textarea, select, [contenteditable], [role="dialog"]')) return;
      if (event.key === 'n') {
        event.preventDefault();
        openCreate();
      }
    };

    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  if (tunnelsLoading.value && list.length === 0) {
    return (
      <div class="flex justify-center py-24">
        <Spinner size="md" label={t('loading')} />
      </div>
    );
  }

  if (tunnelsError.value && list.length === 0) {
    return (
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
    );
  }

  const choosing = stage?.moment === 'ready' || stage?.moment === 'failed';
  const selectedId = stage?.moment === 'ready' ? (stage.tunnel?.id ?? null) : null;
  const chips = view.presets.length > 0 && (
    <PresetChips
      presets={view.presets}
      selectedId={selectedId}
      mode={stage?.moment === 'ready' ? 'select' : 'start'}
      onSelect={selectPreset}
      onCreate={openCreate}
    />
  );

  return (
    <div class="flex flex-1 flex-col gap-4">
      <UpdateBanner />

      {view.strips.length > 0 && (
        <div class="flex flex-col gap-3">
          {view.strips.map((tunnel) => (
            <SessionStrip key={tunnel.id} tunnel={tunnel} />
          ))}
        </div>
      )}

      {stage ? (
        <Reveal
          key={`${stage.moment}:${stage.tunnel?.id ?? ''}`}
          className="ftm-stage"
          distance={6}
          speed="normal"
        >
          {stage.moment === 'ready' && (
            <ReadyMoment tunnel={stage.tunnel} onCreate={openCreate} onEdit={openEdit} />
          )}
          {stage.moment === 'opening' && stage.tunnel && <OpeningMoment tunnel={stage.tunnel} />}
          {stage.moment === 'invite' && stage.tunnel && <InviteMoment tunnel={stage.tunnel} />}
          {stage.moment === 'failed' && stage.tunnel && (
            <FailedMoment tunnel={stage.tunnel} onEdit={openEdit} />
          )}
          {choosing && chips && <div class="mt-8">{chips}</div>}
        </Reveal>
      ) : (
        <footer class="ftm-session-footer">
          <NotificationNudge fallback={t('session_watch_note')} />
          {view.presets.length > 0 ? (
            <PresetChips
              presets={view.presets}
              selectedId={null}
              mode="start"
              align="start"
              onSelect={selectPreset}
              onCreate={openCreate}
            />
          ) : (
            <Button variant="quiet" size="sm" onClick={openCreate}>
              {t('new_connection_action')}
            </Button>
          )}
        </footer>
      )}
    </div>
  );
}
