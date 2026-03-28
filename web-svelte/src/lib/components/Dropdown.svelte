<script>
  import { ChevronDown } from 'lucide-svelte';

  let { 
    options = [], 
    onSelect, 
    align = 'right',
    ariaLabel = 'Options',
    class: className = '',
    id = '',
    label = ''
  } = $props();

  let isOpen = $state(false);

  function toggle() {
    isOpen = !isOpen;
  }

  function select(option) {
    isOpen = false;
    onSelect?.(option);
  }

  function handleClickOutside(e) {
    if (!e.target.closest('.dropdown')) {
      isOpen = false;
    }
  }

  $effect(() => {
    if (isOpen) {
      document.addEventListener('click', handleClickOutside);
      return () => document.removeEventListener('click', handleClickOutside);
    }
  });
</script>

<div class="dropdown {className}">
  <button type="button" class="dropdown-trigger" id={id || undefined} onclick={toggle} aria-label={ariaLabel}>
    <span class="trigger-text">
      {label || 'Options'}
    </span>
    <ChevronDown class="chevron {isOpen ? 'open' : ''}" size={16} />
  </button>

  {#if isOpen}
    <div class="dropdown-menu" class:align-left={align === 'left'}>
      {#each options as option}
        {#if option.label === 'separator'}
          <div class="divider"></div>
        {:else}
          <button 
            type="button" 
            class="dropdown-item"
            class:disabled={option.disabled}
            class:danger={option.danger}
            disabled={option.disabled}
            onclick={() => select(option)}
          >
            {#if option.icon}
              <option.icon size={16} />
            {/if}
            <span>{option.label}</span>
          </button>
        {/if}
      {/each}
    </div>
  {/if}
</div>

<style>
  .dropdown {
    position: relative;
    display: flex;
  }

  .dropdown-trigger {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 12px;
    font-size: 13px;
    border-radius: 8px;
    border: 1px solid var(--border-color);
    background: var(--card-bg);
    color: var(--text-color);
    cursor: pointer;
    transition: all 0.15s;
    flex: 1;
    min-height: 36px;
    box-sizing: border-box;
  }

  .dropdown-trigger:hover {
    background: var(--hover-bg);
  }

  .trigger-text {
    font-size: 13px;
    text-align: left;
    flex: 1;
  }

  :global(.chevron) {
    transition: transform 0.2s;
  }

  :global(.chevron.open) {
    transform: rotate(180deg);
  }

  .dropdown-menu {
    position: absolute;
    right: 0;
    top: calc(100% + 5px);
    min-width: 150px;
    max-height: 300px;
    background: var(--card-bg);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    box-shadow: 0 4px 16px rgba(0,0,0,0.2);
    padding: 4px;
    z-index: 100;
    overflow-y: auto;
    animation: fadeIn 0.1s ease;
  }

  .dropdown-menu.align-left {
    right: auto;
    left: 0;
  }

  @keyframes fadeIn {
    from {
      opacity: 0;
      transform: translateY(4px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .dropdown-item {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 8px 12px;
    font-size: 13px;
    border: none;
    border-radius: 8px;
    background: none;
    color: var(--text-color);
    cursor: pointer;
    text-align: left;
    transition: background 0.1s;
  }

  .dropdown-item:hover:not(:disabled) {
    background: var(--hover-bg);
  }

  .dropdown-item.disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .dropdown-item.danger {
    color: var(--btn-danger-bg, #ef4444);
  }

  .dropdown-item.danger:hover:not(:disabled) {
    background: color-mix(in srgb, var(--btn-danger-bg, #ef4444) 10%, transparent);
  }

  .divider {
    height: 1px;
    background: var(--border-color);
    margin: 4px 8px;
  }
</style>
