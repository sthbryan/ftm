<script>
  import { useTheme } from '$lib/stores/theme.svelte';
  import { useSound } from '$lib/stores/sound.svelte';
  
  const theme = useTheme();
  
  const themeIcons = {
    light: '☀️',
    dark: '🌙',
    sepia: '📜',
    contrast: '⚡',
    red: '🔴',
    blue: '🔵',
    dracula: '🦇'
  };
  
  const themeLabels = {
    light: 'Light',
    dark: 'Dark',
    sepia: 'Sepia',
    contrast: 'High Contrast',
    red: 'Red',
    blue: 'Blue',
    dracula: 'Dracula'
  };

  const sound = useSound();
</script>

<div class="theme-switcher" role="group" aria-label="Theme and sound controls">
  <div class="theme-select">
    <label for="theme-select" class="sr-only">Theme</label>
    <select id="theme-select" on:change={(e) => theme.set(e.target.value)} bind:value={theme.current} aria-label="Select theme">
      {#each theme.themes as t}
        <option value={t}>{themeLabels[t]}</option>
      {/each}
    </select>
  </div>

  <div class="controls">
    <button 
      class="sound-button"
      on:click={() => sound.toggle()}
      aria-pressed={sound.enabled}
      title={sound.enabled ? 'Sound on' : 'Sound off'}
    >
      {sound.enabled ? '🔊' : '🔇'}
    </button>
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

  .theme-select select {
    padding: 8px 10px;
    border-radius: 8px;
    border: 1px solid var(--border-color);
    background: var(--card-bg);
    color: var(--text-color);
  }

  .controls {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .sound-button {
    padding: 8px;
    border-radius: 8px;
    border: 1px solid var(--border-color);
    background: var(--card-bg);
    color: var(--text-color);
    cursor: pointer;
  }

  .sr-only { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap; border: 0; }

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
