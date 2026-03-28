import { useToast } from './toast.svelte.js';

let permission = $state('default');
let useOSNotifications = $state(false);
let enabled = $state(false);

const toast = useToast();

const notificationStore = {
  get permission() { return permission; },
  get useOSNotifications() { return useOSNotifications; },
  get enabled() { return enabled; },
  
  init() {
    const saved = localStorage.getItem('ftm-notification-pref');
    if (saved === 'granted') {
      permission = 'granted';
      useOSNotifications = true;
    }
    enabled = localStorage.getItem('ftm-notifications-enabled') === 'true';
  },
  
  async requestPermission() {
    if (!('Notification' in window)) {
      useOSNotifications = false;
      return false;
    }
    
    if (permission === 'default') {
      const result = await Notification.requestPermission();
      permission = result;
      useOSNotifications = result === 'granted';
    }
    
    if (permission === 'granted') {
      enabled = true;
      localStorage.setItem('ftm-notification-pref', 'granted');
      localStorage.setItem('ftm-notifications-enabled', 'true');
    }
    
    return permission === 'granted';
  },
  
  enable() {
    enabled = true;
    localStorage.setItem('ftm-notifications-enabled', 'true');
  },
  
  disable() {
    enabled = false;
    localStorage.setItem('ftm-notifications-enabled', 'false');
  },
  
  notify(title, body, type = 'info') {
    if (!enabled) return;
    
    if (useOSNotifications && permission === 'granted') {
      new Notification(title, { body });
      return;
    }
    
    toast.show(body, type);
  },
  
  notifyOnline(name, url) {
    this.notify('Tunnel Active', `${name} - ${url}`, 'success');
  },
  
  notifyError(name, err) {
    this.notify('Tunnel Error', `${name}: ${err}`, 'error');
  },
  
  notifyExpiring(name, mins) {
    const title = mins <= 1 ? 'Last Minute!' : 'Tunnel Expiring';
    this.notify(title, `${name}: ${mins} min remaining`, mins <= 1 ? 'error' : 'info');
  },
  
  notifyExpired(name) {
    this.notify('Tunnel Expired', `${name} session ended`, 'error');
  }
};

export function useNotifications() {
  return notificationStore;
}
