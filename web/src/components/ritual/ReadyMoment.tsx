import { useEffect, useState } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { IconButton } from '@zovaris/sephiro/icon-button';
import { ArrowClockwise, CheckCircle, Play, Plus, Question, WarningCircle } from '@phosphor-icons/react';
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
 * port, and a single action. Saved connections sit under it as presets.
 */
export function ReadyMoment({ tunnel, onCreate }: { tunnel: Tunnel | null; onCreate: () => void }) {
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    void checkFoundry();
  }, [tunnel?.id]);

  if (!tunnel) {
    return (
      <section class="flex flex-col items-center py-12 text-center" aria-labelledby="ready-title">
        <Mark class="mb-5 size-12" />
        <h1 id="ready-title" class="m-0 font-serif text-[32px] leading-tight font-medium text-text">
          {t('ready_empty_title')}
        </h1>
        <p class="m-0 mt-2 max-w-[44ch] text-sm text-text-muted">{t('ready_empty_body')}</p>
        <Button variant="primary" size="lg" className="mt-7" onClick={onCreate}>
          <Plus size={16} weight="bold" />
          {t('new_connection_action')}
        </Button>
      </section>
    );
  }

  const check = foundryCheck(tunnel.port, busyPorts.value);
  const port = { port: tunnel.port };

  async function open() {
    if (!tunnel) return;
    unlockSound();
    setBusy(true);
    await attempt(() => startTunnel(tunnel.id));
    setBusy(false);
  }

  return (
    <section class="flex flex-col items-center py-10 text-center" aria-labelledby="ready-title">
      <p class="m-0 text-[13px] text-text-muted">{t('ritual_ready')}</p>
      <h1
        id="ready-title"
        class="m-0 mt-1 max-w-full font-serif text-[clamp(32px,9vw,52px)] leading-[1.05] font-medium break-words text-text"
      >
        {tunnel.name}
      </h1>
      <p class="m-0 mt-2 font-mono text-xs text-text-muted">
        {providerLabel(tunnel.provider)} · localhost:{tunnel.port}
      </p>

      <div class="mt-5 flex items-center gap-1.5 text-sm" aria-live="polite">
        {check === 'answers' && (
          <CheckCircle size={18} weight="fill" className="shrink-0 text-status-running" />
        )}
        {check === 'silent' && (
          <WarningCircle size={18} weight="fill" className="shrink-0 text-status-starting" />
        )}
        {check === 'unknown' && <Question size={18} className="shrink-0 text-text-muted" />}
        <span class={check === 'answers' ? 'text-text' : 'text-text-muted'}>
          {t(`foundry_${check}`, port)}
        </span>
        <IconButton
          label={t('foundry_recheck')}
          size="sm"
          variant="ghost"
          loading={checkingPorts.value}
          icon={<ArrowClockwise size={14} />}
          onClick={() => void checkFoundry()}
        />
      </div>

      <Button variant="primary" size="lg" className="mt-7 min-w-[14rem]" loading={busy} onClick={open}>
        <Play size={16} weight="fill" />
        {t('open_table')}
      </Button>
    </section>
  );
}
