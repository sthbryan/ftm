let permission = $state('default');
let useOSNotifications = $state(false);
let enabled = $state(false);
let soundEnabled = $state(true);

let audioContext = null;

function initAudio() {
  if (typeof window !== 'undefined' && !audioContext) {
    audioContext = new (window.AudioContext || window.webkitAudioContext)();
  }
}

function playSound(type) {
  if (!soundEnabled || typeof window === 'undefined') return;
  
  initAudio();
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
  get permission() { return permission; },
  get useOSNotifications() { return useOSNotifications; },
  get enabled() { return enabled; },
  get soundEnabled() { return soundEnabled; },
  set soundEnabled(value) { 
    soundEnabled = value;
    localStorage.setItem('ftm-sound-enabled', value ? 'true' : 'false');
  },
  
  init() {
    const saved = localStorage.getItem('ftm-notification-pref');
    if (saved === 'granted') {
      permission = 'granted';
      useOSNotifications = true;
    }
    enabled = localStorage.getItem('ftm-notifications-enabled') === 'true';
    soundEnabled = localStorage.getItem('ftm-sound-enabled') !== 'false';
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
    if (!enabled && !soundEnabled) return;
    
    playSound(type);
    
    if (!enabled) return;
    
    if (useOSNotifications && permission === 'granted') {
      new Notification(title, { body });
    }
  },
  
  notifyOnline(name, url) {
    this.notify('Tunnel Active', `${name} - ${url}`, 'success');
  },
  
  notifyError(name, err) {
    this.notify('Tunnel Error', `${name}: ${err}`, 'error');
  },
  
  notifyExpiring(name, mins) {
    const title = mins <= 1 ? 'Last Minute!' : 'Tunnel Expiring';
    const type = mins <= 1 ? 'alert' : 'warning';
    this.notify(title, `${name}: ${mins} min remaining`, type);
  },
  
  notifyExpired(name) {
    this.notify('Tunnel Expired', `${name} session ended`, 'error');
  }
};

export function useNotifications() {
  return notificationStore;
}
