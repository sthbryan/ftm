<script>
  import { useTheme } from "$lib/stores/theme.svelte";
  import { useSound } from "$lib/stores/sound.svelte";
  import Dropdown from './Dropdown.svelte';

  const theme = useTheme();
  const sound = useSound();

  const themeLabels = {
    nord: "Nord",
    "nord-light": "Nord Light",
    "rose-pine": "Rose Pine",
    "rose-pine-dawn": "Rose Pine Dawn",
    "tokyo-night": "Tokyo Night",
    "tokyo-night-storm": "Tokyo Night Storm",
    "tokyo-night-light": "Tokyo Night Light",
    "catppuccin-mocha": "Catppuccin Mocha",
    "catppuccin-latte": "Catppuccin Latte",
    "one-dark": "One Dark",
    gruvbox: "Gruvbox",
    "gruvbox-light": "Gruvbox Light",
    "solarized-dark": "Solarized Dark",
    "solarized-light": "Solarized Light",
    dracula: "Dracula",
    red: "Red",
    blue: "Blue",
    purple: "Purple",
  };

  const themeOptions = $derived(theme.themes.map(t => ({
    label: themeLabels[t] || t,
    value: t
  })));

  const selectedTheme = $derived(themeOptions.find(t => t.value === theme.current));

  function selectTheme(option) {
    theme.set(option.value);
  }
</script>

<div class="theme-switcher" role="group" aria-label="Theme and sound controls">
  <Dropdown 
    options={themeOptions} 
    onSelect={selectTheme}
    align="left"
    class="min-w-150"
    ariaLabel="Select theme"
    label={selectedTheme?.label || 'Theme'}
  />

  <button
    class="sound-button"
    onclick={() => sound.toggle()}
    aria-pressed={sound.enabled}
    title={sound.enabled ? "Sound on" : "Sound off"}
  >
    {sound.enabled ? "🔊" : "🔇"}
  </button>
</div>

<style>
  .theme-switcher {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 10px;
  }

  .sound-button {
    height: 36px;
    width: 36px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 6px;
    border: 1px solid var(--border-color);
    background: var(--card-bg);
    color: var(--text-color);
    cursor: pointer;
    font-size: 16px;
  }

  .sound-button:hover {
    background: var(--hover-bg);
  }
</style>
