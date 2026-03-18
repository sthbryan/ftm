<script>
  import { useToast } from '$lib/stores/toast.svelte';
  import { useSound } from '$lib/stores/sound.svelte';

  const toastStore = useToast();
  const soundStore = useSound();

  function toggleSound() {
    soundStore.toggle();
    // mirror into toast store for UI reflect
    toastStore.soundEnabled = soundStore.enabled;
  }
</script>

<div class="toasts-container">
  {#each toastStore.toasts as toast (toast.id)}
    <div 
      class="toast toast-{toast.type}"
      role="alert"
    >
      <span class="toast-message">{toast.message}</span>
      <button 
        class="toast-close" 
        onclick={() => toastStore.remove(toast.id)}
        aria-label="Close notification"
      >
        &times;
      </button>
    </div>
  {/each}
  
  <!-- migrated sound toggle: keep for backward compatibility but hidden (logic now in ThemeSwitcher) -->
  <button 
    class="sound-toggle" 
    onclick={toggleSound}
    title={toastStore.soundEnabled ? 'Sound on' : 'Sound off'}
    aria-label={toastStore.soundEnabled ? 'Disable sound' : 'Enable sound'}
    style="display:none"
  >
    {toastStore.soundEnabled ? '🔊' : '🔇'}
  </button>
</div>

<style>
  .toasts-container {
    position: fixed;
    top: 20px;
    right: 20px;
    z-index: 1000;
    display: flex;
    flex-direction: column;
    gap: 10px;
    pointer-events: none;
  }

  .toast {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 14px 18px;
    border-radius: 10px;
    box-shadow: 0 10px 30px rgba(0,0,0,0.15);
    animation: slideIn 0.3s cubic-bezier(0.16, 1, 0.3, 1);
    pointer-events: auto;
    min-width: 280px;
    max-width: 400px;
  }

  @keyframes slideIn {
    from {
      opacity: 0;
      transform: translateX(100px);
    }
    to {
      opacity: 1;
      transform: translateX(0);
    }
  }

  .toast-success {
    background: linear-gradient(135deg, #16a34a 0%, #22c55e 100%);
    color: white;
  }

  .toast-error {
    background: linear-gradient(135deg, #dc2626 0%, #ef4444 100%);
    color: white;
  }

  .toast-info {
    background: linear-gradient(135deg, #92400e 0%, #b45309 100%);
    color: white;
  }

  .toast-message {
    flex: 1;
    font-size: 14px;
    font-weight: 500;
  }

  .toast-close {
    background: rgba(255,255,255,0.2);
    border: none;
    color: white;
    width: 28px;
    height: 28px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 18px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background 0.15s;
  }

  .toast-close:hover {
    background: rgba(255,255,255,0.3);
  }

  .sound-toggle {
    position: fixed;
    bottom: 20px;
    right: 20px;
    width: 44px;
    height: 44px;
    border-radius: 50%;
    border: none;
    background: white;
    box-shadow: 0 4px 12px rgba(0,0,0,0.15);
    cursor: pointer;
    font-size: 20px;
    display: flex;
    align-items: center;
    justify-content: center;
    pointer-events: auto;
    transition: transform 0.15s, box-shadow 0.15s;
  }

  .sound-toggle:hover {
    transform: scale(1.1);
    box-shadow: 0 6px 16px rgba(0,0,0,0.2);
  }

  @media (max-width: 640px) {
    .toasts-container {
      top: 10px;
      right: 10px;
      left: 10px;
    }

    .toast {
      min-width: auto;
      max-width: none;
      width: 100%;
    }

    .sound-toggle {
      bottom: 10px;
      right: 10px;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .toast {
      animation: none;
    }
  }
</style>
