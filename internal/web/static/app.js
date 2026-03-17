const API_BASE = '';
let ws = null;
let reconnectInterval = null;
let tunnels = [];
let selectedProvider = 'cloudflared';
let currentLogs = '';

const providerIcons = {
    cloudflared: 'CF',
    playitgg: 'P',
    localhostrun: 'LR',
    serveo: 'S',
    pinggy: 'PI',
    tunnelmole: 'TM',
    zrok: 'Z',
    exposesh: 'E',
    localtunnel: 'LT'
};

const providerNames = {
    cloudflared: 'Cloudflared',
    playitgg: 'Playit.gg',
    localhostrun: 'localhost.run',
    serveo: 'Serveo',
    pinggy: 'Pinggy',
    tunnelmole: 'Tunnelmole',
    zrok: 'zrok',
    exposesh: 'expose.sh',
    localtunnel: 'localtunnel'
};

function init() {
    connectWebSocket();
    setupEventListeners();
    loadTunnels();
}

function connectWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;
    
    ws = new WebSocket(wsUrl);
    
    ws.onopen = () => {
        updateConnectionStatus(true);
        if (reconnectInterval) {
            clearInterval(reconnectInterval);
            reconnectInterval = null;
        }
    };
    
    ws.onmessage = (event) => {
        const msg = JSON.parse(event.data);
        handleWebSocketMessage(msg);
    };
    
    ws.onclose = () => {
        updateConnectionStatus(false);
        if (!reconnectInterval) {
            reconnectInterval = setInterval(connectWebSocket, 3000);
        }
    };
    
    ws.onerror = () => {};
}

function updateConnectionStatus(connected) {
    const status = document.getElementById('connection-status');
    if (connected) {
        status.innerHTML = '<span class="status-dot online"></span> Connected';
    } else {
        status.innerHTML = '<span class="status-dot"></span> Reconnecting...';
    }
}

function handleWebSocketMessage(msg) {
    switch (msg.type) {
        case 'initial_state':
            tunnels = msg.tunnels;
            renderTunnels();
            break;
        case 'tunnel_update':
            updateTunnel(msg.tunnelId, msg.status);
            break;
    }
}

async function loadTunnels() {
    try {
        const resp = await fetch(`${API_BASE}/api/tunnels`);
        tunnels = await resp.json();
        renderTunnels();
    } catch (err) {
        showToast('Error loading connections', 'error');
    }
}

function renderTunnels() {
    const container = document.getElementById('tunnels-container');
    
    if (tunnels.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <div class="empty-icon">*</div>
                <h3>No connections configured</h3>
                <p>Click "New Connection" to start your adventure</p>
            </div>
        `;
        return;
    }
    
    container.innerHTML = tunnels.map(t => createTunnelCard(t)).join('');
    
    document.querySelectorAll('.btn-start').forEach(btn => {
        btn.addEventListener('click', () => startTunnel(btn.dataset.id));
    });
    
    document.querySelectorAll('.btn-stop').forEach(btn => {
        btn.addEventListener('click', () => stopTunnel(btn.dataset.id));
    });
    
    document.querySelectorAll('.btn-copy').forEach(btn => {
        btn.addEventListener('click', () => copyUrl(btn.dataset.url));
    });
    
    document.querySelectorAll('.btn-logs').forEach(btn => {
        btn.addEventListener('click', () => showLogs(btn.dataset.id));
    });
    
    document.querySelectorAll('.btn-delete').forEach(btn => {
        btn.addEventListener('click', () => deleteTunnel(btn.dataset.id));
    });
}

function createTunnelCard(tunnel) {
    const icon = providerIcons[tunnel.provider] || 'T';
    const name = providerNames[tunnel.provider] || tunnel.provider;
    
    let statusClass = 'offline';
    let statusText = 'Offline';
    
    if (tunnel.starting) {
        statusClass = 'starting';
        statusText = 'Connecting...';
    } else if (tunnel.running) {
        statusClass = 'online';
        statusText = 'Online';
    }
    
    const urlSection = tunnel.publicUrl ? `
        <div class="tunnel-info-row">
            <span>URL:</span>
            <div class="tunnel-url" onclick="copyUrl('${tunnel.publicUrl}')" title="Click to copy">
                ${tunnel.publicUrl}
            </div>
        </div>
    ` : '';
    
    const errorSection = tunnel.error ? `
        <div class="tunnel-info-row" style="color: var(--danger);">
            <span>Error:</span>
            <span>${escapeHtml(tunnel.error)}</span>
        </div>
    ` : '';
    
    const actionButtons = tunnel.running 
        ? `<button class="btn btn-danger btn-stop" data-id="${tunnel.id}">Stop</button>`
        : `<button class="btn btn-success btn-start" data-id="${tunnel.id}">Start</button>`;
    
    return `
        <div class="tunnel-card ${statusClass}" id="tunnel-${tunnel.id}">
            <div class="tunnel-header">
                <div class="tunnel-name">${escapeHtml(tunnel.name)}</div>
                <div class="tunnel-provider">${name}</div>
            </div>
            <div class="tunnel-info">
                <div class="tunnel-info-row">
                    <span>Port:</span>
                    <span>localhost:${tunnel.port}</span>
                </div>
                <div class="tunnel-info-row">
                    <span>Status:</span>
                    <span class="status-badge status-${statusClass}">${statusText}</span>
                </div>
                ${urlSection}
                ${errorSection}
            </div>
            <div class="tunnel-actions">
                ${actionButtons}
                ${tunnel.publicUrl ? `<button class="btn btn-primary btn-copy" data-url="${tunnel.publicUrl}">Copy URL</button>` : ''}
                <button class="btn btn-secondary btn-logs" data-id="${tunnel.id}">Logs</button>
                <button class="btn btn-secondary btn-delete" data-id="${tunnel.id}">Delete</button>
            </div>
        </div>
    `;
}

function updateTunnel(id, status) {
    const tunnel = tunnels.find(t => t.id === id);
    if (tunnel) {
        tunnel.running = status.running;
        tunnel.starting = status.starting;
        tunnel.publicUrl = status.publicUrl;
        tunnel.error = status.error;
        renderTunnels();
    }
}

async function startTunnel(id) {
    try {
        const resp = await fetch(`${API_BASE}/api/tunnels/${id}/start`, { method: 'POST' });
        if (!resp.ok) {
            const err = await resp.text();
            throw new Error(err);
        }
        showToast('Starting connection...', 'info');
    } catch (err) {
        showToast('Error: ' + err.message, 'error');
    }
}

async function stopTunnel(id) {
    try {
        await fetch(`${API_BASE}/api/tunnels/${id}/stop`, { method: 'POST' });
        showToast('Connection stopped', 'info');
    } catch (err) {
        showToast('Error stopping connection', 'error');
    }
}

async function deleteTunnel(id) {
    if (!confirm('Delete this connection?')) return;
    
    try {
        await fetch(`${API_BASE}/api/tunnels/${id}`, { method: 'DELETE' });
        tunnels = tunnels.filter(t => t.id !== id);
        renderTunnels();
        showToast('Connection deleted', 'success');
    } catch (err) {
        showToast('Error deleting connection', 'error');
    }
}

async function copyUrl(url) {
    try {
        await navigator.clipboard.writeText(url);
        showToast('URL copied to clipboard!', 'success');
    } catch (err) {
        try {
            await fetch(`${API_BASE}/api/copy-url`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ url })
            });
            showToast('URL copied!', 'success');
        } catch (e) {
            showToast('Error copying URL', 'error');
        }
    }
}

function showLogs(id) {
    const tunnel = tunnels.find(t => t.id === id);
    if (!tunnel) return;
    
    document.getElementById('logs-body').textContent = 'Loading logs...';
    document.getElementById('logs-modal').classList.add('active');
    
    fetch(`${API_BASE}/api/logs/${id}`)
        .then(r => r.text())
        .then(text => {
            document.getElementById('logs-body').textContent = text || 'No logs available';
        })
        .catch(() => {
            document.getElementById('logs-body').textContent = 'Error loading logs';
        });
}

function setupEventListeners() {
    document.getElementById('btn-add').addEventListener('click', () => {
        document.getElementById('add-modal').classList.add('active');
    });
    
    document.getElementById('btn-close-modal').addEventListener('click', closeModal);
    document.getElementById('btn-cancel').addEventListener('click', closeModal);
    document.getElementById('btn-create').addEventListener('click', createTunnel);
    document.getElementById('btn-close-logs').addEventListener('click', closeLogs);
    
    document.querySelectorAll('.provider-card').forEach(card => {
        card.addEventListener('click', () => {
            document.querySelectorAll('.provider-card').forEach(c => c.classList.remove('selected'));
            card.classList.add('selected');
            selectedProvider = card.dataset.provider;
        });
    });
    
    document.getElementById('add-modal').addEventListener('click', (e) => {
        if (e.target === document.getElementById('add-modal')) closeModal();
    });
    
    document.getElementById('logs-modal').addEventListener('click', (e) => {
        if (e.target === document.getElementById('logs-modal')) closeLogs();
    });
}

function closeModal() {
    document.getElementById('add-modal').classList.remove('active');
    document.getElementById('add-form').reset();
    document.getElementById('tunnel-port').value = '30000';
}

function closeLogs() {
    document.getElementById('logs-modal').classList.remove('active');
}

async function createTunnel() {
    const name = document.getElementById('tunnel-name').value.trim();
    const port = parseInt(document.getElementById('tunnel-port').value);
    
    if (!name) {
        showToast('Enter a world name', 'error');
        return;
    }
    
    if (!port || port < 1 || port > 65535) {
        showToast('Invalid port', 'error');
        return;
    }
    
    try {
        const resp = await fetch(`${API_BASE}/api/tunnels`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, provider: selectedProvider, port })
        });
        
        if (!resp.ok) throw new Error('Error creating connection');
        
        const tunnel = await resp.json();
        tunnels.push(tunnel);
        renderTunnels();
        closeModal();
        showToast('Connection created!', 'success');
    } catch (err) {
        showToast(err.message, 'error');
    }
}

function showToast(message, type = 'info') {
    const container = document.getElementById('toast-container');
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    toast.textContent = message;
    container.appendChild(toast);
    
    setTimeout(() => {
        toast.style.animation = 'slideIn 0.3s ease-out reverse';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

document.addEventListener('DOMContentLoaded', init);
