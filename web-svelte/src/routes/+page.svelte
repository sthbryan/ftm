<script>
  import { onMount, onDestroy } from 'svelte';
  import { useTunnels } from '$lib/stores/tunnels.svelte';
  import TunnelCard from '$lib/components/TunnelCard.svelte';
  
  const store = useTunnels();
  
  let showForm = $state(false);
  let formData = $state({ name: '', provider: 'cloudflared', localPort: 30000 });
  
  const providers = [
    { id: 'cloudflared', name: 'Cloudflare' },
    { id: 'playitgg', name: 'Playit.gg' },
    { id: 'localhostrun', name: 'localhost.run' },
    { id: 'pinggy', name: 'Pinggy' },
    { id: 'tunnelmole', name: 'Tunnelmole' },
    { id: 'zrok', name: 'Zrok' },
    { id: 'localtunnel', name: 'localtunnel' }
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
    showForm = false;
    formData = { name: '', provider: 'cloudflared', localPort: 30000 };
  }
</script>

<header>
  <div class="brand">
    <img src="/favicon.png" alt="" class="logo" />
    <div>
      <h1>Foundry Tunnel Manager</h1>
      <p>Share your world with players everywhere</p>
    </div>
  </div>
  <button class="add-btn" onclick={() => showForm = !showForm}>
    {showForm ? 'Cancel' : '+ New Connection'}
  </button>
</header>

{#if showForm}
  <form class="create-form" onsubmit={handleSubmit}>
    <h2>New Connection</h2>
    <div class="field">
      <label for="name">World Name</label>
      <input type="text" id="name" bind:value={formData.name} required placeholder="e.g. Storm King's Thunder" />
    </div>
    <div class="field-row">
      <div class="field">
        <label for="port">Port</label>
        <input type="number" id="port" bind:value={formData.localPort} min="1" max="65535" />
      </div>
      <div class="field">
        <label for="provider">Provider</label>
        <select id="provider" bind:value={formData.provider}>
          {#each providers as p}
            <option value={p.id}>{p.name}</option>
          {/each}
        </select>
      </div>
    </div>
    <button type="submit" class="submit-btn">Create Connection</button>
  </form>
{/if}

<main>
  {#if store.loading}
    <div class="loading">Loading connections...</div>
  {:else if store.tunnels.length === 0}
    <div class="empty">
      <span class="icon">📡</span>
      <h3>No connections yet</h3>
      <p>Create your first connection to share your Foundry world.</p>
    </div>
  {:else}
    <div class="count">
      {store.tunnels.length} connection{store.tunnels.length === 1 ? '' : 's'}
    </div>
    {#each store.tunnels as tunnel (tunnel.id)}
      <TunnelCard 
        {tunnel} 
        onStart={store.start}
        onStop={store.stop}
        onDelete={store.delete}
      />
    {/each}
  {/if}
</main>

<style>
  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 32px;
    padding-bottom: 24px;
    border-bottom: 1px solid #e7e5e4;
  }
  
  .brand {
    display: flex;
    align-items: center;
    gap: 16px;
  }
  
  .logo {
    width: 48px;
    height: 48px;
    border-radius: 8px;
  }
  
  h1 {
    font-size: 24px;
    font-weight: 600;
    color: #1c1917;
    margin: 0;
  }
  
  .brand p {
    color: #78716c;
    font-size: 14px;
    margin: 0;
  }
  
  .add-btn {
    background: #92400e;
    color: white;
    border: none;
    padding: 12px 20px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }
  
  .add-btn:hover {
    background: #78350f;
  }
  
  .create-form {
    background: white;
    padding: 24px;
    border-radius: 12px;
    margin-bottom: 24px;
    box-shadow: 0 2px 8px rgba(0,0,0,0.05);
  }
  
  .create-form h2 {
    font-size: 18px;
    margin-bottom: 20px;
    color: #1c1917;
  }
  
  .field {
    margin-bottom: 16px;
  }
  
  .field-row {
    display: grid;
    grid-template-columns: 1fr 2fr;
    gap: 16px;
  }
  
  label {
    display: block;
    font-size: 14px;
    font-weight: 500;
    color: #44403c;
    margin-bottom: 6px;
  }
  
  input, select {
    width: 100%;
    padding: 10px 12px;
    border: 1px solid #d6d3d1;
    border-radius: 6px;
    font-size: 14px;
    font-family: inherit;
  }
  
  input:focus, select:focus {
    outline: none;
    border-color: #92400e;
  }
  
  .submit-btn {
    background: #16a34a;
    color: white;
    border: none;
    padding: 12px 24px;
    border-radius: 6px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    width: 100%;
  }
  
  .submit-btn:hover {
    background: #15803d;
  }
  
  .loading, .empty {
    text-align: center;
    padding: 60px 20px;
    color: #78716c;
  }
  
  .empty .icon {
    font-size: 48px;
    display: block;
    margin-bottom: 16px;
  }
  
  .empty h3 {
    color: #1c1917;
    font-size: 18px;
    margin-bottom: 8px;
  }
  
  .count {
    font-size: 14px;
    color: #78716c;
    margin-bottom: 16px;
  }
</style>
