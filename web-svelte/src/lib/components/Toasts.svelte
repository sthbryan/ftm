<script lang="ts">
  import { animate, spring } from "motion";
  import { toast, type ToastType } from "$lib/stores/toast.svelte";
  import { cn } from "$lib/utils/cn";
  import { translate } from "$lib/i18n";

  let t = $derived($translate);

  const MAX_TOASTS = 5;
  const STACK_Y = 8;
  const STACK_SCALE = 0.04;

  const SPRING_ENTER = { type: spring, stiffness: 220, damping: 24 } as const;
  const SPRING_STACK = { type: spring, stiffness: 600, damping: 40 } as const;

  const gradientMap: Record<ToastType, string> = {
    success: "bg-gradient-to-br from-btn-start to-btn-start-hover",
    error: "bg-gradient-to-br from-btn-stop to-btn-stop-hover",
    info: "bg-gradient-to-br from-btn-primary to-btn-primary-hover",
    warning: "bg-gradient-to-br from-btn-primary to-btn-primary-hover",
    alert: "bg-gradient-to-br from-btn-stop to-btn-stop-hover",
  };

  let stackRefs = $state<Map<number, HTMLDivElement>>(new Map());
  let contentRefs = $state<Map<number, HTMLDivElement>>(new Map());
  let closingIds = $state<Set<number>>(new Set());

  function getStackRef(id: number) {
    return stackRefs.get(id);
  }

  function getContentRef(id: number) {
    return contentRefs.get(id);
  }

  async function animateOut(id: number) {
    const el = getContentRef(id);
    if (el) await animate(el, { opacity: 0, x: 120 }, SPRING_ENTER).finished;
  }

  async function close(id: number) {
    if (closingIds.has(id)) return;
    closingIds.add(id);
    closingIds = new Set(closingIds);
    await animateOut(id);
    toast.remove(id);
    closingIds.delete(id);
    closingIds = new Set(closingIds);
  }

  function setStackRef(node: HTMLDivElement, id: number) {
    stackRefs.set(id, node);
    stackRefs = new Map(stackRefs);

    return {
      destroy() {
        stackRefs.delete(id);
        stackRefs = new Map(stackRefs);
      },
    };
  }

  function setContentRef(node: HTMLDivElement, id: number) {
    contentRefs.set(id, node);
    contentRefs = new Map(contentRefs);

    node.style.opacity = "0";
    node.style.transform = "translateX(96px)";
    requestAnimationFrame(() => {
      animate(node, { x: 0, opacity: 1 }, SPRING_ENTER);
    });

    const duration = toast.toasts.find((t) => t.id === id)?.duration ?? 3000;
    const timer = setTimeout(() => close(id), duration);

    return {
      destroy() {
        clearTimeout(timer);
        contentRefs.delete(id);
        contentRefs = new Map(contentRefs);
      },
    };
  }

  $effect(() => {
    toast.toasts.forEach((t, i) => {
      const el = getStackRef(t.id);
      if (!el) return;
      animate(
        el,
        {
          y: i * STACK_Y,
          scale: 1 - i * STACK_SCALE,
          opacity: i === 0 ? 1 : Math.max(0.4, 1 - i * 0.2),
        },
        SPRING_STACK
      );
    });
  });

  $effect(() => {
    if (toast.toasts.length > MAX_TOASTS) {
      const overflow = toast.toasts.slice(MAX_TOASTS);
      overflow.forEach((item) => {
        requestAnimationFrame(() => close(item.id));
      });
    }
  });
</script>

<div class="fixed top-5 right-5 z-[1000] max-sm:top-2.5 max-sm:right-2.5 pointer-events-none">
  <div class="relative w-[320px] h-14 pointer-events-none">
    {#each toast.toasts as t, i (t.id)}
      <div
        use:setStackRef={t.id}
        style="position:absolute;top:0;right:0;width:100%;z-index:{MAX_TOASTS - i};transform-origin:top center;"
      >
        <div
          use:setContentRef={t.id}
          class={cn(
            "flex items-center gap-3 px-[18px] py-3.5 rounded-[10px] shadow-lg pointer-events-auto",
            "text-heading border border-heading/20 cursor-default",
            gradientMap[t.type] ?? gradientMap.info
          )}
          role="alert"
        >
          <span class="flex-1 text-sm font-medium">{t.message}</span>
          <button
            onclick={() => close(t.id)}
            aria-label={t('close_notification')}
            class="w-7 h-7 rounded-md cursor-pointer text-lg flex items-center justify-center transition-colors duration-150 bg-heading/20 text-heading"
          >
            &times;
          </button>
        </div>
      </div>
    {/each}
  </div>
</div>