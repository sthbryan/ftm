<script lang="ts">
  import { t } from "$lib/stores/i18n.svelte";
  import { statusInfo } from "$lib/utils/status";

  let {
    state,
    percent = null,
    stale = false,
  }: {
    state: string;
    percent?: number | null;
    stale?: boolean;
  } = $props();

  const info = $derived(statusInfo(state));
  // "Needs install" is a stopped tunnel with work to do, so it keeps a badge.
  const needsInstall = $derived(info.textKey === "need_installing");
  const idle = $derived((info.key === "stopped" && !needsInstall) || stale);

  // State carries meaning here, so it is always badge + word, never colour alone.
  const TONE: Record<string, string> = {
    running: "success",
    starting: "accent",
    installing: "warning",
    error: "danger",
    stopped: "warning",
  };
</script>

{#if idle}
  <span class="text-2xs font-medium text-text-muted">{t(info.textKey)}</span>
{:else}
  <span
    class="sph-badge gap-1.5"
    data-variant={TONE[info.key] ?? "neutral"}
    data-size="sm"
  >
    <span>{t(info.textKey)}</span>
    {#if percent !== null}
      <span class="font-mono tabular-nums">{percent}%</span>
    {/if}
  </span>
{/if}
