<script lang="ts">
  import { useToast } from "$lib/stores/toast.svelte";
  import {
    AlertCircle,
    Copy,
    FileText,
    Menu,
    Pause,
    Pencil,
    Play,
    Trash2,
    X,
  } from "lucide-svelte";
  import { logsApi } from "$lib/api";
  import { translate } from "$lib/i18n";
  import { cn } from "$lib/utils/cn";
  import { formatLogs } from "$lib/utils/logs";
  import Button from "./Button.svelte";
  import Dropdown from "./Dropdown.svelte";
  import type {
    DropdownOption,
    LogStream,
    Tunnel,
    TunnelState,
  } from "$lib/types";

  type StatusKey = "running" | "starting" | "installing" | "error" | "stopped";
  type StatusColors = { bg: string; text: string; dot: string };
  type StatusInfo = { key: StatusKey; textKey: string };
  type InstallProgress = { percent: number; step: string };

  interface TunnelCardProps {
    tunnel: Tunnel;
    onStart: (id: string) => void;
    onStop: (id: string) => void;
    onAction: (action: string, data: unknown) => void;
    index?: number;
    totalItems?: number;
    installProgress?: InstallProgress | null;
  }

  let {
    tunnel,
    onStart,
    onStop,
    onAction,
    index = 0,
    totalItems = 1,
    installProgress = null,
  }: TunnelCardProps = $props();

  let t = $derived($translate);
  const dropdownAlign = $derived(index === totalItems - 1 ? "top-left" : "left");

  const toast = useToast();

  let showLogs = $state(false);
  let logs = $state("");
  let loadingLogs = $state(false);
  let logStream: LogStream | null = $state(null);

  const providerNames: Record<string, string> = {
    cloudflared: "Cloudflared",
    tunnelmole: "Tunnelmole",
    localhostrun: "localhost.run",
    serveo: "Serveo",
    pinggy: "Pinggy",
  };

  const statusConfig: Record<StatusKey, StatusColors> = {
    running: {
      bg: "bg-status-running/40",
      text: "text-status-running",
      dot: "bg-status-running/95",
    },
    starting: {
      bg: "bg-status-starting/40",
      text: "text-status-starting",
      dot: "bg-status-starting/95",
    },
    installing: {
      bg: "bg-status-installing/40",
      text: "text-status-installing",
      dot: "bg-status-installing/95",
    },
    error: {
      bg: "bg-status-error/40",
      text: "text-status-error",
      dot: "bg-status-error/95",
    },
    stopped: {
      bg: "bg-status-stopped/40",
      text: "text-status-stopped",
      dot: "bg-status-stopped/95",
    },
  };

  const statusMap: Record<TunnelState, StatusInfo> = {
    online: { key: "running", textKey: "online" },
    starting: { key: "starting", textKey: "starting" },
    connecting: { key: "starting", textKey: "connecting" },
    installing: { key: "installing", textKey: "installing" },
    downloading: { key: "installing", textKey: "downloading" },
    need_installing: { key: "stopped", textKey: "need_installing" },
    stopping: { key: "starting", textKey: "stopping" },
    stopped: { key: "stopped", textKey: "stopped" },
    offline: { key: "stopped", textKey: "offline" },
    timeout: { key: "error", textKey: "timeout" },
    error: { key: "error", textKey: "error" },
  };

  const tunnelState = $derived(tunnel.state as TunnelState);
  const statusInfo = $derived(statusMap[tunnelState] ?? statusMap.error);
  const statusColors = $derived(statusConfig[statusInfo.key]);

  const isRunning = $derived(
    tunnelState === "online" ||
      tunnelState === "starting" ||
      tunnelState === "connecting" ||
      tunnelState === "installing" ||
      tunnelState === "downloading" ||
      tunnelState === "stopping",
  );

  const isInstalling = $derived(
    tunnelState === "installing" || tunnelState === "downloading",
  );

  function copyUrl(url: string) {
    navigator.clipboard.writeText(url);
    toast.info(t("copied"));
  }

  function closeLogs() {
    if (logStream) {
      logStream.close();
      logStream = null;
    }
    loadingLogs = false;
    showLogs = false;
  }

  function loadLogs() {
    if (showLogs) {
      closeLogs();
      return;
    }

    showLogs = true;
    loadingLogs = true;
    logs = "";

    logsApi.get(tunnel.id)
      .then((initial) => {
        logs = formatLogs(initial);
        loadingLogs = false;
      })
      .catch(() => {
        logs = t("error_loading_logs");
        loadingLogs = false;
      });

    logStream = logsApi.createStream(tunnel.id, {
      onLine: (line: string) => {
        logs = logs + "\n" + formatLogs(line);
      },
      onClose: () => {
        logStream = null;
      },
    });
  }

  function handleDropdownAction(option: DropdownOption) {
    switch (option.action) {
      case "edit":
        onAction("edit", tunnel.id);
        break;
      case "logs":
        loadLogs();
        break;
      case "delete":
        onAction("delete", tunnel);
        break;
    }
  }

  const dropdownOptions: DropdownOption[] = $derived([
    { label: t("edit"), action: "edit", icon: Pencil, disabled: isRunning },
    { label: t("logs"), action: "logs", icon: FileText },
    { label: "separator", action: "separator" },
    { label: t("delete"), action: "delete", icon: Trash2, danger: true },
  ]);

  const installPercent = $derived(
    Math.trunc((installProgress?.percent ?? 0) * 100) / 100,
  );
  const installStep = $derived(installProgress?.step ?? t("installing"));

  const providerLabel = $derived(providerNames[tunnel.provider] || tunnel.provider);
</script>

<div
  class="border rounded-3xl cursor-default transition-all duration-150 hover:scale-[1.01] bg-card border-border"
>
  <div class="flex flex-col">
    <div
      class="flex justify-between items-start p-5 gap-4 flex-row sm:items-stretch sm:p-3.5 sm:gap-3"
    >
      <div class="flex-1 min-w-0">
        <div
          class="font-semibold text-[15px] mb-1 whitespace-nowrap overflow-hidden text-ellipsis sm:text-sm"
        >
          {tunnel.name}
        </div>
        <div class="text-xs mb-2 sm:text-xs text-muted">
          {providerLabel} — {t("port")} {tunnel.port}
        </div>
        <div
          class={cn(
            "inline-flex items-center gap-1.5 text-xs font-medium px-2.5 py-1 rounded-full",
            statusColors.bg,
            statusColors.text,
          )}
        >
          <span class={cn("w-1.5 h-1.5 rounded-full", statusColors.dot)}></span>
          <span>{t(statusInfo.textKey)}</span>
          {#if tunnelState === "installing" && installProgress}
            <span class="font-semibold ml-1">{installPercent}%</span>
          {/if}
        </div>
        {#if tunnelState === "installing" && installProgress}
          <div class="w-full h-1 rounded mt-2 overflow-hidden bg-border">
            <div
              class="h-full rounded bg-status-installing"
              style="width: {installPercent}%"
            ></div>
          </div>
          <div
            class="text-[11px] mt-1 whitespace-nowrap overflow-hidden text-ellipsis sm:text-[10px] text-muted"
          >
            {installStep}
          </div>
        {/if}
      </div>
      <div class="flex gap-2 flex-shrink-0 relative">
        {#if isRunning}
          <Button
            variant="error"
            icon={Pause}
            onclick={() => onStop(tunnel.id)}
            disabled={isInstalling || tunnelState === "stopping"}
          >
            {isInstalling ? t("wait") : tunnelState === "stopping" ? t("stopping") : t("stop")}
          </Button>
        {:else}
          <Button
            variant="success"
            icon={Play}
            onclick={() => onStart(tunnel.id)}
          >
            {t("start")}
          </Button>
        {/if}
        <Dropdown
          options={dropdownOptions}
          onSelect={handleDropdownAction}
          align={dropdownAlign}
          ariaLabel={t("tunnel_options")}
        >
          {#snippet children()}
            <Menu size={16} />
          {/snippet}
        </Dropdown>
      </div>
    </div>
    {#if tunnel.publicUrl}
      <button
        type="button"
        class={cn(
          "group flex items-center gap-2.5 px-4 py-2.5 border-t cursor-pointer w-full",
          "sm:px-3.5 sm:py-2.5 bg-url-bg border-t-status-stopped",
          "hover:bg-hover transition-colors",
          {
            "rounded-b-xl": !(tunnel.errorMessage || showLogs),
          },
        )}
        onclick={() => tunnel.publicUrl && copyUrl(tunnel.publicUrl)}
      >
        <span class="w-4 h-4 flex-shrink-0 text-muted"><Copy size={16} /></span>
        <span
          class="flex-1 text-xs font-mono whitespace-nowrap overflow-hidden text-ellipsis text-start text-primary"
          >{tunnel.publicUrl}</span
        >
        <span
          class="text-[10px] text-muted opacity-0 group-hover:opacity-100 transition-opacity duration-200"
          >{t("click_to_copy")}</span
        >
      </button>
    {/if}
    {#if tunnel.errorMessage}
      <div
        class={cn(
          "flex items-center gap-2.5 px-4 py-2.5 border-t sm:px-3.5 sm:py-2.5",
          "bg-status-error/15 border-t-status-error/70 text-status-error",
          {
            "rounded-b-xl": !showLogs,
          },
        )}
      >
        <span class="w-4 h-4 flex-shrink-0"><AlertCircle size={16} /></span>
        <span class="text-xs font-mono sm:text-xs">{tunnel.errorMessage}</span>
      </div>
    {/if}
    <div
      class={cn(
        "transition-max-height duration-300 ease-in-out overflow-hidden rounded-b-xl",
        showLogs ? "max-h-[500px]" : "max-h-0",
      )}
    >
      <div class="overflow-hidden bg-logs-bg">
        <div class="flex items-center justify-between px-4 py-2.5 border-b border-border sm:px-3.5 sm:py-2">
          <span class="text-[11px] font-medium text-muted">{t("live_logs")}</span>
          <Button variant="ghost" size="sm" icon={X} onclick={closeLogs}>
            {t("close")}
          </Button>
        </div>
        {#if loadingLogs}
          <div
            class="flex items-center justify-center gap-3 p-6 sm:p-4 sm:gap-2.5 text-status-stopped"
          >
            <span>{t("loading")}</span>
          </div>
        {:else}
          <pre
            class="m-0 p-4 text-[12px] leading-relaxed whitespace-pre-wrap break-all max-h-[300px] overflow-auto font-mono sm:p-3.5 sm:text-[11px] sm:leading-relaxed text-logs-text">{logs ||
              t("no_logs")}</pre>
        {/if}
      </div>
    </div>
  </div>
</div>