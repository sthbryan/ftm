let tunnels = $state([]);
let loading = $state(true);
let error = $state(null);
let eventSource = $state(null);

export function useTunnels() {
  return {
    get tunnels() { return tunnels; },
    get loading() { return loading; },
    get error() { return error; },
    
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
          if (!msg.id) return;
          
          const idx = tunnels.findIndex(t => t.id === msg.id);
          if (idx < 0) return;
          
          const current = tunnels[idx];
          let newStatus = 'stopped';
          if (msg.starting) newStatus = 'starting';
          else if (msg.running) newStatus = 'running';
          else if (msg.error) newStatus = 'error';
          
          if (current.status !== newStatus || current.publicUrl !== msg.publicUrl || current.error !== msg.error) {
            tunnels = tunnels.map((t, i) => 
              i === idx 
                ? { ...t, status: newStatus, publicUrl: msg.publicUrl || t.publicUrl, error: msg.error }
                : t
            );
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
      if (idx >= 0) {
        tunnels = tunnels.map((t, i) => i === idx ? { ...t, status: 'starting' } : t);
      }
      
      try {
        const res = await fetch(`/api/tunnels/${id}/start`, { method: 'POST' });
        if (!res.ok) throw new Error('Failed to start');
      } catch (e) {
        if (idx >= 0) {
          tunnels = tunnels.map((t, i) => i === idx ? { ...t, status: 'error', error: e.message } : t);
        }
      }
    },
    
    async stop(id) {
      await fetch(`/api/tunnels/${id}/stop`, { method: 'POST' });
      const idx = tunnels.findIndex(t => t.id === id);
      if (idx >= 0) {
        tunnels = tunnels.map((t, i) => i === idx ? { ...t, status: 'stopped', publicUrl: null } : t);
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
