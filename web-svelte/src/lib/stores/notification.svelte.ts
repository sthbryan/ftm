import { settingsApi, type Settings } from '$lib/api';
import { useToast, type ToastType } from './toast.svelte';

type NotificationStatus = 'pending' | 'granted' | 'rejected';
type NotificationChannel = 'toast' | 'os';

interface NotificationEvent {
  channel?: NotificationChannel | string;
  title?: string;
  body?: string;
  toastType?: string;
  soundType?: string;
  soundEnabled?: boolean;
}

let status = $state<NotificationStatus>('pending');
let soundEnabled = $state(true);

const toast = useToast();

let audioContext: AudioContext | null = null;

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

  if (Notification.permission === 'default') {
    return 'pending';
  }

  return 'rejected';
}

async function initAudio() {
  if (typeof window === 'undefined') return;
  if (audioContext) return;

  const AudioContextClass = (window.AudioContext || (window as typeof window & { webkitAudioContext?: typeof AudioContext }).webkitAudioContext);
  if (!AudioContextClass) return;

  try {
    audioContext = new AudioContextClass();
    if (audioContext.state === 'suspended') {
      await audioContext.resume();
    }
  } catch {
    audioContext = null;
  }
}

async function playSound(type: string) {
  if (!soundEnabled || typeof window === 'undefined') return;

  await initAudio();
  if (!audioContext) return;

  const oscillator = audioContext.createOscillator();
  const gainNode = audioContext.createGain();

  oscillator.connect(gainNode);
  gainNode.connect(audioContext.destination);

  if (type === 'success') {
    oscillator.frequency.setValueAtTime(523.25, audioContext.currentTime);
    oscillator.frequency.setValueAtTime(659.25, audioContext.currentTime + 0.1);
    gainNode.gain.setValueAtTime(0.3, audioContext.currentTime);
    gainNode.gain.exponentialRampToValueAtTime(0.01, audioContext.currentTime + 0.3);
    oscillator.start(audioContext.currentTime);
    oscillator.stop(audioContext.currentTime + 0.3);
  } else if (type === 'error') {
    oscillator.frequency.setValueAtTime(200, audioContext.currentTime);
    gainNode.gain.setValueAtTime(0.3, audioContext.currentTime);
    gainNode.gain.exponentialRampToValueAtTime(0.01, audioContext.currentTime + 0.2);
    oscillator.start(audioContext.currentTime);
    oscillator.stop(audioContext.currentTime + 0.2);
  } else if (type === 'warning') {
    oscillator.frequency.setValueAtTime(440, audioContext.currentTime);
    oscillator.frequency.setValueAtTime(880, audioContext.currentTime + 0.15);
    gainNode.gain.setValueAtTime(0.25, audioContext.currentTime);
    gainNode.gain.exponentialRampToValueAtTime(0.01, audioContext.currentTime + 0.3);
    oscillator.start(audioContext.currentTime);
    oscillator.stop(audioContext.currentTime + 0.3);
  } else if (type === 'alert') {
    oscillator.frequency.setValueAtTime(880, audioContext.currentTime);
    oscillator.frequency.setValueAtTime(440, audioContext.currentTime + 0.1);
    oscillator.frequency.setValueAtTime(880, audioContext.currentTime + 0.2);
    gainNode.gain.setValueAtTime(0.35, audioContext.currentTime);
    gainNode.gain.exponentialRampToValueAtTime(0.01, audioContext.currentTime + 0.3);
    oscillator.start(audioContext.currentTime);
    oscillator.stop(audioContext.currentTime + 0.3);
  } else {
    oscillator.frequency.setValueAtTime(440, audioContext.currentTime);
    gainNode.gain.setValueAtTime(0.2, audioContext.currentTime);
    gainNode.gain.exponentialRampToValueAtTime(0.01, audioContext.currentTime + 0.15);
    oscillator.start(audioContext.currentTime);
    oscillator.stop(audioContext.currentTime + 0.15);
  }
}

const notificationStore = {
  get status() { return status; },
  get enabled() { return status === 'granted'; },
  get soundEnabled() { return soundEnabled; },

  applySettings(settings: Settings) {
    soundEnabled = settings.notification_sound;
    status = deriveStatusFromSettings(settings);
  },

  async syncWithSettings() {
    try {
      const settings = await settingsApi.get();
      this.applySettings(settings);
    } catch {
      status = 'pending';
    }
  },

  async requestPermission(): Promise<boolean> {
    if (typeof window === 'undefined' || !('Notification' in window)) {
      status = 'rejected';
      await settingsApi.update({ notifications_enabled: "rejected" });
      return false;
    }

    const result = await Notification.requestPermission();
    const granted = result === 'granted';

    status = granted ? 'granted' : 'rejected';

    try {
      const settings = await settingsApi.update({ notifications_enabled: granted ? "granted" : "rejected" });
      this.applySettings(settings);
    } catch {
      return granted;
    }

    return granted;
  },

  reject() {
    status = 'rejected';
    settingsApi.update({ notifications_enabled: "rejected" });
  },

  notify(title: string, body: string, type: ToastType = 'info') {
    if (soundEnabled) {
      playSound(type);
    }

    if (status === 'granted' && typeof window !== 'undefined' && 'Notification' in window && Notification.permission === 'granted') {
      new Notification(title, { body });
      return;
    }

    toast.show(body, type);
  },

  renderEvent(event: NotificationEvent) {
    const title = event.title ?? 'Notification';
    const body = event.body ?? '';
    const soundType = event.soundType ?? 'info';
    const toastType = normalizeToastType(event.toastType);
    const channel = event.channel ?? 'toast';
    const shouldPlaySound = event.soundEnabled ?? soundEnabled;

    if (shouldPlaySound) {
      playSound(soundType);
    }

    if (channel === 'os') {
      if (status === 'granted' && typeof window !== 'undefined' && 'Notification' in window && Notification.permission === 'granted') {
        new Notification(title, { body });
      }
      return;
    }

    if (channel === 'toast') {
      toast.show(body, toastType);
      return;
    }

    toast.show(body, toastType);
  },

  notifyOnline(name: string, url: string) {
    this.notify('Tunnel Active', `${name} - ${url}`, 'success');
  },

  notifyError(name: string, err: string) {
    this.notify('Tunnel Error', `${name}: ${err}`, 'error');
  },

  notifyExpiring(name: string, mins: number) {
    const title = mins <= 1 ? 'Last Minute!' : 'Tunnel Expiring';
    const type = mins <= 1 ? 'alert' : 'warning';
    this.notify(title, `${name}: ${mins} min remaining`, type as ToastType);
  },

  notifyExpired(name: string) {
    this.notify('Tunnel Expired', `${name} session ended`, 'error');
  }
};

export function useNotifications() {
  return notificationStore;
}
