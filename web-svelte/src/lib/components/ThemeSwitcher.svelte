<script>
  import { useTheme } from "$lib/stores/theme.svelte";
  import { useSound } from "$lib/stores/sound.svelte";

  const theme = useTheme();

  const themeIcons = {
    light: "☀️",
    dark: "🌙",
    sepia: "📜",
    contrast: "⚡",
    red: "🔴",
    blue: "🔵",
    dracula: "🦇",
  };

  const themeLabels = {
    light: "Light",
    dark: "Dark",
    sepia: "Sepia",
    contrast: "High Contrast",
    red: "Red",
    blue: "Blue",
    dracula: "Dracula",
  };

  const sound = useSound();
</script>

<div class="theme-switcher" role="group" aria-label="Theme and sound controls">
  <div class="theme-select">
    <label for="theme-select" class="sr-only">Theme</label>
    <select
      id="theme-select"
      onchange={(e) => theme.set(e.target.value)}
      bind:value={theme.current}
      aria-label="Select theme"
    >
      {#each theme.themes as t}
        <option value={t}>{themeLabels[t]}</option>
      {/each}
    </select>
  </div>

  <div class="controls">
    <button
      class="sound-button"
      onclick={() => sound.toggle()}
      aria-pressed={sound.enabled}
      title={sound.enabled ? "Sound on" : "Sound off"}
    >
      {sound.enabled ? "🔊" : "🔇"}
    </button>
  </div>
</div>

<style>
  .theme-switcher {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 10px;
  }

  .theme-select select {
    padding: 8px 10px;
    height: 36px;
    border-radius: 8px;
    border: 1px solid var(--border-color);
    background: var(--card-bg);
    color: var(--text-color);
    appearance: none;
    cursor: pointer;
  }

  .controls {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .sound-button {
    height: 36px;
    width: 36px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 8px;
    border: 1px solid var(--border-color);
    background: var(--card-bg);
    color: var(--text-color);
    cursor: pointer;
    aspect-ratio: 1 / 1;
  }

  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  }
</style>
