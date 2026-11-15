<script lang="ts">
  import { cn } from "$lib/utils/cn";
  import type { Snippet, ComponentType, SvelteComponentTyped } from "svelte";
  import type { IconProps } from "lucide-svelte";

  type ButtonVariant = "default" | "primary" | "success" | "error" | "ghost";
  type ButtonSize = "sm" | "md" | "lg";

  interface ButtonProps {
    variant?: ButtonVariant;
    size?: ButtonSize;
    disabled?: boolean;
    type?: "button" | "submit" | "reset";
    class?: string;
    ariaLabel?: string;
    onclick?: (e: MouseEvent) => void;
    children?: Snippet;
    icon?: ComponentType<SvelteComponentTyped<IconProps>>;
    iconPosition?: "left" | "right";
  };

  // ftm's vocabulary on top of the Sephiro contract: same class, data-variant.
  const VARIANT: Record<ButtonVariant, string> = {
    default: "secondary",
    primary: "primary",
    success: "success",
    error: "danger",
    ghost: "quiet",
  };

  let {
    variant = "default",
    size = "sm",
    disabled = false,
    type = "button",
    class: className = "",
    ariaLabel,
    onclick,
    children,
    icon: Icon,
    iconPosition = "left",
  }: ButtonProps = $props();

  const hasIcon = $derived(!!Icon);
  const showIconLeft = $derived(hasIcon && iconPosition === "left");
  const showIconRight = $derived(hasIcon && iconPosition === "right");
  const showChildren = $derived(!!children);
  const iconSize = $derived(size === "sm" ? 14 : 15);
</script>

<button
  {type}
  {disabled}
  {onclick}
  aria-label={ariaLabel}
  data-variant={VARIANT[variant]}
  data-size={size}
  class={cn("sph-button", className)}
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
