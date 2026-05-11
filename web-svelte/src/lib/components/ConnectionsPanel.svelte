<script lang="ts">
  import { Radio } from "lucide-svelte";
  import { animate, spring } from "motion";
  import { onMount } from "svelte";
  import { useTunnels } from "$lib/stores/tunnels.svelte";
  import { translate } from "$lib/i18n";
  import { cn } from "$lib/utils/cn";
  import TunnelCard from "./TunnelCard.svelte";

  let { onAction }: { onAction: (action: string, data: unknown) => void } =
    $props();

  const store = useTunnels();
  let t = $derived($translate);

  let sectionEl: HTMLElement | undefined = $state();
  let headerEl: HTMLElement | undefined = $state();
  let contentEl: HTMLElement | undefined = $state();

  onMount(() => {
    requestAnimationFrame(() => {
      if (sectionEl)
        animate(sectionEl, { opacity: 1 }, { duration: 0.4, type: "spring" });
      if (headerEl)
        animate(
          headerEl,
          { opacity: 1 },
          { duration: 0.4, delay: 0.05, type: "spring" },
        );
      if (contentEl)
        animate(
          contentEl,
          { opacity: 1 },
          { duration: 0.4, delay: 0.1, type: "spring" },
        );
    });
  });
</script>

<section
  bind:this={sectionEl}
  style="opacity: 0;"
  class={cn(
    "flex flex-col overflow-hidden rounded-3xl border shadow-sm transition-all duration-200",
    "bg-card border-border",
  )}
>
  <div
    bind:this={headerEl}
    style="opacity: 0;"
    class={cn(
      "flex shrink-0 items-center justify-between px-4 py-3 border-b",
      "bg-url-bg border-border-light",
    )}
  >
    <h2
      class="m-0 text-base font-semibold font-serif text-text-heading flex items-center gap-2"
    >
      {t("connections")}
    </h2>
    <span
      class="rounded-full px-2.5 py-0.5 text-xs font-semibold shadow-sm text-btn-text bg-primary"
    >
      {store.tunnels.length}
    </span>
  </div>
  <div
    bind:this={contentEl}
    style="opacity: 0;"
    class="flex-1 overflow-y-auto p-4"
  >
    {#if store.loading}
      <div
        class="flex flex-col items-center justify-center gap-3 py-10 text-text-muted"
      >
        <div
          class="h-7 w-7 animate-spin rounded-full border-2 border-border border-t-primary"
        ></div>
        <span>{t("loading")}</span>
      </div>
    {:else if store.tunnels.length === 0}
      <div class="py-10 text-center text-text-muted">
        <Radio class="mx-auto mb-3 h-12 w-12" size={48} />
        <h3 class="mb-1.5 mt-0 text-base text-text-heading">
          {t("no_tunnels")}
        </h3>
        <p class="m-0 text-sm leading-relaxed">
          {t("tunnels_desc")}
        </p>
      </div>
    {:else}
      <div class="flex flex-col gap-2.5">
        {#each store.tunnels as tunnel, index (tunnel.id)}
          <TunnelCard
            {tunnel}
            {index}
            totalItems={store.tunnels.length}
            onStart={store.start}
            onStop={store.stop}
            {onAction}
            installProgress={store.installProgress[
              tunnel.provider as keyof typeof store.installProgress
            ]}
          />
        {/each}
      </div>
    {/if}
  </div>
</section>