<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import { subscribeWsMessages } from '$lib/api/ws';

  let unsubscribeWs: (() => void) | null = null;

  onMount(() => {
    unsubscribeWs = subscribeWsMessages(() => {});
  });

  onDestroy(() => {
    if (unsubscribeWs) {
      unsubscribeWs();
      unsubscribeWs = null;
    }
  });
</script>

<div class="max-w-[1000px] mx-auto p-10 min-h-dvh flex max-md:max-h-dvh">
  <slot />
</div>
