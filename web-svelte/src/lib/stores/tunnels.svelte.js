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
      
      fetch('/api/tunnels?format=json')
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
          if (msg.id && (msg.running !== undefined || msg.starting !== undefined)) {
            const idx = tunnels.findIndex(t => t.id === msg.id);
            if (idx >= 0) {
              const t = tunnels[idx];
              let status = 'stopped';
              if (msg.starting) status = 'starting';
              else if (msg.running) status = 'running';
              else if (msg.error) status = 'error';
              
              tunnels[idx] = {
                ...t,
                status,
                publicUrl: msg.publicUrl || t.publicUrl,
                error: msg.error
              };
            }
          }
        } catch (e) {}
      };
      
      eventSource.onerror = () => {
        error = 'Connection lost';
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
        const arr = [...tunnels];
        arr[idx] = { ...arr[idx], status: 'starting' };
        tunnels = arr;
      }
      
      try {
        const res = await fetch(`/api/tunnels/${id}/start`, { method: 'POST' });
        if (!res.ok) throw new Error('Failed to start');
      } catch (e) {
        if (idx >= 0) {
          const arr = [...tunnels];
          arr[idx] = { ...arr[idx], status: 'error', error: e.message };
          tunnels = arr;
        }
      }
    },
    
    async stop(id) {
      await fetch(`/api/tunnels/${id}/stop`, { method: 'POST' });
      const idx = tunnels.findIndex(t => t.id === id);
      if (idx >= 0) {
        const arr = [...tunnels];
        arr[idx] = { ...arr[idx], status: 'stopped', publicUrl: null };
        tunnels = arr;
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
