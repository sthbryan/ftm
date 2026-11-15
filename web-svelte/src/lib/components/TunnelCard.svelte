<script lang="ts">
  import { useToast } from "$lib/stores/toast.svelte";
  import { useProviders } from "$lib/stores/providers.svelte";
  import { useTunnels } from "$lib/stores/tunnels.svelte";
  import { AlertCircle, Check, Copy, Pause, Play } from "lucide-svelte";
  import { t } from "$lib/stores/i18n.svelte";
  import { useClock } from "$lib/stores/clock.svelte";
  import { formatDuration } from "$lib/utils/duration";
  import { providerErrorHint } from "$lib/utils/providerError";
  import { copyText } from "$lib/utils/clipboard";
  import { isInstallingState, isRunningState } from "$lib/utils/status";
  import { onMount } from "svelte";
  import Button from "./Button.svelte";
  import StatusBadge from "./StatusBadge.svelte";
  import type { Tunnel, TunnelState } from "$lib/types";

  type InstallProgress = { percent: number; step: string };

  interface TunnelCardProps {
    tunnel: Tunnel;
    onStart: (id: string) => void;
    onStop: (id: string) => void;
    onAction: (action: string, data: unknown) => void;
    index?: number;
    totalItems?: number;
    selected?: boolean;
    installProgress?: InstallProgress | null;
  }

  let {
    tunnel,
    onStart,
    onStop,
    onAction,
    selected = false,
    installProgress = null,
  }: TunnelCardProps = $props();

  const toast = useToast();
  const providerStore = useProviders();
  const clock = useClock();

  onMount(() => clock.subscribe());

  const tunnelState = $derived(tunnel.state as TunnelState);

  const isRunning = $derived(isRunningState(tunnelState));

  const isInstalling = $derived(isInstallingState(tunnelState));

  const providerLabel = $derived(
    providerStore.providers.find((p) => p.id === tunnel.provider)?.name ??
      tunnel.provider,
  );

  const store = useTunnels();
  const stale = $derived(isRunning && !store.connected);

  const uptime = $derived(
    tunnel.startedAt && isRunning && !stale
      ? formatDuration(clock.now - tunnel.startedAt)
      : "",
  );

  const remaining = $derived(
    tunnel.expiresAt && isRunning ? tunnel.expiresAt - clock.now : 0,
  );

  const expiryLabel = $derived(
    tunnel.expiresAt && isRunning
      ? remaining > 0
        ? t("card_expires", { 0: formatDuration(remaining) })
        : t("card_expired")
      : "",
  );

  const actionLabel = $derived(
    isInstalling
      ? t("wait")
      : tunnelState === "stopping"
        ? t("stopping")
        : t("stop"),
  );

  let copied = $state(false);
  let copiedTimer: ReturnType<typeof setTimeout> | null = null;

  async function copyUrl(url: string) {
    if (!(await copyText(url))) {
      toast.error(t("copy_failed"));
      return;
    }

    toast.info(t("copied"));

    copied = true;
    if (copiedTimer) clearTimeout(copiedTimer);
    copiedTimer = setTimeout(() => {
      copied = false;
    }, 3000);
  }

  const installPercent = $derived(
    Math.trunc((installProgress?.percent ?? 0) * 100) / 100,
  );
  const installStep = $derived(installProgress?.step ?? t("installing"));
  const errorHint = $derived(providerErrorHint(tunnel.errorMessage));
</script>

<article class="sph-card" data-selected={selected ? "true" : undefined}>
  <div class="flex items-start gap-2 p-3">
    <button
      type="button"
      aria-pressed={selected}
      onclick={() => onAction("select", tunnel.id)}
      class="flex min-w-0 flex-1 cursor-pointer items-start gap-2.5 text-start"
    >
      <span
        aria-hidden="true"
        class="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-[var(--sph-radius-xs)] border border-border-light bg-secondary-btn font-mono text-2xs font-semibold text-secondary-btn-text uppercase"
      >
        {providerLabel.slice(0, 2)}
      </span>

      <span class="min-w-0 flex-1">
        <span class="block truncate text-sm leading-snug font-semibold text-text">
          {tunnel.name}
        </span>
        <span
          class="mt-0.5 flex items-center gap-1.5 truncate text-2xs text-text-muted"
        >
          <span class="truncate">{providerLabel}</span>
          <span aria-hidden="true">·</span>
          <span class="font-mono">localhost:{tunnel.port}</span>
        </span>

        <span class="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-1">
          <StatusBadge
            state={tunnelState}
            {stale}
            percent={tunnelState === "installing" && installProgress
              ? installPercent
              : null}
          />
          {#if stale}
            <span class="text-2xs text-text-muted">{t("connection_lost")}</span>
          {/if}
          {#if uptime}
            <span class="font-mono text-2xs text-text-muted tabular-nums"
              >{uptime}</span
            >
          {/if}
          {#if expiryLabel}
            <span
              class="text-2xs"
              class:text-text-error={remaining <= 0}
              class:text-text-muted={remaining > 0}
            >
              {expiryLabel}
            </span>
          {/if}
        </span>
      </span>
    </button>

    <div class="flex shrink-0">
      {#if isRunning}
        <Button
          variant="error"
          size="sm"
          icon={Pause}
          onclick={() => onStop(tunnel.id)}
          disabled={isInstalling || tunnelState === "stopping"}
          ariaLabel={actionLabel}
        >
          <span class="max-sm:hidden">{actionLabel}</span>
        </Button>
      {:else}
        <Button
          variant="success"
          size="sm"
          icon={Play}
          onclick={() => onStart(tunnel.id)}
          ariaLabel={t("start")}
        >
          <span class="max-sm:hidden">{t("start")}</span>
        </Button>
      {/if}
    </div>
  </div>

  {#if tunnelState === "installing" && installProgress}
    <div class="px-3 pb-2.5">
      <div class="h-1 w-full overflow-hidden rounded-full bg-border-light">
        <div
          class="h-full rounded-full bg-status-installing transition-[width] duration-300"
          style="width: {installPercent}%"
        ></div>
      </div>
      <div class="mt-1 truncate text-2xs text-text-muted">{installStep}</div>
    </div>
  {/if}

  {#if tunnel.publicUrl}
    <button
      type="button"
      class="flex w-full cursor-pointer items-center gap-2 border-t border-border-light px-3 py-1.5 transition-colors hover:bg-hover"
      onclick={() => tunnel.publicUrl && copyUrl(tunnel.publicUrl)}
    >
      <span class="flex shrink-0 text-text-muted">
        {#if copied}<Check size={13} />{:else}<Copy size={13} />{/if}
      </span>
      <span
        class="min-w-0 flex-1 truncate text-start font-mono text-2xs text-url-text"
        >{tunnel.publicUrl}</span
      >
      <span class="shrink-0 text-2xs text-text-muted">
        {copied ? t("copied") : t("click_to_copy")}
      </span>
    </button>
  {/if}

  {#if tunnel.errorMessage}
    <div
      class="flex items-start gap-2 border-t border-status-error/40 bg-status-error/10 px-3 py-2 text-text-error"
    >
      <span class="mt-0.5 shrink-0"><AlertCircle size={14} /></span>
      <span class="min-w-0 flex-1">
        {#if errorHint}
          <span class="block text-2xs font-medium">{t(errorHint)}</span>
        {/if}
        <span class="block font-mono text-2xs break-words opacity-80">
          {tunnel.errorMessage}
        </span>
      </span>
    </div>
  {/if}
</article>
