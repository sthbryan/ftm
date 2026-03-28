<script lang="ts">
  import { page } from '$app/stores';
  import ThemeSwitcher from './ThemeSwitcher.svelte';
  import { cn } from '$lib/utils/cn';

  let isSettings = $derived($page.url.pathname === '/settings');
</script>

<header class="flex flex-col md:flex-row md:justify-between md:items-center mb-6 pb-5 border-b flex-shrink-0 z-10 border-border">
  <div class="flex items-center gap-4">
    <img src="/favicon.png" alt="Foundry Tunnel Manager" class="w-10 h-10 sm:w-12 sm:h-12 rounded-xl object-cover" />
    <div class="text">
      <h1 class="font-serif text-2xl sm:text-3xl lg:text-4xl font-bold m-0 mb-1 tracking-tight text-text-heading">Foundry Tunnel Manager</h1>
      <p class="text-xs sm:text-sm m-0 font-medium text-text-muted">Share your world with players everywhere</p>
    </div>
  </div>
  <div class="flex items-center gap-3">
    <a
      href="/settings"
      class={cn(
        "p-2 rounded-lg transition-colors",
        isSettings ? "bg-primary/20 text-primary" : "hover:bg-secondary"
      )}
      aria-label="Settings"
    >
      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
          d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
          d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
      </svg>
    </a>
    <ThemeSwitcher />
  </div>
</header>
