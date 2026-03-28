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
            <button class="btn btn-primary" onclick={request}>Enable</button>
            <button class="btn btn-secondary" onclick={later}>Not Now</button>
        </div>
    </div>
</div>
{/if}

<style>
    .notification-prompt {
        position: fixed;
        bottom: 1rem;
        right: 1rem;
        background: var(--card-bg);
        border: 1px solid var(--border-color);
        border-radius: 12px;
        padding: 1.25rem;
        max-width: 320px;
        box-shadow: 0 10px 30px rgba(0, 0, 0, 0.15);
        z-index: 100;
        animation: slideIn 0.3s cubic-bezier(0.16, 1, 0.3, 1);
    }
    
    @keyframes slideIn {
        from {
            opacity: 0;
            transform: translateY(20px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }
    
    .content {
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
    }
    
    h3 {
        margin: 0;
        font-family: 'Crimson Pro', Georgia, serif;
        font-size: 1.1rem;
        font-weight: 600;
        color: var(--text-heading);
    }
    
    p {
        margin: 0;
        font-size: 0.875rem;
        color: var(--text-muted);
        line-height: 1.4;
    }
    
    .actions {
        display: flex;
        gap: 0.5rem;
        margin-top: 0.25rem;
    }
    
    .actions :global(.btn) {
        flex: 1;
        padding: 8px 12px;
    }
    
    @media (prefers-reduced-motion: reduce) {
        .notification-prompt {
            animation: none;
        }
    }
</style>
