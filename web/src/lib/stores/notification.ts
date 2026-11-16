import { signal } from '@preact/signals';
import { settingsApi, type Settings } from '@/lib/api';
import { toast, type ToastType } from './toast';
import { t } from './i18n';
import { cue, soundOn } from '@/lib/sound';

type NotificationStatus = 'pending' | 'granted' | 'rejected';
type NotificationChannel = 'toast' | 'os';

export interface NotificationEvent {
  channel?: NotificationChannel | string;
  title?: string;
  body?: string;
  toastType?: string;
  soundType?: string;
  soundEnabled?: boolean;
}

export const notificationStatus = signal<NotificationStatus>('pending');
/** The sound setting; `cue` reads the same signal. */
export const soundEnabled = soundOn;

export const notificationsEnabled = signal(false);

const TOAST_TYPES: ToastType[] = ['success', 'error', 'info', 'warning', 'alert'];

function normalizeToastType(type: string | undefined): ToastType {
  if (!type) return 'info';
  return TOAST_TYPES.includes(type as ToastType) ? (type as ToastType) : 'info';
}

function deriveStatusFromSettings(settings: Settings): NotificationStatus {
  if (settings.notifications_enabled) {
    return settings.notifications_enabled as NotificationStatus;
  }

  if (typeof window === 'undefined' || !('Notification' in window)) {
    return 'rejected';
  }

  return Notification.permission === 'default' ? 'pending' : 'rejected';
}

export function applyNotificationSettings(settings: Settings): void {
  soundEnabled.value = settings.notification_sound;
  notificationStatus.value = deriveStatusFromSettings(settings);
  notificationsEnabled.value = notificationStatus.value === 'granted';
}

export async function syncNotificationSettings(): Promise<void> {
  try {
    applyNotificationSettings(await settingsApi.get());
  } catch {
    notificationStatus.value = 'pending';
  }
}

export async function requestNotificationPermission(): Promise<boolean> {
  if (typeof window === 'undefined') return false;

  const isWails = Boolean((window as { wails?: unknown }).wails);

  if (!('Notification' in window) || isWails) {
    try {
      applyNotificationSettings(await settingsApi.update({ notifications_enabled: 'granted' }));
      return true;
    } catch {
      return false;
    }
  }

  const granted = (await Notification.requestPermission()) === 'granted';
  notificationStatus.value = granted ? 'granted' : 'rejected';
  notificationsEnabled.value = granted;

  try {
    applyNotificationSettings(
      await settingsApi.update({ notifications_enabled: granted ? 'granted' : 'rejected' }),
    );
  } catch {
    return granted;
  }

  return granted;
}

export function rejectNotifications(): void {
  notificationStatus.value = 'rejected';
  notificationsEnabled.value = false;
  void settingsApi.update({ notifications_enabled: 'rejected' });
}

function notify(title: string, body: string, type: ToastType = 'info'): void {
  cue(type);

  if (
    notificationStatus.value === 'granted' &&
    typeof window !== 'undefined' &&
    'Notification' in window &&
    Notification.permission === 'granted'
  ) {
    new Notification(title, { body });
    return;
  }

  toast.show(body, type);
}

export function renderNotificationEvent(event: NotificationEvent): void {
  const title = event.title ?? 'Notification';
  const body = event.body ?? '';
  const soundType = event.soundType ?? 'info';
  const toastType = normalizeToastType(event.toastType);
  const channel = event.channel ?? 'toast';
  const shouldPlay = event.soundEnabled ?? soundEnabled.value;

  if (shouldPlay) cue(soundType);

  if (
    channel === 'os' &&
    notificationStatus.value === 'granted' &&
    typeof window !== 'undefined' &&
    'Notification' in window &&
    Notification.permission === 'granted'
  ) {
    new Notification(title, { body });
    return;
  }

  toast.show(body, toastType);
}

export function notifyExpiring(name: string, minutes: number): void {
  const lastMinute = minutes <= 1;
  notify(
    t('tunnel_expiring'),
    t('notification_tunnel_expiring', { name, minutes }),
    lastMinute ? 'alert' : 'warning',
  );
}

export function notifyExpired(name: string): void {
  notify(t('tunnel_expired'), t('notification_tunnel_expired', { name }), 'error');
}
