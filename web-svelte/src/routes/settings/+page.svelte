<script lang="ts">
  import { onMount } from 'svelte';
  import { useSettings } from '$lib/stores/settings.svelte';
  import { useTheme } from '$lib/stores/theme.svelte';
  import { goto } from '$app/navigation';
  import { cn } from '$lib/utils/cn';

  const settingsStore = useSettings();
  const theme = useTheme();

  onMount(() => {
    settingsStore.load();
  });

  function toggleNotifications() {
    settingsStore.update({ notifications_enabled: !settingsStore.settings.notifications_enabled });
  }

  function toggleSound() {
    settingsStore.update({ notification_sound: !settingsStore.settings.notification_sound });
  }

  function setTheme(t: 'light' | 'dark' | 'system') {
    settingsStore.update({ theme: t });
    theme.set(t);
  }
</script>

<div class="max-w-lg mx-auto py-8">
  <div class="flex items-center gap-4 mb-8">
    <a href="/" class="p-2 rounded-lg hover:bg-secondary transition-colors" aria-label="Go back">
      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
      </svg>
    </a>
    <h1 class="text-2xl font-semibold">Settings</h1>
  </div>

  {#if !settingsStore.loaded}
    <div class="flex justify-center py-12">
      <div class="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin"></div>
    </div>
  {:else}
    <div class="space-y-6">
      <section class="p-5 rounded-xl bg-card border border-border">
        <h2 class="text-lg font-semibold mb-4 text-text-heading">Notifications</h2>

        <div class="space-y-4">
          <label class="flex items-center justify-between cursor-pointer">
            <span>Enable Notifications</span>
            <button
              onclick={toggleNotifications}
              class={cn(
                "relative w-12 h-6 rounded-full transition-colors",
                settingsStore.settings.notifications_enabled ? "bg-primary" : "bg-secondary"
              )}
              aria-label="Toggle notifications"
              aria-pressed={settingsStore.settings.notifications_enabled}
            >
              <span
                class={cn(
                  "absolute top-1 w-4 h-4 bg-white rounded-full transition-transform",
                  settingsStore.settings.notifications_enabled ? "translate-x-7" : "translate-x-1"
                )}
              ></span>
            </button>
          </label>

          <label class="flex items-center justify-between cursor-pointer">
            <span>Sound Effects</span>
            <button
              onclick={toggleSound}
              class={cn(
                "relative w-12 h-6 rounded-full transition-colors",
                settingsStore.settings.notification_sound ? "bg-primary" : "bg-secondary"
              )}
              aria-label="Toggle sound"
              aria-pressed={settingsStore.settings.notification_sound}
            >
              <span
                class={cn(
                  "absolute top-1 w-4 h-4 bg-white rounded-full transition-transform",
                  settingsStore.settings.notification_sound ? "translate-x-7" : "translate-x-1"
                )}
              ></span>
            </button>
          </label>
        </div>
      </section>

      <section class="p-5 rounded-xl bg-card border border-border">
        <h2 class="text-lg font-semibold mb-4 text-text-heading">Appearance</h2>

        <div class="flex gap-2">
          {#each ['light', 'dark', 'system'] as t}
            <button
              onclick={() => setTheme(t as 'light' | 'dark' | 'system')}
              class={cn(
                "flex-1 py-2 px-3 rounded-lg text-sm font-medium transition-colors capitalize",
                settingsStore.settings.theme === t
                  ? "bg-primary text-heading"
                  : "bg-secondary-btn text-secondary-btn-text hover:bg-secondary"
              )}
            >
              {t}
            </button>
          {/each}
        </div>
      </section>
    </div>
  {/if}
</div>
