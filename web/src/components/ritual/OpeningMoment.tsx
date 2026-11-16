import { useState } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { Progress } from '@zovaris/sephiro/progress';
import { t } from '@/lib/stores/i18n';
import { providerLabel } from '@/lib/stores/providers';
import { installProgress, stopTunnel } from '@/lib/stores/tunnels';
import { isInstallingState } from '@/lib/utils/status';
import type { Tunnel } from '@/lib/types';
import { attempt } from './shared';
import { AsciiDie } from './AsciiDie';
import { ProviderIcon } from './ProviderIcon';
import { Ticket } from './Ticket';

/**
 * Moment two. The provider binary downloading (first run only) and the tunnel
 * connecting, with real progress from the server and a way out. The ticket's
 * stub rolls a d20 drawn in ASCII meanwhile.
 */
export function OpeningMoment({ tunnel }: { tunnel: Tunnel }) {
  const [busy, setBusy] = useState(false);
  const installing = isInstallingState(tunnel.state);
  const progress = installProgress.value[tunnel.provider];
  const provider = providerLabel(tunnel.provider);
  const percent = installing && progress ? Math.round(progress.percent) : undefined;

  async function cancel() {
    setBusy(true);
    await attempt(() => stopTunnel(tunnel.id));
    setBusy(false);
  }

  return (
    <section class="ftm-moment" aria-labelledby="opening-title" aria-busy="true">
      <Ticket label={tunnel.name} state="opening" stub={<AsciiDie mode="rolling" />}>
        <p class="ftm-ticket__eyebrow">
          <ProviderIcon provider={tunnel.provider} />
          {provider}
        </p>
        <h1 id="opening-title" class="ftm-ticket__world">
          {tunnel.name}
        </h1>
        <div class="w-full max-w-96" aria-live="polite">
          <Progress value={percent} label={t(tunnel.state)} size="sm" />
          <div class="mt-2.5 flex items-baseline gap-3 text-xs text-text-muted">
            <p class="m-0 min-w-0 flex-1 truncate">
              {installing
                ? progress?.step || t('hero_installing', { 0: provider })
                : t('hero_connecting', { 0: provider })}
            </p>
            {percent !== undefined && (
              <span class="shrink-0 font-mono text-2xs tabular-nums">{percent}%</span>
            )}
          </div>
        </div>
        <div class="ftm-ticket__actions">
          <Button variant="quiet" loading={busy} onClick={cancel}>
            {t('cancel')}
          </Button>
        </div>
      </Ticket>
    </section>
  );
}
