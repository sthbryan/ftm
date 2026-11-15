import { useState } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { ArrowClockwise, PencilSimple } from '@phosphor-icons/react';
import { unlockSound } from '@/lib/sound';
import { t } from '@/lib/stores/i18n';
import { startTunnel } from '@/lib/stores/tunnels';
import { providerErrorHint } from '@/lib/utils/providerError';
import type { Tunnel } from '@/lib/types';
import { attempt } from './shared';

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
    <section class="flex flex-col items-center py-10 text-center" aria-labelledby="failed-title">
      <p class="m-0 max-w-full truncate text-[13px] text-text-muted">{tunnel.name}</p>
      <h1
        id="failed-title"
        class="m-0 mt-1 font-serif text-[clamp(26px,7vw,36px)] leading-tight font-medium text-text"
      >
        {hint ? t(hint) : t('hero_failed')}
      </h1>
      {tunnel.errorMessage && (
        <p class="m-0 mt-3 max-w-[60ch] font-mono text-xs break-words text-text-muted">
          {tunnel.errorMessage}
        </p>
      )}
      <div class="mt-7 flex flex-wrap justify-center gap-2">
        <Button variant="primary" size="lg" loading={busy} onClick={retry}>
          <ArrowClockwise size={16} />
          {t('retry')}
        </Button>
        <Button variant="secondary" size="lg" onClick={() => onEdit(tunnel.id)}>
          <PencilSimple size={16} />
          {t('edit')}
        </Button>
      </div>
    </section>
  );
}
