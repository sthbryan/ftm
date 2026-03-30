<script lang="ts">
  import { Plus } from "lucide-svelte";
  import { animate, spring } from "motion";
  import { onMount } from "svelte";
  import { useProviders, detectPort } from "$lib/stores/providers.svelte";
  import { useToast } from "$lib/stores/toast.svelte";
  import { useTunnels } from "$lib/stores/tunnels.svelte";
  import Button from "./Button.svelte";
  import Dropdown from "./Dropdown.svelte";
  import type { DropdownOption } from "$lib/types";

  const store = useTunnels();
  const toast = useToast();
  const providerStore = useProviders();

  let sectionEl: HTMLElement | undefined = $state();
  let headerEl: HTMLElement | undefined = $state();
  let contentEl: HTMLElement | undefined = $state();

  let formData = $state({
    name: "",
    provider: "cloudflared",
    localPort: 30000,
  });

  const providerOptions: DropdownOption[] = $derived(
    providerStore.providers.map((p) => ({
      label: p.name,
      value: p.id,
    })),
  );

  const selectedProvider = $derived(
    providerOptions.find((p) => p.value === formData.provider),
  );

  onMount(async () => {
    const detectedPort = await detectPort();
    formData.localPort = detectedPort;

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

  function selectProvider(option: DropdownOption) {
    if (option.value) formData.provider = option.value;
  }

  async function handleSubmit(e: Event) {
    e.preventDefault();
    const name = formData.name;
    await store.create({ ...formData });
    const detectedPort = await detectPort({ forceRefresh: true });
    formData = {
      name: "",
      provider: "cloudflared",
      localPort: detectedPort,
    };
    toast.success(`Connection "${name}" created`);
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
    class="mb-5"
  >
    <h2
      class="text-base font-semibold text-text-heading flex items-center gap-2"
    >
      New Connection
    </h2>
  </div>
  <div bind:this={contentEl} style="opacity: 0; transform: translateY(8px);">
    <form onsubmit={handleSubmit}>
      <div class="mb-4">
        <label
          for="name"
          class="block text-xs font-medium mb-1.5 text-text-muted"
          >Connection Name</label
        >
        <input
          type="text"
          id="name"
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
            for="port"
            class="block text-xs font-medium mb-1.5 text-text-muted">Port</label
          >
          <input
            type="number"
            id="port"
            bind:value={formData.localPort}
            min="1"
            max="65535"
            required
            class="w-full px-3 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-offset-1 bg-input-bg text-text border-border focus:ring-primary transition-all duration-200"
          />
        </div>
        <div class="mb-4">
          <label
            for="provider"
            class="block text-xs font-medium mb-1.5 text-text-muted"
            >Provider</label
          >
          <Dropdown
            id="provider"
            class="w-full"
            options={providerOptions}
            onSelect={selectProvider}
            align="left"
            ariaLabel="Select provider"
            label={selectedProvider?.label || "Select"}
          />
        </div>
      </div>
      <div class="mt-5">
        <Button
          variant="primary"
          size="lg"
          type="submit"
          class="w-full"
          icon={Plus}>Create Connection</Button
        >
      </div>
    </form>
  </div>
</section>
