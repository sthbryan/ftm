<script lang="ts">
  import ThemeButton from './ThemeButton.svelte';
  import { useTheme } from '$lib/stores/theme.svelte';
  import { translate } from '$lib/i18n';

  interface ThemeGroup {
    name: string;
    themes: { id: string; color: string }[];
  }

  interface Props {
    groups: ThemeGroup[];
  }

  let { groups }: Props = $props();

  const theme = useTheme();

  let t = $derived($translate);

  const themeNames: Record<string, string> = {
    'dracula': 'Dracula',
    'nord': 'Nord',
    'nord-light': 'Nord Light',
    'tokyo-night': 'Tokyo Night',
    'tokyo-night-storm': 'Tokyo Storm',
    'tokyo-night-light': 'Tokyo Light',
    'catppuccin-mocha': 'Catppuccin',
    'catppuccin-latte': 'Catppuccin Latte',
    'one-dark': 'One Dark',
    'gruvbox': 'Gruvbox',
    'gruvbox-light': 'Gruvbox Light',
    'solarized-dark': 'Solarized',
    'solarized-light': 'Solarized Light',
    'rose-pine': 'Rose Pine',
    'rose-pine-dawn': 'Rose Pine Dawn',
    'red': 'Red',
    'red-light': 'Red Light',
    'blue': 'Blue',
    'blue-light': 'Blue Light',
    'purple': 'Purple',
    'purple-light': 'Purple Light',
  };

  function getName(id: string): string {
    return themeNames[id] || id;
  }

  function getCurrentColor(): string {
    for (const group of groups) {
      const found = group.themes.find(t => t.id === theme.current);
      if (found) return found.color;
    }
    return '#bd93f9';
  }
</script>

{#each groups as group}
  <div class="mb-4 last:mb-0">
    <h3 class="text-xs text-text-muted mb-2 font-medium">{group.name}</h3>
    <div class="flex flex-wrap gap-2">
      {#each group.themes as t}
        <ThemeButton
          id={t.id}
          color={t.color}
          selected={theme.current === t.id}
          label={getName(t.id)}
          onclick={() => theme.set(t.id)}
        />
      {/each}
    </div>
  </div>
{/each}

<div class="mt-4 pt-4 border-t border-border flex items-center gap-3">
  <div 
    class="w-8 h-8 rounded-full shadow-md flex-shrink-0"
    style="background: {getCurrentColor()};"
  ></div>
  <div>
    <p class="font-medium text-sm">{getName(theme.current)}</p>
    <p class="text-xs text-text-muted">{t('current_theme')}</p>
  </div>
</div>
