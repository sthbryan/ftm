<script lang="ts">
  import { onMount } from "svelte";
  import { Activity, Hourglass, PlugZap, Radio, Timer, Users } from "lucide-svelte";
  import { useTunnels } from "$lib/stores/tunnels.svelte";
  import { useClock } from "$lib/stores/clock.svelte";
  import { t } from "$lib/stores/i18n.svelte";
  import { formatDuration } from "$lib/utils/duration";

  const store = useTunnels();
  const clock = useClock();

  onMount(() => clock.subscribe());

  const ACTIVE = ["online", "starting", "connecting"];

  const active = $derived(
    store.tunnels.filter((tunnel) => ACTIVE.includes(tunnel.state)),
  );

  const failing = $derived(
    store.tunnels.filter((tunnel) =>
      ["error", "timeout"].includes(tunnel.state),
    ).length,
  );

  const players = $derived(
    active.length > 0
      ? active.reduce((sum, tunnel) => sum + (tunnel.activeSessions ?? 0), 0)
      : null,
  );

  const nextExpiry = $derived(
    active
      .filter((tunnel) => (tunnel.expiresAt ?? 0) > clock.now)
      .sort((a, b) => (a.expiresAt ?? 0) - (b.expiresAt ?? 0))[0] ?? null,
  );

  const oldestStart = $derived(
    active
      .filter((tunnel) => tunnel.startedAt)
      .sort((a, b) => (a.startedAt ?? 0) - (b.startedAt ?? 0))[0] ?? null,
  );

  const third = $derived.by(() => {
    if (!store.connected && store.tunnels.length > 0) {
      return {
        icon: PlugZap,
        value: "—",
        label: t("overview_reconnecting"),
        urgent: true,
      };
    }
    if (nextExpiry) {
      return {
        icon: Hourglass,
        value: formatDuration((nextExpiry.expiresAt ?? 0) - clock.now),
        label: t("overview_next_expiry", { name: nextExpiry.name }),
        urgent: (nextExpiry.expiresAt ?? 0) - clock.now < 5 * 60 * 1000,
      };
    }
    if (oldestStart) {
      return {
        icon: Timer,
        value: formatDuration(clock.now - (oldestStart.startedAt ?? 0)),
        label: t("overview_uptime"),
        urgent: false,
      };
    }
    return {
      icon: Timer,
      value: "—",
      label: t("overview_idle"),
      urgent: false,
    };
  });
</script>

{#snippet metric(
  Icon: typeof Radio,
  value: string,
  label: string,
  tone: "muted" | "good" | "bad",
)}
  <div class="sph-card flex min-w-0 items-center gap-2.5 px-3 py-2">
    <span
      class="shrink-0"
      class:text-status-running={tone === "good"}
      class:text-text-error={tone === "bad"}
      class:text-text-muted={tone === "muted"}
    >
      <Icon size={14} />
    </span>
    <span class="min-w-0">
      <span
        class="block font-mono text-base leading-none font-semibold text-text tabular-nums"
        >{value}</span
      >
      <span class="mt-1 block truncate text-2xs text-text-muted">{label}</span>
    </span>
  </div>
{/snippet}

<section class="ftm-enter mb-app flex shrink-0 flex-wrap items-stretch gap-1.5">
  {#if active.length > 0}
    {@render metric(Radio, String(active.length), t("overview_online"), "good")}
  {:else}
    {@render metric(Radio, "—", t("overview_idle"), "muted")}
  {/if}

  {#if failing > 0}
    {@render metric(Activity, String(failing), t("overview_failing"), "bad")}
  {:else if players !== null}
    {@render metric(Users, String(players), t("overview_players"), "good")}
  {/if}

  {#if active.length > 0 || !store.connected}
    {@render metric(
      third.icon,
      third.value,
      third.label,
      third.urgent ? "bad" : "muted",
    )}
  {/if}
</section>
