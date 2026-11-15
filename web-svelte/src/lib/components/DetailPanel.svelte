<script lang="ts">
  import {
    Check,
    ChevronDown,
    Copy,
    MousePointerClick,
    Pencil,
    PictureInPicture2,
    QrCode as QrCodeIcon,
    Trash2,
  } from "lucide-svelte";
  import { useProviders } from "$lib/stores/providers.svelte";
  import { useTunnels } from "$lib/stores/tunnels.svelte";
  import { useToast } from "$lib/stores/toast.svelte";
  import { useClock } from "$lib/stores/clock.svelte";
  import { logsApi, pipApi, statusApi } from "$lib/api";
  import { openDocumentPip, supportsDocumentPip } from "$lib/utils/pip";
  import { t } from "$lib/stores/i18n.svelte";
  import { cn } from "$lib/utils/cn";
  import { formatDuration } from "$lib/utils/duration";
  import { formatLogs } from "$lib/utils/logs";
  import { providerErrorHint } from "$lib/utils/providerError";
  import { revealDuration } from "$lib/utils/motion";
  import { copyText } from "$lib/utils/clipboard";
  import { slide } from "svelte/transition";
  import Button from "./Button.svelte";
  import IconButton from "./IconButton.svelte";
  import QrCode from "./QrCode.svelte";
  import StatusBadge from "./StatusBadge.svelte";
  import type { LogStream, Tunnel } from "$lib/types";
  import { onDestroy, onMount } from "svelte";

  let {
    tunnel,
    onAction,
  }: {
    tunnel: Tunnel | null;
    onAction: (action: string, data: unknown) => void;
  } = $props();

  const providerStore = useProviders();
  const toast = useToast();
  const clock = useClock();

  onMount(() => clock.subscribe());

  const providerLabel = $derived(
    providerStore.providers.find((p) => p.id === tunnel?.provider)?.name ??
      tunnel?.provider ??
      "",
  );

  const isOnline = $derived(tunnel?.state === "online");
  const errorHint = $derived(providerErrorHint(tunnel?.errorMessage));

  const tunnelStore = useTunnels();
  const stale = $derived(isOnline && !tunnelStore.connected);

  const uptime = $derived(
    tunnel?.startedAt && isOnline && !stale
      ? formatDuration(clock.now - tunnel.startedAt)
      : "",
  );

  const remaining = $derived(
    tunnel?.expiresAt && isOnline ? tunnel.expiresAt - clock.now : 0,
  );

  const LOGS_OPEN_KEY = "ftm-detail-logs-open";
  const QR_OPEN_KEY = "ftm-detail-qr-open";

  let nativePip = $state(false);
  let documentPip = $state(false);
  const pipAvailable = $derived(nativePip || documentPip);

  onMount(async () => {
    documentPip = supportsDocumentPip();
    try {
      nativePip = (await statusApi.get()).nativePip === true;
    } catch {
      nativePip = false;
    }
  });

  async function openPip() {
    if (!tunnel) return;

    try {
      if (nativePip) {
        await pipApi.open(tunnel.id);
        return;
      }
      await openDocumentPip(tunnel.id);
    } catch {
      toast.error(t("pip_failed"));
    }
  }

  let qrDataUrl = $state("");
  let qrOpen = $state(false);
  let copied = $state(false);
  let copiedTimer: ReturnType<typeof setTimeout> | null = null;
  let logsOpen = $state(true);
  let logs = $state("");
  let logPre: HTMLPreElement | undefined = $state();
  let followBottom = $state(true);
  let stream: LogStream | null = null;

  const isRunning = $derived(
    ["online", "starting", "connecting"].includes(tunnel?.state ?? ""),
  );

  onMount(() => {
    logsOpen = localStorage.getItem(LOGS_OPEN_KEY) !== "false";
    qrOpen = localStorage.getItem(QR_OPEN_KEY) === "true";
  });

  function toggleLogs() {
    logsOpen = !logsOpen;
    localStorage.setItem(LOGS_OPEN_KEY, String(logsOpen));
  }

  function toggleQr() {
    qrOpen = !qrOpen;
    localStorage.setItem(QR_OPEN_KEY, String(qrOpen));
  }

  function closeStream() {
    stream?.close();
    stream = null;
  }

  onDestroy(closeStream);

  $effect(() => {
    const id = tunnel?.id;
    const running = isRunning;
    const open = logsOpen;

    closeStream();
    logs = "";
    followBottom = true;

    if (!id || !running || !open) return;

    logsApi
      .get(id)
      .then((initial) => {
        if (tunnel?.id === id) logs = formatLogs(initial);
      })
      .catch(() => {
        if (tunnel?.id === id) logs = t("error_loading_logs");
      });

    stream = logsApi.createStream(id, {
      onLine: (line) => {
        logs = logs + "\n" + formatLogs(line);
      },
      onClose: () => {
        stream = null;
      },
    });

    return closeStream;
  });

  $effect(() => {
    logs;
    queueMicrotask(() => {
      if (logPre && followBottom) logPre.scrollTop = logPre.scrollHeight;
    });
  });

  function onLogScroll() {
    if (!logPre) return;
    followBottom =
      logPre.scrollHeight - logPre.scrollTop - logPre.clientHeight < 24;
  }

  async function copy() {
    if (!tunnel?.publicUrl) return;

    if (!(await copyText(tunnel.publicUrl))) {
      toast.error(t("copy_failed"));
      return;
    }

    toast.success(t("overview_copied"));

    copied = true;
    if (copiedTimer) clearTimeout(copiedTimer);
    copiedTimer = setTimeout(() => {
      copied = false;
    }, 3000);
  }

  function downloadQr(blob: Blob) {
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `${tunnel?.name ?? "tunnel"}-qr.png`;
    link.click();
    URL.revokeObjectURL(url);
    toast.success(t("qr_downloaded"));
  }

  async function copyQr() {
    if (!qrDataUrl) return;

    const blob = await (await fetch(qrDataUrl)).blob();

    if (typeof ClipboardItem === "undefined" || !navigator.clipboard?.write) {
      downloadQr(blob);
      return;
    }

    try {
      await navigator.clipboard.write([
        new ClipboardItem({ "image/png": blob }),
      ]);
      toast.success(t("qr_copied"));
    } catch {
      downloadQr(blob);
    }
  }
</script>

<section
  class="ftm-enter sph-card flex h-full min-h-0 flex-col max-md:h-auto"
>
  <div
    class="flex shrink-0 items-center justify-between gap-2 border-b border-border px-3.5 py-2.5"
  >
    <h2 class="m-0 truncate text-sm font-semibold tracking-tight text-text">
      {t("detail_title")}
    </h2>
    {#if tunnel}
      <div class="flex shrink-0 gap-1">
        {#if pipAvailable}
          <IconButton
            icon={PictureInPicture2}
            label={t("pip_open")}
            size="sm"
            onclick={openPip}
          />
        {/if}
        <IconButton
          icon={Pencil}
          label={t("edit")}
          size="sm"
          disabled={isRunning}
          onclick={() => onAction("edit", tunnel.id)}
        />
        <IconButton
          icon={Trash2}
          label={t("delete")}
          variant="danger"
          size="sm"
          disabled={isRunning}
          onclick={() => onAction("delete", tunnel)}
        />
      </div>
    {/if}
  </div>

  <div
    class="min-h-0 flex-1 overflow-y-auto p-3.5 max-md:h-auto max-md:overflow-visible"
  >
    {#if !tunnel}
      <div class="sph-empty-state" data-compact="true">
        <span class="sph-empty-state__icon"><MousePointerClick size={20} /></span>
        <p class="sph-empty-state__description">{t("detail_empty")}</p>
      </div>
    {:else}
      <h3
        class="m-0 mb-1 truncate text-base font-semibold text-text-heading"
        title={tunnel.name}
      >
        {tunnel.name}
      </h3>
      <p class="m-0 mb-3 text-xs text-text-muted">
        {providerLabel} · <span class="font-mono">localhost:{tunnel.port}</span>
      </p>

      <dl class="m-0 mb-3 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
        <dt class="text-text-muted">{t("status_label")}</dt>
        <dd class="m-0">
          <StatusBadge state={tunnel.state} {stale} />
        </dd>
        {#if stale}
          <dt class="text-text-muted">{t("connection_label")}</dt>
          <dd class="m-0 font-medium text-text-error">
            {t("connection_lost")}
          </dd>
        {/if}
        {#if uptime}
          <dt class="text-text-muted">{t("detail_uptime")}</dt>
          <dd class="m-0 font-mono text-text">{uptime}</dd>
        {/if}
        {#if remaining > 0}
          <dt class="text-text-muted">{t("detail_expires")}</dt>
          <dd class="m-0 font-mono text-text">{formatDuration(remaining)}</dd>
        {/if}
        {#if isOnline}
          <dt class="text-text-muted">{t("detail_sessions")}</dt>
          <dd class="m-0 font-mono text-text">{tunnel.activeSessions ?? 0}</dd>
          <dt class="text-text-muted">{t("detail_visitors")}</dt>
          <dd class="m-0 font-mono text-text">{tunnel.visitors ?? 0}</dd>
        {/if}
      </dl>

      {#if tunnel.publicUrl}
        <div class="sph-well mb-3 flex items-center gap-1 py-1 pr-1 pl-2.5">
          <span
            class="min-w-0 flex-1 truncate font-mono text-xs text-url-text"
            title={tunnel.publicUrl}
          >
            {tunnel.publicUrl}
          </span>
          <button
            type="button"
            onclick={copy}
            aria-label={copied ? t("link_copied") : t("copy_link")}
            title={copied ? t("link_copied") : t("copy_link")}
            class={cn(
              "inline-flex h-7 w-7 shrink-0 cursor-pointer items-center justify-center rounded-control transition-colors",
              copied ? "text-status-running" : "text-url-text hover:bg-hover",
            )}
          >
            {#if copied}
              <Check size={14} />
            {:else}
              <Copy size={14} />
            {/if}
          </button>
        </div>

        <div class="mb-3 border-t border-border-light pt-3">
          <button
            type="button"
            onclick={toggleQr}
            aria-expanded={qrOpen}
            class="flex w-full cursor-pointer items-center justify-between gap-2 rounded-[var(--sph-radius-xs)] px-1 py-1.5 text-2xs font-medium text-text-muted transition-colors hover:text-text"
          >
            <span>{t("qr_for_phones")}</span>
            <ChevronDown
              size={14}
              class={cn("transition-transform", !qrOpen && "-rotate-90")}
            />
          </button>
          {#if qrOpen}
            <div
              class="mt-2 flex flex-col items-center gap-2"
              transition:slide={{ duration: revealDuration() }}
            >
              <div class="flex h-[148px] w-[148px] items-center justify-center">
                <QrCode
                  value={tunnel.publicUrl}
                  size={140}
                  bind:dataUrl={qrDataUrl}
                />
              </div>
              <Button
                variant="default"
                icon={QrCodeIcon}
                disabled={!qrDataUrl}
                onclick={copyQr}
              >
                {t("qr_copy")}
              </Button>
            </div>
          {/if}
        </div>
      {/if}

      {#if tunnel.errorMessage}
        <div
          class="m-0 mb-3 rounded-control border border-status-error/40 bg-status-error/10 px-2.5 py-1.5 text-text-error"
        >
          {#if errorHint}
            <p class="m-0 mb-1 text-xs font-medium">{t(errorHint)}</p>
          {/if}
          <p class="m-0 font-mono text-xs break-words opacity-80">
            {tunnel.errorMessage}
          </p>
        </div>
      {/if}

      <div class="mt-3 border-t border-border-light pt-3">
        <button
          type="button"
          onclick={toggleLogs}
          aria-expanded={logsOpen}
          class="mb-1.5 flex w-full cursor-pointer items-center justify-between gap-2 rounded-[var(--sph-radius-xs)] px-1 py-1.5 text-2xs font-medium text-text-muted transition-colors hover:text-text"
        >
          <span>{t("detail_logs")}</span>
          <span class="flex items-center gap-1">
            <span class="text-2xs font-normal">
              {logsOpen ? t("detail_logs_hide") : t("detail_logs_show")}
            </span>
            <ChevronDown
              size={14}
              class={cn("transition-transform", !logsOpen && "-rotate-90")}
            />
          </span>
        </button>
        {#if logsOpen}
          <div transition:slide={{ duration: revealDuration() }}>
            {#if !isRunning}
              <p class="m-0 text-xs text-text-muted italic">
                {t("detail_logs_idle")}
              </p>
            {:else}
              <pre
                bind:this={logPre}
                onscroll={onLogScroll}
                class="m-0 max-h-48 overflow-x-hidden overflow-y-auto rounded-control bg-logs-bg p-2 font-mono text-2xs leading-relaxed break-all whitespace-pre-wrap text-logs-text">{logs ||
                  t("no_logs")}</pre>
            {/if}
          </div>
        {/if}
      </div>

    {/if}
  </div>
</section>
