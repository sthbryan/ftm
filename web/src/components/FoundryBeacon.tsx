import { useEffect } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { Tooltip } from '@zovaris/sephiro/tooltip';
import { Users } from '@phosphor-icons/react';
import {
  checkFoundry,
  foundryPorts,
  watchFoundry,
  worldOn,
  worldTitle,
} from '@/lib/stores/foundry';
import { t } from '@/lib/stores/i18n';
import { formatDuration } from '@/lib/utils/duration';
import type { FoundryWorld } from '@/lib/stores/foundry';

/**
 * A quiet line in the header that says whether Foundry is running on this
 * computer, kept fresh every few seconds. When Foundry appears the dot lands
 * with one ring; nothing pulses while it simply stays up. Pressing it checks
 * at once.
 */
export function FoundryBeacon() {
  useEffect(() => watchFoundry(), []);

  const found = foundryPorts.value;
  if (found === null) return null;

  const port = found[0];
  const live = port !== undefined;
  const status = live ? worldOn(port) : undefined;
  const world = status?.active && status.world ? worldTitle(status.world) : '';

  let label = t('beacon_missing');
  let tip = t('beacon_missing_tip');
  if (live && world) {
    label = t('beacon_world', { world });
    tip = t('beacon_world_tip', {
      world,
      system: status?.system ?? '',
      version: status?.version ?? '',
      users: status?.users ?? 0,
      port,
    });
  } else if (live && status?.version) {
    label = t('beacon_setup');
    tip = t('beacon_setup_tip', { version: status.version, port });
  } else if (live) {
    label = t('beacon_found', { port });
    tip = t('beacon_found_tip', { port });
  }

  return (
    <Tooltip content={status && (world || status.version) ? <BeaconTip status={status} world={world} /> : tip} side="bottom">
      <Button
        variant="quiet"
        size="sm"
        className="ftm-beacon"
        data-live={live ? '' : undefined}
        data-setup={live && !world && status?.version ? '' : undefined}
        aria-label={tip}
        onClick={() => void checkFoundry()}
      >
        <span key={live ? `on-${port}` : 'off'} class="ftm-beacon__dot" aria-hidden="true" />
        <span key={label} class="ftm-beacon__label" aria-live="polite">
          {label}
        </span>
        {world && status && (
          <span class="ftm-beacon__users">
            <Users size={13} aria-hidden="true" />
            {status.users}
          </span>
        )}
      </Button>
    </Tooltip>
  );
}

/**
 * What Foundry says about itself, as a small card: the world and its system,
 * then port, who is connected and how long the world has been open, and how
 * to check again. The button's aria-label carries the same in one sentence.
 */
function BeaconTip({ status, world }: { status: FoundryWorld; world: string }) {
  const players = Math.max(0, status.users - 1);
  return (
    <span class="ftm-beacon-tip">
      <span class="ftm-beacon-tip__world">{world || t('beacon_setup')}</span>
      <span class="ftm-beacon-tip__meta">
        {[status.system, status.version && `Foundry ${status.version}`].filter(Boolean).join(' · ')}
      </span>
      <span class="ftm-beacon-tip__rows">
        <span>{t('beacon_tip_port')}</span>
        <span>{status.port}</span>
        {world && (
          <>
            <span>{t('beacon_tip_connected')}</span>
            <span>
              {players === 0
                ? t('beacon_tip_only_you')
                : t('beacon_tip_you_and', { count: players })}
            </span>
            {status.uptimeMs !== undefined && status.uptimeMs > 0 && (
              <>
                <span>{t('beacon_tip_open_for')}</span>
                <span>{formatDuration(status.uptimeMs)}</span>
              </>
            )}
          </>
        )}
      </span>
      <span class="ftm-beacon-tip__hint">{t('beacon_tip_recheck')}</span>
    </span>
  );
}
