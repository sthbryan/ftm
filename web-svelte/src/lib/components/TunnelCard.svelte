<script>
  let { tunnel, onStart, onStop, onDelete, onShowDelete, index = 0 } = $props();
  
  let showLogs = $state(false);
  let logs = $state('');
  let loadingLogs = $state(false);
  let justStarted = $state(false);
  let prevStatus = $state('');

  
  $effect(() => {
    const currentStatus = tunnel.status;
    if (currentStatus === 'running' && prevStatus !== 'running' && prevStatus !== '') {
      justStarted = true;
      setTimeout(() => justStarted = false, 600);
    }
    prevStatus = currentStatus;
  });
  
  const providerNames = {
    cloudflared: 'Cloudflared',
    playitgg: 'Playit.gg',
    tunnelmole: 'Tunnelmole',
    localhostrun: 'localhost.run',
    serveo: 'Serveo',
    pinggy: 'Pinggy'
  };
  
  function getStatusClass() {
    if (tunnel.status === 'running') return 'running';
    if (tunnel.status === 'starting') return 'starting';
    if (tunnel.status === 'error') return 'error';
    return 'stopped';
  }
  
  function getStatusText() {
    if (tunnel.status === 'running') return 'Running';
    if (tunnel.status === 'starting') return 'Starting...';
    if (tunnel.status === 'error') return 'Error';
    return 'Stopped';
  }
  
  function copyUrl(url) {
    navigator.clipboard.writeText(url);
  }
  
  async function loadLogs() {
    showLogs = !showLogs;
    if (!showLogs) return;
    
    loadingLogs = true;
    try {
      const res = await fetch(`/api/logs/${tunnel.id}`);
      logs = await res.text();
    } catch (e) {
      logs = 'Failed to load logs';
    }
    loadingLogs = false;
  }
</script>

<div 
  class="connection-item {getStatusClass()}" 
  style="--stagger-delay: {index * 50}ms"
>
  <div class="connection-content">
    <div class="connection-main">
      <div class="connection-info">
        <div class="connection-name">{tunnel.name}</div>
        <div class="connection-meta">{providerNames[tunnel.provider] || tunnel.provider} — Port {tunnel.port}</div>
        <div class="connection-status status-{getStatusClass()}" class:just-started={justStarted}>
          <span class="status-dot"></span>
          <span class="status-text">{getStatusText()}</span>
        </div>
      </div>
      <div class="connection-actions">
        {#if tunnel.status === 'running' || tunnel.status === 'starting'}
          <button type="button" class="btn btn-stop" onclick={() => onStop(tunnel.id)}>Stop</button>
        {:else}
          <button type="button" class="btn btn-start" onclick={() => onStart(tunnel.id)}>Start</button>
        {/if}
        <button type="button" class="btn" onclick={loadLogs}>
          <span class="logs-label">{showLogs ? 'Hide' : 'Logs'}</span>
        </button>
        <button type="button" class="btn btn-danger" onclick={() => onShowDelete(tunnel)}>Delete</button>
      </div>
    </div>
    {#if tunnel.publicUrl}
      <button type="button" class="connection-url-row" onclick={() => copyUrl(tunnel.publicUrl)}>
        <svg class="copy-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
          <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
        </svg>
        <span class="url-text">{tunnel.publicUrl}</span>
        <span class="copy-hint">Click to copy</span>
      </button>
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
    background: white;
    border: 1px solid #e7e5e4;
    border-radius: 12px;
    overflow: hidden;
    transition: all 0.2s cubic-bezier(0.25, 1, 0.5, 1);
    opacity: 0;
    transform: translateY(20px);
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
    border-color: #d6d3d1;
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
    color: #1c1917;
    margin-bottom: 4px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .connection-meta {
    font-size: 13px;
    color: #78716c;
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
    background: #dcfce7;
    color: #166534;
  }

  .status-running .status-dot {
    background: #22c55e;
  }

  .status-starting {
    background: #fef3c7;
    color: #92400e;
  }

  .status-starting .status-dot {
    background: #f59e0b;
    animation: pulse 1.5s infinite;
  }

  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.4; }
  }

  .status-error {
    background: #fee2e2;
    color: #991b1b;
  }

  .status-error .status-dot {
    background: #ef4444;
  }

  .status-stopped {
    background: #f5f5f4;
    color: #78716c;
  }

  .status-stopped .status-dot {
    background: #a8a29e;
  }

  .connection-actions {
    display: flex;
    gap: 8px;
    flex-shrink: 0;
  }

  .btn {
    padding: 8px 14px;
    font-size: 13px;
    border-radius: 6px;
    border: 1px solid #d6d3d1;
    background: white;
    color: #44403c;
    cursor: pointer;
    transition: all 0.15s cubic-bezier(0.25, 1, 0.5, 1);
    position: relative;
    overflow: hidden;
  }

  .btn:hover {
    background: #f5f5f4;
    transform: translateY(-1px);
    box-shadow: 0 2px 4px rgba(0,0,0,0.08);
  }

  .btn:active {
    transform: translateY(1px);
    box-shadow: 0 1px 2px rgba(0,0,0,0.05);
  }

  .btn-start {
    background: #16a34a;
    color: white;
    border-color: #16a34a;
  }

  .btn-start:hover {
    background: #15803d;
    border-color: #15803d;
    box-shadow: 0 4px 12px rgba(22, 163, 74, 0.3);
  }

  .btn-stop {
    background: #dc2626;
    color: white;
    border-color: #dc2626;
  }

  .btn-stop:hover {
    background: #b91c1c;
    border-color: #b91c1c;
    box-shadow: 0 4px 12px rgba(220, 38, 38, 0.3);
  }

  .btn-danger {
    color: #dc2626;
    border-color: #fecaca;
    background: #fef2f2;
  }

  .btn-danger:hover {
    background: #fee2e2;
    box-shadow: 0 2px 4px rgba(220, 38, 38, 0.1);
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
    background: #1c1917;
  }

  .logs-loading {
    padding: 24px;
    color: #a8a29e;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 12px;
  }

  .logs-spinner {
    width: 16px;
    height: 16px;
    border: 2px solid #44403c;
    border-top-color: #a8a29e;
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .logs-content {
    margin: 0;
    padding: 16px;
    color: #d6d3d1;
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
    background: #fafaf9;
    border-top: 1px solid #f5f5f4;
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
    background: #f5f5f4;
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
    width: 14px;
    height: 14px;
    color: #78716c;
    flex-shrink: 0;
  }

  .url-text {
    flex: 1;
    font-size: 13px;
    color: #92400e;
    font-family: ui-monospace, SFMono-Regular, monospace;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .copy-hint {
    font-size: 11px;
    color: #a8a29e;
    opacity: 0;
    transition: opacity 0.15s;
  }

  .connection-url-row:hover .copy-hint {
    opacity: 1;
  }

  @media (max-width: 600px) {
    .connection-main {
      flex-direction: column;
      align-items: stretch;
    }

    .connection-actions {
      justify-content: flex-start;
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
