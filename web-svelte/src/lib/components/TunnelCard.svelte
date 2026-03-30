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
  type StatusInfo = { key: StatusKey; text: string };
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

  let zIndex = $derived(totalItems - index);

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
      bg: "bg-status-running",
      text: "text-status-running-text",
      dot: "bg-status-running-dot",
    },
    starting: {
      bg: "bg-status-starting",
      text: "text-status-starting-text",
      dot: "bg-status-starting-dot",
    },
    installing: {
      bg: "bg-status-installing",
      text: "text-status-installing-text",
      dot: "bg-status-installing-dot",
    },
    error: {
      bg: "bg-status-error",
      text: "text-status-error-text",
      dot: "bg-status-error-dot",
    },
    stopped: {
      bg: "bg-status-stopped",
      text: "text-status-stopped-text",
      dot: "bg-status-stopped-dot",
    },
  };

  const statusMap: Record<TunnelState, StatusInfo> = {
    online: { key: "running", text: "Online" },
    starting: { key: "starting", text: "Starting..." },
    connecting: { key: "starting", text: "Connecting..." },
    installing: { key: "installing", text: "Installing..." },
    downloading: { key: "installing", text: "Downloading..." },
    stopping: { key: "starting", text: "Stopping..." },
    stopped: { key: "stopped", text: "Stopped" },
    offline: { key: "stopped", text: "Offline" },
    timeout: { key: "error", text: "Timeout" },
    error: { key: "error", text: "Error" },
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
    toast.info("URL copied to clipboard");
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
        logs = "Failed to load logs";
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
    { label: "Edit", action: "edit", icon: Pencil, disabled: isRunning },
    { label: "Logs", action: "logs", icon: FileText },
    { label: "separator", action: "separator" },
    { label: "Delete", action: "delete", icon: Trash2, danger: true },
  ]);

  const installPercent = $derived(installProgress?.percent ?? 0);
  const installStep = $derived(installProgress?.step ?? "Installing...");
</script>

<div
  class={cn("border rounded-xl cursor-default", "bg-card border-border")}
  style="z-index: {zIndex};"
>
  <div class="flex flex-col">
    <div
      class="flex justify-between items-start p-4 gap-4 flex-row sm:items-stretch sm:p-3.5 sm:gap-3"
    >
      <div class="flex-1 min-w-0">
        <div
          class="font-semibold text-[15px] mb-1 whitespace-nowrap overflow-hidden text-ellipsis sm:text-sm"
        >
          {tunnel.name}
        </div>
        <div class="text-xs mb-2 sm:text-xs text-muted">
          {providerNames[tunnel.provider] || tunnel.provider} — Port {tunnel.port}
        </div>
        <div
          class={cn(
            "inline-flex items-center gap-1.5 text-xs font-medium px-2.5 py-1 rounded-full",
            statusColors.bg,
            statusColors.text,
          )}
        >
          <span class={cn("w-1.5 h-1.5 rounded-full", statusColors.dot)}></span>
          <span>{statusInfo.text}</span>
          {#if tunnelState === "installing" && installProgress}
            <span class="font-semibold ml-1">{installPercent}%</span>
          {/if}
        </div>
        {#if tunnelState === "installing" && installProgress}
          <div class="w-full h-1 rounded mt-2 overflow-hidden bg-border">
            <div
              class="h-full rounded bg-status-installing-dot"
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
      <div class="flex gap-2 flex-shrink-0 relative z-10">
        {#if isRunning}
          <Button
            variant="error"
            icon={Pause}
            onclick={() => onStop(tunnel.id)}
            disabled={isInstalling || tunnelState === "stopping"}
          >
            {isInstalling ? "Wait..." : tunnelState === "stopping" ? "Stopping..." : "Stop"}
          </Button>
        {:else}
          <Button
            variant="success"
            icon={Play}
            onclick={() => onStart(tunnel.id)}
          >
            Start
          </Button>
        {/if}
        <Dropdown
          options={dropdownOptions}
          onSelect={handleDropdownAction}
          ariaLabel="Tunnel options"
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
          >Click to copy</span
        >
      </button>
    {/if}
    {#if tunnel.errorMessage}
      <div
        class={cn(
          "flex items-center gap-2.5 px-4 py-2.5 border-t sm:px-3.5 sm:py-2.5",
          "bg-status-error border-t-status-error text-status-error-text",
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
          <span class="text-[11px] font-medium text-muted">Live logs</span>
          <Button variant="ghost" size="sm" icon={X} onclick={closeLogs}>
            Close
          </Button>
        </div>
        {#if loadingLogs}
          <div
            class="flex items-center justify-center gap-3 p-6 sm:p-4 sm:gap-2.5 text-status-stopped-dot"
          >
            <span>Loading logs...</span>
          </div>
        {:else}
          <pre
            class="m-0 p-4 text-[12px] leading-relaxed whitespace-pre-wrap break-all max-h-[300px] overflow-auto font-mono sm:p-3.5 sm:text-[11px] sm:leading-relaxed text-logs-text">{logs ||
              "No logs available"}</pre>
        {/if}
      </div>
    </div>
  </div>
</div>
