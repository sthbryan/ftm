<script>
  import { onMount, onDestroy } from 'svelte';
  import { useTunnels } from '$lib/stores/tunnels.svelte';
  import { useToast } from '$lib/stores/toast.svelte';
  import { useProviders } from '$lib/stores/providers.svelte';
  import { useTheme } from '$lib/stores/theme.svelte';
  import Header from '$lib/components/Header.svelte';
  import Footer from '$lib/components/Footer.svelte';
  import DeleteModal from '$lib/components/DeleteModal.svelte';
  import Toasts from '$lib/components/Toasts.svelte';
  import ConnectionsPanel from '$lib/components/ConnectionsPanel.svelte';
  import NewConnection from '$lib/components/NewConnection.svelte';
  import EditConnection from '$lib/components/EditConnection.svelte';
  import '$lib/styles/components.css';
  
  const store = useTunnels();
  const toast = useToast();
  const providerStore = useProviders();
  const theme = useTheme();

  let deleteTunnel = $state(null);
  let editingTunnelId = $state(null);

  onMount(async () => {
    theme.init();
    store.connect();
    providerStore.fetch();
  });
  
  onDestroy(() => {
    store.disconnect();
  });
  
  function handleAction(action, data) {
    switch (action) {
      case 'edit':
        editingTunnelId = data;
        break;
      case 'delete':
        deleteTunnel = data;
        break;
    }
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

  function handleEditCancel() {
    editingTunnelId = null;
  }

  function handleEditSaved() {
    editingTunnelId = null;
  }
</script>

<svelte:head>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Crimson+Pro:wght@400;500;600;700&family=Inter:wght@300;400;500;600&display=swap" rel="stylesheet">
</svelte:head>

<div class="app">
  <Header />

  <main class="app-main">
    {#if editingTunnelId}
      <EditConnection 
        tunnelId={editingTunnelId} 
        onCancel={handleEditCancel}
        onSaved={handleEditSaved}
      />
    {:else}
      <NewConnection />
    {/if}

    <ConnectionsPanel onAction={handleAction} />
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
  }
</style>
