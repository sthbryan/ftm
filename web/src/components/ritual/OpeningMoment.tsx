import { useState } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { Progress } from '@zovaris/sephiro/progress';
import { X } from '@phosphor-icons/react';
import { t } from '@/lib/stores/i18n';
import { providerLabel } from '@/lib/stores/providers';
import { installProgress, stopTunnel } from '@/lib/stores/tunnels';
import { isInstallingState } from '@/lib/utils/status';
import type { Tunnel } from '@/lib/types';
import { attempt } from './shared';

/**
 * Moment two. The provider binary downloading (first run only) and the tunnel
 * connecting, with real progress from the server and a way out.
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
    <section class="flex flex-col items-center py-12 text-center" aria-labelledby="opening-title">
      <h1
        id="opening-title"
        class="m-0 max-w-full font-serif text-[clamp(26px,7vw,36px)] leading-tight font-medium break-words text-text"
      >
        {t('opening_title', { name: tunnel.name })}
      </h1>

      <div class="mt-8 w-full max-w-md text-start" aria-live="polite">
        <Progress value={percent} label={t(tunnel.state)} />
        <div class="mt-3 flex items-baseline gap-3 text-sm text-text-muted">
          <p class="m-0 min-w-0 flex-1 truncate">
            {installing
              ? progress?.step || t('hero_installing', { 0: provider })
              : t('hero_connecting', { 0: provider })}
          </p>
          {percent !== undefined && (
            <span class="shrink-0 font-mono text-xs tabular-nums">{percent}%</span>
          )}
        </div>
      </div>

      <Button variant="quiet" className="mt-6" loading={busy} onClick={cancel}>
        <X size={16} />
        {t('cancel')}
      </Button>
    </section>
  );
}
