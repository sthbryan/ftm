<script>
  import { onMount, onDestroy } from 'svelte';
  import { useTunnels } from '$lib/stores/tunnels.svelte';
  import { useToast } from '$lib/stores/toast.svelte';
  import { useProviders, detectPort } from '$lib/stores/providers.svelte';
  import { useTheme } from '$lib/stores/theme.svelte';
  import TunnelCard from '$lib/components/TunnelCard.svelte';
  import DeleteModal from '$lib/components/DeleteModal.svelte';
  import Toasts from '$lib/components/Toasts.svelte';
  import ThemeSwitcher from '$lib/components/ThemeSwitcher.svelte';
  
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

<div class="app">
  <header class="app-header">
    <div class="brand">
      <img src="/favicon.png" alt="Foundry Tunnel Manager" class="d20-badge" />
      <div class="brand-text">
        <h1>Foundry Tunnel Manager</h1>
        <p class="tagline">Share your world with players everywhere</p>
      </div>
    </div>
    <ThemeSwitcher />
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

<Toasts />

<style>
  :root {
    --bg-color: #fbf9f6;
    --card-bg: #ffffff;
    --text-color: #44403c;
    --text-heading: #1c1917;
    --text-muted: #78716c;
    --border-color: #e7e5e4;
    --border-light: #f5f5f4;
    --primary-color: #92400e;
    --primary-hover: #78350f;
    --hover-bg: #f5f5f4;
    --status-running-bg: #dcfce7;
    --status-running-text: #166534;
    --status-starting-bg: #fef3c7;
    --status-starting-text: #92400e;
    --status-installing-bg: #dbeafe;
    --status-installing-text: #1e40af;
    --status-error-bg: #fee2e2;
    --status-error-text: #991b1b;
    --status-stopped-bg: #f5f5f4;
    --status-stopped-text: #78716c;
    --logs-bg: #1c1917;
    --logs-text: #d6d3d1;
    --url-bg: #fafaf9;
    --url-text: #92400e;
  }

  :root[data-theme="dark"] {
    --bg-color: #0c0a09;
    --card-bg: #1c1917;
    --text-color: #d6d3d1;
    --text-heading: #fafaf9;
    --text-muted: #a8a29e;
    --border-color: #292524;
    --border-light: #1c1917;
    --primary-color: #fbbf24;
    --primary-hover: #f59e0b;
    --hover-bg: #292524;
    --status-running-bg: #14532d;
    --status-running-text: #86efac;
    --status-starting-bg: #713f12;
    --status-starting-text: #fcd34d;
    --status-installing-bg: #1e3a8a;
    --status-installing-text: #93c5fd;
    --status-error-bg: #7f1d1d;
    --status-error-text: #fca5a5;
    --status-stopped-bg: #292524;
    --status-stopped-text: #a8a29e;
    --logs-bg: #0c0a09;
    --logs-text: #d6d3d1;
    --url-bg: #292524;
    --url-text: #fbbf24;
  }

  :root[data-theme="sepia"] {
    --bg-color: #f4ecd8;
    --card-bg: #fdf6e3;
    --text-color: #433422;
    --text-heading: #2d2416;
    --text-muted: #8b7355;
    --border-color: #d4c5a9;
    --border-light: #e8dcc8;
    --primary-color: #8b4513;
    --primary-hover: #654321;
    --hover-bg: #e8dcc8;
    --status-running-bg: #d4edda;
    --status-running-text: #155724;
    --status-starting-bg: #fff3cd;
    --status-starting-text: #856404;
    --status-installing-bg: #cce5ff;
    --status-installing-text: #004085;
    --status-error-bg: #f8d7da;
    --status-error-text: #721c24;
    --status-stopped-bg: #e8dcc8;
    --status-stopped-text: #8b7355;
    --logs-bg: #2d2416;
    --logs-text: #e8dcc8;
    --url-bg: #e8dcc8;
    --url-text: #8b4513;
  }

  :root[data-theme="red"] {
    --bg-color: #fff6f6;
    --card-bg: #ffffff;
    --text-color: #2b0b0b;
    --text-heading: #1b0a09;
    --text-muted: #6b2626;
    --border-color: #f2dede;
    --border-light: #fff6f6;
    --primary-color: #dc2626;
    --primary-hover: #b91c1c;
    --hover-bg: #fff2f2;
    --status-running-bg: #fee2e2;
    --status-running-text: #7f1d1d;
    --status-starting-bg: #fff2e6;
    --status-starting-text: #b45309;
    --status-installing-bg: #dbeafe;
    --status-installing-text: #1e40af;
    --status-error-bg: #fee2e2;
    --status-error-text: #7f1d1d;
    --status-stopped-bg: #fff6f6;
    --status-stopped-text: #6b2626;
    --logs-bg: #2b0b0b;
    --logs-text: #f7eaea;
    --url-bg: #fff6f6;
    --url-text: #dc2626;
  }

  :root[data-theme="blue"] {
    --bg-color: #f3f8ff;
    --card-bg: #ffffff;
    --text-color: #0b1220;
    --text-heading: #071029;
    --text-muted: #40577a;
    --border-color: #e6f0ff;
    --border-light: #f3f8ff;
    --primary-color: #2563eb;
    --primary-hover: #1d4ed8;
    --hover-bg: #eef6ff;
    --status-running-bg: #e6f8ef;
    --status-running-text: #0b5132;
    --status-starting-bg: #fff7e6;
    --status-starting-text: #d97706;
    --status-installing-bg: #e6f0ff;
    --status-installing-text: #1e3a8a;
    --status-error-bg: #ffeef0;
    --status-error-text: #9b1b1b;
    --status-stopped-bg: #f3f8ff;
    --status-stopped-text: #40577a;
    --logs-bg: #071029;
    --logs-text: #eaf3ff;
    --url-bg: #eef6ff;
    --url-text: #2563eb;
  }

  :root[data-theme="dracula"] {
    --bg-color: #282a36;
    --card-bg: #3b3a4a;
    --text-color: #f8f8f2;
    --text-heading: #f8f8f2;
    --text-muted: #b8bfd9;
    --border-color: #454655;
    --border-light: #2d2f39;
    --primary-color: #ff79c6;
    --primary-hover: #ff6bb0;
    --hover-bg: #3e3f4b;
    --status-running-bg: #50fa7b;
    --status-running-text: #0b2b12;
    --status-starting-bg: #f1fa8c;
    --status-starting-text: #2b2b0b;
    --status-installing-bg: #8be9fd;
    --status-installing-text: #062033;
    --status-error-bg: #ff6b6b;
    --status-error-text: #2b0b0b;
    --status-stopped-bg: #3b3a4a;
    --status-stopped-text: #e6e6f0;
    --logs-bg: #23232c;
    --logs-text: #e6e6e9;
    --url-bg: #2f3139;
    --url-text: #8be9fd;
    --input-bg: #2b2d36;
    --input-text: #f8f8f2;
    --input-placeholder: #9aa0c7;
    --input-disabled: #3b3f55;
  }

  :root[data-theme="contrast"] {
    --bg-color: #000000;
    --card-bg: #000000;
    --text-color: #ffffff;
    --text-heading: #ffffff;
    --text-muted: #cccccc;
    --border-color: #ffffff;
    --border-light: #333333;
    --primary-color: #ffff00;
    --primary-hover: #ffff00;
    --hover-bg: #333333;
    --status-running-bg: #00ff00;
    --status-running-text: #000000;
    --status-starting-bg: #ffff00;
    --status-starting-text: #000000;
    --status-installing-bg: #00ffff;
    --status-installing-text: #000000;
    --status-error-bg: #ff0000;
    --status-error-text: #ffffff;
    --status-stopped-bg: #333333;
    --status-stopped-text: #ffffff;
    --logs-bg: #000000;
    --logs-text: #00ff00;
    --url-bg: #333333;
    --url-text: #ffff00;
  }

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

  .app-header {
    margin-bottom: 24px;
    padding-bottom: 20px;
    border-bottom: 1px solid var(--border-color);
    opacity: 0;
    transform: translateY(-20px);
    animation: headerIn 0.5s cubic-bezier(0.16, 1, 0.3, 1) forwards;
    flex-shrink: 0;
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
    line-height: normal;
    font-weight: 700;
    color: var(--text-heading);
    margin: 0 0 4px 0;
    letter-spacing: -0.01em;
  }

  .tagline {
    font-size: 14px;
    color: var(--text-muted);
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
    
    .brand-text h1 {
      font-size: 24px;
    }
    
    .tagline {
      font-size: 13px;
    }
    
    .d20-badge {
      width: 40px;
      height: 40px;
    }
    
    .app-header {
      margin-bottom: 16px;
      padding-bottom: 16px;
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

  .app-footer {
    margin-top: 16px;
    padding-top: 16px;
    border-top: 1px solid #e7e5e4;
    text-align: center;
    flex-shrink: 0;
  }

  .app-footer p {
    font-size: 12px;
    color: var(--text-muted);
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
