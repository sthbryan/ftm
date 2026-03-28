<script>
  import ThemeSwitcher from './ThemeSwitcher.svelte';
</script>

<header class="app-header">
  <div class="brand">
    <img src="/favicon.png" alt="Foundry Tunnel Manager" class="d20-badge" />
    <div class="brand-text">
      <h1>Foundry Tunnel Manager</h1>
      <p class="tagline">Share your world with players everywhere</p>
    </div>
  </div>
  <ThemeSwitcher />
</header>

<style>
  .app-header {
    display: flex;
    flex-direction: column;
    margin-bottom: 24px;
    padding-bottom: 20px;
    border-bottom: 1px solid var(--border-color);
    opacity: 0;
    transform: translateY(-20px);
    animation: headerIn 0.5s cubic-bezier(0.16, 1, 0.3, 1) forwards;
    flex-shrink: 0;
    z-index: 1;
  }

  @media (min-width: 768px) {
    .app-header {
      flex-direction: row;
      justify-content: space-between;
      align-items: center;
    }
  }

  @keyframes headerIn {
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .d20-badge {
    width: 48px;
    height: 48px;
    border-radius: 12px;
    object-fit: cover;
  }

  .brand-text h1 {
    font-family: 'Crimson Pro', Georgia, serif;
    font-size: 32px;
    line-height: normal;
    font-weight: 700;
    color: var(--text-heading);
    margin: 0 0 4px 0;
    letter-spacing: -0.01em;
  }

  .tagline {
    font-size: 14px;
    color: var(--text-muted);
    margin: 0;
    font-weight: 500;
  }

  @media (max-width: 767px) {
    .brand-text h1 {
      font-size: 24px;
    }
    
    .tagline {
      font-size: 13px;
    }
    
    .d20-badge {
      width: 40px;
      height: 40px;
    }
    
    .app-header {
      margin-bottom: 16px;
      padding-bottom: 16px;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .app-header {
      animation: none;
      opacity: 1;
      transform: none;
    }
  }
</style>
