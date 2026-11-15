import { Button } from '@zovaris/sephiro/button';
import { ArrowCircleUp, ArrowSquareOut } from '@phosphor-icons/react';
import { applyUpdate, updateApplying, updateError, updateInfo } from '@/lib/stores/update';
import { t } from '@/lib/stores/i18n';

export function UpdateBanner() {
  const info = updateInfo.value;
  const applying = updateApplying.value;
  const error = updateError.value;

  if (!info?.hasUpdate && !error) return null;

  const selfUpdatable = info?.method === 'self';

  const hint =
    info?.method === 'store'
      ? t('update_manual_store')
      : info?.method === 'homebrew'
        ? t('update_manual_homebrew')
        : info?.method === 'download'
          ? t('update_manual_download')
          : '';

  return (
    <div class="flex shrink-0 flex-col gap-2">
      {info?.hasUpdate && (
        <div class="sph-card flex flex-wrap items-center gap-3 px-3 py-2">
          <span class="flex min-w-0 items-center gap-2">
            <ArrowCircleUp size={16} className="shrink-0 text-primary" />
            <span class="truncate text-xs font-medium text-text">
              {t('update_web_banner', { latest: info.latest })}
            </span>
          </span>

          {hint && <span class="min-w-0 truncate text-2xs text-text-muted">{hint}</span>}

          <div class="ml-auto flex shrink-0 items-center gap-2">
            <a
              href={info.releaseUrl}
              target="_blank"
              rel="noreferrer"
              class="flex items-center gap-1 text-2xs text-text-muted underline-offset-2 hover:text-text hover:underline"
            >
              {t('update_web_notes')}
              <ArrowSquareOut size={12} />
            </a>

            {applying ? (
              <span class="text-2xs text-text-muted">{t('update_web_restart')}</span>
            ) : selfUpdatable ? (
              <Button variant="primary" size="sm" onClick={() => void applyUpdate()}>
                {t('update_web_button')}
              </Button>
            ) : (
              <Button
                variant="secondary"
                size="sm"
                onClick={() => window.open(info.releaseUrl, '_blank', 'noreferrer')}
              >
                {t('update_web_download')}
              </Button>
            )}
          </div>
        </div>
      )}

      {error && (
        <div class="sph-card border-status-error/40 px-3 py-2 text-2xs text-text-error">
          {t('update_apply_failed', { 0: error })}
        </div>
      )}
    </div>
  );
}
