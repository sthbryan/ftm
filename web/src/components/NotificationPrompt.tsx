import { Button } from '@zovaris/sephiro/button';
import { notificationsEnabled, notificationStatus, rejectNotifications, requestNotificationPermission } from '@/lib/stores/notification';
import { t } from '@/lib/stores/i18n';

export function NotificationPrompt() {
  if (notificationStatus.value !== 'pending' || notificationsEnabled.value) return null;

  return (
    <aside class="pointer-events-auto fixed right-3 bottom-3 z-50 w-[19rem] max-w-[calc(100vw-1.5rem)] rounded-[var(--sph-radius-panel)] border border-border bg-card p-3 shadow-lg">
      <h3 class="m-0 text-xs font-semibold text-text">{t('enable_notifications_web')}</h3>
      <p class="m-0 mt-1 text-2xs leading-relaxed text-text-muted">{t('notifications_prompt')}</p>
      <div class="mt-2 flex gap-2">
        <Button
          variant="primary"
          size="sm"
          className="flex-1"
          onClick={() => void requestNotificationPermission()}
        >
          {t('enable')}
        </Button>
        <Button variant="quiet" size="sm" onClick={() => rejectNotifications()}>
          {t('not_now')}
        </Button>
      </div>
    </aside>
  );
}
