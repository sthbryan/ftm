<script>
    import { useNotifications } from '$lib/stores/notification.svelte.js';
    
    const notifications = useNotifications();
    
    let show = $derived(notifications.permission === 'default');
    
    async function request() {
        await notifications.requestPermission();
    }
    
    function later() {
        notifications.disable();
    }
</script>

{#if show}
<div class="notification-prompt">
    <div class="content">
        <h3>Enable Notifications</h3>
        <p>Get notified when tunnels go online, offline, or are about to expire.</p>
        <div class="actions">
            <button class="primary" onclick={request}>Enable</button>
            <button class="secondary" onclick={later}>Not Now</button>
        </div>
    </div>
</div>
{/if}

<style>
    .notification-prompt {
        position: fixed;
        bottom: 1rem;
        right: 1rem;
        background: var(--surface-2, #2a2a2a);
        border: 1px solid var(--border, #404040);
        border-radius: 0.75rem;
        padding: 1rem;
        max-width: 320px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
        z-index: 100;
    }
    
    h3 {
        margin: 0 0 0.5rem;
        font-size: 1rem;
        font-weight: 600;
    }
    
    p {
        margin: 0 0 1rem;
        font-size: 0.875rem;
        opacity: 0.8;
    }
    
    .actions {
        display: flex;
        gap: 0.5rem;
    }
    
    button {
        flex: 1;
        padding: 0.5rem 1rem;
        border: none;
        border-radius: 0.5rem;
        font-size: 0.875rem;
        cursor: pointer;
        transition: opacity 0.2s;
    }
    
    button:hover {
        opacity: 0.9;
    }
    
    .primary {
        background: var(--primary, #6366f1);
        color: white;
    }
    
    .secondary {
        background: var(--surface-3, #3a3a3a);
        color: var(--text, #fff);
    }
</style>
