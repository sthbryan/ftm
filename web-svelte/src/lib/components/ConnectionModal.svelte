<script lang="ts">
  import ConnectionForm from "./ConnectionForm.svelte";

  let {
    show,
    tunnelId = null,
    onClose,
  }: {
    show: boolean;
    tunnelId?: string | null;
    onClose: () => void;
  } = $props();

  let dialogEl: HTMLDivElement | undefined = $state();

  $effect(() => {
    if (!show || !dialogEl) return;
    const previous = document.activeElement as HTMLElement | null;
    dialogEl.focus();
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = "";
      previous?.focus?.();
    };
  });

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Escape") onClose();
  }
</script>

<svelte:window onkeydown={show ? handleKeydown : undefined} />

{#if show}
  <div
    class="sph-dialog__backdrop"
    role="presentation"
    onclick={onClose}
  >
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_interactive_supports_focus -->
    <div
      bind:this={dialogEl}
      role="dialog"
      aria-modal="true"
      tabindex="-1"
      class="sph-dialog"
      onclick={(e) => e.stopPropagation()}
    >
      {#if tunnelId}
        <ConnectionForm mode="edit" {tunnelId} onCancel={onClose} onSaved={onClose} />
      {:else}
        <ConnectionForm mode="create" onCancel={onClose} onSaved={onClose} />
      {/if}
    </div>
  </div>
{/if}
