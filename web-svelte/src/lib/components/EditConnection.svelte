<script>
  import { useTunnels } from '$lib/stores/tunnels.svelte';
  import { useToast } from '$lib/stores/toast.svelte';
  import { useProviders } from '$lib/stores/providers.svelte';
  import Dropdown from './Dropdown.svelte';

  let { tunnelId, onCancel, onSaved } = $props();

  const store = useTunnels();
  const toast = useToast();
  const providerStore = useProviders();

  let formData = $state({ 
    name: '', 
    provider: 'cloudflared', 
    localPort: 30000,
  });

  let currentTunnelId = '';

  const providerOptions = $derived(providerStore.providers.map(p => ({
    label: p.name,
    value: p.id
  })));

  const selectedProvider = $derived(providerOptions.find(p => p.value === formData.provider));

  $effect(() => {
    
    const tunnels = store.tunnels;

    if (!tunnelId) {
      return;
    }

    const tunnel = store.getById(tunnelId);
    if (!tunnel) {
      
      return;
    }

    if (currentTunnelId === tunnelId) {
      
      return;
    }

    currentTunnelId = tunnelId;
    formData = {
      name: tunnel.name || '',
      provider: tunnel.provider || 'cloudflared',
      localPort: tunnel.port || 30000,
    };
  });

  function selectProvider(option) {
    formData.provider = option.value;
  }

  async function handleSubmit(e) {
    e.preventDefault();
    
    try {
      await store.update(tunnelId, {
        name: formData.name,
        provider: formData.provider,
        localPort: formData.localPort,
      });
      toast.success(`Connection "${formData.name}" updated`);
      onSaved?.();
    } catch (err) {
      toast.error(`Failed to update: ${err.message}`);
    }
  }

  function handleCancel() {
    onCancel?.();
  }
</script>

<section class="panel edit-panel">
  <div class="panel-header">
    <h2>Edit Connection</h2>
    <button type="button" class="btn-cancel" onclick={handleCancel}>✕</button>
  </div>
  <div class="panel-body">
    <form class="edit-form" onsubmit={handleSubmit}>
      <div class="field-group">
        <label for="edit-name">Connection Name</label>
        <input type="text" id="edit-name" bind:value={formData.name} placeholder="e.g. Storm King's Thunder" required autocomplete="off" />
      </div>

      <div class="field-row">
        <div class="field-group">
          <label for="edit-port">Port</label>
          <input type="number" id="edit-port" bind:value={formData.localPort} min="1" max="65535" required />
        </div>
        <div class="field-group">
          <label for="edit-provider" class="field-label">Provider</label>
          <Dropdown 
            id="edit-provider"
            class="w-full"
            options={providerOptions} 
            onSelect={selectProvider}
            align="left"
            ariaLabel="Select provider"
            label={selectedProvider?.label || 'Select'}
          />
        </div>
      </div>

      <div class="button-row">
        <button type="button" class="btn btn-secondary" onclick={handleCancel}>
          Cancel
        </button>
        <button type="submit" class="btn btn-primary btn-full">
          Save Changes
        </button>
      </div>
    </form>
  </div>
</section>

<style>
  .btn-cancel {
    background: none;
    border: none;
    font-size: 18px;
    color: var(--text-muted);
    cursor: pointer;
    padding: 4px 8px;
    border-radius: 4px;
    transition: all 0.15s;
  }

  .btn-cancel:hover {
    background: var(--hover-bg);
    color: var(--text-color);
  }

  .button-row {
    display: flex;
    gap: 10px;
    margin-top: 20px;
  }

  .btn-full {
    flex: 1;
  }
</style>
