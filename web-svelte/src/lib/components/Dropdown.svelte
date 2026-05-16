<script lang="ts">
  import { cn } from "$lib/utils/cn";
  import { ChevronDown } from "lucide-svelte";
  import { animate, spring } from "motion";
  import { translate } from "$lib/i18n";
  import type { DropdownOption } from "$lib/types";
  import type { Snippet } from "svelte";

  interface DropdownProps {
    options?: DropdownOption[];
    onSelect?: (option: DropdownOption) => void;
    align?: "left" | "right" | "top-left" | "top-right";
    ariaLabel?: string;
    label?: string;
    class?: string;
    id?: string;
    children?: Snippet;
  }

  const POSITION_MAP: Record<NonNullable<DropdownProps["align"]>, string> = {
    left: "left-auto right-0",
    right: "right-auto left-0",
    "top-left": "bottom-full mb-1.5 left-auto right-0",
    "top-right": "bottom-full mb-1.5 right-auto left-0"
  };

  let t = $derived($translate);

  let {
    options = [],
    onSelect,
    align = "left",
    ariaLabel = t("options"),
    label = t("options"),
    class: className = "",
    id = "",
    children,
  }: DropdownProps = $props();

  let isOpen = $state(false);
  let menuEl: HTMLDivElement | undefined = $state();

  const menuPosition = $derived.by(() => {
    const vert = align.startsWith("top") ? "" : "top-full mt-1.5";
    return `${POSITION_MAP[align]} ${vert}`;
  });

  function open() {
    if (isOpen || !menuEl) return;
    isOpen = true;
    animate(menuEl, { opacity: 1, scale: 1, y: 0 }, { type: "spring" });
  }

  function close() {
    if (!isOpen || !menuEl) return;
    isOpen = false;
    animate(menuEl, { opacity: 0, scale: 1, y: -4 }, { type: "spring" });
  }

  function toggle() {
    isOpen ? close() : open();
  }

  function handleOutsideClick(e: MouseEvent) {
    if (!isOpen) return;
    const target = e.target as HTMLElement;
    if (
      target.closest(".dropdown-trigger") ||
      target.closest(".dropdown-menu")
    ) return;
    close();
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Escape") close();
  }

  $effect(() => {
    if (isOpen) {
      document.addEventListener("click", handleOutsideClick, true);
      document.addEventListener("keydown", handleKeydown);
    } else {
      document.removeEventListener("click", handleOutsideClick, true);
      document.removeEventListener("keydown", handleKeydown);
    }

    return () => {
      document.removeEventListener("click", handleOutsideClick, true);
      document.removeEventListener("keydown", handleKeydown);
    };
  });
</script>

<div class={cn("dropdown-container h-fit relative flex", className)}>
  <button
    type="button"
    {id}
    onclick={(e) => { e.stopPropagation(); toggle(); }}
    aria-label={ariaLabel}
    aria-expanded={isOpen}
    aria-haspopup="true"
    class={cn(
      "dropdown-trigger flex items-center gap-1.5 px-3 py-2 text-xs h-9 rounded-xl border min-h-9 cursor-pointer",
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

  <!-- Always in DOM, just hidden via opacity when closed -->
  <div
    bind:this={menuEl}
    id={id ? `${id}-menu` : undefined}
    role="menu"
    aria-orientation="vertical"
    style="opacity: 0; scale: 0.95; transform: translateY(-4px);"
    class={cn(
      "dropdown-menu absolute min-w-[150px] max-h-[300px] rounded-2xl border p-1 z-[9999] overflow-y-auto cursor-default",
      menuPosition,
      "bg-card border-border pointer-events-none",
      isOpen && "pointer-events-auto",
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
            "flex items-center gap-2 w-full px-3 py-2 text-xs rounded-xl text-left cursor-pointer",
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
</div>