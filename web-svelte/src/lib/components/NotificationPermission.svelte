<script lang="ts">
  import { useNotifications } from '$lib/stores/notification.svelte.js';
  import { cn } from '$lib/utils/cn';
  import { animate } from 'motion';

  const notifications = useNotifications();

  let show = $derived(notifications.status === 'pending');
  let cardRef: HTMLDivElement | undefined = $state();
  let isAnimatingOut = $state(false);

  $effect(() => {
    if (show && cardRef) {
      cardRef.style.opacity = '0';
      cardRef.style.transform = 'translateY(16px) scale(0.96)';
      requestAnimationFrame(() => {
        animate(
          cardRef!,
          { opacity: 1, y: 0, scale: 1 },
          { type: 'spring', stiffness: 300, damping: 26 }
        );
      });
    }
  });

  function animateOut(): Promise<void> {
    if (!cardRef) return Promise.resolve();
    return animate(
      cardRef,
      { opacity: 0, y: 16, scale: 0.96 },
      { type: 'spring', stiffness: 500, damping: 35 }
    ).finished.then(() => {});
  }

  async function request() {
    if (isAnimatingOut) return;
    isAnimatingOut = true;
    await animateOut();
    await notifications.requestPermission();
    isAnimatingOut = false;
  }

  async function later() {
    if (isAnimatingOut) return;
    isAnimatingOut = true;
    await animateOut();
    notifications.reject();
    isAnimatingOut = false;
  }
</script>

{#if show || isAnimatingOut}
  <div
    bind:this={cardRef}
    class={cn(
      'fixed bottom-4 right-4 max-w-[320px] rounded-xl p-5 z-50',
      'bg-card border border-border shadow-lg'
    )}
  >
    <div class="flex flex-col gap-2">
      <h3 class="m-0 text-[1.1rem] font-semibold text-text-heading">Enable Notifications</h3>
      <p class="m-0 text-sm leading-relaxed text-text-muted">Get notified when tunnels go online, offline, or are about to expire.</p>
      <div class="flex gap-2 mt-1">
        <button
          onclick={request}
          class={cn(
            'inline-flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-sm font-medium flex-1 cursor-pointer',
            'transition-all duration-150 hover:-translate-y-px',
            'bg-primary text-heading border-none'
          )}
        >
          Enable
        </button>
        <button
          onclick={later}
          class={cn(
            'inline-flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-sm font-medium flex-1 cursor-pointer',
            'transition-all duration-150 hover:-translate-y-px',
            'bg-secondary-btn text-secondary-btn-text border border-border'
          )}
        >
          Not Now
        </button>
      </div>
    </div>
  </div>
{/if}