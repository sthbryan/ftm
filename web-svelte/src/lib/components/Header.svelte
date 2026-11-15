<script lang="ts">
  import { page } from "$app/stores";
  import { goto } from "$app/navigation";
  import { t } from "$lib/stores/i18n.svelte";
  import { Settings, ChevronLeft } from "lucide-svelte";
  import Mark from "./Mark.svelte";
  import Button from "./Button.svelte";
  import IconButton from "./IconButton.svelte";

  const isHome = $derived($page.url.pathname === "/");
  const isSettings = $derived($page.url.pathname === "/settings");

  function handleBack() {
    if (history.length > 1) {
      history.back();
    } else {
      goto("/");
    }
  }
</script>

<header
  class="z-10 mb-3 flex shrink-0 items-center justify-between gap-3 border-b border-border pb-2.5"
>
  <a
    href="/"
    class="flex min-w-0 items-center gap-2.5 rounded-control py-0.5 transition-opacity hover:opacity-80"
    aria-label={t("go_home")}
  >
    <Mark class="size-8 shrink-0" />
    <div class="min-w-0">
      <h1
        class="m-0 truncate text-lg leading-tight font-semibold tracking-tight text-text-heading"
      >
        {t("app_name")}
      </h1>
      <p class="m-0 truncate text-2xs text-text-muted">{t("app_tagline")}</p>
    </div>
  </a>

  {#if isSettings}
    <Button variant="ghost" icon={ChevronLeft} onclick={handleBack}>
      {t("go_home")}
    </Button>
  {:else if isHome}
    <IconButton
      icon={Settings}
      label={t("settings")}
      size="md"
      onclick={() => goto("/settings")}
    />
  {/if}
</header>
