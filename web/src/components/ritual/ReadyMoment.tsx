import { useEffect, useState } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { IconAction } from '@/components/IconAction';
import { ArrowClockwise, DoorOpen, PencilSimple, Plus } from '@phosphor-icons/react';
import { foundryCheck } from '@/lib/ritual';
import { unlockSound } from '@/lib/sound';
import {
  busyPorts,
  checkFoundry,
  checkingPorts,
  foundryPorts,
  scannedPorts,
  watchPort,
  worldOn,
  worldTitle,
} from '@/lib/stores/foundry';
import { t } from '@/lib/stores/i18n';
import { startTunnel, updateTunnel } from '@/lib/stores/tunnels';
import type { Tunnel } from '@/lib/types';
import { ProviderIcon } from './ProviderIcon';
import { Ticket } from './Ticket';
import { AsciiDie } from './AsciiDie';
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
    watchPort(tunnel?.port);
    return () => watchPort(undefined);
  }, [tunnel?.id, tunnel?.port]);

  if (!tunnel) {
    return (
      <section class="ftm-moment" aria-labelledby="ready-title">
        <AsciiDie mode="idle" />
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

  const check = foundryCheck(tunnel.port, busyPorts.value, scannedPorts.value, foundryPorts.value);
  // Foundry is up, just not where this connection looks for it.
  const elsewhere =
    check !== 'answers' ? foundryPorts.value?.find((port) => port !== tunnel.port) : undefined;

  async function usePort(port: number) {
    if (!tunnel) return;
    await attempt(() => updateTunnel(tunnel.id, { localPort: port }));
  }

  async function open() {
    if (!tunnel) return;
    unlockSound();
    setBusy(true);
    await attempt(() => startTunnel(tunnel.id));
    setBusy(false);
  }

  // What Foundry says about itself on this port: the world it runs, or that
  // it sits on the setup screen with no world launched (players can't join).
  const status = check === 'answers' ? worldOn(tunnel.port) : undefined;
  const setupScreen = status?.version !== undefined && !status.active;
  const foundryLine = elsewhere
    ? t('foundry_elsewhere', { port: elsewhere, expected: tunnel.port })
    : status?.active && status.world
      ? t('foundry_world', { world: worldTitle(status.world), port: tunnel.port })
      : setupScreen
        ? t('foundry_setup', { port: tunnel.port })
        : t(`foundry_${check}`, { port: tunnel.port });

  return (
    <section class="ftm-moment" aria-labelledby="ready-title">
      <Ticket
        label={tunnel.name}
        state="ready"
        stub={
          <>
            <span class="ftm-ticket__label">{t('port_label')}</span>
            <span class="ftm-ticket__seat">{tunnel.port}</span>
            <span class="ftm-ticket__label inline-flex items-center gap-1.5">
              <span class="ftm-dot" data-check={setupScreen ? 'silent' : check} aria-hidden="true" />
              Foundry
            </span>
          </>
        }
      >
        <p class="ftm-ticket__eyebrow">
          <ProviderIcon provider={tunnel.provider} />
          {t(`provider_note_${tunnel.provider}`)}
        </p>
        <h1 id="ready-title" class="ftm-ticket__world">
          {tunnel.name}
        </h1>
        <p class="ftm-ticket__facts" aria-live="polite">
          <span class="ftm-dot" data-check={setupScreen ? 'silent' : check} aria-hidden="true" />
          <span class={check === 'answers' && !setupScreen ? 'text-text' : ''}>{foundryLine}</span>
          {elsewhere !== undefined && (
            <Button variant="secondary" size="sm" onClick={() => void usePort(elsewhere)}>
              {t('foundry_use_port', { port: elsewhere })}
            </Button>
          )}
          {check !== 'answers' && elsewhere === undefined && (
            <IconAction
              label={t('foundry_recheck')}
              size="sm"
              loading={checkingPorts.value}
              icon={<ArrowClockwise size={14} />}
              onClick={() => void checkFoundry(tunnel.port)}
            />
          )}
        </p>
        <div class="ftm-ticket__actions">
          <Button variant="primary" size="lg" loading={busy} onClick={open}>
            <DoorOpen size={18} />
            {t('open_table')}
          </Button>
          <Button variant="quiet" size="lg" onClick={() => onEdit(tunnel.id)}>
            <PencilSimple size={18} />
            {t('edit_connection')}
          </Button>
        </div>
      </Ticket>
    </section>
  );
}
