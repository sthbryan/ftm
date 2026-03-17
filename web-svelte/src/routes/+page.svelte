<script>
  import { onMount, onDestroy } from 'svelte';
  import { useTunnels } from '$lib/stores/tunnels.svelte';
  import TunnelCard from '$lib/components/TunnelCard.svelte';
  
  const store = useTunnels();
  
  let formData = $state({ name: '', provider: 'cloudflared', localPort: 30000 });
  
  const providers = [
    { id: 'cloudflared', name: 'Cloudflared' },
    { id: 'playitgg', name: 'Playit.gg' },
    { id: 'tunnelmole', name: 'Tunnelmole' },
    { id: 'localhostrun', name: 'localhost.run' },
    { id: 'serveo', name: 'Serveo' },
    { id: 'pinggy', name: 'Pinggy' }
  ];
  
  onMount(() => {
    store.connect();
  });
  
  onDestroy(() => {
    store.disconnect();
  });
  
  async function handleSubmit(e) {
    e.preventDefault();
    await store.create(formData);
    formData = { name: '', provider: 'cloudflared', localPort: 30000 };
  }
</script>

<svelte:head>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Crimson+Pro:wght@400;500;600&family=Inter:wght@400;500;600&display=swap" rel="stylesheet">
</svelte:head>

<div class="app">
  <header class="app-header">
    <div class="brand">
      <img src="/favicon.png" alt="Foundry Tunnel Manager" class="d20-badge" />
      <div class="brand-text">
        <h1>Foundry Tunnel Manager</h1>
        <p class="tagline">Share your world with players everywhere</p>
      </div>
    </div>
  </header>

  <main class="app-main">
    <section class="panel create-panel">
      <div class="panel-header">
        <h2>New Connection</h2>
      </div>
      <div class="panel-body">
        <form class="create-form" onsubmit={handleSubmit}>
          <div class="field-group">
            <label for="name">Connection Name</label>
            <input type="text" id="name" bind:value={formData.name} placeholder="e.g. Storm King's Thunder" required autocomplete="off" />
          </div>

          <div class="field-row">
            <div class="field-group">
              <label for="port">Port</label>
              <input type="number" id="port" bind:value={formData.localPort} min="1" max="65535" required />
            </div>
            <div class="field-group">
              <label for="provider">Provider</label>
              <div class="select-wrap">
                <select id="provider" bind:value={formData.provider} required>
                  {#each providers as p}
                    <option value={p.id}>{p.name}</option>
                  {/each}
                </select>
              </div>
            </div>
          </div>

          <button type="submit" class="btn btn-primary btn-full">
            <span>Create Connection</span>
          </button>
        </form>
      </div>
    </section>

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
            <div class="empty-state-icon">📡</div>
            <h3>No connections yet</h3>
            <p>Create your first connection to share your Foundry world with players.</p>
          </div>
        {:else}
          <div class="connection-list">
            {#each store.tunnels as tunnel (tunnel.id)}
              <TunnelCard 
                {tunnel} 
                onStart={store.start}
                onStop={store.stop}
                onDelete={store.delete}
              />
            {/each}
          </div>
        {/if}
      </div>
    </section>
  </main>

  <footer class="app-footer">
    <p>Press <kbd>W</kbd> in your terminal to open this dashboard</p>
  </footer>
</div>

<style>
  :global(body) {
    margin: 0;
    font-family: 'Inter', system-ui, sans-serif;
    background: #fafaf9;
    color: #44403c;
  }

  .app {
    max-width: 1200px;
    margin: 0 auto;
    padding: 32px 24px;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }

  .app-header {
    margin-bottom: 32px;
    padding-bottom: 24px;
    border-bottom: 1px solid #e7e5e4;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .d20-badge {
    width: 56px;
    height: 56px;
    border-radius: 12px;
    object-fit: cover;
  }

  .brand-text h1 {
    font-family: 'Crimson Pro', Georgia, serif;
    font-size: 28px;
    font-weight: 600;
    color: #1c1917;
    margin: 0 0 4px 0;
  }

  .tagline {
    font-size: 14px;
    color: #78716c;
    margin: 0;
  }

  .app-main {
    display: grid;
    grid-template-columns: 380px 1fr;
    gap: 24px;
    flex: 1;
  }

  @media (max-width: 900px) {
    .app-main {
      grid-template-columns: 1fr;
    }
  }

  .panel {
    background: white;
    border-radius: 12px;
    box-shadow: 0 1px 3px rgba(0,0,0,0.05);
    border: 1px solid #e7e5e4;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .panel-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid #f5f5f4;
    background: #fafaf9;
  }

  .panel-header h2 {
    font-family: 'Crimson Pro', Georgia, serif;
    font-size: 18px;
    font-weight: 600;
    color: #1c1917;
    margin: 0;
  }

  .connection-count {
    background: #92400e;
    color: white;
    font-size: 12px;
    font-weight: 600;
    padding: 2px 10px;
    border-radius: 12px;
  }

  .panel-body {
    padding: 20px;
    flex: 1;
    overflow-y: auto;
  }

  .connections-scroll {
    max-height: 600px;
  }

  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 60px 20px;
    color: #78716c;
    gap: 16px;
  }

  .spinner {
    width: 32px;
    height: 32px;
    border: 2px solid #e7e5e4;
    border-top-color: #92400e;
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .empty-state {
    text-align: center;
    padding: 60px 20px;
    color: #78716c;
  }

  .empty-state-icon {
    font-size: 48px;
    margin-bottom: 16px;
  }

  .empty-state h3 {
    font-size: 18px;
    color: #1c1917;
    margin: 0 0 8px 0;
  }

  .empty-state p {
    margin: 0;
    font-size: 14px;
  }

  .connection-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .field-group {
    margin-bottom: 16px;
  }

  .field-row {
    display: grid;
    grid-template-columns: 100px 1fr;
    gap: 12px;
  }

  @media (max-width: 500px) {
    .field-row {
      grid-template-columns: 1fr;
    }
  }

  label {
    display: block;
    font-size: 13px;
    font-weight: 500;
    color: #57534e;
    margin-bottom: 6px;
  }

  input, select {
    width: 100%;
    padding: 10px 12px;
    border: 1px solid #d6d3d1;
    border-radius: 8px;
    font-size: 14px;
    font-family: inherit;
    background: white;
    box-sizing: border-box;
  }

  input:focus, select:focus {
    outline: none;
    border-color: #92400e;
    box-shadow: 0 0 0 3px rgba(146, 64, 14, 0.1);
  }

  .select-wrap {
    position: relative;
  }

  .btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 10px 16px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    border: 1px solid #d6d3d1;
    background: white;
    color: #44403c;
    transition: all 0.15s;
  }

  .btn:hover {
    background: #f5f5f4;
  }

  .btn-primary {
    background: #92400e;
    color: white;
    border-color: #92400e;
  }

  .btn-primary:hover {
    background: #78350f;
    border-color: #78350f;
  }

  .btn-full {
    width: 100%;
    padding: 12px;
  }

  .app-footer {
    margin-top: 32px;
    padding-top: 24px;
    border-top: 1px solid #e7e5e4;
    text-align: center;
  }

  .app-footer p {
    font-size: 13px;
    color: #78716c;
    margin: 0;
  }

  kbd {
    background: #f5f5f4;
    border: 1px solid #d6d3d1;
    border-radius: 4px;
    padding: 2px 6px;
    font-family: inherit;
    font-size: 12px;
  }
</style>
