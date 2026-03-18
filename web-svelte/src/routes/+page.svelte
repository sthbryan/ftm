<script>
  import { onMount, onDestroy } from 'svelte';
  import { useTunnels } from '$lib/stores/tunnels.svelte';
  import TunnelCard from '$lib/components/TunnelCard.svelte';
  import DeleteModal from '$lib/components/DeleteModal.svelte';
  
  const store = useTunnels();
  
  let formData = $state({ name: '', provider: 'cloudflared', localPort: 30000 });
  let deleteTunnel = $state(null);
  
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
  
  function handleShowDelete(tunnel) {
    deleteTunnel = tunnel;
  }
  
  function handleConfirmDelete() {
    if (deleteTunnel) {
      store.delete(deleteTunnel.id);
      deleteTunnel = null;
    }
  }
  
  function handleCancelDelete() {
    deleteTunnel = null;
  }
</script>

<svelte:head>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Crimson+Pro:wght@400;500;600;700&family=Inter:wght@300;400;500;600&display=swap" rel="stylesheet">
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
            Create Connection
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
            <p>Create your first tunnel to share your Foundry VTT world with players.</p>
          </div>
        {:else}
          <div class="connection-list">
            {#each store.tunnels as tunnel, index (tunnel.id)}
              <TunnelCard 
                {tunnel} 
                {index}
                onStart={store.start}
                onStop={store.stop}
                onDelete={store.delete}
                onShowDelete={handleShowDelete}
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

<DeleteModal 
  show={deleteTunnel !== null} 
  name={deleteTunnel?.name || ''} 
  onConfirm={handleConfirmDelete} 
  onCancel={handleCancelDelete}
/>

<style>
  :global(body) {
    margin: 0;
    font-family: 'Inter', system-ui, sans-serif;
    background: #fbf9f6;
    color: #44403c;
  }

  :global(html, body) {
    overflow: hidden;
  }

  .app {
    max-width: 1200px;
    margin: 0 auto;
    padding: 24px;
    display: flex;
    flex: 1;
    flex-direction: column;
    box-sizing: border-box;
  }

  .app-header {
    margin-bottom: 24px;
    padding-bottom: 20px;
    border-bottom: 1px solid #e7e5e4;
    opacity: 0;
    transform: translateY(-20px);
    animation: headerIn 0.5s cubic-bezier(0.16, 1, 0.3, 1) forwards;
    flex-shrink: 0;
  }

  @keyframes headerIn {
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .d20-badge {
    width: 48px;
    height: 48px;
    border-radius: 12px;
    object-fit: cover;
  }

  .brand-text h1 {
    font-family: 'Crimson Pro', Georgia, serif;
    font-size: 32px;
    font-weight: 700;
    color: #1c1917;
    margin: 0 0 4px 0;
    letter-spacing: -0.01em;
  }

  .tagline {
    font-size: 14px;
    color: #78716c;
    margin: 0;
    font-weight: 500;
  }

  .app-main {
    display: grid;
    grid-template-columns: 360px 1fr;
    gap: 20px;
    flex: 1;
    min-height: 0;
  }

  @media (max-width: 900px) {
    .app-main {
      grid-template-columns: 1fr;
      overflow-y: auto;
    }
    
    :global(html, body) {
      height: auto;
      overflow: auto;
    }
    
    .app {
      height: auto;
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
    transition: box-shadow 0.2s ease, transform 0.2s ease;
    opacity: 0;
    transform: translateY(30px);
    animation: panelIn 0.5s cubic-bezier(0.16, 1, 0.3, 1) forwards;
    min-height: 0;
  }

  .create-panel {
    animation-delay: 0.1s;
  }

  .connections-panel {
    animation-delay: 0.2s;
  }

  @keyframes panelIn {
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .panel:hover {
    box-shadow: 0 8px 24px rgba(0,0,0,0.08), 0 2px 6px rgba(0,0,0,0.04);
    transform: translateY(-1px);
  }

  .panel-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 14px 18px;
    border-bottom: 1px solid #f5f5f4;
    background: #fafaf9;
    flex-shrink: 0;
  }

  .panel-header h2 {
    font-family: 'Crimson Pro', Georgia, serif;
    font-size: 17px;
    font-weight: 600;
    color: #1c1917;
    margin: 0;
  }

  .connection-count {
    background: linear-gradient(135deg, #92400e 0%, #b45309 100%);
    color: white;
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
    color: #78716c;
    gap: 12px;
  }

  .spinner {
    width: 28px;
    height: 28px;
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
    padding: 40px 16px;
    color: #78716c;
  }

  .empty-state-icon {
    font-size: 40px;
    margin-bottom: 12px;
  }

  .empty-state h3 {
    font-size: 16px;
    color: #1c1917;
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

  .field-group {
    margin-bottom: 14px;
  }

  .field-row {
    display: grid;
    grid-template-columns: 90px 1fr;
    gap: 10px;
  }

  @media (max-width: 500px) {
    .field-row {
      grid-template-columns: 1fr;
    }
  }

  label {
    display: block;
    font-size: 12px;
    font-weight: 500;
    color: #57534e;
    margin-bottom: 5px;
  }

  input, select {
    width: 100%;
    padding: 8px 10px;
    border: 1px solid #d6d3d1;
    border-radius: 8px;
    font-size: 13px;
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
    transition: all 0.2s ease;
  }

  .btn:hover {
    background: #f5f5f4;
  }

  .btn-primary {
    background: linear-gradient(135deg, #92400e 0%, #b45309 100%);
    color: white;
    border-color: #92400e;
    box-shadow: 0 4px 12px rgba(146, 64, 14, 0.3), 0 2px 4px rgba(146, 64, 14, 0.2);
  }

  .btn-primary:hover {
    background: linear-gradient(135deg, #78350f 0%, #92400e 100%);
    border-color: #78350f;
    box-shadow: 0 6px 16px rgba(146, 64, 14, 0.35), 0 3px 6px rgba(146, 64, 14, 0.25);
    transform: translateY(-1px);
  }

  .btn-full {
    width: 100%;
    padding: 12px;
  }

  .app-footer {
    margin-top: 16px;
    padding-top: 16px;
    border-top: 1px solid #e7e5e4;
    text-align: center;
    flex-shrink: 0;
  }

  .app-footer p {
    font-size: 12px;
    color: #78716c;
    margin: 0;
  }

  kbd {
    background: #f5f5f4;
    border: 1px solid #d6d3d1;
    border-radius: 4px;
    padding: 2px 6px;
    font-family: inherit;
    font-size: 11px;
  }

  @media (prefers-reduced-motion: reduce) {
    .app-header,
    .panel {
      animation: none;
      opacity: 1;
      transform: none;
    }
  }
</style>
