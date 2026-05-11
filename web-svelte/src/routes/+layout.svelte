<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import Toasts from "$lib/components/Toasts.svelte";
  import { subscribeWsMessages } from "$lib/api/ws";
  import { i18n } from "$lib/i18n";

  import "../styles/app.css";

  let unsubscribeWs: (() => void) | null = null;

  onMount(async () => {
    unsubscribeWs = subscribeWsMessages(() => {});
    await i18n.init();
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

<Toasts />
