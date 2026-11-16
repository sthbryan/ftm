import { useState } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { ArrowClockwise, PencilSimple } from '@phosphor-icons/react';
import { unlockSound } from '@/lib/sound';
import { t } from '@/lib/stores/i18n';
import { startTunnel } from '@/lib/stores/tunnels';
import { providerErrorHint } from '@/lib/utils/providerError';
import type { Tunnel } from '@/lib/types';
import { attempt } from './shared';
import { AsciiDie } from './AsciiDie';

/** When opening fails: what went wrong in plain words, the raw error, and a retry. */
export function FailedMoment({ tunnel, onEdit }: { tunnel: Tunnel; onEdit: (id: string) => void }) {
  const [busy, setBusy] = useState(false);
  const hint = providerErrorHint(tunnel.errorMessage);

  async function retry() {
    unlockSound();
    setBusy(true);
    await attempt(() => startTunnel(tunnel.id));
    setBusy(false);
  }

  return (
    <section class="ftm-moment" aria-labelledby="failed-title">
      <AsciiDie mode="one" />
      <div class="flex max-w-full flex-col items-center">
        <h1 id="failed-title" class="ftm-title">
          {tunnel.name}
        </h1>
        <p class="m-0 mt-2 flex items-center gap-2 text-sm text-text" role="status">
          <span class="ftm-dot" data-check="failed" aria-hidden="true" />
          {hint ? t(hint) : t('hero_failed')}
        </p>
        {tunnel.errorMessage && (
          <p class="m-0 mt-2 max-w-[60ch] font-mono text-2xs break-words text-text-muted">
            {tunnel.errorMessage}
          </p>
        )}
      </div>
      <div class="flex flex-wrap justify-center gap-2">
        <Button variant="primary" size="lg" className="ftm-hero-action" loading={busy} onClick={retry}>
          <ArrowClockwise size={20} />
          {t('retry')}
        </Button>
      </div>
      <Button variant="quiet" size="sm" onClick={() => onEdit(tunnel.id)}>
        <PencilSimple size={16} />
        {t('edit_connection')}
      </Button>
    </section>
  );
}
