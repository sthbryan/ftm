<script lang="ts">
  import { cn } from "$lib/utils/cn";
  import { ChevronDown } from "lucide-svelte";
  import { animate, spring } from "motion";
  import type { DropdownOption } from "$lib/types";
  import type { Snippet } from "svelte";

  interface DropdownProps {
    options?: DropdownOption[];
    onSelect?: (option: DropdownOption) => void;
    align?: "left" | "right" | "top" | "center";
    ariaLabel?: string;
    label?: string;
    class?: string;
    id?: string;
    children?: Snippet;
  };

  const POSITION_MAP: Record<NonNullable<DropdownProps["align"]>, string> = {
    left: "left-auto right-0",
    right: "right-auto left-0",
    top: "bottom-full mb-1.5 left-0 right-auto",
    center: "left-1/2 -translate-x-1/2",
  };

  let {
    options = [],
    onSelect,
    align = "left",
    ariaLabel = "Options",
    label = "Options",
    class: className = "",
    id = "",
    children,
  }: DropdownProps = $props();

  let isOpen = $state(false);
  let isAnimating = $state(false);
  let menuEl: HTMLDivElement | undefined = $state();

  const isVisible = $derived(isOpen || isAnimating);
  const menuPosition = $derived.by(() => {
    const vert = align === "top" ? "" : "top-full mt-1.5";
    return `${POSITION_MAP[align]} ${vert}`;
  });

  function open() {
    if (isOpen) return;
    isOpen = true;
    isAnimating = true;
    requestAnimationFrame(() => {
      if (!menuEl) return;
      // @ts-ignore
      animate(menuEl, { opacity: 1, scale: 1, y: 0 }, spring()).finished.then(
        () => {
          isAnimating = false;
        },
      );
    });
  }

  function close() {
    if (!isOpen || !menuEl) {
      isOpen = false;
      isAnimating = false;
      return;
    }
    isOpen = false;
    isAnimating = true;
    // @ts-ignore
    animate(menuEl, { opacity: 0, scale: 1, y: -4 }, spring()).finished.then(
      () => {
        isAnimating = false;
      },
    );
  }

  function toggle() {
    isOpen ? close() : open();
  }

  function handleOutsideClick(e: MouseEvent) {
    if (!(e.target as HTMLElement).closest(".dropdown-container")) close();
  }

  $effect(() => {
    if (!isOpen) return;
    document.addEventListener("click", handleOutsideClick);
    document.addEventListener("keydown", handleKeydown);
    return () => {
      document.removeEventListener("click", handleOutsideClick);
      document.removeEventListener("keydown", handleKeydown);
    };
  });

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Escape") close();
  }
</script>

<div class="dropdown-container h-fit relative flex {className}">
  <button
    type="button"
    {id}
    onclick={() => toggle()}
    aria-label={ariaLabel}
    aria-expanded={isOpen}
    aria-haspopup="true"
    class={cn(
      "flex items-center gap-1.5 px-3 py-2 text-xs h-9 rounded-lg border min-h-9 cursor-pointer",
      "bg-card border-border text-text hover:bg-hover flex-1",
    )}
  >
    {#if children}
      {@render children()}
    {:else}
      <span class="flex-1 text-left text-sm">{label}</span>
      <ChevronDown
        size={16}
        class={cn("transition-transform duration-200", isOpen && "rotate-180")}
      />
    {/if}
  </button>

  {#if isVisible}
    <div
      bind:this={menuEl}
      id={id ? `${id}-menu` : undefined}
      role="menu"
      aria-orientation="vertical"
      style="opacity: 0; scale: 0.95; transform: translateY(-4px);"
      class={cn(
        "absolute min-w-[150px] max-h-[300px] rounded-lg border p-1 z-50 overflow-y-auto cursor-default",
        menuPosition,
        "bg-card border-border",
      )}
    >
      {#each options as option}
        {#if option.label === "separator"}
          <div class="h-px my-1 mx-2 bg-border"></div>
        {:else}
          <button
            type="button"
            role="menuitem"
            disabled={option.disabled}
            onclick={() => {
              close();
              onSelect?.(option);
            }}
            class={cn(
              "flex items-center gap-2 w-full px-3 py-2 text-xs rounded-lg text-left cursor-pointer",
              "text-text bg-transparent border-none hover:bg-hover",
              "disabled:opacity-50 disabled:cursor-not-allowed",
              option.danger && "text-red-500 hover:bg-red-500/10",
            )}
          >
            {#if option.icon}
              {@const IconComponent =
                option.icon as import("svelte").Component<{
                  size?: number;
                }>}
              <IconComponent size={16} />
            {/if}
            <span>{option.label}</span>
          </button>
        {/if}
      {/each}
    </div>
  {/if}
</div>
