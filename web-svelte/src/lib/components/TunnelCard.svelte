<script>
  let { tunnel, onStart, onStop, onDelete } = $props();
  
  const icons = {
    cloudflared: '☁️',
    playitgg: '🎮',
    localhostrun: '🌐',
    serveo: '🔌',
    pinggy: '📡',
    tunnelmole: '🕳️',
    zrok: '🔒',
    exposesh: '🚀',
    localtunnel: '🔧'
  };
  
  function copyUrl(url) {
    navigator.clipboard.writeText(url);
  }
</script>

<div class="card {tunnel.status}">
  <div class="header">
    <span class="icon">{icons[tunnel.provider] || '🔗'}</span>
    <div class="info">
      <h3>{tunnel.name}</h3>
      <span class="provider">{tunnel.provider}</span>
    </div>
    <span class="status {tunnel.status}">
      {tunnel.status === 'running' ? '● Online' : 
       tunnel.status === 'starting' ? '◐ Starting...' : 
       tunnel.status === 'error' ? '● Error' : '● Offline'}
    </span>
  </div>
  
  <div class="details">
    <div>Port: <code>localhost:{tunnel.localPort}</code></div>
    {#if tunnel.publicUrl}
      <div class="url" onclick={() => copyUrl(tunnel.publicUrl)}>
        {tunnel.publicUrl}
      </div>
    {/if}
  </div>
  
  <div class="actions">
    {#if tunnel.status === 'running'}
      <button class="stop" onclick={() => onStop(tunnel.id)}>Stop</button>
      {#if tunnel.publicUrl}
        <button onclick={() => copyUrl(tunnel.publicUrl)}>Copy</button>
      {/if}
    {:else if tunnel.status === 'starting'}
      <button disabled>Starting...</button>
    {:else}
      <button class="start" onclick={() => onStart(tunnel.id)}>Start</button>
    {/if}
    <button class="delete" onclick={() => onDelete(tunnel.id)}>Delete</button>
  </div>
</div>

<style>
  .card {
    background: white;
    border-radius: 12px;
    padding: 20px;
    margin-bottom: 16px;
    box-shadow: 0 2px 4px rgba(0,0,0,0.05);
    transition: all 0.2s;
  }
  
  .card:hover {
    box-shadow: 0 4px 12px rgba(0,0,0,0.1);
    transform: translateY(-2px);
  }
  
  .card.running {
    background: #f0fdf4;
    border: 1px solid #86efac;
  }
  
  .card.starting {
    background: #fffbeb;
    border: 1px solid #fcd34d;
  }
  
  .card.error {
    background: #fef2f2;
    border: 1px solid #fca5a5;
  }
  
  .header {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 12px;
  }
  
  .icon {
    font-size: 24px;
  }
  
  .info {
    flex: 1;
  }
  
  h3 {
    margin: 0;
    font-size: 18px;
    color: #1c1917;
  }
  
  .provider {
    font-size: 12px;
    color: #78716c;
    text-transform: capitalize;
  }
  
  .status {
    font-size: 14px;
    font-weight: 500;
  }
  
  .status.running { color: #16a34a; }
  .status.starting { color: #d97706; }
  .status.error { color: #dc2626; }
  .status.stopped { color: #78716c; }
  
  .details {
    margin-bottom: 16px;
    font-size: 14px;
    color: #44403c;
  }
  
  code {
    background: #f5f5f4;
    padding: 2px 6px;
    border-radius: 4px;
    font-family: monospace;
  }
  
  .url {
    margin-top: 8px;
    color: #16a34a;
    cursor: pointer;
    text-decoration: underline;
  }
  
  .actions {
    display: flex;
    gap: 8px;
  }
  
  button {
    padding: 8px 16px;
    border: none;
    border-radius: 6px;
    font-size: 14px;
    cursor: pointer;
    transition: all 0.2s;
  }
  
  button:hover:not(:disabled) {
    opacity: 0.8;
  }
  
  button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  
  button.start {
    background: #16a34a;
    color: white;
  }
  
  button.stop {
    background: #78716c;
    color: white;
  }
  
  button.delete {
    background: transparent;
    color: #dc2626;
    border: 1px solid #dc2626;
  }
  
  button.delete:hover {
    background: #dc2626;
    color: white;
  }
</style>
