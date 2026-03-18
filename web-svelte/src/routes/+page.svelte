<script>
  import { onMount, onDestroy } from 'svelte';
  import { useTunnels } from '$lib/stores/tunnels.svelte';
  import { useToast } from '$lib/stores/toast.svelte';
  import { useProviders, detectPort } from '$lib/stores/providers.svelte';
  import { useTheme } from '$lib/stores/theme.svelte';
  import Header from '$lib/components/Header.svelte';
  import Footer from '$lib/components/Footer.svelte';
  import TunnelCard from '$lib/components/TunnelCard.svelte';
  import DeleteModal from '$lib/components/DeleteModal.svelte';
  import Toasts from '$lib/components/Toasts.svelte';
  
  const store = useTunnels();
  const toast = useToast();
  const providerStore = useProviders();
  const theme = useTheme();
  
  let formData = $state({ name: '', provider: 'cloudflared', localPort: 30000 });
  let deleteTunnel = $state(null);
  
  onMount(async () => {
    theme.init();
    store.connect();
    providerStore.fetch();
    const detectedPort = await detectPort();
    formData.localPort = detectedPort;
  });
  
  onDestroy(() => {
    store.disconnect();
  });
  
  async function handleSubmit(e) {
    e.preventDefault();
    const name = formData.name;
    await store.create(formData);
    formData = { name: '', provider: 'cloudflared', localPort: 30000 };
    toast.success(`Connection "${name}" created`);
  }
  
  function handleShowDelete(tunnel) {
    deleteTunnel = tunnel;
  }
  
  function handleConfirmDelete() {
    if (deleteTunnel) {
      const name = deleteTunnel.name;
      store.delete(deleteTunnel.id);
      toast.success(`Connection "${name}" deleted`);
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

<link rel="stylesheet" href="/themes.css" />

<div class="app">
  <Header />

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
                  {#each providerStore.providers as p}
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
                installProgress={store.installProgress[tunnel.provider]}
              />
            {/each}
          </div>
        {/if}
      </div>
    </section>
  </main>

  <Footer />
</div>

<DeleteModal 
  show={deleteTunnel !== null} 
  name={deleteTunnel?.name || ''} 
  onConfirm={handleConfirmDelete} 
  onCancel={handleCancelDelete}
/>

<Toasts />

<style>
  :global(body) {
    margin: 0;
    font-family: 'Inter', system-ui, sans-serif;
    background: var(--bg-color);
    color: var(--text-color);
    transition: background-color 0.3s, color 0.3s;
  }

  :global(html, body) {
    overflow: hidden;
  }

  .app {
    max-width: 1200px;
    margin: 0 auto;
    display: flex;
    flex: 1;
    flex-direction: column;
    box-sizing: border-box;
  }

  /* Inputs and form elements - ensure readable in dark themes */
  .create-form input,
  .create-form select {
    width: 100%;
    padding: 10px 12px;
    border-radius: 8px;
    border: 1px solid var(--border-color);
    background: var(--card-bg);
    color: var(--text-color);
    font-size: 14px;
    outline: none;
    transition: box-shadow 0.15s, border-color 0.15s;
  }

  .create-form input::placeholder {
    color: var(--input-placeholder, var(--text-muted));
  }

  .create-form input:disabled,
  .create-form select:disabled {
    background: var(--input-disabled, var(--hover-bg));
    color: var(--text-muted);
  }

  @keyframes headerIn {
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .app-main {
    display: grid;
    grid-template-columns: 360px 1fr;
    gap: 20px;
    flex: 1;
    min-height: 0;
  }

  @media (max-width: 1024px) and (min-width: 768px) {
    .app-main {
      grid-template-columns: 320px 1fr;
      gap: 16px;
    }
    
    .panel-body {
      padding: 14px;
    }
  }

  @media (max-width: 767px) {
    .app-main {
      grid-template-columns: 1fr;
      overflow-y: auto;
      gap: 16px;
    }
    
    :global(html, body) {
      height: auto;
      overflow: auto;
    }
    
    .app {
      height: auto;
      padding: 16px;
    }
    
    .panel-header {
      padding: 12px 16px;
    }
    
    .panel-body {
      padding: 16px;
    }
  }

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

  .empty-state-icon {
    font-size: 40px;
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

  .field-group {
    margin-bottom: 14px;
  }

  .field-group input {
    height: 42px;
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
    color: var(--text-muted);
    margin-bottom: 5px;
  }

  input, select {
    width: 100%;
    padding: 8px 10px;
    border: 1px solid var(--border-color);
    border-radius: 8px !important;
    font-size: 13px;
    font-family: inherit;
    background: var(--card-bg);
    box-sizing: border-box;
  }

  input:focus, select:focus {
    outline: none;
    border-color: var(--primary-color);
    box-shadow: 0 0 0 3px rgba(0,0,0,0.08);
  }

  .select-wrap {
    position: relative;
  }

  .select-wrap select {
    height: 42px;
    appearance: none;
    cursor: pointer;
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
    background: var(--card-bg);
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

  @media (prefers-reduced-motion: reduce) {
    .panel {
      animation: none;
      opacity: 1;
      transform: none;
    }
  }
</style>
