import { EmptyState } from '@zovaris/sephiro/empty-state';
import { Button } from '@zovaris/sephiro/button';
import { Kbd } from '@zovaris/sephiro/kbd';
import { Plus } from '@phosphor-icons/react';
import { t } from '@/lib/stores/i18n';
import { Mark } from './Mark';

const STEPS: [string, string][] = [
  ['welcome_step_provider_title', 'welcome_step_provider_body'],
  ['welcome_step_port_title', 'welcome_step_port_body'],
  ['welcome_step_share_title', 'welcome_step_share_body'],
];

export function Welcome({ onCreate }: { onCreate: () => void }) {
  return (
    <div class="flex h-full items-center justify-center">
      <div class="sph-card w-full max-w-[34rem] px-6 py-7">
        <EmptyState
          icon={<Mark class="size-7" />}
          title={t('welcome_title')}
          description={t('welcome_lead')}
          action={
            <div class="flex flex-col items-center gap-2">
              <Button variant="primary" size="lg" onClick={onCreate}>
                <Plus size={16} weight="bold" />
                {t('create_first')}
              </Button>
              <span class="flex items-center gap-1.5 text-2xs text-text-muted">
                <Kbd keys="N" /> {t('list_hint_new')}
              </span>
            </div>
          }
        />

        <ol class="mx-auto mt-5 grid w-full max-w-[26rem] list-none gap-3 p-0 text-start">
          {STEPS.map(([title, body], index) => (
            <li key={title} class="flex items-start gap-2.5">
              <span class="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full bg-primary font-mono text-2xs font-semibold text-btn-text">
                {index + 1}
              </span>
              <div class="min-w-0">
                <p class="m-0 text-xs font-medium text-text">{t(title)}</p>
                <p class="m-0 text-2xs leading-relaxed text-text-muted">{t(body)}</p>
              </div>
            </li>
          ))}
        </ol>
      </div>
    </div>
  );
}
