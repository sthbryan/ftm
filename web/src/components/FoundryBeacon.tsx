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
    <Tooltip content={tip} side="bottom">
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
