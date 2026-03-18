const STORAGE_KEY = 'ftm-sound-enabled';

let soundEnabled = $state(true);

function getInitial() {
  if (typeof window === 'undefined') return true;
  const saved = localStorage.getItem(STORAGE_KEY);
  return saved === null ? true : saved === 'true';
}

soundEnabled = getInitial();

export function useSound() {
  return {
    get enabled() { return soundEnabled; },
    set(enabled) {
      soundEnabled = !!enabled;
      if (typeof window !== 'undefined') localStorage.setItem(STORAGE_KEY, soundEnabled);
    },
    toggle() {
      this.set(!soundEnabled);
    }
  };
}
