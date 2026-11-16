import { Button } from '@zovaris/sephiro/button';
import { Bell, X } from '@phosphor-icons/react';
import { IconAction } from '@/components/IconAction';
import {
  notificationsEnabled,
  notificationStatus,
  rejectNotifications,
  requestNotificationPermission,
} from '@/lib/stores/notification';
import { t } from '@/lib/stores/i18n';

/**
 * Asks once for notifications, inline where they matter: one quiet row next to
 * an open table, not a card floating over the main action. Settings keeps the
 * switch for later.
 */
export function NotificationNudge({ fallback }: { fallback?: string }) {
  const pending = notificationStatus.value === 'pending' && !notificationsEnabled.value;

  if (!pending) {
    return fallback ? <p class="m-0 text-xs text-text-muted">{fallback}</p> : null;
  }

  return (
    <div class="ftm-nudge" role="group" aria-label={t('enable_notifications_web')}>
      <Bell size={16} className="shrink-0 text-text-muted" aria-hidden="true" />
      <p class="m-0 min-w-0 text-xs text-text-muted">{t('notifications_nudge')}</p>
      <span class="flex shrink-0 items-center">
        <Button variant="quiet" size="sm" className="ftm-nudge-action" onClick={() => void requestNotificationPermission()}>
          {t('enable')}
        </Button>
        <IconAction
          label={t('not_now')}
          size="sm"
          icon={<X size={14} />}
          onClick={() => rejectNotifications()}
        />
      </span>
    </div>
  );
}
