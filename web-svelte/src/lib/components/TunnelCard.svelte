<script>
  import { useToast } from '$lib/stores/toast.svelte';
  import { Copy, AlertCircle, FileText, Pencil, Trash2 } from 'lucide-svelte';
  import { createLogStream, getLogs } from '$lib/api';
  import Dropdown from './Dropdown.svelte';
  
  let { tunnel, onStart, onStop, onAction, index = 0, totalItems = 1, installProgress = null } = $props();

  let zIndex = $derived(totalItems - index);
  
  const toast = useToast();
  
  let showLogs = $state(false);
  let logs = $state('');
  let loadingLogs = $state(false);
  let justStarted = $state(false);
  let logStream = $state(null);
  let hasStartedOnce = $state(false);
  
  $effect(() => {
    if (tunnel.state === 'online' && !hasStartedOnce) {
      hasStartedOnce = true;
      justStarted = true;
      setTimeout(() => justStarted = false, 600);
    }
  });
  
  const providerNames = {
    cloudflared: 'Cloudflared',
    tunnelmole: 'Tunnelmole',
    localhostrun: 'localhost.run',
    serveo: 'Serveo',
    pinggy: 'Pinggy'
  };
  
  const statusMap = {
    online: { class: 'running', text: 'Running' },
    starting: { class: 'starting', text: 'Starting...' },
    connecting: { class: 'starting', text: 'Connecting...' },
    installing: { class: 'installing', text: 'Installing...' },
    downloading: { class: 'installing', text: 'Installing...' },
    timeout: { class: 'error', text: 'Timeout' },
    error: { class: 'error', text: 'Error' }
  };

  const statusInfo = $derived(statusMap[tunnel.state] || statusMap.error);
  
  const isRunning = $derived(
    tunnel.state === 'online' || 
    tunnel.state === 'starting' || 
    tunnel.state === 'connecting' || 
    tunnel.state === 'installing' || 
    tunnel.state === 'downloading'
  );

  const isInstalling = $derived(
    tunnel.state === 'installing' || tunnel.state === 'downloading'
  );
  

  
  function copyUrl(url) {
    navigator.clipboard.writeText(url);
    toast.info('URL copied to clipboard');
  }
  
  function loadLogs() {
    if (showLogs) {
      if (logStream) {
        logStream.close();
        logStream = null;
      }
      showLogs = false;
      return;
    }
    
    showLogs = true;
    loadingLogs = true;
    logs = '';
    
    getLogs(tunnel.id)
      .then(initial => {
        logs = initial;
        loadingLogs = false;
      })
      .catch(() => {
        logs = 'Failed to load logs';
        loadingLogs = false;
      });
    
    logStream = createLogStream(tunnel.id, {
      onLine: (line) => {
        logs = logs + '\n' + line;
      },
      onClose: () => {
        logStream = null;
      }
    });
  }
  
  function handleDropdownAction(option) {
    switch (option.action) {
      case 'edit':
        onAction?.('edit', tunnel.id);
        break;
      case 'logs':
        loadLogs();
        break;
      case 'delete':
        onAction?.('delete', tunnel);
        break;
    }
  }

  const dropdownOptions = $derived([
    { label: 'Edit', action: 'edit', icon: Pencil, disabled: isRunning },
    { label: 'Logs', action: 'logs', icon: FileText },
    { label: 'separator', action: 'separator' },
    { label: 'Delete', action: 'delete', icon: Trash2, danger: true }
  ]);
  
  const installPercent = $derived(installProgress?.percent || 0);
  const installStep = $derived(installProgress?.step || 'Installing...');
</script>

<div 
  class="connection-item {statusInfo.class}" 
  class:animate={!hasStartedOnce}
  style="--stagger-delay: {index * 50}ms; z-index: {zIndex}"
>
  <div class="connection-content">
    <div class="connection-main">
      <div class="connection-info">
        <div class="connection-name">{tunnel.name}</div>
        <div class="connection-meta">{providerNames[tunnel.provider] || tunnel.provider} — Port {tunnel.port}</div>
        <div class="connection-status status-{statusInfo.class}" class:just-started={justStarted}>
          <span class="status-dot"></span>
          <span class="status-text">{statusInfo.text}</span>
          {#if tunnel.state === 'installing' && installProgress}
            <span class="install-percent">{installPercent}%</span>
          {/if}
        </div>
        {#if tunnel.state === 'installing' && installProgress}
          <div class="install-bar">
            <div class="install-progress" style="width: {installPercent}%"></div>
          </div>
          <div class="install-step">{installStep}</div>
        {/if}
      </div>
      <div class="connection-actions">
        {#if isRunning}
          <button type="button" class="btn btn-stop" onclick={() => onStop(tunnel.id)} disabled={isInstalling}>
            {isInstalling ? 'Wait...' : 'Stop'}
          </button>
        {:else}
          <button type="button" class="btn btn-start" onclick={() => onStart(tunnel.id)}>Start</button>
        {/if}
          <Dropdown 
            options={dropdownOptions} 
            onSelect={handleDropdownAction}
            width="140px"
          >
            <span slot="trigger-text">Options</span>
          </Dropdown>
      </div>
    </div>
    {#if tunnel.publicUrl}
      <button type="button" class="connection-url-row" onclick={() => copyUrl(tunnel.publicUrl)}>
        <Copy class="copy-icon" size={16} />
        <span class="url-text">{tunnel.publicUrl}</span>
        <span class="copy-hint">Click to copy</span>
      </button>
    {/if}
    {#if tunnel.errorMessage}
      <div class="error-row">
        <AlertCircle class="error-icon" size={16} />
        <span class="error-text">{tunnel.errorMessage}</span>
      </div>
    {/if}
    <div class="logs-wrapper" class:expanded={showLogs}>
      <div class="logs-panel">
        {#if loadingLogs}
          <div class="logs-loading">
            <span class="logs-spinner"></span>
            <span>Loading logs...</span>
          </div>
        {:else}
          <pre class="logs-content">{logs || 'No logs available'}</pre>
        {/if}
      </div>
    </div>
  </div>
</div>

<style>
  .connection-item {
    background: var(--card-bg, #ffffff);
    border: 1px solid var(--border-color, #e7e5e4);
    border-radius: 12px;
    transition: all 0.2s cubic-bezier(0.25, 1, 0.5, 1);
    opacity: 0;
    transform: translateY(20px);
  }

  .connection-item.animate {
    animation: slideIn 0.4s cubic-bezier(0.16, 1, 0.3, 1) forwards;
    animation-delay: var(--stagger-delay, 0ms);
  }

  @keyframes slideIn {
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .connection-item:hover {
    border-color: var(--border-color, #d6d3d1);
    box-shadow: 0 8px 24px rgba(0,0,0,0.08);
    transform: translateY(-2px);
  }

  .connection-content {
    display: flex;
    flex-direction: column;
  }

  .connection-main {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    padding: 16px;
    gap: 16px;
  }

  .connection-info {
    flex: 1;
    min-width: 0;
  }

  .connection-name {
    font-weight: 600;
    font-size: 15px;
    color: var(--text-heading, #1c1917);
    margin-bottom: 4px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .connection-meta {
    font-size: 13px;
    color: var(--text-muted, #78716c);
    margin-bottom: 8px;
  }

  .connection-status {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    font-weight: 500;
    padding: 4px 10px;
    border-radius: 12px;
    transition: transform 0.2s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  .connection-status.just-started {
    animation: celebrate 0.6s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  @keyframes celebrate {
    0% { transform: scale(1); }
    50% { transform: scale(1.1); }
    100% { transform: scale(1); }
  }

  .status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
  }

  .status-running {
    background: var(--status-running-bg, #dcfce7);
    color: var(--status-running-text, #166534);
  }

  .status-running .status-dot {
    background: var(--status-running-dot, #22c55e);
  }

  .status-starting {
    background: var(--status-starting-bg, #fef3c7);
    color: var(--status-starting-text, #92400e);
  }

  .status-starting .status-dot {
    background: var(--status-starting-dot, #f59e0b);
    animation: pulse 1.5s infinite;
  }

  .status-installing {
    background: var(--status-installing-bg, #dbeafe);
    color: var(--status-installing-text, #1e40af);
  }

  .status-installing .status-dot {
    background: var(--status-installing-dot, #3b82f6);
    animation: pulse 1.5s infinite;
  }

  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.4; }
  }

  .status-error {
    background: var(--status-error-bg, #fee2e2);
    color: var(--status-error-text, #991b1b);
  }

  .status-error .status-dot {
    background: var(--status-error-dot, #ef4444);
  }

  .status-stopped {
    background: var(--status-stopped-bg, #f5f5f4);
    color: var(--status-stopped-text, #78716c);
  }

  .status-stopped .status-dot {
    background: var(--status-stopped-dot, #a8a29e);
  }

  .install-percent {
    font-weight: 600;
    margin-left: 4px;
  }

  .install-bar {
    width: 100%;
    height: 4px;
    background: var(--border-color, #e5e7eb);
    border-radius: 2px;
    margin-top: 8px;
    overflow: hidden;
  }

  .install-progress {
    height: 100%;
    background: linear-gradient(90deg, var(--status-installing-dot, #3b82f6), #60a5fa);
    border-radius: 2px;
    transition: width 0.3s ease;
  }

  .install-step {
    font-size: 11px;
    color: var(--text-muted, #6b7280);
    margin-top: 4px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .connection-actions {
    display: flex;
    gap: 8px;
    flex-shrink: 0;
    position: relative;
    z-index: 10;
  }

  .btn {
    padding: 8px 14px;
    font-size: 13px;
    border-radius: 6px;
    border: 1px solid var(--border-color);
    background: var(--card-bg);
    color: var(--text-color);
    cursor: pointer;
    transition: all 0.15s cubic-bezier(0.25, 1, 0.5, 1);
    position: relative;
    overflow: hidden;
  }

  .btn:hover:not(:disabled) {
    background: var(--hover-bg);
    transform: translateY(-1px);
    box-shadow: 0 2px 4px rgba(0,0,0,0.08);
  }

  .btn:active:not(:disabled) {
    transform: translateY(1px);
    box-shadow: 0 1px 2px rgba(0,0,0,0.05);
  }

  .btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .btn-start {
    background: var(--btn-start-bg);
    color: var(--badge-text);
    border-color: var(--btn-start-bg);
  }

  .btn-start:hover:not(:disabled) {
    background: var(--btn-start-hover-bg);
    border-color: var(--btn-start-hover-bg);
    box-shadow: 0 4px 12px color-mix(in srgb, var(--btn-start-bg) 30%, transparent);
  }

  .btn-stop {
    background: var(--btn-stop-bg);
    color: var(--badge-text);
    border-color: var(--btn-stop-bg);
  }

  .btn-stop:hover:not(:disabled) {
    background: var(--btn-stop-hover-bg);
    border-color: var(--btn-stop-hover-bg);
    box-shadow: 0 4px 12px color-mix(in srgb, var(--btn-stop-bg) 30%, transparent);
  }

  .btn-danger {
    color: var(--btn-danger-bg);
    border-color: var(--btn-danger-bg);
    background: color-mix(in srgb, var(--btn-danger-bg) 10%, transparent);
  }

  .btn-danger:hover:not(:disabled) {
    background: color-mix(in srgb, var(--btn-danger-bg) 20%, transparent);
  }

  .btn-edit {
    color: var(--primary-color);
    border-color: var(--primary-color);
    background: color-mix(in srgb, var(--primary-color) 10%, transparent);
  }

  .btn-edit:hover:not(:disabled) {
    background: color-mix(in srgb, var(--primary-color) 20%, transparent);
  }

  .logs-label {
    transition: all 0.15s;
  }

  .logs-wrapper {
    display: grid;
    grid-template-rows: 0fr;
    transition: grid-template-rows 0.35s cubic-bezier(0.16, 1, 0.3, 1);
  }

  .logs-wrapper.expanded {
    grid-template-rows: 1fr;
  }

  .logs-panel {
    overflow: hidden;
    background: var(--logs-bg, #1c1917);
  }

  .logs-loading {
    padding: 24px;
    color: var(--status-stopped-dot, #a8a29e);
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 12px;
  }

  .logs-spinner {
    width: 16px;
    height: 16px;
    border: 2px solid var(--text-color, #44403c);
    border-top-color: var(--status-stopped-dot, #a8a29e);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .logs-content {
    margin: 0;
    padding: 16px;
    color: var(--logs-text, #d6d3d1);
    font-family: ui-monospace, SFMono-Regular, monospace;
    font-size: 12px;
    line-height: 1.6;
    white-space: pre-wrap;
    word-break: break-all;
    max-height: 300px;
    overflow: auto;
  }

  .connection-url-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 16px;
    background: var(--url-bg, #fafaf9);
    border-top: 1px solid var(--status-stopped-bg, #f5f5f4);
    cursor: pointer;
    transition: all 0.15s cubic-bezier(0.25, 1, 0.5, 1);
    border: none;
    width: 100%;
    font: inherit;
    text-align: left;
    position: relative;
  }

  .connection-url-row::after {
    content: '';
    position: absolute;
    inset: 0;
    background: linear-gradient(90deg, transparent, rgba(146, 64, 14, 0.1), transparent);
    opacity: 0;
    transition: opacity 0.3s;
  }

  .connection-url-row:hover {
    background: var(--hover-bg, #f5f5f4);
  }

  .connection-url-row:hover::after {
    opacity: 1;
    animation: shimmer 1.5s infinite;
  }

  @keyframes shimmer {
    0% { transform: translateX(-100%); }
    100% { transform: translateX(100%); }
  }

  .copy-icon {
    color: var(--text-muted, #78716c);
    flex-shrink: 0;
  }

  .url-text {
    flex: 1;
    font-size: 13px;
    color: var(--url-text, var(--primary-color, #92400e));
    font-family: ui-monospace, SFMono-Regular, monospace;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .copy-hint {
    font-size: 11px;
    color: var(--status-stopped-dot, #a8a29e);
    opacity: 0;
    transition: opacity 0.15s;
  }

  .connection-url-row:hover .copy-hint {
    opacity: 1;
  }

  .error-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 16px;
    background: var(--status-error-bg, #fee2e2);
    border-top: 1px solid var(--status-error-dot, #ef4444);
    color: var(--status-error-text, #991b1b);
  }

  .error-icon {
    flex-shrink: 0;
  }

  .error-text {
    font-size: 13px;
    font-family: ui-monospace, SFMono-Regular, monospace;
  }

  @media (max-width: 640px) {
    .connection-main {
      flex-direction: column;
      align-items: stretch;
      padding: 14px;
      gap: 12px;
    }

    .connection-actions {
      justify-content: flex-start;
      gap: 6px;
    }
    
    .btn {
      padding: 8px 12px;
      font-size: 12px;
      flex: 1;
      max-width: 80px;
    }
    
    .connection-name {
      font-size: 14px;
    }
    
    .connection-meta {
      font-size: 12px;
    }
    
    .connection-status {
      font-size: 11px;
      padding: 3px 8px;
    }
    
    .url-text {
      font-size: 12px;
    }
    
    .copy-hint {
      display: none;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .connection-item,
    .btn,
    .connection-status,
    .connection-url-row,
    .logs-wrapper {
      animation: none;
      transition: none;
    }
    .connection-item {
      opacity: 1;
      transform: none;
    }
    .logs-wrapper {
      grid-template-rows: 1fr;
    }
    .logs-wrapper:not(.expanded) {
      display: none;
    }
  }
</style>
