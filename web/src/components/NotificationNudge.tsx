import { Button } from '@zovaris/sephiro/button';
import { Bell } from '@phosphor-icons/react';
import {
  notificationsEnabled,
  notificationStatus,
  rejectNotifications,
  requestNotificationPermission,
} from '@/lib/stores/notification';
import { t } from '@/lib/stores/i18n';

/**
 * Asks once for notifications, inline where they matter: next to an open
 * table, not as a card floating over the main action. Settings keeps the switch.
 */
export function NotificationNudge({ fallback }: { fallback?: string }) {
  const pending = notificationStatus.value === 'pending' && !notificationsEnabled.value;

  if (!pending) {
    return fallback ? <p class="m-0 text-xs text-text-muted">{fallback}</p> : null;
  }

  return (
    <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-text-muted">
      <p class="m-0 flex min-w-0 items-start gap-2">
        <Bell size={16} className="mt-px shrink-0" aria-hidden="true" />
        {t('notifications_prompt')}
      </p>
      <span class="flex items-center gap-1">
        <Button variant="secondary" size="sm" onClick={() => void requestNotificationPermission()}>
          {t('enable')}
        </Button>
        <Button variant="quiet" size="sm" onClick={() => rejectNotifications()}>
          {t('not_now')}
        </Button>
      </span>
    </div>
  );
}
