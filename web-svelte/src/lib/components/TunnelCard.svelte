<script>
  let { tunnel, onStart, onStop, onDelete } = $props();
  
  let showLogs = $state(false);
  let logs = $state('');
  let loadingLogs = $state(false);
  
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

<div class="connection-item {getStatusClass()}">
  <div class="connection-content">
    <div class="connection-main">
      <div class="connection-info">
        <div class="connection-name">{tunnel.name}</div>
        <div class="connection-meta">{providerNames[tunnel.provider] || tunnel.provider} — Port {tunnel.port}</div>
        <div class="connection-status status-{getStatusClass()}">
          <span class="status-dot"></span>
          <span class="status-text">{getStatusText()}</span>
        </div>
      </div>
      <div class="connection-actions">
        {#if tunnel.status === 'running'}
          <button class="btn" onclick={() => onStop(tunnel.id)}>Stop</button>
        {:else}
          <button class="btn btn-start" onclick={() => onStart(tunnel.id)}>Start</button>
        {/if}
        <button class="btn" onclick={loadLogs}>{showLogs ? 'Hide' : 'Logs'}</button>
        <button class="btn" onclick={() => onDelete(tunnel.id)}>Delete</button>
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
    {#if showLogs}
      <div class="logs-panel">
        {#if loadingLogs}
          <div class="logs-loading">Loading...</div>
        {:else}
          <pre class="logs-content">{logs || 'No logs available'}</pre>
        {/if}
      </div>
    {/if}
  </div>
</div>

<style>
  .connection-item {
    background: white;
    border: 1px solid #e7e5e4;
    border-radius: 10px;
    overflow: hidden;
    transition: all 0.2s;
  }

  .connection-item:hover {
    border-color: #d6d3d1;
    box-shadow: 0 2px 8px rgba(0,0,0,0.04);
  }

  .connection-content {
    display: flex;
    flex-direction: column;
  }

  .connection-main {
    display: flex;
    justify-content: space-between;
    align-items: center;
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

  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.4; }
  }

  .connection-actions {
    display: flex;
    gap: 8px;
  }

  .btn {
    padding: 8px 14px;
    font-size: 13px;
    border-radius: 6px;
    border: 1px solid #d6d3d1;
    background: white;
    color: #44403c;
    cursor: pointer;
    transition: all 0.15s;
  }

  .btn:hover {
    background: #f5f5f4;
  }

  .btn-start {
    background: #16a34a;
    color: white;
    border-color: #16a34a;
  }

  .btn-start:hover {
    background: #15803d;
    border-color: #15803d;
  }

  .connection-url-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 16px;
    background: #fafaf9;
    border-top: 1px solid #f5f5f4;
    cursor: pointer;
    transition: all 0.15s;
    border: none;
    width: 100%;
    font: inherit;
    text-align: left;
  }

  .connection-url-row:hover {
    background: #f5f5f4;
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

  .logs-panel {
    border-top: 1px solid #f5f5f4;
    background: #1c1917;
    max-height: 300px;
    overflow: auto;
  }

  .logs-loading {
    padding: 20px;
    color: #a8a29e;
    text-align: center;
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
  }
</style>
