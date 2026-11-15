<script lang="ts">
  import { Trash2 } from "lucide-svelte";
  import { t } from "$lib/stores/i18n.svelte";
  import Button from "./Button.svelte";

  let {
    show,
    name,
    title,
    body,
    onConfirm,
    onCancel,
  }: {
    show: boolean;
    name?: string;
    title?: string;
    body?: string;
    onConfirm: () => void;
    onCancel: () => void;
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
    if (e.key === "Escape") onCancel();
  }
</script>

<svelte:window onkeydown={show ? handleKeydown : undefined} />

{#if show}
  <div
    class="sph-dialog__backdrop"
    role="presentation"
    onclick={onCancel}
  >
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_interactive_supports_focus -->
    <div
      bind:this={dialogEl}
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
      tabindex="-1"
      class="sph-dialog"
      onclick={(e) => e.stopPropagation()}
    >
      <div class="sph-dialog__header">
        <div class="sph-dialog__heading">
          <h2
            id="modal-title"
            class="sph-dialog__title flex items-center gap-2"
          >
            <Trash2 size={18} class="text-text-error" />
            {title ?? t("confirm_delete_title")}
          </h2>
        </div>
        <button
          type="button"
          class="sph-dialog__close"
          aria-label={t("close")}
          onclick={onCancel}
        >
          ×
        </button>
      </div>

      <div class="sph-dialog__content">
        <p class="m-0 text-sm leading-relaxed text-text-muted">
          {body ?? t("confirm_delete_body", { 0: name ?? "" })}
        </p>
      </div>

      <div class="sph-dialog__footer">
        <Button variant="default" onclick={onCancel}>{t("cancel")}</Button>
        <Button variant="error" onclick={onConfirm}>{t("delete")}</Button>
      </div>
    </div>
  </div>
{/if}
