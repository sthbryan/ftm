<script>
  import { useTunnels } from '$lib/stores/tunnels.svelte';
  import TunnelCard from './TunnelCard.svelte';
  import { Radio } from 'lucide-svelte';

  let { onAction } = $props();

  const store = useTunnels();
</script>

<section class="panel connections-panel">
  <div class="panel-header">
    <h2>Your Connections</h2>
    <span class="connection-count">{store.tunnels.length}</span>
  </div>
  <div class="panel-body connections-scroll">
    {#if store.loading}
      <div class="loading-state">
        <div class="spinner"></div>
        <span>Loading connections...</span>
      </div>
    {:else if store.tunnels.length === 0}
      <div class="empty-state">
        <Radio class="empty-state-icon" size={48} />
        <h3>No connections yet</h3>
        <p>Create your first tunnel to share your Foundry VTT world with players.</p>
      </div>
    {:else}
      <div class="connection-list">
        {#each store.tunnels as tunnel, index (tunnel.id)}
          <TunnelCard
            {tunnel}
            {index}
            totalItems={store.tunnels.length}
            onStart={store.start}
            onStop={store.stop}
            {onAction}
            installProgress={store.installProgress[tunnel.provider]}
          />
        {/each}
      </div>
    {/if}
  </div>
</section>

<style>
  .panel {
    background: var(--card-bg);
    border-radius: 12px;
    box-shadow: 0 1px 3px rgba(0,0,0,0.05);
    border: 1px solid var(--border-color);
    display: flex;
    flex-direction: column;
    overflow: hidden;
    transition: box-shadow 0.2s ease, transform 0.2s ease;
    opacity: 0;
    transform: translateY(30px);
    animation: panelIn 0.5s cubic-bezier(0.16, 1, 0.3, 1) forwards;
    animation-delay: 0.2s;
    min-height: 0;
  }

  .panel:hover {
    box-shadow: 0 8px 24px rgba(0,0,0,0.08), 0 2px 6px rgba(0,0,0,0.04);
    transform: translateY(-1px);
  }

  @keyframes panelIn {
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .panel-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 14px 18px;
    border-bottom: 1px solid var(--border-light);
    background: var(--url-bg);
    flex-shrink: 0;
  }

  .panel-header h2 {
    font-family: 'Crimson Pro', Georgia, serif;
    font-size: 17px;
    font-weight: 600;
    color: var(--text-heading);
    margin: 0;
  }

  .connection-count {
    background: linear-gradient(135deg, var(--primary-color) 0%, var(--primary-hover) 100%);
    color: var(--badge-text);
    font-size: 12px;
    font-weight: 600;
    padding: 2px 10px;
    border-radius: 12px;
    box-shadow: 0 2px 4px rgba(146, 64, 14, 0.25);
  }

  .panel-body {
    padding: 18px;
    flex: 1;
    overflow-y: auto;
    min-height: 0;
  }

  .connections-scroll {
    overflow-y: auto;
  }

  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 40px 20px;
    color: var(--text-muted);
    gap: 12px;
  }

  .spinner {
    width: 28px;
    height: 28px;
    border: 2px solid var(--border-color);
    border-top-color: var(--primary-color);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .empty-state {
    text-align: center;
    padding: 40px 16px;
    color: var(--text-muted);
  }

  :global(.empty-state-icon) {
    color: var(--text-muted);
    margin-bottom: 12px;
  }

  .empty-state h3 {
    font-size: 16px;
    color: var(--text-heading);
    margin: 0 0 6px 0;
  }

  .empty-state p {
    margin: 0;
    font-size: 13px;
    line-height: 1.5;
  }

  .connection-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  @media (prefers-reduced-motion: reduce) {
    .panel {
      animation: none;
      opacity: 1;
      transform: none;
    }
  }
</style>
