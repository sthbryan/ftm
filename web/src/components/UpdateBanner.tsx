import { Button } from '@zovaris/sephiro/button';
import { Card } from '@zovaris/sephiro/card';
import { Link } from '@zovaris/sephiro/link';
import { Notice } from '@zovaris/sephiro/notice';
import { ArrowCircleUp } from '@phosphor-icons/react';
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
        <Card className="ftm-banner">
          <div class="flex flex-wrap items-center gap-3">
          <span class="flex min-w-0 items-center gap-2">
            <ArrowCircleUp size={16} className="shrink-0 text-primary" />
            <span class="truncate text-xs font-medium text-text">
              {t('update_web_banner', { latest: info.latest })}
            </span>
          </span>

          {hint && <span class="min-w-0 truncate text-2xs text-text-muted">{hint}</span>}

          <div class="ml-auto flex shrink-0 items-center gap-2">
            <Link href={info.releaseUrl} variant="muted" external className="text-2xs">
              {t('update_web_notes')}
            </Link>

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
        </Card>
      )}

      {error && <Notice variant="danger">{t('update_apply_failed', { 0: error })}</Notice>}
    </div>
  );
}
