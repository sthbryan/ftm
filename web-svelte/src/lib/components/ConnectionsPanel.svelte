<script lang="ts">
  import { Plus, PlugZap, Radio, RefreshCw, Search } from "lucide-svelte";
  import { useTunnels } from "$lib/stores/tunnels.svelte";
  import { useToast } from "$lib/stores/toast.svelte";
  import { t } from "$lib/stores/i18n.svelte";
  import TunnelCard from "./TunnelCard.svelte";
  import Button from "./Button.svelte";

  let {
    onAction,
    onCreateFirst,
    onCreate,
    selectedId = null,
  }: {
    onAction: (action: string, data: unknown) => void;
    onCreateFirst?: () => void;
    onCreate?: () => void;
    selectedId?: string | null;
  } = $props();

  const store = useTunnels();
  const toast = useToast();

  const SEARCH_THRESHOLD = 8;

  async function start(id: string) {
    try {
      await store.start(id);
    } catch (err) {
      toast.error((err as Error).message);
    }
  }

  async function stop(id: string) {
    try {
      await store.stop(id);
    } catch (err) {
      toast.error((err as Error).message);
    }
  }

  let query = $state("");

  const searchable = $derived(store.tunnels.length >= SEARCH_THRESHOLD);

  const STATE_ORDER: Record<string, number> = {
    error: 0,
    timeout: 0,
    online: 1,
    starting: 2,
    connecting: 2,
    stopping: 2,
    installing: 3,
    downloading: 3,
  };

  const visible = $derived.by(() => {
    const term = query.trim().toLowerCase();
    const matches =
      !term || !searchable
        ? store.tunnels
        : store.tunnels.filter(
            (tunnel) =>
              tunnel.name.toLowerCase().includes(term) ||
              tunnel.provider.toLowerCase().includes(term) ||
              String(tunnel.port).includes(term),
          );

    return [...matches].sort(
      (a, b) => (STATE_ORDER[a.state] ?? 9) - (STATE_ORDER[b.state] ?? 9),
    );
  });
</script>

<section
  class="ftm-enter sph-card flex h-full min-h-56 w-full flex-col max-md:h-auto"
>
  <div class="sph-card__header">
    <div class="flex items-center justify-between gap-2">
      <div class="flex min-w-0 items-center gap-2">
        <h2 class="m-0 truncate text-sm font-semibold tracking-tight text-text">
          {t("connections")}
        </h2>
        <span class="sph-badge font-mono tabular-nums" data-size="sm">
          {store.tunnels.length}
        </span>
      </div>
      {#if onCreate}
        <Button
          variant="primary"
          size="sm"
          icon={Plus}
          onclick={onCreate}
          class="max-sm:aspect-square max-sm:px-0"
          ariaLabel={t("new_connection_action")}
        >
          <span class="max-sm:hidden">{t("new_connection_action")}</span>
        </Button>
      {/if}
    </div>
  </div>

  {#if searchable}
    <div class="shrink-0 border-b border-border-light px-3 py-2">
      <div class="relative">
        <Search
          size={14}
          class="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-text-muted"
        />
        <input
          type="search"
          bind:value={query}
          placeholder={t("search_placeholder")}
          aria-label={t("search_placeholder")}
          class="sph-input sph-input--icon"
          data-size="sm"
        />
      </div>
    </div>
  {/if}

  <div class="ftm-enter ftm-enter-delay-2 min-h-0 flex-1 overflow-y-auto p-2 max-md:h-auto max-md:overflow-visible">
    {#if store.loading}
      <div class="flex flex-col items-center justify-center gap-3 py-10">
        <span class="sph-spinner sph-spinner--md" aria-hidden="true"></span>
        <span class="text-2xs text-text-muted">{t("loading")}</span>
      </div>
    {:else if store.error}
      <div class="sph-empty-state" data-compact="true">
        <span class="sph-empty-state__icon text-text-error">
          <PlugZap size={20} />
        </span>
        <h3 class="sph-empty-state__title">{t("panel_unreachable")}</h3>
        <p class="sph-empty-state__description">
          {t("panel_unreachable_desc")}
        </p>
        <p class="m-0 max-w-xs font-mono text-2xs break-words text-text-muted">
          {store.error}
        </p>
        <div class="sph-empty-state__action">
          <Button variant="primary" icon={RefreshCw} onclick={() => store.retry()}>
            {t("retry")}
          </Button>
        </div>
      </div>
    {:else if store.tunnels.length === 0}
      <div class="sph-empty-state" data-compact="true">
        <span class="sph-empty-state__icon"><Radio size={20} /></span>
        <h3 class="sph-empty-state__title">{t("no_tunnels")}</h3>
        {#if onCreateFirst}
          <div class="sph-empty-state__action">
            <Button variant="primary" icon={Plus} onclick={onCreateFirst}>
              {t("new_connection_action")}
            </Button>
          </div>
        {/if}
      </div>
    {:else if visible.length === 0}
      <p class="m-0 px-3 py-8 text-center text-2xs text-text-muted">
        {t("search_empty")}
      </p>
    {:else}
      <div class="flex flex-col gap-1.5">
        {#each visible as tunnel (tunnel.id)}
          <TunnelCard
            {tunnel}
            selected={tunnel.id === selectedId}
            onStart={start}
            onStop={stop}
            {onAction}
            installProgress={store.installProgress[
              tunnel.provider as keyof typeof store.installProgress
            ]}
          />
        {/each}
      </div>
    {/if}
  </div>

  {#if store.tunnels.length > 0}
    <p
      class="m-0 shrink-0 border-t border-border-light px-3 py-1.5 text-center text-2xs text-text-muted max-md:hidden"
    >
      {t("list_hint_new")}
    </p>
  {/if}
</section>
