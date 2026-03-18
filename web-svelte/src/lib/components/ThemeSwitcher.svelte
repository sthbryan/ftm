<script>
  import { useTheme } from '$lib/stores/theme.svelte';
  
  const theme = useTheme();
  
  const themeIcons = {
    light: '☀️',
    dark: '🌙',
    sepia: '📜',
    contrast: '⚡'
  };
  
  const themeLabels = {
    light: 'Light',
    dark: 'Dark',
    sepia: 'Sepia',
    contrast: 'High Contrast'
  };
</script>

<div class="theme-switcher">
  <button 
    class="theme-button" 
    onclick={() => theme.toggle()}
    title={themeLabels[theme.current]}
    aria-label={`Current theme: ${themeLabels[theme.current]}. Click to change.`}
  >
    <span class="theme-icon">{themeIcons[theme.current]}</span>
    <span class="theme-name">{themeLabels[theme.current]}</span>
  </button>
  
  <div class="theme-options">
    {#each theme.themes as t}
      <button 
        class="theme-option"
        class:active={t === theme.current}
        onclick={() => theme.set(t)}
        title={themeLabels[t]}
        aria-label={`Switch to ${themeLabels[t]}`}
      >
        {themeIcons[t]}
      </button>
    {/each}
  </div>
</div>

<style>
  .theme-switcher {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .theme-button {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 12px;
    border-radius: 8px;
    border: 1px solid var(--border-color);
    background: var(--card-bg);
    color: var(--text-color);
    cursor: pointer;
    transition: all 0.15s;
    font-size: 13px;
  }

  .theme-button:hover {
    background: var(--hover-bg);
  }

  .theme-icon {
    font-size: 16px;
  }

  .theme-name {
    font-weight: 500;
  }

  @media (max-width: 640px) {
    .theme-name {
      display: none;
    }
  }

  .theme-options {
    display: flex;
    gap: 4px;
  }

  .theme-option {
    width: 32px;
    height: 32px;
    border-radius: 6px;
    border: 1px solid var(--border-color);
    background: var(--card-bg);
    cursor: pointer;
    font-size: 14px;
    display: flex;
    align-items: center;
    justify-content: center;
    opacity: 0.5;
    transition: all 0.15s;
  }

  .theme-option:hover {
    opacity: 0.8;
    transform: scale(1.05);
  }

  .theme-option.active {
    opacity: 1;
    border-color: var(--primary-color);
    box-shadow: 0 0 0 2px var(--primary-color);
  }
</style>
