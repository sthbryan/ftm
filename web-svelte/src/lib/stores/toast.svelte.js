let toasts = $state([]);
let soundEnabled = $state(true);

const audioContext = typeof window !== 'undefined' ? new (window.AudioContext || window.webkitAudioContext)() : null;

try {
  const { useSound } = await import('./sound.svelte.js');
  const soundStore = useSound();
  soundEnabled = soundStore.enabled;
} catch (e) {
}

function playSound(type) {
  if (!soundEnabled || !audioContext) return;
  
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
  } else if (type === 'info') {
    oscillator.frequency.setValueAtTime(440, audioContext.currentTime);
    gainNode.gain.setValueAtTime(0.2, audioContext.currentTime);
    gainNode.gain.exponentialRampToValueAtTime(0.01, audioContext.currentTime + 0.15);
    oscillator.start(audioContext.currentTime);
    oscillator.stop(audioContext.currentTime + 0.15);
  }
}

export function useToast() {
  return {
    get toasts() { return toasts; },
    get soundEnabled() { return soundEnabled; },
    set soundEnabled(value) { 
      soundEnabled = value;
      try { if (typeof window !== 'undefined') localStorage.setItem('ftm-sound-enabled', value); } catch(e){}
    },
    
    show(message, type = 'info', duration = 3000) {
      const id = Date.now() + Math.random();
      const toast = { id, message, type, duration };
      toasts = [...toasts, toast];
      
      playSound(type);
      
      setTimeout(() => {
        toasts = toasts.filter(t => t.id !== id);
      }, duration);
    },
    
    success(message, duration) {
      this.show(message, 'success', duration);
    },
    
    error(message, duration) {
      this.show(message, 'error', duration);
    },
    
    info(message, duration) {
      this.show(message, 'info', duration);
    },
    
    remove(id) {
      toasts = toasts.filter(t => t.id !== id);
    }
  };
}
