<script lang="ts">
  import { X } from "lucide-svelte";
  import { animate, spring } from "motion";
  import { onMount } from "svelte";
  import { useProviders, detectPort } from "$lib/stores/providers.svelte";
  import { useToast } from "$lib/stores/toast.svelte";
  import { useTunnels } from "$lib/stores/tunnels.svelte";
  import Button from "./Button.svelte";
  import Dropdown from "./Dropdown.svelte";
  import type { DropdownOption } from "$lib/types";

  let { tunnelId, onCancel, onSaved } = $props();

  let sectionEl: HTMLElement | undefined = $state();
  let headerEl: HTMLElement | undefined = $state();
  let contentEl: HTMLElement | undefined = $state();

  let formData = $state({
    name: "",
    provider: "cloudflared",
    localPort: 30000,
  });
  let currentTunnelId = "";

  const providerOptions: DropdownOption[] = $derived(
    useProviders().providers.map((p) => ({
      label: p.name,
      value: p.id,
    })),
  );

  const selectedProvider = $derived(
    providerOptions.find((p) => p.value === formData.provider),
  );

  onMount(() => {
    requestAnimationFrame(() => {
      if (sectionEl)
        animate(
          sectionEl,

          { opacity: 1, y: 0 },
          { duration: 0.4, type: "spring" },
        );
      if (headerEl)
        animate(
          headerEl,
          { opacity: 1, y: 0 },
          { duration: 0.4, delay: 0.05, type: "spring" },
        );
      if (contentEl)
        animate(
          contentEl,
          { opacity: 1, y: 0 },
          { duration: 0.4, delay: 0.1, type: "spring" },
        );
    });
  });

  $effect(() => {
    const store = useTunnels();
    if (!tunnelId) return;

    const tunnel = store.getById(tunnelId);
    if (!tunnel || currentTunnelId === tunnelId) return;

    currentTunnelId = tunnelId;
    formData = {
      name: tunnel.name || "",
      provider: tunnel.provider || "cloudflared",
      localPort: tunnel.port || 30000,
    };

    if (!tunnel.port) {
      detectPort().then((port) => {
        if (currentTunnelId === tunnelId) {
          formData = { ...formData, localPort: port };
        }
      });
    }
  });

  function selectProvider(option: DropdownOption) {
    if (option.value) formData.provider = option.value;
  }

  async function handleSubmit(e: Event) {
    e.preventDefault();
    const store = useTunnels();
    const toast = useToast();

    try {
      await store.update(tunnelId, {
        name: formData.name,
        provider: formData.provider,
        localPort: formData.localPort,
      });
      toast.success(`Connection "${formData.name}" updated`);
      onSaved?.();
    } catch (err) {
      toast.error(`Failed to update: ${(err as Error).message}`);
    }
  }
</script>

<section
  bind:this={sectionEl}
  style="opacity: 0; transform: translateY(12px);"
  class="rounded-xl p-5 bg-card border border-border"
>
  <div
    bind:this={headerEl}
    style="opacity: 0; transform: translateY(8px);"
    class="flex items-center justify-between mb-5"
  >
    <h2
      class="text-base font-semibold text-text-heading flex items-center gap-2"
    >
      Edit Connection
    </h2>
    <button
      type="button"
      onclick={onCancel}
      class="p-1 rounded text-lg bg-transparent border-none text-text-muted cursor-pointer transition-colors hover:bg-hover hover:text-text"
    >
      <X size={18} />
    </button>
  </div>
  <div bind:this={contentEl} style="opacity: 0; transform: translateY(8px);">
    <form onsubmit={handleSubmit}>
      <div class="mb-4">
        <label
          for="edit-name"
          class="block text-xs font-medium mb-1.5 text-text-muted"
          >Connection Name</label
        >
        <input
          type="text"
          id="edit-name"
          bind:value={formData.name}
          placeholder="e.g. Storm King's Thunder"
          required
          autocomplete="off"
          class="w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-offset-1 bg-input-bg text-text border-border focus:ring-primary transition-all duration-200"
        />
      </div>
      <div class="grid grid-cols-2 gap-4">
        <div class="mb-4">
          <label
            for="edit-port"
            class="block text-xs font-medium mb-1.5 text-text-muted">Port</label
          >
          <input
            type="number"
            id="edit-port"
            bind:value={formData.localPort}
            min="1"
            max="65535"
            required
            class="w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-offset-1 bg-input-bg text-text border-border focus:ring-primary transition-all duration-200"
          />
        </div>
        <div class="mb-4">
          <label
            for="edit-provider"
            class="block text-xs font-medium mb-1.5 text-text-muted"
            >Provider</label
          >
          <Dropdown
            id="edit-provider"
            class="w-full"
            options={providerOptions}
            onSelect={selectProvider}
            align="left"
            ariaLabel="Select provider"
            label={selectedProvider?.label || "Select"}
          />
        </div>
      </div>
      <div class="flex gap-3 mt-5">
        <Button
          variant="default"
          size="lg"
          type="button"
          onclick={onCancel}
          class="flex-1">Cancel</Button
        >
        <Button variant="primary" size="lg" type="submit" class="flex-1"
          >Save</Button
        >
      </div>
    </form>
  </div>
</section>
