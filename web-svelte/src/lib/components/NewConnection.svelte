<script>
  import { onMount } from 'svelte';
  import { useTunnels } from '$lib/stores/tunnels.svelte';
  import { useToast } from '$lib/stores/toast.svelte';
  import { useProviders, detectPort } from '$lib/stores/providers.svelte';
  import Dropdown from './Dropdown.svelte';

  const store = useTunnels();
  const toast = useToast();
  const providerStore = useProviders();

  let formData = $state({ name: '', provider: 'cloudflared', localPort: 30000, autoStart: false });

  const providerOptions = $derived(providerStore.providers.map(p => ({
    label: p.name,
    value: p.id
  })));

  const selectedProvider = $derived(providerOptions.find(p => p.value === formData.provider));

  onMount(async () => {
    const detectedPort = await detectPort();
    formData.localPort = detectedPort;
  });

  function selectProvider(option) {
    formData.provider = option.value;
  }

  function toggleAutoStart() {
    formData.autoStart = !formData.autoStart;
  }

  async function handleSubmit(e) {
    e.preventDefault();
    const name = formData.name;
    await store.create({ ...formData });
    formData = { name: '', provider: 'cloudflared', localPort: formData.localPort, autoStart: false };
    toast.success(`Connection "${name}" created`);
  }
</script>

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
          <label for="provider" class="field-label">Provider</label>
          <Dropdown 
            id="provider"
            class="w-full"
            options={providerOptions} 
            onSelect={selectProvider}
            align="left"
            ariaLabel="Select provider"
            label={selectedProvider?.label || 'Select'}
          />
        </div>
      </div>

      <button type="submit" class="btn btn-primary btn-full mt-20">
        Create Connection
      </button>
    </form>
  </div>
</section>

<style>
  .mt-20 {
    margin-top: 20px;
  }
</style>
