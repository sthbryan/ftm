let tunnels = $state([]);
let loading = $state(true);
let error = $state(null);
let eventSource = $state(null);
let installProgress = $state({});

export function useTunnels() {
  return {
    get tunnels() { return tunnels; },
    get loading() { return loading; },
    get error() { return error; },
    get installProgress() { return installProgress; },
    
    connect() {
      if (eventSource) return;
      
      loading = true;
      
      fetch('/api/tunnels')
        .then(r => r.json())
        .then(data => {
          tunnels = data;
          loading = false;
        })
        .catch(e => {
          error = e.message;
          loading = false;
        });
      
      eventSource = new EventSource('/api/events');
      
      eventSource.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          
          if (msg.type === 'install') {
            installProgress = {
              ...installProgress,
              [msg.provider]: {
                step: msg.step,
                percent: msg.percent,
                downloaded: msg.downloaded,
                total: msg.total
              }
            };
            return;
          }
          
          if (!msg.id) return;
          
          const idx = tunnels.findIndex(t => t.id === msg.id);
          if (idx < 0) return;
          
          const current = tunnels[idx];
          const newState = msg.state || 'stopped';
          
          if (current.state !== newState || current.publicUrl !== msg.publicUrl || current.errorMessage !== msg.errorMessage) {
            tunnels = tunnels.map((t, i) => 
              i === idx 
                ? { ...t, state: newState, publicUrl: msg.publicUrl || t.publicUrl, errorMessage: msg.errorMessage || t.errorMessage }
                : t
            );
          }
          
          if (newState === 'online' && current.state !== 'online') {
            installProgress = { ...installProgress };
            delete installProgress[current.provider];
          }
        } catch (e) {}
      };
      
      eventSource.onerror = () => {
        error = 'Connection lost. Retrying...';
        setTimeout(() => {
          if (eventSource.readyState === EventSource.CLOSED) {
            this.disconnect();
            this.connect();
          }
        }, 3000);
      };
    },
    
    disconnect() {
      if (eventSource) {
        eventSource.close();
        eventSource = null;
      }
    },
    
    async start(id) {
      const idx = tunnels.findIndex(t => t.id === id);
      if (idx < 0) return;
      
      const tunnel = tunnels[idx];
      
      tunnels = tunnels.map((t, i) => i === idx ? { ...t, state: 'starting' } : t);
      
      try {
        const res = await fetch(`/api/tunnels/${id}/start`, { method: 'POST' });
        const data = await res.json();
        
        if (data.status === 'installing') {
          tunnels = tunnels.map((t, i) => i === idx ? { ...t, state: 'installing' } : t);
        }
      } catch (e) {
        tunnels = tunnels.map((t, i) => i === idx ? { ...t, state: 'error', errorMessage: e.message } : t);
      }
    },
    
    async stop(id) {
      await fetch(`/api/tunnels/${id}/stop`, { method: 'POST' });
      const idx = tunnels.findIndex(t => t.id === id);
      if (idx >= 0) {
        tunnels = tunnels.map((t, i) => i === idx ? { ...t, state: 'stopped', publicUrl: null } : t);
      }
    },
    
    async delete(id) {
      await fetch(`/api/tunnels/${id}`, { method: 'DELETE' });
      tunnels = tunnels.filter(t => t.id !== id);
    },
    
    async create(data) {
      const res = await fetch('/api/tunnels', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data)
      });
      const newTunnel = await res.json();
      tunnels = [...tunnels, newTunnel];
    }
  };
}
