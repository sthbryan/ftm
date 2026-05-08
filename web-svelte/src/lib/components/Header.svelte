<script lang="ts">
  import { page } from '$app/stores';
  import { cn } from '$lib/utils/cn';
  import { translate } from '$lib/i18n';
  import { Settings } from 'lucide-svelte';

  let isSettings = $derived($page.url.pathname === '/settings');
  let t = $derived($translate);
</script>

<header class="flex justify-between items-center mb-6 pb-5 border-b flex-shrink-0 z-10 border-border">
  <div class="flex items-center gap-4">
    <img src="/favicon.png" alt={t('app_name')} class="w-10 h-10 sm:w-12 sm:h-12 rounded-xl object-cover" />
    <div class="text">
      <h1 class="font-serif text-2xl sm:text-3xl lg:text-4xl font-bold m-0 mb-1 tracking-tight text-text-heading">{t('app_name')}</h1>
      <p class="text-xs sm:text-sm m-0 font-medium text-text-muted">{t('app_tagline')}</p>
    </div>
  </div>
  
  <a
    href="/settings"
    class={cn(
      "p-2 rounded-lg transition-colors",
      isSettings ? "bg-primary/20 text-primary" : "hover:bg-secondary"
    )}
    aria-label={t('settings')}
  >
    <Settings size={20} />
  </a>
</header>