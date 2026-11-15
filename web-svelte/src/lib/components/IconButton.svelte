<script lang="ts">
  import { cn } from "$lib/utils/cn";
  import type { ComponentType, SvelteComponentTyped } from "svelte";
  import type { IconProps } from "lucide-svelte";

  type IconButtonVariant = "ghost" | "outline" | "danger";
  type IconButtonSize = "sm" | "md";

  interface IconButtonProps {
    icon: ComponentType<SvelteComponentTyped<IconProps>>;
    label: string;
    variant?: IconButtonVariant;
    size?: IconButtonSize;
    disabled?: boolean;
    type?: "button" | "submit" | "reset";
    class?: string;
    onclick?: (e: MouseEvent) => void;
  }

  // ghost is Sephiro's default icon button; the other two are data-variant values.
  const VARIANT: Record<IconButtonVariant, string | undefined> = {
    ghost: undefined,
    outline: "outline",
    danger: "danger",
  };

  let {
    icon: Icon,
    label,
    variant = "ghost",
    size = "md",
    disabled = false,
    type = "button",
    class: className = "",
    onclick,
  }: IconButtonProps = $props();

  const iconSize = $derived(size === "sm" ? 14 : 16);
</script>

<button
  {type}
  {disabled}
  {onclick}
  aria-label={label}
  title={label}
  data-variant={VARIANT[variant]}
  data-size={size}
  class={cn("sph-icon-button", className)}
>
  <Icon size={iconSize} />
</button>
