<script lang="ts">
  import { X, Trash2 } from "lucide-svelte";
  import { animate } from "motion";
  import { cn } from "$lib/utils/cn";
  import { translate } from "$lib/i18n";
  import Button from "./Button.svelte";

  let {
    show,
    name,
    onConfirm,
    onCancel,
  }: {
    show: boolean;
    name: string;
    onConfirm: () => void;
    onCancel: () => void;
  } = $props();

  let t = $derived($translate);

  let modalRef: HTMLDivElement | undefined = $state();
  let backdropRef: HTMLDivElement | undefined = $state();
  let isAnimatingOut = $state(false);

  $effect(() => {
    if (show && modalRef && backdropRef && !isAnimatingOut) {
      backdropRef.style.opacity = "0";
      animate(backdropRef, { opacity: 1 }, { duration: 0.2 });

      modalRef.style.opacity = "0";
      modalRef.style.transform = "scale(0.92)";
      requestAnimationFrame(() => {
        animate(
          modalRef!,
          { opacity: 1, scale: 1 },
          { type: "spring", stiffness: 280, damping: 24 },
        );
      });
    }
  });

  function animateOut(): Promise<void> {
    return Promise.all([
      backdropRef
        ? animate(backdropRef, { opacity: 0 }, { duration: 0.18 }).finished
        : Promise.resolve(),
      modalRef
        ? animate(
            modalRef,
            { opacity: 0, scale: 0.92 },
            { type: "spring", stiffness: 500, damping: 35 },
          ).finished
        : Promise.resolve(),
    ]).then(() => {});
  }

  function handleCancel() {
    if (isAnimatingOut) return;
    isAnimatingOut = true;
    animateOut().then(() => {
      isAnimatingOut = false;
      onCancel();
    });
  }

  function handleConfirm() {
    if (isAnimatingOut) return;
    isAnimatingOut = true;
    animateOut().then(() => {
      isAnimatingOut = false;
      onConfirm();
    });
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Escape" && !isAnimatingOut) handleCancel();
  }
</script>

<svelte:window onkeydown={show || isAnimatingOut ? handleKeydown : undefined} />

{#if show || isAnimatingOut}
  <div
    bind:this={backdropRef}
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm"
    role="presentation"
    onclick={handleCancel}
  >
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_interactive_supports_focus -->
    <div
      bind:this={modalRef}
      class={cn("w-[90%] max-w-md rounded-2xl bg-card shadow-2xl")}
      onclick={(e) => e.stopPropagation()}
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
    >
      <div class="flex items-center justify-between px-6 pt-6 max-md:px-5">
        <h3
          id="modal-title"
          class="m-0 text-lg font-semibold text-text-heading flex items-center gap-2"
        >
          {t("delete_connection")}
        </h3>
        <button
          class={cn(
            "w-8 h-8 flex items-center justify-center rounded-xl cursor-pointer",
            "bg-transparent border-none text-text-muted transition-all duration-150 hover:rotate-90 hover:bg-hover"
          )}
          onclick={handleCancel}
          aria-label={t("close")}
        >
          <X size={18} />
        </button>
      </div>
      <div class="px-6 py-5 max-md:px-5">
        <p class="m-0 mb-2 text-text">
          {t("confirm_delete", { name })}
        </p>
        <p class="m-0 text-sm text-text-muted">{t("cannot_undo")}</p>
      </div>
      <div
        class="flex gap-3 px-6 pb-6 max-md:flex-col-reverse max-md:gap-2 max-md:px-5 max-md:pb-5"
      >
        <Button
          variant="default"
          size="lg"
          onclick={handleCancel}
          class="flex-1 max-md:w-full">{t("cancel")}</Button
        >
        <Button
          variant="error"
          size="lg"
          onclick={handleConfirm}
          class="flex-1 max-md:w-full">{t("delete")}</Button
        >
      </div>
    </div>
  </div>
{/if}
