<script lang="ts">
  import { cn } from "$lib/utils/cn";
  import type { Snippet, ComponentType, SvelteComponentTyped } from "svelte";
  import type {  IconProps } from "lucide-svelte";

  type ButtonVariant = "default" | "primary" | "success" | "error" | "ghost";
  type ButtonSize = "sm" | "md" | "lg";

  interface ButtonProps {
    variant?: ButtonVariant;
    size?: ButtonSize;
    disabled?: boolean;
    type?: "button" | "submit" | "reset";
    class?: string;
    onclick?: (e: MouseEvent) => void;
    children?: Snippet;
    icon?: ComponentType<SvelteComponentTyped<IconProps>>;
    iconPosition?: "left" | "right";
  };

  const VARIANT_CLASSES: Record<ButtonVariant, { base: string; hover: string }> = {
    default: {
      base: "bg-secondary-btn text-secondary-btn-text border-border",
      hover: "hover:bg-hover hover:border-border-light",
    },
    primary: {
      base: "bg-btn-primary text-btn-text border-btn-primary",
      hover: "hover:bg-btn-primary-hover",
    },
    success: {
      base: "bg-btn-start text-btn-text border-btn-start",
      hover: "hover:bg-btn-start-hover",
    },
    error: {
      base: "bg-btn-stop text-btn-text border-btn-stop",
      hover: "hover:bg-btn-stop-hover",
    },
    ghost: {
      base: "bg-transparent text-text border-transparent",
      hover: "hover:bg-hover",
    },
  };

  const SIZE_CLASSES: Record<ButtonSize, string> = {
    sm: "h-8 px-2.5 text-xs gap-1.5",
    md: "h-9 px-3.5 text-sm gap-2",
    lg: "h-11 px-5 text-sm gap-2",
  };

  let {
    variant = "default",
    size = "md",
    disabled = false,
    type = "button",
    class: className = "",
    onclick,
    children,
    icon: Icon,
    iconPosition = "left",
  }: ButtonProps = $props();

  const variantClasses = $derived(VARIANT_CLASSES[variant]);
  const sizeClasses = $derived(SIZE_CLASSES[size]);
  const hasIcon = $derived(!!Icon);
  const showIconLeft = $derived(hasIcon && iconPosition === "left");
  const showIconRight = $derived(hasIcon && iconPosition === "right");
  const showChildren = $derived(!!children);
  const iconSize = $derived(size === "sm" ? 12 : 14);

</script>

<button
  {type}
  {disabled}
  {onclick}
  class={cn(
    "inline-flex items-center justify-center rounded-lg border font-medium cursor-pointer",
    "transition-colors duration-150",
    "disabled:opacity-50 disabled:cursor-not-allowed",
    variantClasses.base,
    variantClasses.hover,
    sizeClasses,
    className,
  )}
>
  {#if showIconLeft && Icon}
    <Icon size={iconSize} />
  {/if}
  {#if showChildren && children}
    {@render children()}
  {/if}
  {#if showIconRight && Icon}
    <Icon size={iconSize} />
  {/if}
</button>
