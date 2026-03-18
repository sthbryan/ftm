<script>
  import { onMount } from "svelte";
</script>

<div class="app">
  <slot />
</div>

<style>
  :global(*) {
    box-sizing: border-box;
    margin: 0;
    padding: 0;
  }

  :global(body) {
    font-family:
      system-ui,
      -apple-system,
      sans-serif;
    background: #faf9f7;
    color: #1c1917;
    line-height: 1.6;
  }

  .app {
    max-width: 1000px;
    margin: 0 auto;
    padding: 40px 20px;
    min-height: 100dvh;
    display: flex;
  }

  @media (min-width: 768px) {
    .app {
      max-height: 100dvh;
    }
  }
</style>
