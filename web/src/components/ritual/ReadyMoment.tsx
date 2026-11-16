import { useEffect, useState } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { IconAction } from '@/components/IconAction';
import { ArrowClockwise, DoorOpen, PencilSimple, Plus } from '@phosphor-icons/react';
import { foundryCheck } from '@/lib/ritual';
import { unlockSound } from '@/lib/sound';
import { busyPorts, checkFoundry, checkingPorts } from '@/lib/stores/foundry';
import { t } from '@/lib/stores/i18n';
import { providerLabel } from '@/lib/stores/providers';
import { startTunnel } from '@/lib/stores/tunnels';
import type { Tunnel } from '@/lib/types';
import { Mark } from '../Mark';
import { attempt } from './shared';

/**
 * Moment one. The world about to be opened, whether Foundry answers on its
 * port, and one large action. The presets sit under it.
 */
export function ReadyMoment({
  tunnel,
  onCreate,
  onEdit,
}: {
  tunnel: Tunnel | null;
  onCreate: () => void;
  onEdit: (id: string) => void;
}) {
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    void checkFoundry();
  }, [tunnel?.id]);

  if (!tunnel) {
    return (
      <section class="ftm-moment" aria-labelledby="ready-title">
        <Mark class="size-12" />
        <div>
          <h1 id="ready-title" class="ftm-title">
            {t('ready_empty_title')}
          </h1>
          <p class="m-0 mt-2 max-w-[44ch] text-sm text-text-muted">{t('ready_empty_body')}</p>
        </div>
        <Button variant="primary" size="lg" className="ftm-hero-action" onClick={onCreate}>
          <Plus size={20} weight="bold" />
          {t('new_connection_action')}
        </Button>
      </section>
    );
  }

  const check = foundryCheck(tunnel.port, busyPorts.value);

  async function open() {
    if (!tunnel) return;
    unlockSound();
    setBusy(true);
    await attempt(() => startTunnel(tunnel.id));
    setBusy(false);
  }

  return (
    <section class="ftm-moment" aria-labelledby="ready-title">
      <div class="flex max-w-full flex-col items-center">
        <h1 id="ready-title" class="ftm-title">
          {tunnel.name}
        </h1>
        <p class="m-0 mt-2 flex flex-wrap items-center justify-center gap-x-2 gap-y-1 text-sm text-text-muted">
          <span class="ftm-dot" data-check={check} aria-hidden="true" />
          <span class={check === 'answers' ? 'text-text' : ''} aria-live="polite">
            {t(`foundry_${check}`, { port: tunnel.port })}
          </span>
          {check !== 'answers' && (
            <IconAction
              label={t('foundry_recheck')}
              size="sm"
              loading={checkingPorts.value}
              icon={<ArrowClockwise size={14} />}
              onClick={() => void checkFoundry()}
            />
          )}
          <span class="inline-flex items-center gap-2 whitespace-nowrap">
            <span aria-hidden="true">·</span>
            <span>{providerLabel(tunnel.provider)}</span>
            <IconAction
              label={t('edit_connection')}
              size="sm"
              icon={<PencilSimple size={14} />}
              onClick={() => onEdit(tunnel.id)}
            />
          </span>
        </p>
      </div>

      <Button
        variant="primary"
        size="lg"
        className="ftm-hero-action"
        loading={busy}
        onClick={open}
      >
        <DoorOpen size={20} />
        {t('open_table')}
      </Button>

    </section>
  );
}
